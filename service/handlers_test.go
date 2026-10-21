package main

import (
	"crypto/tls"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/jamesread/httpauthshim/authpublic"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"easypour/service/gen/easypour/v1/easypourv1connect"
	"easypour/service/internal/config"
	"easypour/service/internal/order"
)

func TestFormatWebhookItemString(t *testing.T) {
	cases := []struct {
		name        string
		addSugar    bool
		addMilk     bool
		sugarAmount int32
		want        string
	}{
		{"no extras", false, false, 0, "Coffee (no sugar, no milk)"},
		{"sugar defaults to one", true, false, 0, "Coffee (1 sugar, no milk)"},
		{"one sugar", true, true, 1, "Coffee (1 sugar, milk)"},
		{"several sugars", true, true, 3, "Coffee (3 sugars, milk)"},
		{"amount ignored without sugar", false, true, 2, "Coffee (no sugar, milk)"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, formatWebhookItemString("Coffee", tc.addSugar, tc.addMilk, tc.sugarAmount, 0))
		})
	}
}

func TestValidOrderStatusTransition(t *testing.T) {
	cases := []struct {
		from, to string
		isAdmin  bool
		want     bool
	}{
		{"pending", "preparing", true, true},
		{"pending", "preparing", false, false},
		{"pending", "delivered", false, true},
		{"preparing", "delivered", false, true},
		{"delivered", "pending", true, false},
		{"pending", "pending", true, false},
		{"pending", "bogus", true, false},
		{"bogus", "delivered", true, false},
	}
	for _, tc := range cases {
		t.Run(tc.from+"->"+tc.to, func(t *testing.T) {
			assert.Equal(t, tc.want, validOrderStatusTransition(tc.from, tc.to, tc.isAdmin))
		})
	}
}

func TestUserInAdminGroup(t *testing.T) {
	assert.True(t, userInAdminGroup(&authpublic.AuthenticatedUser{UsergroupLine: "staff, Admin"}))
	assert.False(t, userInAdminGroup(&authpublic.AuthenticatedUser{UsergroupLine: "staff,administrators"}))
	assert.False(t, userInAdminGroup(&authpublic.AuthenticatedUser{}))
}

func TestCanViewOrder_OwnerOrAdminOnly(t *testing.T) {
	o := &order.Order{Username: "alice"}
	assert.True(t, canViewOrder(&authpublic.AuthenticatedUser{Username: "alice"}, o))
	assert.True(t, canViewOrder(&authpublic.AuthenticatedUser{Username: "bob", UsergroupLine: "admin"}, o))
	assert.False(t, canViewOrder(&authpublic.AuthenticatedUser{Username: "bob"}, o))
}

func newTestSPA(t *testing.T) http.Handler {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "index.html"), []byte("<html>index</html>"), 0o600))
	require.NoError(t, os.Mkdir(filepath.Join(dir, "assets"), 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "assets", "app.js"), []byte("console.log(1)"), 0o600))
	handler, err := spaFileServer(dir)
	require.NoError(t, err)
	return handler
}

func serveSPA(t *testing.T, handler http.Handler, method, path string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), method, path, nil))
	return rec
}

func TestSPAFileServer_ServesIndexAtRoot(t *testing.T) {
	rec := serveSPA(t, newTestSPA(t), http.MethodGet, "/")
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "index")
}

func TestSPAFileServer_ServesStaticFiles(t *testing.T) {
	rec := serveSPA(t, newTestSPA(t), http.MethodGet, "/assets/app.js")
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "console.log(1)", rec.Body.String())
}

func TestSPAFileServer_ServesIndexForClientRoutes(t *testing.T) {
	rec := serveSPA(t, newTestSPA(t), http.MethodGet, "/orders/123")
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "index")
}

func TestSPAFileServer_ServesIndexForPathsOutsideRoot(t *testing.T) {
	rec := serveSPA(t, newTestSPA(t), http.MethodGet, "/../../etc/passwd")
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "index")
}

func TestSPAFileServer_RejectsWrites(t *testing.T) {
	rec := serveSPA(t, newTestSPA(t), http.MethodPost, "/")
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestWithAuth_AllowsPublicProceduresWithoutSession(t *testing.T) {
	var sawRequestInContext bool
	inner := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		sawRequestInContext = requestFromContext(r.Context()) != nil
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, easypourv1connect.EasyPourServiceInitProcedure, nil)
	withAuth(nil, inner).ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.True(t, sawRequestInContext)
}

func TestIsSecureRequest(t *testing.T) {
	plain := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/login", nil)
	assert.False(t, isSecureRequest(plain))

	tlsReq := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/login", nil)
	tlsReq.TLS = &tls.ConnectionState{}
	assert.True(t, isSecureRequest(tlsReq))

	proxied := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/login", nil)
	proxied.Header.Set("X-Forwarded-Proto", "HTTPS")
	assert.True(t, isSecureRequest(proxied))
}

func TestImageExtension_DefaultsToJPG(t *testing.T) {
	assert.Equal(t, ".png", imageExtension("photo.png"))
	assert.Equal(t, ".jpg", imageExtension("photo"))
}

func TestClampToInt32(t *testing.T) {
	assert.Equal(t, int32(42), clampToInt32(42))
	assert.Equal(t, int32(math.MaxInt32), clampToInt32(math.MaxInt32+1))
	assert.Equal(t, int32(math.MinInt32), clampToInt32(math.MinInt32-1))
}

func TestOAuthProvidersFromConfig_SkipsIncompleteEntries(t *testing.T) {
	providers := oauthProvidersFromConfig([]config.OAuthProviderConfig{
		{ID: "google", Name: "Google", AuthURL: "https://example.com/auth"},
		{ID: "github", Name: "GitHub"},
	})
	require.Len(t, providers, 1)
	assert.Equal(t, "google", providers[0].GetId())
}

func TestWriteJSON_SetsContentType(t *testing.T) {
	rec := httptest.NewRecorder()
	writeJSON(rec, map[string]string{"ok": "true"})
	body, err := io.ReadAll(rec.Body)
	require.NoError(t, err)
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))
	assert.JSONEq(t, `{"ok":"true"}`, string(body))
}
