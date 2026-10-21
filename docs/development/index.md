# Development

All build, test, and lint steps run through `make` from the repository root.

| Target | Purpose |
|--------|---------|
| `make` | Build the service and frontend |
| `make setup` | Install dependencies and generate protocol code |
| `make generate` | Regenerate Go and TypeScript code from `protocol/` with buf |
| `make test` | Generate protocol code, then run service and frontend unit tests |
| `make lint` | Run `go vet` and ESLint |
| `make integration-test` | Build the service, start it with `-configdir integration-tests/tests`, and run mocha |
| `make migrate` | Apply SQLite migrations (set `DB_PATH`) |
| `make dev-service` | Run the service on `$PORT` (default `9654`) |
| `make dev-frontend` | Run the Vite dev server on port 3000, proxying to the service on `$PORT` (default `9654`) |

Buf plugin versions are pinned in `protocol/buf.gen.yaml` so `make generate` is reproducible; bump them deliberately alongside the matching Go and npm runtime libraries.

## Metrics

The service exposes Prometheus metrics at `/metrics` (unauthenticated):

| Metric | Type | Description |
|--------|------|-------------|
| `easypour_orders_created_total` | counter | Orders created |
| `easypour_order_status_updates_total{status}` | counter | Order status changes, by new status |
| `go_*`, `process_*` | various | Go runtime and process metrics |
