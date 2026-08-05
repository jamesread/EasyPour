package apprise

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNotify_EmptyURLIsNoop(t *testing.T) {
	err := Notify(nil, "", Payload{Title: "t", Body: "b"})
	assert.NoError(t, err)
}

func TestNotify_PostsJSONPayload(t *testing.T) {
	var gotMethod string
	var gotContentType string
	var gotBody Payload
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotContentType = r.Header.Get("Content-Type")
		require.NoError(t, json.NewDecoder(r.Body).Decode(&gotBody))
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	err := Notify(srv.Client(), srv.URL, Payload{Title: "New order", Body: "Coffee", Type: "info"})
	require.NoError(t, err)
	assert.Equal(t, http.MethodPost, gotMethod)
	assert.Equal(t, "application/json", gotContentType)
	assert.Equal(t, "New order", gotBody.Title)
	assert.Equal(t, "Coffee", gotBody.Body)
	assert.Equal(t, "info", gotBody.Type)
}

func TestNotify_DefaultsTypeToInfo(t *testing.T) {
	var gotBody Payload
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, json.NewDecoder(r.Body).Decode(&gotBody))
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	require.NoError(t, Notify(srv.Client(), srv.URL, Payload{Title: "t", Body: "b"}))
	assert.Equal(t, "info", gotBody.Type)
}

func TestNotify_RetriesOnFailureThenSucceeds(t *testing.T) {
	var attempts atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := attempts.Add(1)
		io.Copy(io.Discard, r.Body)
		if n < 2 {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	err := Notify(srv.Client(), srv.URL, Payload{Title: "t", Body: "b"})
	require.NoError(t, err)
	assert.Equal(t, int32(2), attempts.Load())
}

func TestFormatOrderBody(t *testing.T) {
	assert.Equal(t,
		"Order #abcd1234\n• Coffee (1 sugar, milk)",
		FormatOrderBody("abcd1234ef", []string{"Coffee (1 sugar, milk)"}, ""),
	)
	assert.Equal(t,
		"Ordered by alice\nOrder #order01\n• Espresso",
		FormatOrderBody("order01", []string{"Espresso"}, "alice"),
	)
	assert.Equal(t,
		"• Tea",
		FormatOrderBody("", []string{"Tea"}, ""),
	)
	assert.Equal(t,
		"Ordered by bob\nOrder #group123\n• Latte (no sugar, milk)\n• Marmite toast (no sugar, no milk)",
		FormatOrderBody("group12345", []string{
			"Latte (no sugar, milk)",
			"Marmite toast (no sugar, no milk)",
		}, "bob"),
	)
}
