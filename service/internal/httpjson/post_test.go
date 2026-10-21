package httpjson

import (
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPostWithRetry_SendsJSONBody(t *testing.T) {
	var gotContentType, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotContentType = r.Header.Get("Content-Type")
		body, _ := io.ReadAll(r.Body)
		gotBody = string(body)
	}))
	defer srv.Close()

	require.NoError(t, PostWithRetry(srv.Client(), srv.URL, []byte(`{"a":1}`)))
	assert.Equal(t, "application/json", gotContentType)
	assert.JSONEq(t, `{"a":1}`, gotBody)
}

func TestPostWithRetry_RetriesUntilSuccess(t *testing.T) {
	var attempts atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if attempts.Add(1) < 2 {
			w.WriteHeader(http.StatusBadGateway)
		}
	}))
	defer srv.Close()

	require.NoError(t, PostWithRetry(srv.Client(), srv.URL, []byte(`{}`)))
	assert.Equal(t, int32(2), attempts.Load())
}

func TestPostWithRetry_ReturnsLastErrorAfterAllAttempts(t *testing.T) {
	var attempts atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		attempts.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	err := PostWithRetry(srv.Client(), srv.URL, []byte(`{}`))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "HTTP 500")
	assert.Equal(t, int32(MaxAttempts), attempts.Load())
}

func TestBackoffDelay_DoublesAfterFirstRetry(t *testing.T) {
	assert.Zero(t, backoffDelay(0))
	assert.Equal(t, baseRetryDelay, backoffDelay(1))
	assert.Equal(t, 2*baseRetryDelay, backoffDelay(2))
}
