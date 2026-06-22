# ZapLink Architecture

[English](#english) | [Русский](ARCHITECTURE.ru.md)

---

## English

Comprehensive architectural documentation for ZapLink URL shortener service.

## Table of Contents

- [Overview](#overview)
- [System Architecture](#system-architecture)
- [Technology Choices](#technology-choices)
- [Data Model](#data-model)
- [Request Flows](#request-flows)
- [Caching Strategy](#caching-strategy)
- [Error Handling](#error-handling)
- [Performance Optimizations](#performance-optimizations)
- [Design Decisions](#design-decisions)

---

## Overview

ZapLink is a **high-performance URL shortener** built with Go, designed to handle thousands of redirects per second with sub-20ms latency.

**Core Features:**
- URL shortening with unique code generation
- Fast redirection with Redis caching
- Click analytics tracking
- Prometheus metrics integration
- Structured logging with slog

---

## System Architecture

ZapLink follows a **layered architecture** pattern with clear separation of concerns:

![System Architecture](docs/images/system_architecture.png)

### Layer Responsibilities

| Layer      | Responsibilities | Does NOT handle |
|------------|------------------|-----------------|
| **Handler** | HTTP request/response, JSON marshaling, input validation | Business logic, database access |
| **Service** | Business logic, short code generation, orchestration | Direct database queries, HTTP concerns |
| **Repository** | PostgreSQL data access, SQL queries | Caching, business rules |
| **Cache** | Redis operations, TTL management | Business logic, validation |

**Critical Design Rule:** Repository layer does NOT handle caching. Service orchestrates both Repository and Cache independently.

---

## Technology Choices

### Go 1.25
**Why Go:**
- Excellent HTTP performance (native `net/http`)
- Simple concurrency model (goroutines for async click tracking)
- Strong standard library (context, slog, sql)
- Fast compilation and deployment

### Chi Router
**Why Chi:**
- Lightweight and idiomatic
- Excellent middleware support
- Context-based routing
- Compatible with `net/http.Handler`

**Alternatives considered:**
- Gin (more batteries-included, heavier)
- Echo (similar performance, different API style)

### PostgreSQL 17
**Why PostgreSQL:**
- ACID guarantees for link creation
- Rich indexing (unique constraint on `short_code`)
- JSONB support for future extensibility
- Robust migration tooling (golang-migrate)

**Schema design:**
- `links` table: main storage with `short_code` unique index
- `clicks` table: analytics with foreign key to `links`

### Redis 7
**Why Redis:**
- Sub-millisecond lookups (p99 < 1ms)
- Simple key-value model (short_code → serialized JSON)
- TTL support for automatic expiration
- Optional (app runs without Redis if unavailable)

**Cache strategy:** Write-through with 1-hour TTL (configurable via `REDIS_TTL_SECONDS`)

### Prometheus + Grafana
**Why Prometheus:**
- Industry standard for metrics
- Pull-based scraping (no instrumentation overhead)
- Powerful PromQL query language
- Native histogram support for latencies

**Metrics exposed:**
- HTTP request counters (by method, route, status)
- Request duration histograms
- Business metrics (links created, redirects served, clicks tracked)

---

## Data Model

### Links Table

```sql
CREATE TABLE links (
    id BIGSERIAL PRIMARY KEY,
    short_code VARCHAR(8) UNIQUE NOT NULL,
    original_url TEXT NOT NULL,
    is_active BOOLEAN DEFAULT true NOT NULL,
    created_at TIMESTAMPTZ DEFAULT now() NOT NULL
);

CREATE UNIQUE INDEX idx_links_short_code ON links(short_code);
CREATE INDEX idx_links_created_at ON links(created_at DESC);
```

| Column       | Type         | Constraints              | Purpose                           |
|--------------|--------------|--------------------------|-----------------------------------|
| id           | BIGSERIAL    | PRIMARY KEY              | Internal identifier               |
| short_code   | VARCHAR(8)   | UNIQUE NOT NULL          | User-facing short code (base62)   |
| original_url | TEXT         | NOT NULL                 | Original long URL                 |
| is_active    | BOOLEAN      | DEFAULT true NOT NULL    | Soft delete flag                  |
| created_at   | TIMESTAMPTZ  | DEFAULT now() NOT NULL   | Creation timestamp                |

**Index strategy:**
- `idx_links_short_code`: Fast lookup on redirect (most common query)
- `idx_links_created_at`: Analytics queries (recent links)

### Clicks Table

```sql
CREATE TABLE clicks (
    id BIGSERIAL PRIMARY KEY,
    link_id BIGINT NOT NULL REFERENCES links(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ DEFAULT now() NOT NULL,
    user_agent TEXT,
    ip_address INET
);

CREATE INDEX idx_clicks_link_id ON clicks(link_id);
CREATE INDEX idx_clicks_created_at ON clicks(created_at DESC);
```

| Column     | Type        | Constraints                            | Purpose                  |
|------------|-------------|----------------------------------------|--------------------------|
| id         | BIGSERIAL   | PRIMARY KEY                            | Internal identifier      |
| link_id    | BIGINT      | FOREIGN KEY → links(id) ON DELETE CASCADE | Reference to link     |
| created_at | TIMESTAMPTZ | DEFAULT now() NOT NULL                 | Click timestamp          |
| user_agent | TEXT        | NULL                                   | Browser user-agent       |
| ip_address | INET        | NULL                                   | Client IP address        |

**Index strategy:**
- `idx_clicks_link_id`: Fast click count aggregation per link
- `idx_clicks_created_at`: Time-series analytics queries

---

## Request Flows

### 1. Create Short Link

![Create Short Link](docs/images/create_short_link.png)

**Key Steps:**
1. Handler validates request body (JSON unmarshaling)
2. Service validates URL format (regex check)
3. Service generates unique short code (base62 encoding of timestamp + random bytes)
4. Repository inserts into PostgreSQL with unique constraint
5. If conflict (rare): retry with new short code (max 3 attempts)

**Metrics incremented:** `zaplink_links_created_total`

---

### 2. Redirect (Hot Path)

![Redirect](docs/images/redirect.png)

**Cache Hit (95% of requests):**
1. Handler extracts `short_code` from URL path
2. Service queries Redis via Cache layer
3. **Cache HIT** (p99 < 1ms)
4. Handler returns 302 redirect immediately
5. **Async**: Spawn goroutine to track click in background (doesn't block response)

**Cache Miss (5% of requests):**

```
  │                 │               │                 │                │                │
  │                 │               ├─GET short_code─>│                │                │
  │                 │               │<─MISS───────────┤                │                │
  │                 │               │                 │                │                │
  │                 │               ├─GetByShortCode()───────────────>│                │
  │                 │               │                 │                ├─SELECT * ─────>│
  │                 │               │                 │                │<─link row──────┤
  │                 │               │<─link─────────────────────────────┤                │
  │                 │               │                 │                │                │
  │                 │               ├─SET short_code, TTL=3600────────>│                │
  │                 │               │<─OK─────────────┤                │                │
  │                 │<─original_url─┤                 │                │                │
  │<─302 Redirect───┤               │                 │                │                │
```

**Cache miss flow:**
1. Redis returns empty (key not found)
2. Service queries PostgreSQL via Repository
3. If found: Service populates Redis with 1-hour TTL
4. Handler returns 302 redirect
5. **Async**: Track click in background


**Metrics incremented:** 
- `zaplink_redirects_total` (synchronous, always)
- `zaplink_clicks_tracked_total` (async, may lag or fail silently)

---

### 3. Get Link Info

![Get Link Info](docs/images/get_link_info.png)

**Key Points:**
- Does NOT use cache (always fresh data from PostgreSQL)
- Two queries: link lookup + click count aggregation
- Future optimization: cache click_count with short TTL (5 seconds)

---

## Caching Strategy

### Write-Through Cache Pattern

**On Read (Cache Hit):**
```
Service → Cache.Get(short_code) → Redis GET
          └─ HIT → return link data
```

**On Read (Cache Miss):**
```
Service → Cache.Get(short_code) → Redis GET
          └─ MISS → Repository.GetByShortCode() → PostgreSQL SELECT
                 → Cache.Set(short_code, link, TTL=3600) → Redis SET
                 → return link data
```

**On Write (Link Creation):**
```
Service → Repository.CreateLink() → PostgreSQL INSERT
       → (do NOT populate cache - let first read populate it)
```

**On Update (Link Modification):**
```
Service → Repository.UpdateLink() → PostgreSQL UPDATE
       → Cache.Delete(short_code) → Redis DEL (invalidate)
```


### Cache Configuration

| Parameter          | Default | Environment Variable  | Purpose                          |
|--------------------|---------|-----------------------|----------------------------------|
| TTL                | 3600s   | `REDIS_TTL_SECONDS`   | How long to cache links          |
| Connection Timeout | 5s      | -                     | Redis connect timeout            |
| Operation Timeout  | 2s      | -                     | Redis GET/SET timeout            |

### Graceful Degradation

If Redis is **unavailable at startup**:
- App logs warning and continues without cache
- All requests hit PostgreSQL directly
- Performance degrades but service remains available

```go
// cmd/api/main.go
redisCache, err := cache.NewRedisCache(cfg.Redis)
if err != nil {
    log.Warn("Redis unavailable, running without cache", slog.Any("error", err))
    redisCache = cache.NewNoOpCache() // Fallback to no-op cache
}
```

If Redis **fails during runtime**:
- Cache operations return errors
- Service falls back to Repository (PostgreSQL)
- Errors logged but not surfaced to client

---

## Error Handling

### Typed Errors (apperror package)

ZapLink uses **domain-specific errors** to distinguish business logic failures from technical failures:

```go
// internal/apperror/error.go
type Error struct {
    StatusCode int    // HTTP status code
    Code       string // Machine-readable error code
    Message    string // Human-readable message
    Err        error  // Underlying error (optional)
}
```

**Usage Example:**

```go
// Service layer
if !isValidURL(url) {
    return nil, apperror.New(http.StatusBadRequest, "invalid_url", "invalid url format")
}

// Repository layer (wrapping database error)
if err != nil {
    return nil, apperror.Wrap(err, http.StatusInternalServerError, "db_error", "failed to create link")
}
```

### Error Flow

```
Repository → Returns apperror.Error
    ↓
Service → Checks error type, may add context
    ↓
Handler → Calls writeError() helper
    ↓
Client ← Receives JSON error response
```

**Handler helper (internal/http/handler/helpers.go):**

```go
func (h *Handler) writeError(w http.ResponseWriter, err error) {
    var appErr *apperror.Error
    if errors.As(err, &appErr) {
        w.WriteHeader(appErr.StatusCode)
        json.NewEncoder(w).Encode(map[string]interface{}{
            "error": map[string]string{
                "code":    appErr.Code,
                "message": appErr.Message,
            },
        })
    } else {
        // Unexpected error - log and return 500
        w.WriteHeader(http.StatusInternalServerError)
        json.NewEncoder(w).Encode(map[string]interface{}{
            "error": map[string]string{
                "code":    "internal_error",
                "message": "internal server error",
            },
        })
    }
}
```

### Common Error Codes

| HTTP Status | Error Code         | Layer     | Cause                           |
|-------------|--------------------|-----------|---------------------------------|
| 400         | `invalid_url`      | Service   | URL validation failed           |
| 400         | `invalid_request`  | Handler   | JSON unmarshal failed           |
| 404         | `not_found`        | Repository| Short code does not exist       |
| 404         | `link_inactive`    | Service   | Link exists but is_active=false |
| 500         | `db_error`         | Repository| PostgreSQL query failed         |
| 500         | `internal_error`   | Handler   | Unexpected panic or error       |

---

## Performance Optimizations

### 1. Redis Caching
- **Impact:** 95% cache hit rate reduces DB load by 20x
- **Latency:** p99 < 20ms vs p99 < 100ms without cache

### 2. Async Click Tracking
- **Impact:** Click tracking does NOT block redirect response
- **Implementation:** Goroutine with separate context and 5s timeout

```go
// internal/http/handler/link.go:84-96
go func() {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    
    if err := h.service.TrackClick(ctx, short_code, userAgent, ipAddress); err != nil {
        log.Error("failed to track click", slog.String("short_code", short_code), slog.Any("error", err))
    }
}()
```

**Trade-off:** Click count may be slightly inaccurate if tracking fails (acceptable for analytics).

### 3. Database Indexes
- `idx_links_short_code`: B-tree index on `short_code` (most critical)
- `idx_clicks_link_id`: Foreign key index for click count aggregation

### 4. Connection Pooling
```go
// internal/repository/postgres.go
db.SetMaxOpenConns(25)      // Max concurrent connections
db.SetMaxIdleConns(5)       // Idle connection pool
db.SetConnMaxLifetime(5m)   // Recycle connections
```

### 5. Context Timeouts
- HTTP handler timeout: 30s (Chi middleware)
- Database query timeout: 10s (via `context.WithTimeout`)
- Redis operation timeout: 2s

---

## Design Decisions

### 1. Why Repository Does NOT Handle Caching

**Problem:** Mixing caching logic in Repository violates Single Responsibility Principle.

**Solution:** Service orchestrates both Repository (persistent storage) and Cache (ephemeral storage) independently.

**Benefits:**
- Repository remains a pure data access layer (testable without Redis)
- Cache can be swapped (e.g., Memcached) without changing Repository
- Clear separation: Repository = source of truth, Cache = optimization

**Code Example:**

```go
// Service layer (internal/service/link_service.go)
func (s *linkService) GetByShortCode(ctx context.Context, shortCode string) (*Link, error) {
    // Try cache first
    if link, err := s.cache.Get(ctx, shortCode); err == nil {
        return link, nil
    }
    
    // Cache miss - query database
    link, err := s.repo.GetByShortCode(ctx, shortCode)
    if err != nil {
        return nil, err
    }
    
    // Populate cache (fire-and-forget)
    go s.cache.Set(context.Background(), shortCode, link, s.cacheTTL)
    
    return link, nil
}
```

### 2. Why Async Click Tracking

**Problem:** Writing click records to PostgreSQL adds ~10-20ms latency to redirect response.

**Solution:** Spawn goroutine to track click in background.

**Trade-offs:**
- ✅ Redirect latency reduced by 20ms (p99 < 20ms vs < 40ms)
- ✅ User experience improved (faster redirects)
- ❌ Click count may be slightly inaccurate if tracking fails
- ❌ No backpressure if database is slow (goroutines may accumulate)

**Acceptable because:** Analytics data is not mission-critical. 99.5% tracking accuracy is sufficient.

### 3. Why Graceful Cache Degradation

**Problem:** Hard dependency on Redis means single point of failure.

**Solution:** If Redis is unavailable, run without cache (PostgreSQL-only mode).

**Benefits:**
- Service remains available even if Redis crashes
- Easier local development (no Redis required)
- Reduces operational complexity

**Trade-off:** Performance degrades (10x higher latency) but service stays online.

### 4. Why Base62 Short Codes

**Problem:** Need short, URL-safe identifiers without special characters.

**Solution:** Base62 encoding (0-9, A-Z, a-z) of timestamp + random bytes.

**Benefits:**
- URL-safe (no escaping needed)
- Case-sensitive (more entropy per character)
- Collision-resistant (timestamp ensures ordering)

**Alternative considered:**
- UUIDv4: Too long (36 chars vs 7 chars)
- Base64: Contains `/` and `+` (requires URL encoding)
- Hashids: Deterministic (security concern)

### 5. Why Chi Router Over Gin

**Decision:** Use `go-chi/chi` instead of `gin-gonic/gin`.

**Rationale:**
- Chi is more idiomatic Go (standard `net/http.Handler` interface)
- Lighter weight (fewer dependencies)
- Better middleware composability
- Gin has more features but heavier (not needed for this project)

---

## Future Enhancements

- [ ] Add distributed tracing (OpenTelemetry)
- [ ] Implement rate limiting (token bucket)
- [ ] Add custom short code support
- [ ] Implement link expiration (TTL)
- [ ] Add analytics API (click stats by time range)
- [ ] Geo-location tracking (MaxMind GeoIP2)
- [ ] Admin dashboard UI
- [ ] GraphQL API

---

**For API documentation, see [API.md](API.md)**  
**For observability setup, see [OBSERVABILITY.md](OBSERVABILITY.md)**
