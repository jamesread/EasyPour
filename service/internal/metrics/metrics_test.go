package metrics

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func scrapeMetrics(t *testing.T) string {
	t.Helper()
	rec := httptest.NewRecorder()
	Handler().ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/metrics", nil))
	require.Equal(t, http.StatusOK, rec.Code)
	body, err := io.ReadAll(rec.Body)
	require.NoError(t, err)
	return string(body)
}

func TestHandler_ExposesGoRuntimeMetrics(t *testing.T) {
	assert.Contains(t, scrapeMetrics(t), "go_goroutines")
}

func TestRecordOrderCreated_IncrementsOrdersCreatedCounter(t *testing.T) {
	RecordOrderCreated()

	assert.Contains(t, scrapeMetrics(t), "easypour_orders_created_total")
}

func TestRecordOrderStatusUpdate_LabelsCounterByStatus(t *testing.T) {
	RecordOrderStatusUpdate("delivered")

	assert.Contains(t, scrapeMetrics(t), `easypour_order_status_updates_total{status="delivered"}`)
}
