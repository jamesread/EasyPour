package main

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/sirupsen/logrus"
)

// httpError carries the status code and client-facing message for plain HTTP handlers.
type httpError struct {
	status  int
	message string
}

func (e *httpError) Error() string {
	return e.message
}

var (
	errMethodNotAllowed = &httpError{http.StatusMethodNotAllowed, "Method not allowed"}
	errBadRequest       = &httpError{http.StatusBadRequest, "Bad request"}
	errUnauthorized     = &httpError{http.StatusUnauthorized, "Unauthorized"}
	errForbidden        = &httpError{http.StatusForbidden, "Forbidden"}
)

// writeHTTPError writes an httpError as-is; any other error becomes a generic 500 so internals are not leaked.
func writeHTTPError(w http.ResponseWriter, err error) {
	var he *httpError
	if errors.As(err, &he) {
		http.Error(w, he.message, he.status)
		return
	}
	logrus.Warnf("HTTP handler error: %v", err)
	http.Error(w, "Internal server error", http.StatusInternalServerError)
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		logrus.Warnf("Write JSON response: %v", err)
	}
}

func isReadMethod(method string) bool {
	return method == http.MethodGet || method == http.MethodHead
}
