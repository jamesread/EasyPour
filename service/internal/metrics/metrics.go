package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// A private registry keeps metrics isolated from anything registered globally by dependencies.
var (
	registry = prometheus.NewRegistry()

	ordersCreated = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "easypour_orders_created_total",
		Help: "Number of orders created.",
	})

	orderStatusUpdates = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "easypour_order_status_updates_total",
		Help: "Number of order status changes, by new status.",
	}, []string{"status"})
)

func init() {
	registry.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
		ordersCreated,
		orderStatusUpdates,
	)
}

// Handler serves the Prometheus exposition format for the service registry.
func Handler() http.Handler {
	return promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
}

// RecordOrderCreated increments the orders created counter.
func RecordOrderCreated() {
	ordersCreated.Inc()
}

// RecordOrderStatusUpdate increments the status update counter for the given status.
func RecordOrderStatusUpdate(status string) {
	orderStatusUpdates.WithLabelValues(status).Inc()
}
