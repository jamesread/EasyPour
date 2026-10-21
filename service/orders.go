package main

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/jamesread/httpauthshim/authpublic"
	"github.com/sirupsen/logrus"

	easypourv1 "easypour/service/gen/easypour/v1"
	"easypour/service/internal/metrics"
	"easypour/service/internal/order"
)

var allowedStatuses = map[string]bool{"pending": true, "preparing": true, "delivered": true}

type statusTransition struct {
	from, to string
}

// orderStatusTransitions lists every permitted status change; the value is true when only admins may make it.
var orderStatusTransitions = map[statusTransition]bool{
	{"pending", "preparing"}:   true,
	{"pending", "delivered"}:   false,
	{"preparing", "delivered"}: false,
}

// validOrderStatusTransition returns true if the status change is allowed; pending→preparing is admin-only.
func validOrderStatusTransition(from, to string, isAdmin bool) bool {
	adminOnly, ok := orderStatusTransitions[statusTransition{from, to}]
	return ok && (isAdmin || !adminOnly)
}

// OrderDrink places an order for a drink and persists it with status "pending".
func (s *EasyPourServer) OrderDrink(
	ctx context.Context,
	req *connect.Request[easypourv1.OrderRequest],
) (*connect.Response[easypourv1.OrderResponse], error) {
	o := newPendingOrder(req.Msg, s.getUsernameFromContext(ctx))
	if err := s.orderStore.Create(ctx, o); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("create order: %w", err))
	}
	metrics.RecordOrderCreated()
	logrus.Infof("Order received: %s - item %s (Sugar: %v, Milk: %v)", o.ID, o.MenuItemID, o.AddSugar, o.AddMilk)
	s.notifyOrderPlaced(ctx, o)
	return connect.NewResponse(orderResponse(o)), nil
}

func newPendingOrder(msg *easypourv1.OrderRequest, username string) *order.Order {
	orderID := uuid.New().String()
	groupID := msg.GetGroupId()
	if groupID == "" {
		groupID = orderID
	}
	return &order.Order{
		ID:          orderID,
		MenuItemID:  msg.GetMenuItemId(),
		Username:    username,
		AddSugar:    msg.GetAddSugar(),
		AddMilk:     msg.GetAddMilk(),
		SugarAmount: msg.GetSugarAmount(),
		MilkAmount:  msg.GetMilkAmount(),
		Status:      "pending",
		GroupID:     groupID,
	}
}

func orderResponse(o *order.Order) *easypourv1.OrderResponse {
	return &easypourv1.OrderResponse{
		OrderId:     o.ID,
		MenuItemId:  o.MenuItemID,
		AddSugar:    o.AddSugar,
		AddMilk:     o.AddMilk,
		SugarAmount: o.SugarAmount,
		MilkAmount:  o.MilkAmount,
		Status:      o.Status,
		CreatedAt:   o.CreatedAt,
		GroupId:     o.GroupID,
	}
}

// GetOrder returns a single order by id. Caller must own the order or be admin.
func (s *EasyPourServer) GetOrder(
	ctx context.Context,
	req *connect.Request[easypourv1.GetOrderRequest],
) (*connect.Response[easypourv1.GetOrderResponse], error) {
	user, err := s.requireAuth(ctx)
	if err != nil {
		return nil, err
	}
	o, err := s.findVisibleOrder(ctx, user, req.Msg.GetOrderId())
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&easypourv1.GetOrderResponse{Order: o.ToProto()}), nil
}

func (s *EasyPourServer) findVisibleOrder(ctx context.Context, user *authpublic.AuthenticatedUser, orderID string) (*order.Order, error) {
	if orderID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("order_id required"))
	}
	o, err := s.findOrder(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if !canViewOrder(user, o) {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("not your order"))
	}
	return o, nil
}

func canViewOrder(user *authpublic.AuthenticatedUser, o *order.Order) bool {
	return userInAdminGroup(user) || o.Username == user.Username
}

func (s *EasyPourServer) findOrder(ctx context.Context, orderID string) (*order.Order, error) {
	o, err := s.orderStore.Get(ctx, orderID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("get order: %w", err))
	}
	if o == nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("order not found"))
	}
	return o, nil
}

// ListOrders returns orders for the caller (own) or all orders if admin.
func (s *EasyPourServer) ListOrders(
	ctx context.Context,
	_ *connect.Request[easypourv1.ListOrdersRequest],
) (*connect.Response[easypourv1.ListOrdersResponse], error) {
	user, err := s.requireAuth(ctx)
	if err != nil {
		return nil, err
	}
	list, err := s.orderStore.List(ctx, user.Username, userInAdminGroup(user))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("list orders: %w", err))
	}
	orders := make([]*easypourv1.Order, 0, len(list))
	for _, o := range list {
		orders = append(orders, o.ToProto())
	}
	return connect.NewResponse(&easypourv1.ListOrdersResponse{Orders: orders}), nil
}

// UpdateOrderStatus updates an order's status and broadcasts the change via SSE.
func (s *EasyPourServer) UpdateOrderStatus(
	ctx context.Context,
	req *connect.Request[easypourv1.UpdateOrderStatusRequest],
) (*connect.Response[easypourv1.UpdateOrderStatusResponse], error) {
	if _, err := s.requireAuth(ctx); err != nil {
		return nil, err
	}
	orderID, status, err := parseOrderStatusUpdate(req.Msg)
	if err != nil {
		return nil, err
	}
	updated, err := s.applyOrderStatus(ctx, orderID, status)
	if err != nil {
		return nil, err
	}
	metrics.RecordOrderStatusUpdate(updated.Status)
	s.sseBroadcast.BroadcastOrderEvent(orderEvent{Type: "status_update", OrderID: updated.ID, Status: updated.Status})
	return connect.NewResponse(&easypourv1.UpdateOrderStatusResponse{Order: updated.ToProto()}), nil
}

func parseOrderStatusUpdate(msg *easypourv1.UpdateOrderStatusRequest) (orderID, status string, err error) {
	orderID = msg.GetOrderId()
	status = strings.TrimSpace(strings.ToLower(msg.GetStatus()))
	if orderID == "" || status == "" {
		return "", "", connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("order_id and status required"))
	}
	if !allowedStatuses[status] {
		return "", "", connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("status must be pending, preparing, or delivered"))
	}
	return orderID, status, nil
}

func (s *EasyPourServer) applyOrderStatus(ctx context.Context, orderID, status string) (*order.Order, error) {
	o, err := s.findOrder(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if !s.canUpdateOrderStatus(ctx, o, status) {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("cannot update order status"))
	}
	return s.storeOrderStatus(ctx, orderID, status)
}

// canUpdateOrderStatus returns true if the user can set the order to the new status. Admins may make any
// permitted transition; owners only the non-admin ones (which all end in "delivered").
func (s *EasyPourServer) canUpdateOrderStatus(ctx context.Context, o *order.Order, newStatus string) bool {
	user, err := s.requireAuth(ctx)
	if err != nil {
		return false
	}
	if userInAdminGroup(user) {
		return validOrderStatusTransition(o.Status, newStatus, true)
	}
	return o.Username == user.Username && validOrderStatusTransition(o.Status, newStatus, false)
}

func (s *EasyPourServer) storeOrderStatus(ctx context.Context, orderID, status string) (*order.Order, error) {
	updated, err := s.orderStore.UpdateStatus(ctx, orderID, status)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("update status: %w", err))
	}
	if updated == nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("order not found"))
	}
	return updated, nil
}
