package main

import (
	"context"
	"fmt"

	"connectrpc.com/connect"

	easypourv1 "easypour/service/gen/easypour/v1"
	"easypour/service/internal/cvar"
	"easypour/service/internal/settings"
)

func toProtoCvar(row *settings.CvarRow) *easypourv1.Cvar {
	return &easypourv1.Cvar{
		Key: row.Key, MainType: row.MainType,
		ValueInt: int32(row.ValueInt), ValueString: row.ValueString,
		Title: row.Title, Description: row.Description,
		Category: row.Category, Ordinal: int32(row.Ordinal),
	}
}

func appendProtoCvars(dst []*easypourv1.Cvar, rows []settings.CvarRow) []*easypourv1.Cvar {
	for i := range rows {
		dst = append(dst, toProtoCvar(&rows[i]))
	}
	return dst
}

func (s *EasyPourServer) siteTitle(ctx context.Context) string {
	row := s.findCvarValue(ctx, cvar.KeySiteTitle)
	if row == "" {
		return cvar.DefaultSiteTitle
	}
	return row
}

func (s *EasyPourServer) findCvarValue(ctx context.Context, key string) string {
	if s.settingsStore == nil {
		return ""
	}
	row, err := s.settingsStore.FindCvar(ctx, key)
	if err != nil || row == nil {
		return ""
	}
	return row.ValueString
}

func (s *EasyPourServer) initFeatures(_ context.Context) *easypourv1.Features {
	return &easypourv1.Features{}
}

func (s *EasyPourServer) ListCvars(ctx context.Context, _ *connect.Request[easypourv1.ListCvarsRequest]) (*connect.Response[easypourv1.ListCvarsResponse], error) {
	if _, err := s.requireAdmin(ctx); err != nil {
		return nil, err
	}
	if s.settingsStore == nil {
		return connect.NewResponse(&easypourv1.ListCvarsResponse{}), nil
	}
	rows, err := s.settingsStore.ListCvars(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&easypourv1.ListCvarsResponse{
		Cvars: appendProtoCvars(nil, rows),
	}), nil
}

func validateCvarUpdate(row *settings.CvarRow, valueInt int32, valueString string) (int, string, error) {
	switch row.MainType {
	case cvar.TypeString:
		return validateStringCvar(row.Key, valueString)
	case cvar.TypeInt:
		return int(valueInt), "", nil
	case cvar.TypeBool:
		return validateBoolCvar(valueInt), "", nil
	default:
		return 0, "", fmt.Errorf("unsupported cvar type")
	}
}

func validateStringCvar(key, valueString string) (int, string, error) {
	if len(valueString) > 255 {
		return 0, "", fmt.Errorf("value too long")
	}
	if key == cvar.KeyAppriseURL {
		return 0, valueString, nil
	}
	if valueString == "" {
		return 0, "", fmt.Errorf("value required")
	}
	return 0, valueString, nil
}

func validateBoolCvar(valueInt int32) int {
	if valueInt != 0 {
		return 1
	}
	return 0
}

func (s *EasyPourServer) UpdateCvar(ctx context.Context, req *connect.Request[easypourv1.UpdateCvarRequest]) (*connect.Response[easypourv1.Cvar], error) {
	if _, err := s.requireAdmin(ctx); err != nil {
		return nil, err
	}
	return s.applyCvarUpdate(ctx, req.Msg.Key, req.Msg.ValueInt, req.Msg.ValueString)
}

func (s *EasyPourServer) applyCvarUpdate(ctx context.Context, key string, valueInt int32, valueString string) (*connect.Response[easypourv1.Cvar], error) {
	row, findErr := s.requireCvar(ctx, key)
	if findErr != nil {
		return nil, findErr
	}
	vi, vs, valErr := validateCvarUpdate(row, valueInt, valueString)
	if valErr != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, valErr)
	}
	if err := s.settingsStore.UpdateCvar(ctx, row.Key, vi, vs); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	updated, _ := s.settingsStore.FindCvar(ctx, row.Key)
	return connect.NewResponse(toProtoCvar(updated)), nil
}

func (s *EasyPourServer) requireCvar(ctx context.Context, key string) (*settings.CvarRow, error) {
	if s.settingsStore == nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("settings store unavailable"))
	}
	row, err := s.settingsStore.FindCvar(ctx, key)
	if err != nil || row == nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("cvar not found"))
	}
	return row, nil
}
