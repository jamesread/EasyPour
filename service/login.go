package main

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/google/uuid"
	auth "github.com/jamesread/httpauthshim"
	"github.com/jamesread/httpauthshim/providers/haslocal"
)

const sessionMaxAgeSeconds = 7 * 24 * 3600

var errMissingCredentials = &httpError{http.StatusBadRequest, "username and password required"}

// loginRequest is the JSON body for POST /login
type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// loginOrSPA handles /login: POST goes to handleLogin; GET/HEAD when spa is set serves index.html
// so auth redirects to /login still load the SPA instead of 405.
func loginOrSPA(authCtx *auth.AuthShimContext, spa http.Handler) http.Handler {
	login := handleLogin(authCtx)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost:
			login(w, r)
		case spa != nil && isReadMethod(r.Method):
			r = r.Clone(r.Context())
			r.URL.Path = "/"
			spa.ServeHTTP(w, r)
		default:
			writeHTTPError(w, errMethodNotAllowed)
		}
	})
}

// handleLogin handles POST /login for session-based auth. Validates username/password,
// creates a session, and sets the session cookie.
func handleLogin(authCtx *auth.AuthShimContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeHTTPError(w, errMethodNotAllowed)
			return
		}
		username, err := authenticateLogin(authCtx, r)
		if err != nil {
			writeHTTPError(w, err)
			return
		}
		startLocalSession(w, r, authCtx, username)
	}
}

func authenticateLogin(authCtx *auth.AuthShimContext, r *http.Request) (string, error) {
	username, password, err := parseLoginRequest(r)
	if err != nil {
		return "", err
	}
	if !haslocal.CheckUserPassword(authCtx.Config, username, password) {
		return "", errUnauthorized
	}
	return username, nil
}

func parseLoginRequest(r *http.Request) (username, password string, err error) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return "", "", errBadRequest
	}
	username = strings.TrimSpace(req.Username)
	password = strings.TrimSpace(req.Password)
	if username == "" || password == "" {
		return "", "", errMissingCredentials
	}
	return username, password, nil
}

func startLocalSession(w http.ResponseWriter, r *http.Request, authCtx *auth.AuthShimContext, username string) {
	usergroup := ""
	if cfgUser := authCtx.Config.FindUserByUsername(username); cfgUser != nil {
		usergroup = cfgUser.UsergroupLine
	}
	sid := uuid.New().String()
	authCtx.RegisterUserSession("local", sid, username, strings.Split(usergroup, ",")...)
	http.SetCookie(w, sessionCookie(r, authCtx.Config.GetLocalSessionCookieName(), sid, sessionMaxAgeSeconds))
	writeJSON(w, map[string]string{"username": username})
}

// handleLogout clears the session cookie so the client is logged out.
func handleLogout(authCtx *auth.AuthShimContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeHTTPError(w, errMethodNotAllowed)
			return
		}
		http.SetCookie(w, sessionCookie(r, authCtx.Config.GetLocalSessionCookieName(), "", -1))
		writeJSON(w, map[string]string{"ok": "true"})
	}
}

// sessionCookie only sets Secure for HTTPS requests so plain-HTTP LAN installs can still log in.
func sessionCookie(r *http.Request, name, value string, maxAge int) *http.Cookie {
	return &http.Cookie{ //nolint:gosec // Secure is decided per request; see function comment
		Name:     name,
		Value:    value,
		Path:     "/",
		HttpOnly: true,
		Secure:   isSecureRequest(r),
		SameSite: http.SameSiteLaxMode,
		MaxAge:   maxAge,
	}
}

func isSecureRequest(r *http.Request) bool {
	return r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")
}
