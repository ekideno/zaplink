# Observability Guide

[English](#english) | [Русский](OBSERVABILITY.ru.md)

---

## English

Complete guide to monitoring, logging, and observability in ZapLink.

## Table of Contents

- [Overview](#overview)
- [Architecture](#architecture)
- [Prometheus Metrics](#prometheus-metrics)
- [Grafana Dashboards](#grafana-dashboards)
- [Loki Logging](#loki-logging)
- [Setup Instructions](#setup-instructions)
- [Querying and Alerting](#querying-and-alerting)
- [Troubleshooting](#troubleshooting)

---

## Overview

ZapLink includes a **full observability stack** for monitoring application health, performance, and behavior:

- **Prometheus** - Metrics collection and storage
- **Grafana** - Visualization and dashboards
- **Loki** - Log aggregation
- **Promtail** - Log shipping from Docker containers

**Stack Architecture:**

```
┌─────────────┐
│  ZapLink    │ (Go app on :8080)
│   App       │
└──┬────────┬─┘
   │        │
   │ /metrics │ stdout logs
   │        │
   ▼        ▼
┌──────┐  ┌───────────┐
│Prome-│  │ Promtail  │ (Docker log collector)
│theus │  └─────┬─────┘
│:9090 │        │
└──┬───┘        │
   │            ▼
   │       ┌────────┐
   │       │  Loki  │ (Log aggregation)
   │       │ :3100  │
   │       └────┬───┘
   │            │
   └────────────┼─────────┐
                │         │
                ▼         ▼
           ┌──────────────────┐
           │     Grafana      │ (Visualization)
           │      :3000       │
           └──────────────────┘
```

---

## Architecture

### Components

| Component  | Purpose                          | Port | URL                        |
|------------|----------------------------------|------|----------------------------|
| Prometheus | Scrape and store metrics         | 9090 | http://localhost:9090      |
| Grafana    | Dashboards and visualization     | 3000 | http://localhost:3000      |
| Loki       | Log aggregation and querying     | 3100 | http://localhost:3100      |
| Promtail   | Collect Docker logs              | 9080 | (internal)                 |
| ZapLink    | Application metrics + logs       | 8080 | http://localhost:8080/metrics |

### Data Flow

**Metrics:**
1. ZapLink exposes `/metrics` endpoint (Prometheus format)
2. Prometheus scrapes every 15 seconds
3. Prometheus stores time-series data
4. Grafana queries Prometheus for dashboard visualization

**Logs:**
1. ZapLink writes structured logs to stdout (slog)
2. Docker captures stdout as container logs
3. Promtail reads Docker logs and ships to Loki
4. Loki stores logs with labels (service, container, level)
5. Grafana queries Loki for log exploration

---

## Prometheus Metrics

### Available Metrics

ZapLink exposes 5 core business and infrastructure metrics:

#### 1. HTTP Request Counter

**Name:** `zaplink_http_requests_total`  
**Type:** Counter  
**Labels:** `method`, `route`, `status`  
**Description:** Total number of HTTP requests processed

**Example values:**
```
zaplink_http_requests_total{method="GET",route="/{short_code}",status="302"} 15234
zaplink_http_requests_total{method="POST",route="/links",status="201"} 428
zaplink_http_requests_total{method="GET",route="/health",status="200"} 892
```

**Use cases:**
- Request rate per endpoint
- Error rate (status >= 400)
- Traffic distribution

---

#### 2. HTTP Request Duration

**Name:** `zaplink_http_request_duration_seconds`  
**Type:** Histogram  
**Labels:** `method`, `route`  
**Buckets:** `[0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10]` (seconds)  
**Description:** HTTP request latency distribution

**Example histogram:**
```
zaplink_http_request_duration_seconds_bucket{method="GET",route="/{short_code}",le="0.005"} 1234
zaplink_http_request_duration_seconds_bucket{method="GET",route="/{short_code}",le="0.01"} 1456
zaplink_http_request_duration_seconds_bucket{method="GET",route="/{short_code}",le="0.025"} 1489
zaplink_http_request_duration_seconds_sum{method="GET",route="/{short_code}"} 18.45
zaplink_http_request_duration_seconds_count{method="GET",route="/{short_code}"} 1523
```

**Use cases:**
- p50, p95, p99 latency calculation
- SLO monitoring (e.g., "95% of requests < 100ms")
- Performance regression detection

---

#### 3. Links Created Counter

**Name:** `zaplink_links_created_total`  
**Type:** Counter  
**Labels:** (none)  
**Description:** Total number of short links successfully created

**Example value:**
```
zaplink_links_created_total 428
```

**Use cases:**
- Link creation rate (links/second)
- Business growth tracking
- Capacity planning

---

#### 4. Redirects Served Counter

**Name:** `zaplink_redirects_total`  
**Type:** Counter  
**Labels:** (none)  
**Description:** Total number of redirects served (successful /{short_code} requests)

**Example value:**
```
zaplink_redirects_total 15234
```

**Use cases:**
- Redirect rate (redirects/second)
- Cache effectiveness (compare with DB queries)
- User engagement tracking

---

#### 5. Clicks Tracked Counter

**Name:** `zaplink_clicks_tracked_total`  
**Type:** Counter  
**Labels:** (none)  
**Description:** Total number of clicks successfully persisted to database

**Example value:**
```
zaplink_clicks_tracked_total 15198
```

**Use cases:**
- Click tracking success rate (compare with `redirects_total`)
- Data loss detection (if tracking fails)
- Analytics pipeline health

**Note:** This counter increments asynchronously. Slight lag or loss is expected.

---

### Prometheus Configuration

**File:** `prometheus.yml`

```yaml
global:
  scrape_interval: 15s      # Scrape every 15 seconds
  scrape_timeout: 10s       # Timeout if scrape takes > 10s

scrape_configs:
  - job_name: "zaplink"
    metrics_path: /metrics
    static_configs:
      - targets:
          - "host.docker.internal:8080"  # ZapLink app
```

**Key settings:**
- `scrape_interval: 15s` - Balances freshness vs storage cost
- `host.docker.internal` - Docker Desktop DNS for host machine

---

## Grafana Dashboards

### Setup

1. **Access Grafana:**  
   Open http://localhost:3000  
   Login: `admin` / `admin` (change on first login)

2. **Add Prometheus Data Source:**
   - Go to **Configuration** → **Data Sources** → **Add data source**
   - Select **Prometheus**
   - URL: `http://prometheus:9090`
   - Click **Save & Test**

3. **Add Loki Data Source:**
   - Go to **Configuration** → **Data Sources** → **Add data source**
   - Select **Loki**
   - URL: `http://loki:3100`
   - Click **Save & Test**

---

### Recommended Dashboard Panels

#### Panel 1: Request Rate

**Type:** Graph (Time series)  
**Query:**
```promql
rate(zaplink_http_requests_total[1m])
```
**Legend:** `{{method}} {{route}} {{status}}`

Shows requests per second over time, broken down by endpoint.

---

#### Panel 2: Error Rate

**Type:** Graph (Time series)  
**Query:**
```promql
sum(rate(zaplink_http_requests_total{status=~"5.."}[1m]))
```
**Legend:** `5xx errors/sec`

Tracks server errors (500, 503, etc.)

---

#### Panel 3: p99 Latency

**Type:** Graph (Time series)  
**Query:**
```promql
histogram_quantile(0.99, 
  rate(zaplink_http_request_duration_seconds_bucket[1m])
)
```
**Legend:** `p99 latency`

Shows 99th percentile request duration (SLO monitoring).

---

#### Panel 4: Redirect Performance

**Type:** Stat (Single value)  
**Query:**
```promql
histogram_quantile(0.99, 
  rate(zaplink_http_request_duration_seconds_bucket{route="/{short_code}"}[5m])
)
```
**Unit:** seconds  
**Thresholds:** Green < 0.05, Yellow < 0.1, Red >= 0.1

Highlights redirect latency (main performance metric).

---

#### Panel 5: Links Created Rate

**Type:** Graph (Time series)  
**Query:**
```promql
rate(zaplink_links_created_total[5m])
```
**Legend:** `links/sec`

Business metric: link creation velocity.

---

#### Panel 6: Click Tracking Loss

**Type:** Stat (Single value)  
**Query:**
```promql
(zaplink_redirects_total - zaplink_clicks_tracked_total) / zaplink_redirects_total * 100
```
**Unit:** percent  
**Thresholds:** Green < 1%, Yellow < 5%, Red >= 5%

Tracks data loss in async click tracking.

---

#### Panel 7: Traffic by Status Code

**Type:** Pie chart  
**Query:**
```promql
sum by (status) (zaplink_http_requests_total)
```

Visual breakdown of 2xx, 3xx, 4xx, 5xx responses.

---

### Example Dashboard JSON

Save this to import a pre-built dashboard:

```json
{
  "dashboard": {
    "title": "ZapLink Monitoring",
    "panels": [
      {
        "title": "Request Rate",
        "targets": [
          {
            "expr": "rate(zaplink_http_requests_total[1m])",
            "legendFormat": "{{method}} {{route}} {{status}}"
          }
        ],
        "type": "graph"
      },
      {
        "title": "p99 Latency",
        "targets": [
          {
            "expr": "histogram_quantile(0.99, rate(zaplink_http_request_duration_seconds_bucket[1m]))"
          }
        ],
        "type": "graph"
      }
    ]
  }
}
```

---

## Loki Logging

### Log Format

ZapLink uses **structured logging** with Go's `slog` package:

**Example log output:**
```json
{
  "time": "2026-06-18T12:34:56.789Z",
  "level": "INFO",
  "msg": "HTTP request",
  "method": "GET",
  "path": "/abc123",
  "status": 302,
  "duration_ms": 12.4,
  "user_agent": "Mozilla/5.0...",
  "ip": "192.168.1.100"
}
```

**Log levels:**
- `DEBUG` - Detailed diagnostic information
- `INFO` - Normal operation events (HTTP requests)
- `WARN` - Cache failures, degraded mode
- `ERROR` - Database errors, click tracking failures

---

### Promtail Configuration

**File:** `promtail.yml`

```yaml
server:
  http_listen_port: 9080
  grpc_listen_port: 0

clients:
  - url: http://loki:3100/loki/api/v1/push

scrape_configs:
  - job_name: docker
    docker_sd_configs:
      - host: unix:///var/run/docker.sock
        refresh_interval: 5s

    relabel_configs:
      - source_labels: [__meta_docker_container_name]
        target_label: container
      - source_labels: [__meta_docker_container_label_com_docker_compose_service]
        target_label: service
      - source_labels: [__meta_docker_container_label_com_docker_compose_project]
        target_label: project

    pipeline_stages:
      - cri: {}  # Parse CRI-formatted logs
```

**Key features:**
- Auto-discovers Docker containers
- Extracts labels: `container`, `service`, `project`
- Ships logs to Loki every 5 seconds

---

### Querying Logs in Grafana

**Access:** Grafana → Explore → Select "Loki" data source

#### Example Queries

**1. All logs from ZapLink app:**
```logql
{service="app"}
```

**2. Only ERROR level logs:**
```logql
{service="app"} |= "ERROR"
```

**3. Failed cache operations:**
```logql
{service="app"} |= "cache" |= "failed"
```

**4. Requests to specific short code:**
```logql
{service="app"} |= "abc123"
```

**5. Slow requests (> 100ms):**
```logql
{service="app"} | json | duration_ms > 100
```

**6. 5xx errors:**
```logql
{service="app"} | json | status >= 500
```

---

### Log Labels

Promtail automatically adds these labels:

| Label     | Example Value      | Source                          |
|-----------|--------------------|----------------------------------|
| container | `zaplink-app`      | Docker container name            |
| service   | `app`              | Docker Compose service name      |
| project   | `zaplink`          | Docker Compose project name      |
| level     | `INFO`, `ERROR`    | Parsed from log JSON             |

**Filter by label:**
```logql
{service="app", level="ERROR"}
```

---

## Setup Instructions

### Start Full Stack

```bash
# Start all observability services
docker compose up -d prometheus grafana loki promtail

# Check service health
docker compose ps

# View logs
docker compose logs -f grafana
```

### Verify Metrics Endpoint

```bash
curl http://localhost:8080/metrics
```

Expected output:
```
# HELP zaplink_http_requests_total Total number of HTTP requests
# TYPE zaplink_http_requests_total counter
zaplink_http_requests_total{method="GET",route="/health",status="200"} 5
...
```

### Verify Prometheus Scraping

1. Open http://localhost:9090
2. Go to **Status** → **Targets**
3. Check that `zaplink` target is **UP**

### Verify Loki Ingestion

1. Open Grafana → Explore
2. Select Loki data source
3. Query: `{service="app"}`
4. Should see recent logs

---

## Querying and Alerting

### Useful PromQL Queries

**Request rate (last 5 minutes):**
```promql
rate(zaplink_http_requests_total[5m])
```

**Average latency:**
```promql
rate(zaplink_http_request_duration_seconds_sum[5m]) 
/ 
rate(zaplink_http_request_duration_seconds_count[5m])
```

**Error rate (percentage):**
```promql
sum(rate(zaplink_http_requests_total{status=~"5.."}[5m])) 
/ 
sum(rate(zaplink_http_requests_total[5m])) * 100
```

**Cache hit rate (estimated):**
```promql
rate(zaplink_redirects_total[5m]) / 
(rate(zaplink_redirects_total[5m]) + rate(postgres_queries[5m]))
```

---

### Alerting Rules (Example)

Create `alerts.yml` and mount in Prometheus:

```yaml
groups:
  - name: zaplink_alerts
    interval: 30s
    rules:
      - alert: HighErrorRate
        expr: |
          sum(rate(zaplink_http_requests_total{status=~"5.."}[1m])) 
          / sum(rate(zaplink_http_requests_total[1m])) > 0.05
        for: 2m
        annotations:
          summary: "High error rate (> 5%)"
        
      - alert: HighLatency
        expr: |
          histogram_quantile(0.99, 
            rate(zaplink_http_request_duration_seconds_bucket[1m])
          ) > 0.5
        for: 5m
        annotations:
          summary: "p99 latency > 500ms"
      
      - alert: ClickTrackingDegraded
        expr: |
          (zaplink_redirects_total - zaplink_clicks_tracked_total) 
          / zaplink_redirects_total > 0.1
        for: 5m
        annotations:
          summary: "Click tracking loss > 10%"
```

---

## Troubleshooting

### Prometheus Not Scraping

**Symptom:** Targets show as DOWN in Prometheus UI

**Check:**
```bash
# From inside Prometheus container
docker exec zaplink-prometheus wget -O- http://host.docker.internal:8080/metrics
```

**Fix:** Ensure ZapLink app is running on port 8080

---

### Grafana Cannot Connect to Prometheus

**Symptom:** "Bad Gateway" error in data source

**Fix:** Use `http://prometheus:9090` (Docker network name), not `localhost`

---

### No Logs in Loki

**Symptom:** Empty results in Grafana Explore

**Check Promtail:**
```bash
docker logs zaplink-promtail
```

**Common issues:**
- Promtail cannot access Docker socket → mount `/var/run/docker.sock`
- Label mismatch → check `{service="app"}` exists

---

### Metrics Not Incrementing

**Symptom:** Counters stuck at 0

**Check:**
1. Are metrics registered? (`metrics.Register()` called in `main.go`)
2. Are handlers incrementing metrics? (e.g., `metrics.LinksCreatedTotal.Inc()`)
3. Check for typos in metric names

---

## Best Practices

1. **Set up alerts early** - Don't wait for incidents
2. **Monitor SLOs** - Define p99 latency and error rate targets
3. **Use labels wisely** - Avoid high-cardinality labels (e.g., user IDs)
4. **Dashboard hygiene** - Delete unused panels, organize by priority
5. **Log sampling** - In high-traffic production, sample verbose logs (DEBUG level)
6. **Retention policies** - Configure Prometheus retention (default: 15 days)

---

**For architecture details, see [ARCHITECTURE.md](ARCHITECTURE.md)**  
**For API documentation, see [API.md](API.md)**
