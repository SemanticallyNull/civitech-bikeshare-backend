# CLAUDE.md - AI Agent Guide for Booking Engine Backend

## Overview

This is a Go backend for a bike rental mobile app. Users can make advance or on-the-spot bookings. Billing is per-unlock + per-minute.

## Quick Reference

| What | Where |
|------|-------|
| Entry point | `cmd/server/main.go` |
| HTTP handlers | `api/` |
| Domain modules | `bike/`, `customer/`, `ride/`, `station/` |
| Middleware | `internal/middleware/` |
| Observability | `internal/o11y/` |
| Migrations | `sql/migrations/` |
| E2E tests | `acceptance/` |

## Architecture

```
cmd/server/main.go          # Config (Kong), DI, server setup
    │
    ├── api/api.go          # Gin router, middleware chain
    │   ├── bike.go         # GET /bikes/nearby, /bikes/:id, /bikes/:id/unlock
    │   ├── station.go      # GET /stations, /stations/:id
    │   ├── customer.go     # POST /customer/session, /setupintent, /paymentmethod
    │   └── ride.go         # POST /ride/start, /ride/end, GET /ride/current
    │
    └── {domain}/           # Each domain has:
        ├── {domain}.go     #   - Model structs (db tags for sqlx)
        └── sql.go          #   - Repository with raw SQL queries
```

## Key Libraries

- **HTTP**: `gin-gonic/gin`
- **Database**: `jmoiron/sqlx` + `jackc/pgx/v5` (PostgreSQL)
- **Auth**: Auth0 via `auth0/go-jwt-middleware/v2`
- **Payments**: `stripe/stripe-go/v84`
- **Config**: `alecthomas/kong` (env vars)
- **Observability**: `prometheus/client_golang`, `go.opentelemetry.io/otel`

## Development Workflow

**We use TDD (Red-Green-Refactor)**:
1. Write failing e2e test in `acceptance/`
2. Implement minimal code to pass
3. Refactor while keeping tests green

**Run locally**:
```bash
docker-compose up -d                    # PostgreSQL on :5432
export DATABASE_URL="postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable"
go run ./cmd/server/main.go
```

## Adding a New Feature

### 1. Create Domain Module

```go
// newdomain/newdomain.go
package newdomain

type Thing struct {
    ID        uuid.UUID `db:"id"`
    Name      string    `db:"name"`
    CreatedAt time.Time `db:"created_at"`
}

// newdomain/sql.go
package newdomain

var ErrNotFound = errors.New("thing not found")

type Repository struct {
    db *sqlx.DB
}

func NewRepository(db *sqlx.DB) *Repository {
    return &Repository{db: db}
}

func (r *Repository) GetThing(ctx context.Context, id uuid.UUID) (Thing, error) {
    var t Thing
    err := r.db.GetContext(ctx, &t, "SELECT * FROM things WHERE id = $1", id)
    if errors.Is(err, sql.ErrNoRows) {
        return Thing{}, ErrNotFound
    }
    return t, err
}
```

### 2. Add Migration

```sql
-- sql/migrations/YYYYMMDDHHMMSS_add_things.up.sql
CREATE TABLE things (
    id         uuid PRIMARY KEY,
    name       text NOT NULL,
    created_at timestamp with time zone DEFAULT now()
);

-- sql/migrations/YYYYMMDDHHMMSS_add_things.down.sql
DROP TABLE things;
```

### 3. Create Handler

```go
// api/thing.go
package api

type thingResponse struct {
    ID   uuid.UUID `json:"id"`
    Name string    `json:"name"`
}

func (a *API) thingHandler(c *gin.Context) {
    logger := middleware.GetLogger(c)

    id, err := uuid.Parse(c.Param("id"))
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
        return
    }

    thing, err := a.thingRepo.GetThing(c, id)
    if err != nil {
        if errors.Is(err, newdomain.ErrNotFound) {
            c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
            return
        }
        logger.ErrorContext(c, "failed to get thing", "error", err)
        c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
        return
    }

    c.JSON(http.StatusOK, thingResponse{ID: thing.ID, Name: thing.Name})
}
```

