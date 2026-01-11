## Relevant Files

- `internal/o11y/o11y.go` - Observability setup that needs Prometheus registry added
- `internal/middleware/metrics.go` - New file to create for RED metrics middleware
- `cmd/server/main.go` - Server initialization where metrics credentials configuration will be added
- `api/api.go` - API setup that needs refactoring to use route groups and basic auth for metrics endpoint

### Notes

- The application uses Gin web framework with existing middleware pattern
- Prometheus client library (`github.com/prometheus/client_golang v1.19.1`) is already installed
- Existing middleware: Tracing, Logging, JWT authentication
- JWT authentication should only apply to business routes, not `/metrics`
- Grafana Cloud will scrape `/metrics` using HTTP basic auth

## Instructions for Completing Tasks

**IMPORTANT:** As you complete each task, you must check it off in this markdown file by changing `- [ ]` to `- [x]`. This helps track progress and ensures you don't skip any steps.

Example:
- `- [ ] 1.1 Read file` → `- [x] 1.1 Read file` (after completing)

Update the file after completing each sub-task, not just after completing an entire parent task.

## Tasks

- [x] 0.0 Create feature branch
  - [x] 0.1 Create and checkout a new branch for this feature (e.g., `git checkout -b feature/red-metrics-prometheus`)
- [x] 1.0 Update observability to add Prometheus registry
  - [x] 1.1 Read internal/o11y/o11y.go to understand current structure
  - [x] 1.2 Import prometheus package (`github.com/prometheus/client_golang/prometheus`)
  - [x] 1.3 Add `Registry *prometheus.Registry` field to Observability struct
  - [x] 1.4 Initialize registry in Setup() function with `prometheus.NewRegistry()`
  - [x] 1.5 Assign registry to Observability struct before returning
- [x] 2.0 Create metrics middleware for RED metrics
  - [x] 2.1 Create internal/middleware/metrics.go file with package declaration
  - [x] 2.2 Import required packages (gin, prometheus, time, strconv)
  - [x] 2.3 Define `httpRequestsTotal` CounterVec with labels: method, path, status
  - [x] 2.4 Define `httpRequestErrorsTotal` CounterVec with labels: method, path, status, error_type
  - [x] 2.5 Define `httpRequestDuration` HistogramVec with labels: method, path, status
  - [x] 2.6 Implement `Metrics(reg *prometheus.Registry) gin.HandlerFunc` function
  - [x] 2.7 Register all three metrics with the provided registry using `reg.MustRegister()`
  - [x] 2.8 In returned HandlerFunc, capture start time with `time.Now()` before `c.Next()`
  - [x] 2.9 After `c.Next()`, extract status code (`c.Writer.Status()`), path (`c.FullPath()`), method (`c.Request.Method`)
  - [x] 2.10 Calculate duration with `time.Since(start).Seconds()`
  - [x] 2.11 Increment `httpRequestsTotal` counter with extracted labels
  - [x] 2.12 Add conditional logic: if status >= 400 && < 500, increment `httpRequestErrorsTotal` with `error_type="client"`
  - [x] 2.13 Add conditional logic: if status >= 500, increment `httpRequestErrorsTotal` with `error_type="server"`
  - [x] 2.14 Observe duration in `httpRequestDuration` histogram with extracted labels
- [x] 3.0 Update main.go for metrics credentials configuration
  - [x] 3.1 Read cmd/server/main.go to understand current CLI structure
  - [x] 3.2 Add `MetricsUsername string` field to cli struct with Kong tag `env:"METRICS_USERNAME"`
  - [x] 3.3 Add `MetricsPassword string` field to cli struct with Kong tag `env:"METRICS_PASSWORD"`
  - [x] 3.4 Update `api.New()` call to pass `cli.MetricsUsername` and `cli.MetricsPassword` as additional parameters
- [x] 4.0 Refactor API to use route groups with basic auth for metrics
  - [x] 4.1 Read api/api.go to understand current structure and imports
  - [x] 4.2 Import `os` and `github.com/prometheus/client_golang/prometheus/promhttp` packages
  - [x] 4.3 Update `New()` function signature to accept `metricsUsername string` and `metricsPassword string` parameters
  - [x] 4.4 Keep recovery, tracing, and logging middleware as global (applied with `a.r.Use()`)
  - [x] 4.5 Add `a.r.Use(middleware.Metrics(o.Registry))` after logging middleware to apply metrics globally
  - [x] 4.6 Add conditional check: if `metricsUsername != "" && metricsPassword != ""`, create metrics endpoint
  - [x] 4.7 Inside conditional, create authorized group with `gin.BasicAuth(gin.Accounts{metricsUsername: metricsPassword})`
  - [x] 4.8 Add `/metrics` endpoint to authorized group using `gin.WrapH(promhttp.HandlerFor(o.Registry, promhttp.HandlerOpts{}))`
  - [x] 4.9 Create protected route group with `a.r.Group("/")`
  - [x] 4.10 Apply JWT middleware to protected group using `protected.Use(a.jwtValidator.EnsureValidToken())`
  - [x] 4.11 Move `GET /bikes/nearby` to protected group (change `a.r.GET` to `protected.GET`)
  - [x] 4.12 Move `GET /bikes/:id` to protected group
  - [x] 4.13 Move `GET /stations` to protected group
  - [x] 4.14 Move `GET /stations/:id` to protected group
  - [x] 4.15 Remove global JWT middleware application (delete line with `a.r.Use(a.jwtValidator.EnsureValidToken())`)
