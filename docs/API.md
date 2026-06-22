# API Reference

[English](#english) | [Русский](API.ru.md)

---

## English

ZapLink REST API documentation with detailed endpoint specifications, request/response examples, and error codes.

**Base URL:** `http://localhost:8080`

**Content-Type:** All requests and responses use `application/json`

---

## Table of Contents

- [Endpoints](#endpoints)
  - [Create Short Link](#create-short-link)
  - [Get Link Info](#get-link-info)
  - [Redirect to Original URL](#redirect-to-original-url)
  - [Health Check](#health-check)
  - [Prometheus Metrics](#prometheus-metrics)
- [Error Responses](#error-responses)
- [Examples](#examples)

---

## Endpoints

### Create Short Link

Generate a shortened URL for a given original URL.

**Endpoint:** `POST /links`

**Request Body:**

```json
{
  "url": "https://example.com/very/long/url/path"
}
```

| Field | Type   | Required | Description                           |
|-------|--------|----------|---------------------------------------|
| url   | string | Yes      | Original URL to shorten (must be valid HTTP/HTTPS) |

**Success Response:** `201 Created`

```json
{
  "short_code": "a1b2c3d",
  "short_url": "http://localhost:8080/a1b2c3d"
}
```

| Field      | Type   | Description                          |
|------------|--------|--------------------------------------|
| short_code | string | Generated unique short code (7-8 chars) |
| short_url  | string | Full shortened URL                   |

**Error Responses:**

- `400 Bad Request` - Invalid URL format
- `500 Internal Server Error` - Database or server error

**Example:**

```bash
curl -X POST http://localhost:8080/links \
  -H "Content-Type: application/json" \
  -d '{
    "url": "https://github.com/ekideno/zaplink"
  }'
```

---

### Get Link Info

Retrieve information about a shortened link, including click count.

**Endpoint:** `GET /links/{short_code}`

**Path Parameters:**

| Parameter  | Type   | Description                    |
|------------|--------|--------------------------------|
| short_code | string | Short code of the link         |

**Success Response:** `200 OK`

```json
{
  "id": 42,
  "short_code": "a1b2c3d",
  "original_url": "https://github.com/ekideno/zaplink",
  "is_active": true,
  "created_at": "2026-06-18T12:34:56Z",
  "click_count": 127
}
```

| Field        | Type    | Description                              |
|--------------|---------|------------------------------------------|
| id           | integer | Internal database ID                     |
| short_code   | string  | Unique short code                        |
| original_url | string  | Original long URL                        |
| is_active    | boolean | Whether link is active (true/false)      |
| created_at   | string  | Creation timestamp (ISO 8601 UTC)        |
| click_count  | integer | Total number of redirects/clicks         |

**Error Responses:**

- `404 Not Found` - Short code does not exist
- `500 Internal Server Error` - Database error

**Example:**

```bash
curl http://localhost:8080/links/a1b2c3d
```

---

### Redirect to Original URL

Redirect to the original URL associated with the short code. **This is the main redirect endpoint.**

**Endpoint:** `GET /{short_code}`

**Path Parameters:**

| Parameter  | Type   | Description            |
|------------|--------|------------------------|
| short_code | string | Short code of the link |

**Success Response:** `302 Found`

```
HTTP/1.1 302 Found
Location: https://github.com/ekideno/zaplink
```

Browser automatically follows the redirect to the original URL.

**Side Effects:**
- Click is tracked asynchronously (user-agent, IP address, timestamp)
- Does NOT block redirect response
- Cache is updated if Redis is available

**Error Responses:**

- `404 Not Found` - Short code does not exist or link is inactive
- `500 Internal Server Error` - Database error

**Example:**

```bash
# Follow redirect
curl -L http://localhost:8080/a1b2c3d

# View redirect header only
curl -I http://localhost:8080/a1b2c3d
```

**Performance:**
- Cached redirects: **p99 < 20ms**
- Database redirects: **p99 < 100ms**

---

### Health Check

Check service health status, including PostgreSQL and Redis connectivity.

**Endpoint:** `GET /health`

**Success Response:** `200 OK`

```json
{
  "status": "healthy",
  "timestamp": "2026-06-18T12:34:56Z",
  "checks": {
    "database": "ok",
    "redis": "ok"
  }
}
```

| Field     | Type   | Description                              |
|-----------|--------|------------------------------------------|
| status    | string | Overall health: "healthy" or "unhealthy" |
| timestamp | string | Check timestamp (ISO 8601 UTC)           |
| checks    | object | Individual component statuses            |

**Degraded Response:** `200 OK` (when Redis is unavailable)

```json
{
  "status": "healthy",
  "timestamp": "2026-06-18T12:34:56Z",
  "checks": {
    "database": "ok",
    "redis": "unavailable"
  }
}
```

**Note:** Service continues operating without Redis (cache gracefully degrades to database-only).

**Error Response:** `503 Service Unavailable` (when PostgreSQL is down)

```json
{
  "status": "unhealthy",
  "timestamp": "2026-06-18T12:34:56Z",
  "checks": {
    "database": "error",
    "redis": "ok"
  }
}
```

**Example:**

```bash
curl http://localhost:8080/health
```

---

### Prometheus Metrics

Expose Prometheus-compatible metrics for monitoring.

**Endpoint:** `GET /metrics`

**Success Response:** `200 OK` (text/plain)

```
# HELP zaplink_http_requests_total Total HTTP requests
# TYPE zaplink_http_requests_total counter
zaplink_http_requests_total{method="GET",route="/links/{short_code}",status="200"} 1523

# HELP zaplink_http_request_duration_seconds HTTP request duration
# TYPE zaplink_http_request_duration_seconds histogram
zaplink_http_request_duration_seconds_bucket{le="0.005"} 1234
zaplink_http_request_duration_seconds_bucket{le="0.01"} 1456
zaplink_http_request_duration_seconds_bucket{le="0.025"} 1489
...

# HELP zaplink_links_created_total Total links created
# TYPE zaplink_links_created_total counter
zaplink_links_created_total 428

# HELP zaplink_redirects_total Total redirects served
# TYPE zaplink_redirects_total counter
zaplink_redirects_total 15234

# HELP zaplink_clicks_tracked_total Total clicks persisted
# TYPE zaplink_clicks_tracked_total counter
zaplink_clicks_tracked_total 15198
```

**Available Metrics:**

| Metric Name                            | Type      | Labels                      | Description                      |
|----------------------------------------|-----------|-----------------------------|----------------------------------|
| `zaplink_http_requests_total`          | Counter   | method, route, status       | Total HTTP requests              |
| `zaplink_http_request_duration_seconds`| Histogram | -                           | Request duration distribution    |
| `zaplink_links_created_total`          | Counter   | -                           | Total links created              |
| `zaplink_redirects_total`              | Counter   | -                           | Total redirects served           |
| `zaplink_clicks_tracked_total`         | Counter   | -                           | Total clicks persisted to DB     |

**Example:**

```bash
curl http://localhost:8080/metrics
```

**Note:** This endpoint is public (no authentication). Consider restricting access in production.

---

## Error Responses

All error responses follow this format:

```json
{
  "error": {
    "code": "error_code",
    "message": "Human-readable error message"
  }
}
```

### Common Error Codes

| HTTP Status | Error Code         | Description                                 |
|-------------|--------------------|---------------------------------------------|
| 400         | `invalid_url`      | URL format is invalid or empty              |
| 400         | `invalid_request`  | Request body is malformed or missing fields |
| 404         | `not_found`        | Short code does not exist                   |
| 404         | `link_inactive`    | Link exists but is marked inactive          |
| 500         | `db_error`         | Database operation failed                   |
| 500         | `internal_error`   | Unexpected server error                     |

**Example Error Response:**

```bash
curl -X POST http://localhost:8080/links \
  -H "Content-Type: application/json" \
  -d '{"url": "not-a-valid-url"}'
```

```json
{
  "error": {
    "code": "invalid_url",
    "message": "invalid url format"
  }
}
```

---

## Examples

### JavaScript (fetch)

```javascript
// Create short link
async function createShortLink(originalUrl) {
  const response = await fetch('http://localhost:8080/links', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ url: originalUrl })
  });
  
  if (!response.ok) {
    const error = await response.json();
    throw new Error(error.error.message);
  }
  
  return await response.json();
}

// Usage
createShortLink('https://example.com/long/url')
  .then(data => console.log('Short URL:', data.short_url))
  .catch(err => console.error('Error:', err));
```

### Python (requests)

```python
import requests

# Create short link
def create_short_link(original_url):
    response = requests.post(
        'http://localhost:8080/links',
        json={'url': original_url}
    )
    response.raise_for_status()
    return response.json()

# Get link info
def get_link_info(short_code):
    response = requests.get(f'http://localhost:8080/links/{short_code}')
    response.raise_for_status()
    return response.json()

# Usage
result = create_short_link('https://example.com/long/url')
print(f"Short URL: {result['short_url']}")

info = get_link_info(result['short_code'])
print(f"Clicks: {info['click_count']}")
```

### Go

```go
package main

import (
    "bytes"
    "encoding/json"
    "fmt"
    "net/http"
)

type CreateLinkRequest struct {
    URL string `json:"url"`
}

type CreateLinkResponse struct {
    ShortCode string `json:"short_code"`
    ShortURL  string `json:"short_url"`
}

func createShortLink(originalURL string) (*CreateLinkResponse, error) {
    reqBody, _ := json.Marshal(CreateLinkRequest{URL: originalURL})
    
    resp, err := http.Post(
        "http://localhost:8080/links",
        "application/json",
        bytes.NewBuffer(reqBody),
    )
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    
    var result CreateLinkResponse
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, err
    }
    
    return &result, nil
}

func main() {
    result, err := createShortLink("https://example.com/long/url")
    if err != nil {
        panic(err)
    }
    fmt.Printf("Short URL: %s\n", result.ShortURL)
}
```

---

## Authentication

⚠️ **Not implemented.** All endpoints are public.

Future consideration:
- API key authentication for link creation
- Admin endpoints with JWT
- Public read-only access for redirects

---

## CORS

Cross-Origin Resource Sharing (CORS) is **not configured** by default.

To enable CORS for frontend applications, add middleware in `internal/http/router.go`:

```go
import "github.com/go-chi/cors"

r.Use(cors.Handler(cors.Options{
    AllowedOrigins: []string{"https://your-frontend.com"},
    AllowedMethods: []string{"GET", "POST", "OPTIONS"},
    AllowedHeaders: []string{"Content-Type"},
}))
```

---

**For more details on architecture and implementation, see [ARCHITECTURE.md](ARCHITECTURE.md)**
