# ⚡ ZapLink

> High-performance URL shortener built with Go, Redis, and PostgreSQL

[English](#english) | [Русский](README.ru.md)

---

## English

[![Go Version](https://img.shields.io/badge/Go-1.25-00ADD8?style=flat&logo=go)](https://golang.org)
![Tests](https://github.com/<owner>/<repo>/actions/workflows/tests.yml/badge.svg)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Docker](https://img.shields.io/badge/docker-ready-blue?logo=docker)](https://www.docker.com/)

### ✨ Features

- **High Performance**: Sub-20ms p99 latency for cached redirects
- **Layered Architecture**: Clean separation with Handler → Service → Repository pattern
- **Redis Caching**: Write-through cache strategy with configurable TTL
- **Click Analytics**: Async click tracking with user-agent and IP logging
- **Observability Ready**: Prometheus metrics, Grafana dashboards, Loki logs
- **Production Grade**: Health checks, graceful shutdown, structured logging
- **Type-Safe Errors**: Domain-specific error handling with HTTP status mapping

### 🚀 Quick Start

```bash
# Clone and start with Docker Compose (full stack)
git clone https://github.com/ekideno/zaplink.git
cd zaplink
cp .env.example .env
docker compose up -d

# Service available at http://localhost:8080
# Grafana dashboard at http://localhost:3000 (admin/admin)
```

**Create a short link:**
```bash
curl -X POST http://localhost:8080/links \
  -H "Content-Type: application/json" \
  -d '{"url": "https://github.com/ekideno/zaplink"}'

# Response: {"short_code":"abc123","short_url":"http://localhost:8080/abc123"}
```

**Redirect:**
```bash
curl -I http://localhost:8080/abc123
# HTTP/1.1 302 Found
# Location: https://github.com/ekideno/zaplink
```

### 🏗️ Architecture

```
┌─────────────┐
│   Client    │
└──────┬──────┘
       │
┌──────▼──────────────────────────────────────┐
│           HTTP Handler Layer                │
│  (validation, JSON marshaling, routing)     │
└──────┬──────────────────────────────────────┘
       │
┌──────▼──────────────────────────────────────┐
│           Service Layer                     │
│  (business logic, short code generation)    │
└──────┬──────────────────┬───────────────────┘
       │                  │
┌──────▼──────┐    ┌──────▼──────┐
│  PostgreSQL │    │    Redis    │
│ (persistent)│    │   (cache)   │
└─────────────┘    └─────────────┘
```

**Key Design Decisions:**
- Repository layer does NOT handle caching (service orchestrates both)
- Click tracking is async (doesn't block redirects)
- Cache gracefully degrades if Redis is unavailable
- Typed errors distinguish business vs technical failures

See [ARCHITECTURE.md](docs/ARCHITECTURE.md) for detailed flow diagrams.

### 🛠️ Tech Stack

| Component       | Technology                                      |
|-----------------|-------------------------------------------------|
| Language        | Go 1.25                                         |
| HTTP Router     | [chi](https://github.com/go-chi/chi)            |
| Database        | PostgreSQL 17                                   |
| Cache           | Redis 7                                         |
| Migrations      | [golang-migrate](https://github.com/golang-migrate/migrate) |
| Metrics         | Prometheus + Grafana                            |
| Logging         | slog + Loki + Promtail                          |
| Containerization| Docker + Docker Compose                         |

### 📊 Performance

| Metric                  | Value          |
|-------------------------|----------------|
| Redirect latency (cached) | p99 < 20ms   |
| Redirect latency (DB)   | p99 < 100ms    |
| Throughput              | 10k+ RPS       |
| Cache hit rate          | ~95% (typical) |

Benchmarks run on MacBook Pro M1, 16GB RAM, local Docker.

### 📚 Documentation

- **[API Reference](docs/API.md)** - Endpoint specs, request/response examples
- **[Architecture](docs/ARCHITECTURE.md)** - System design, data flow, patterns
- **[Development Guide](docs/DEVELOPMENT.md)** - Project structure, testing, debugging
- **[Observability](docs/OBSERVABILITY.md)** - Prometheus metrics, Grafana dashboards, logs
- **[Deployment](docs/DEPLOYMENT.md)** - Production setup, CI/CD, migrations

### 🧪 Development

```bash
# Install dependencies
go mod download

# Run migrations
make up

# Run tests
go test ./...
```


### 📦 Project Structure

```
zaplink/
├── cmd/api/              # Application entry point
├── internal/
│   ├── apperror/         # Typed error definitions
│   ├── cache/            # Redis cache interface
│   ├── config/           # Configuration management
│   ├── http/
│   │   ├── handler/      # HTTP handlers
│   │   └── middleware/   # Logging, metrics, recovery
│   ├── metrics/          # Prometheus metrics
│   ├── repository/       # PostgreSQL data access
│   └── service/          # Business logic
├── migrations/           # SQL migrations
├── docs/                 # Documentation
└── docker-compose.yml    # Full stack setup
```


### 🐳 Docker Services

```bash
docker compose up -d  # Start all services

# Available services:
# - app (Go API)         :8080
# - postgres (DB)        :5432
# - redis (Cache)        :6379
# - prometheus           :9090
# - grafana              :3000
# - loki (Logs)          :3100
```


---

### 🎯 Learning Objectives

This project demonstrates:
- ✅ Clean Architecture / Layered Architecture pattern
- ✅ Interface-driven design for testability
- ✅ Redis caching strategies (write-through)
- ✅ Async processing (background click tracking)
- ✅ Production observability (metrics, logs, tracing)
- ✅ Database migrations management
- ✅ Docker multi-service orchestration
- ✅ RESTful API design
- ✅ Error handling patterns
- ✅ Unit testing with mocks

### 📈 Roadmap

- [ ] Add rate limiting (token bucket)
- [ ] Implement custom short codes
- [ ] Add link expiration
- [ ] GraphQL API
- [ ] Distributed tracing (OpenTelemetry)
- [ ] Kubernetes deployment manifests
- [ ] Admin dashboard UI
- [ ] Analytics API (click stats, geo location)

---

**Star ⭐ this repo if you find it useful!**
