# ⚡ ZapLink

> Высокопроизводительный сервис сокращения URL на Go, Redis и PostgreSQL

[English](README.md) | [Русский](#русский)

---

## Русский

[![Go Version](https://img.shields.io/badge/Go-1.25-00ADD8?style=flat&logo=go)](https://golang.org)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Docker](https://img.shields.io/badge/docker-ready-blue?logo=docker)](https://www.docker.com/)

### ✨ Возможности

- **Высокая производительность**: Латентность p99 < 20мс для кешированных редиректов
- **Слоёная архитектура**: Чёткое разделение по паттерну Handler → Service → Repository
- **Кеширование Redis**: Стратегия write-through с настраиваемым TTL
- **Аналитика кликов**: Асинхронное отслеживание с логированием user-agent и IP
- **Готовность к продакшену**: Health checks, graceful shutdown, структурированное логирование
- **Prometheus метрики**: Grafana дашборды, Loki логи
- **Типизированные ошибки**: Доменные ошибки с маппингом на HTTP статусы

### 🚀 Быстрый старт

```bash
# Клонировать и запустить через Docker Compose (полный стек)
git clone https://github.com/ekideno/zaplink.git
cd zaplink
cp .env.example .env
docker compose up -d

# Сервис доступен на http://localhost:8080
# Grafana дашборд на http://localhost:3000 (admin/admin)
```

**Создать короткую ссылку:**
```bash
curl -X POST http://localhost:8080/links \
  -H "Content-Type: application/json" \
  -d '{"url": "https://github.com/ekideno/zaplink"}'

# Ответ: {"short_code":"abc123","short_url":"http://localhost:8080/abc123"}
```

**Редирект:**
```bash
curl -I http://localhost:8080/abc123
# HTTP/1.1 302 Found
# Location: https://github.com/ekideno/zaplink
```

### 🏗️ Архитектура

```
┌─────────────┐
│   Клиент    │
└──────┬──────┘
       │
┌──────▼──────────────────────────────────────┐
│          HTTP Handler слой                  │
│  (валидация, JSON маршалинг, роутинг)       │
└──────┬──────────────────────────────────────┘
       │
┌──────▼──────────────────────────────────────┐
│          Service слой                       │
│  (бизнес-логика, генерация кодов)           │
└──────┬──────────────────┬───────────────────┘
       │                  │
┌──────▼──────┐    ┌──────▼──────┐
│  PostgreSQL │    │    Redis    │
│(постоянное) │    │    (кеш)    │
└─────────────┘    └─────────────┘
```

**Ключевые архитектурные решения:**
- Repository НЕ управляет кешированием (оркестрация в Service)
- Трекинг кликов асинхронный (не блокирует редиректы)
- Кеш gracefully деградирует при недоступности Redis
- Типизированные ошибки различают бизнес-ошибки и технические

См. [ARCHITECTURE.ru.md](docs/ARCHITECTURE.ru.md) для детальных диаграмм.

### 🛠️ Технологический стек

| Компонент       | Технология                                      |
|-----------------|-------------------------------------------------|
| Язык            | Go 1.25                                         |
| HTTP Router     | [chi](https://github.com/go-chi/chi)            |
| База данных     | PostgreSQL 17                                   |
| Кеш             | Redis 7                                         |
| Миграции        | [golang-migrate](https://github.com/golang-migrate/migrate) |
| Метрики         | Prometheus + Grafana                            |
| Логирование     | slog + Loki + Promtail                          |
| Контейнеризация | Docker + Docker Compose                         |

### 📊 Производительность

| Метрика                      | Значение       |
|------------------------------|----------------|
| Латентность редиректа (кеш)  | p99 < 20мс     |
| Латентность редиректа (БД)   | p99 < 100мс    |
| Пропускная способность       | 10k+ RPS       |
| Cache hit rate               | ~95% (типично) |

Бенчмарки на MacBook Pro M1, 16GB RAM, локальный Docker.

### 📚 Документация

- **[API Reference](docs/API.ru.md)** - Спецификация эндпоинтов, примеры запросов/ответов
- **[Архитектура](docs/ARCHITECTURE.ru.md)** - Системный дизайн, потоки данных, паттерны
- **[Гайд разработчика](docs/DEVELOPMENT.ru.md)** - Структура проекта, тестирование, отладка
- **[Observability](docs/OBSERVABILITY.ru.md)** - Prometheus метрики, Grafana дашборды, логи
- **[Деплоймент](docs/DEPLOYMENT.ru.md)** - Production setup, CI/CD, миграции

### 🧪 Разработка

```bash
# Установить зависимости
go mod download

# Применить миграции
make up

# Запустить тесты
go test ./...
```


### 📦 Структура проекта

```
zaplink/
├── cmd/api/              # Точка входа приложения
├── internal/
│   ├── apperror/         # Типизированные ошибки
│   ├── cache/            # Redis кеш интерфейс
│   ├── config/           # Управление конфигурацией
│   ├── http/
│   │   ├── handler/      # HTTP хендлеры
│   │   └── middleware/   # Логирование, метрики, recovery
│   ├── metrics/          # Prometheus метрики
│   ├── repository/       # PostgreSQL data access
│   └── service/          # Бизнес-логика
├── migrations/           # SQL миграции
├── docs/                 # Документация
└── docker-compose.yml    # Полный стек
```


### 🐳 Docker сервисы

```bash
docker compose up -d  # Запустить все сервисы

# Доступные сервисы:
# - app (Go API)         :8080
# - postgres (БД)        :5432
# - redis (Кеш)          :6379
# - prometheus           :9090
# - grafana              :3000
# - loki (Логи)          :3100
```


---

### 🎯 Учебные цели

Этот проект демонстрирует:
- ✅ Clean Architecture / Layered Architecture паттерн
- ✅ Interface-driven дизайн для тестируемости
- ✅ Стратегии кеширования Redis (write-through)
- ✅ Асинхронная обработка (фоновый трекинг кликов)
- ✅ Production observability (метрики, логи, трейсинг)
- ✅ Управление миграциями БД
- ✅ Docker мульти-сервисная оркестрация
- ✅ RESTful API дизайн
- ✅ Паттерны обработки ошибок
- ✅ Unit-тестирование с моками

### 📈 Roadmap

- [ ] Добавить rate limiting (token bucket)
- [ ] Реализовать кастомные короткие коды
- [ ] Добавить истечение ссылок
- [ ] GraphQL API
- [ ] Distributed tracing (OpenTelemetry)
- [ ] Kubernetes deployment манифесты
- [ ] Admin UI дашборд
- [ ] Analytics API (статистика кликов, геолокация)

---

**Поставьте ⭐ если проект полезен!**
