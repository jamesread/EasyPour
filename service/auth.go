package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"connectrpc.com/connect"
	auth "github.com/jamesread/httpauthshim"
	"github.com/jamesread/httpauthshim/authpublic"
	"github.com/jamesread/httpauthshim/providers/haslocal"
	"github.com/jamesread/httpauthshim/sessions"
	"github.com/sirupsen/logrus"

	easypourv1 "easypour/service/gen/easypour/v1"
	"easypour/service/gen/easypour/v1/easypourv1connect"
	"easypour/service/internal/config"
)

// contextKey stores the *http.Request in RPC contexts so handlers can resolve the session user.
type contextKey string

const httpRequestKey contextKey = "httpRequest"

func requestFromContext(ctx context.Context) *http.Request {
	r, _ := ctx.Value(httpRequestKey).(*http.Request)
	return r
}

// authenticatedUser returns the logged-in, non-guest user for r, or nil.
func authenticatedUser(authCtx *auth.AuthShimContext, r *http.Request) *authpublic.AuthenticatedUser {
	if r == nil {
		return nil
	}
	user := authCtx.AuthFromHttpReq(r)
	if user == nil || user.IsGuest() {
		return nil
	}
	return user
}

func (s *EasyPourServer) userFromContext(ctx context.Context) *authpublic.AuthenticatedUser {
	if s.authCtx == nil {
		return nil
	}
	return authenticatedUser(s.authCtx, requestFromContext(ctx))
}

// getUsernameFromContext returns the authenticated username, or "" if auth disabled or guest.
func (s *EasyPourServer) getUsernameFromContext(ctx context.Context) string {
	if user := s.userFromContext(ctx); user != nil {
		return user.Username
	}
	return ""
}

// requireAuth returns the authenticated user or error. Used by RPCs that need any logged-in user.
func (s *EasyPourServer) requireAuth(ctx context.Context) (*authpublic.AuthenticatedUser, error) {
	if s.authCtx == nil {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("auth required"))
	}
	r := requestFromContext(ctx)
	if r == nil {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("request context missing"))
	}
	user := authenticatedUser(s.authCtx, r)
	if user == nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("login required"))
	}
	return user, nil
}

// requireAdmin returns a permission-denied error if the request user is not in the admin group.
func (s *EasyPourServer) requireAdmin(ctx context.Context) error {
	user, err := s.requireAuth(ctx)
	if err != nil {
		return err
	}
	if !userInAdminGroup(user) {
		return connect.NewError(connect.CodePermissionDenied, fmt.Errorf("admin required"))
	}
	return nil
}

// userInAdminGroup returns true if the user's usergroups contain "admin"
func userInAdminGroup(user *authpublic.AuthenticatedUser) bool {
	for g := range strings.SplitSeq(user.UsergroupLine, ",") {
		if strings.EqualFold(strings.TrimSpace(g), "admin") {
			return true
		}
	}
	return false
}

// GetCurrentUser returns the authenticated user when auth is enabled (including is_admin for "admin" group),
// and the list of configured OAuth2 providers for the login form. Callable unauthenticated to obtain providers.
func (s *EasyPourServer) GetCurrentUser(
	ctx context.Context,
	_ *connect.Request[easypourv1.GetCurrentUserRequest],
) (*connect.Response[easypourv1.GetCurrentUserResponse], error) {
	resp := &easypourv1.GetCurrentUserResponse{OauthProviders: s.oauthProviders}
	if user := s.userFromContext(ctx); user != nil {
		resp.IsAuthenticated = true
		resp.Username = user.Username
		resp.IsAdmin = userInAdminGroup(user)
	}
	return connect.NewResponse(resp), nil
}

// withAuth wraps an http.Handler with httpauthshim authentication (session-based).
// Unauthenticated requests receive 401 without WWW-Authenticate so the browser does not show Basic auth.
// Init and GetCurrentUser are allowed without auth so the login form can fetch the user and OAuth provider list.
func withAuth(authCtx *auth.AuthShimContext, h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !isPublicProcedure(r.URL.Path) && authenticatedUser(authCtx, r) == nil {
			writeHTTPError(w, errUnauthorized)
			return
		}
		ctx := context.WithValue(r.Context(), httpRequestKey, r)
		h.ServeHTTP(w, r.WithContext(ctx))
	})
}

func isPublicProcedure(path string) bool {
	return path == easypourv1connect.EasyPourServiceInitProcedure ||
		path == easypourv1connect.EasyPourServiceGetCurrentUserProcedure
}

// requireSession applies withAuth only when auth is enabled.
func requireSession(authCtx *auth.AuthShimContext, h http.Handler) http.Handler {
	if authCtx == nil {
		return h
	}
	return withAuth(authCtx, h)
}

// setupAuth creates an AuthShimContext from the given config when auth is enabled.
// Returns nil when auth is disabled or misconfigured.
func setupAuth(appConfig *config.Config) (*auth.AuthShimContext, error) {
	authCfg := localAuthConfig(appConfig)
	if authCfg == nil {
		return nil, nil
	}
	authCtx, err := auth.NewAuthShimContext(authCfg, sessions.NewSessionStorage(sessions.NewYAMLPersistence()))
	if err != nil {
		return nil, err
	}
	authCtx.AddProvider(haslocal.CheckUserFromLocalSession)
	logrus.Info("Authentication enabled (httpauthshim, session-based login)")
	return authCtx, nil
}

func localAuthConfig(appConfig *config.Config) *authpublic.Config {
	if !localAuthEnabled(appConfig) {
		return nil
	}
	if len(appConfig.Auth.LocalUsers.Users) == 0 {
		logrus.Warn("Auth enabled but no users configured; auth disabled")
		return nil
	}
	return appConfig.Auth
}

func localAuthEnabled(appConfig *config.Config) bool {
	return appConfig != nil && appConfig.Auth != nil && appConfig.Auth.LocalUsers.Enabled
}

func shutdownAuth(authCtx *auth.AuthShimContext) {
	if authCtx == nil {
		return
	}
	if err := authCtx.Shutdown(); err != nil {
		logrus.Warnf("Auth shutdown error: %v", err)
	}
}