### 4. Register Route

```go
// api/api.go - in NewAPI()
protected.GET("/things/:id", a.thingHandler)
```

### 5. Wire in main.go

```go
thingRepo := newdomain.NewRepository(db)
api := api.NewAPI(/* existing repos */, thingRepo)
```

## Handler Patterns

### Getting User Context
```go
userID, ok := middleware.GetUserID(c)  // Auth0 subject claim
logger := middleware.GetLogger(c)       // Request-scoped logger with trace IDs
```

### Request Binding
```go
var req struct {
    BikeID string `json:"bike_id"`
}
if err := c.Bind(&req); err != nil {
    c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
    return
}
```

### Error Responses
```go
// 400 - Bad request (validation)
// 404 - Not found
// 409 - Conflict (resource unavailable)
// 412 - Precondition failed (e.g., no payment method)
// 500 - Internal error
c.JSON(http.StatusNotFound, gin.H{"error": "bike not found"})
```

## Database Patterns

### Repository Pattern
- Each domain has a `Repository` struct with `*sqlx.DB`
- Use raw SQL queries (no ORM)
- Use `db:"column_name"` tags for column mapping
- Define sentinel errors: `var ErrNotFound = errors.New("not found")`

### Transactions
```go
tx, err := r.db.BeginTxx(ctx, nil)
if err != nil {
    return err
}
defer tx.Rollback()

// Use SELECT ... FOR UPDATE for pessimistic locking
var bike Bike
err = tx.GetContext(ctx, &bike, "SELECT * FROM bikes WHERE label = $1 FOR UPDATE", id)

// Do updates...

return tx.Commit()
```

### PostGIS Points
```go
// Model
Location pgtype.Point `db:"location"`

// Access coordinates
lat := thing.Location.P.X
lng := thing.Location.P.Y
```

## Testing

### E2E Tests (acceptance/)

All tests are e2e. Run the full application against a test database. Use fakes (not mocks) for external APIs.

```go
// acceptance/thing_test.go
func TestCreateThing(t *testing.T) {
    // Setup: start server with test DB
    // Act: make HTTP request
    // Assert: check response and DB state
}
```

External service fakes should simulate real behavior, not just return canned responses.

## Middleware Chain

All protected routes use this middleware stack (in order):
1. `gin.Recovery()` - Panic recovery
2. `Tracing()` - OpenTelemetry spans
3. `Logging()` - slog with trace IDs
4. `Metrics()` - Prometheus counters/histograms
5. `JWT()` - Auth0 token validation

## Environment Variables

| Variable | Purpose |
|----------|---------|
| DATABASE_URL | PostgreSQL connection string |
| PORT | Server port (default: 8080) |
| AUTH0_DOMAIN | Auth0 tenant domain |
| AUDIENCE | JWT audience |
| STRIPE_PK | Stripe publishable key |
| STRIPE_SK | Stripe secret key |
| METRICS_USERNAME | Basic auth for /metrics |
| METRICS_PASSWORD | Basic auth for /metrics |

## Code Style

- Max line length: 120 chars
- No naked returns
- No `pkg/errors` (use stdlib `errors`)
- Use `errors.Is()` for error checking
- Logging: `logger.ErrorContext(ctx, "message", "key", value)`

## Common Tasks

### Add new endpoint to existing domain
1. Add repository method in `{domain}/sql.go`
2. Add handler in `api/{domain}.go`
3. Register route in `api/api.go`

### Add field to existing model
1. Add to struct in `{domain}/{domain}.go`
2. Create migration in `sql/migrations/`
3. Update relevant queries in `{domain}/sql.go`

### Debug with logs
```go
logger := middleware.GetLogger(c)
logger.InfoContext(c, "debug message", "key", value)
```
Logs include trace IDs for correlation.
