package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"

	"easypour/service/internal/apprise"
	"easypour/service/internal/httpjson"
	"easypour/service/internal/order"
)

const appriseGroupSettle = 500 * time.Millisecond

// webhookItem is a single item in the webhook payload's items array.
type webhookItem struct {
	MenuItemId  string `json:"menu_item_id"`
	Name        string `json:"name,omitempty"`
	AddSugar    bool   `json:"add_sugar"`
	AddMilk     bool   `json:"add_milk"`
	SugarAmount int32  `json:"sugar_amount"`
	MilkAmount  int32  `json:"milk_amount"`
}

// orderWebhookPayload is the JSON body sent to webhook URLs when an order is submitted.
type orderWebhookPayload struct {
	OrderId     string        `json:"order_id"`
	Status      string        `json:"status"`
	CreatedAt   int64         `json:"created_at"`
	OrderString string        `json:"order_string"`
	Items       []webhookItem `json:"items"`
}

// formatWebhookItemString returns a short description for one item, e.g. "Coffee (no sugar, no milk)" or "Espresso (2 sugars, milk)".
func formatWebhookItemString(name string, addSugar, addMilk bool, sugarAmount, _ int32) string {
	return fmt.Sprintf("%s (%s, %s)", name, sugarDescription(addSugar, sugarAmount), milkDescription(addMilk))
}

// sugarDescription treats a non-positive amount as one sugar, since the flag alone means "with sugar".
func sugarDescription(addSugar bool, amount int32) string {
	if !addSugar {
		return "no sugar"
	}
	if amount <= 1 {
		return "1 sugar"
	}
	return fmt.Sprintf("%d sugars", amount)
}

func milkDescription(addMilk bool) string {
	if addMilk {
		return "milk"
	}
	return "no milk"
}

func describeOrder(name string, o *order.Order) string {
	return formatWebhookItemString(name, o.AddSugar, o.AddMilk, o.SugarAmount, o.MilkAmount)
}

func (s *EasyPourServer) notifyOrderPlaced(ctx context.Context, o *order.Order) {
	s.sendOrderWebhooks(o, nameOrID(s.menuNameByID(ctx), o.MenuItemID))
	s.scheduleAppriseGroupNotify(o.GroupID, o.Username)
}

// sendOrderWebhooks posts the order to every configured webhook in the background, with retries.
func (s *EasyPourServer) sendOrderWebhooks(o *order.Order, itemName string) {
	urls := s.webhookURLs()
	if len(urls) == 0 {
		return
	}
	logrus.Infof("Sending order %s to %d webhook(s)", o.ID, len(urls))
	body, err := json.Marshal(newOrderWebhookPayload(o, itemName))
	if err != nil {
		logrus.Warnf("Marshal webhook payload for order %s: %v", o.ID, err)
		return
	}
	for _, url := range urls {
		go s.postWebhook(url, body)
	}
}

func (s *EasyPourServer) webhookURLs() []string {
	var urls []string
	for _, wh := range s.webhooks {
		if wh.URL != "" {
			urls = append(urls, wh.URL)
		}
	}
	return urls
}

func newOrderWebhookPayload(o *order.Order, itemName string) orderWebhookPayload {
	return orderWebhookPayload{
		OrderId:     o.ID,
		Status:      o.Status,
		CreatedAt:   o.CreatedAt,
		OrderString: describeOrder(itemName, o),
		Items: []webhookItem{{
			MenuItemId:  o.MenuItemID,
			Name:        itemName,
			AddSugar:    o.AddSugar,
			AddMilk:     o.AddMilk,
			SugarAmount: o.SugarAmount,
			MilkAmount:  o.MilkAmount,
		}},
	}
}

func (s *EasyPourServer) postWebhook(url string, body []byte) {
	if err := httpjson.PostWithRetry(s.webhookClient, url, body); err != nil {
		logrus.Warnf("Webhook %s: %v", url, err)
	}
}

// scheduleAppriseGroupNotify coalesces notifications for a multi-item basket into one message.
func (s *EasyPourServer) scheduleAppriseGroupNotify(groupID, username string) {
	if groupID == "" || s.settingsStore == nil {
		return
	}
	s.appriseMu.Lock()
	defer s.appriseMu.Unlock()
	if existing, ok := s.appriseTimers[groupID]; ok {
		existing.Stop()
	}
	s.setAppriseTimer(groupID, time.AfterFunc(appriseGroupSettle, func() {
		s.fireAppriseGroupNotify(groupID, username)
	}))
}

func (s *EasyPourServer) setAppriseTimer(groupID string, timer *time.Timer) {
	if s.appriseTimers == nil {
		s.appriseTimers = make(map[string]*time.Timer)
	}
	s.appriseTimers[groupID] = timer
}

func (s *EasyPourServer) fireAppriseGroupNotify(groupID, username string) {
	s.appriseMu.Lock()
	delete(s.appriseTimers, groupID)
	s.appriseMu.Unlock()
	s.sendAppriseGroupNotify(context.Background(), groupID, username)
}

// sendAppriseGroupNotify loads all items in the group and sends a single Apprise notification.
func (s *EasyPourServer) sendAppriseGroupNotify(ctx context.Context, groupID, username string) {
	url := s.appriseNotifyURL(ctx)
	if url == "" {
		return
	}
	payload, ok := s.groupOrderPayload(ctx, groupID, username)
	if !ok {
		return
	}
	go func() {
		if err := apprise.Notify(s.webhookClient, url, payload); err != nil {
			logrus.Warnf("Apprise notify for order group %s: %v", groupID, err)
		}
	}()
}

func (s *EasyPourServer) appriseNotifyURL(ctx context.Context) string {
	url, err := s.savedAppriseURL(ctx)
	if err != nil {
		logrus.Warnf("Load settings for Apprise: %v", err)
	}
	return url
}

func (s *EasyPourServer) groupOrderPayload(ctx context.Context, groupID, username string) (apprise.Payload, bool) {
	members, err := s.orderStore.ListByGroupID(ctx, groupID)
	if err != nil {
		logrus.Warnf("List order group %s for Apprise: %v", groupID, err)
		return apprise.Payload{}, false
	}
	if len(members) == 0 {
		return apprise.Payload{}, false
	}
	if username == "" {
		username = members[0].Username
	}
	return apprise.Payload{
		Title: "New EasyPour order",
		Body:  apprise.FormatOrderBody(groupID, s.describeOrders(ctx, members), username),
		Type:  "info",
	}, true
}

func (s *EasyPourServer) describeOrders(ctx context.Context, orders []*order.Order) []string {
	names := s.menuNameByID(ctx)
	descriptions := make([]string, 0, len(orders))
	for _, o := range orders {
		descriptions = append(descriptions, describeOrder(nameOrID(names, o.MenuItemID), o))
	}
	return descriptions
}

func nameOrID(names map[string]string, id string) string {
	if name := names[id]; name != "" {
		return name
	}
	return id
}
