# ZapLink - Проблемы и TODO

Дата первичного анализа: 2026-06-07
Дата code review (ошибки, логи, Redis): 2026-06-16

## 🔴 Критические проблемы

### 1. Отсутствие бизнес-логики
**Пакет**: `internal/service/`, `internal/http/handler/`

- `internal/service/service.go` — полностью пустой файл (2 строки)
- `internal/http/handler/handler.go` — полностью пустой файл (1 строка)
- Нет реализации core-функциональности: создание коротких ссылок, редирект, статистика

**Приоритет**: P0  
**Блокирует**: Весь функционал приложения

### 2. Broken dependency injection
**Файл**: `cmd/api/main.go:52-60`

```go
database, err := db.NewPostgres(connectCtx, cfg.DB.URL)
// ...
srv := &http.Server{
    Handler: apphttp.NewRouter(log), // ❌ database не передается
}
```

- Database создается, но не передается в router/handlers
- Handlers не имеют доступа к слою данных
- Невозможно выполнить ни один запрос к БД

**Приоритет**: P0  
**Блокирует**: Интеграция слоев приложения

### 3. Отсутствие пакета для typed errors
**Пакет**: `internal/apperror/` (не создан)

- Упоминается в CLAUDE.md и AGENTS.md
- Необходим для правильной обработки ошибок (404, validation, DB failures)
- Без него невозможно отличить business errors от technical errors

**Приоритет**: P1  
**Блокирует**: Proper error handling в handlers

---

## ⚠️ Важные проблемы

### 4. Дублирование middleware для логирования
**Файл**: `internal/http/router.go:18-21`

```go
r.Use(middleware.Logger)           // ❌ Chi встроенный логгер
r.Use(appmiddleware.RequestLogger(log)) // ❌ Наш кастомный логгер
```

- Каждый запрос логируется дважды
- Избыточный overhead
- Разные форматы логов

**Приоритет**: P2  
**Решение**: Удалить `middleware.Logger`, оставить только `RequestLogger`

### 5. Некорректная обработка UpdateLink
**Файл**: `internal/repository/postgres_links.go:72-90`

```go
func (r *LinkPostgresRepository) UpdateLink(ctx context.Context, link *model.Link) error {
    // ...
    if ct.RowsAffected() == 0 {
        return nil // ❌ Возвращает nil даже если запись не найдена
    }
    return nil
}
```

- Не отличить "запись обновлена" от "запись не найдена"
- Service layer не может определить успешность операции

**Приоритет**: P2  
**Решение**: Вернуть typed error если RowsAffected == 0

### 6. Неоптимальное использование sql.NullString
**Файл**: `internal/repository/postgres_clicks.go:5, 26-34`

```go
import "database/sql" // Только для NullString

var userAgent sql.NullString
var ipAddress sql.NullString
```

- Импорт `database/sql` только для NullString
- pgx имеет собственный `pgtype` пакет для nullable типов
- Лишняя зависимость

**Приоритет**: P3  
**Решение**: Использовать `pgtype.Text` или передавать pointer напрямую в Scan

---

## 📋 Отсутствующие компоненты

### 7. Нет HTTP DTO структур
**Пакет**: `internal/http/dto/` (не создан)

- Нет request/response моделей для API
- Domain models (`model.Link`, `model.Click`) не должны напрямую использоваться в HTTP слое
- Необходимы для validation и versioning API

**Приоритет**: P1

### 8. Redis/Cache layer не реализован
**Упоминается в**: CLAUDE.md, AGENTS.md

- Нет пакета `internal/cache/`
- Нет Redis конфигурации в `config.Config`
- Для high-performance URL shortener критичен кеш hot URLs

**Приоритет**: P2

### 9. Отсутствуют Service интерфейсы
**Пакет**: `internal/service/`

- Repository имеют интерфейсы (`LinkRepository`, `ClickRepository`)
- Service layer не имеет интерфейсов
- Усложняет mock для тестирования handlers

**Приоритет**: P2

### 10. Нет генератора short_code
**Требуется в**: `internal/service/`

- Основная бизнес-логика URL shortener
- Нужен алгоритм генерации уникальных коротких кодов
- Варианты: base62, nanoid, custom alphabet

**Приоритет**: P0

---

## 🧪 Тестирование

### 11. Полное отсутствие тестов
**Статус**: 0 тестов во всем проекте

```bash
go test ./...
# [no test files] для всех пакетов
```

- Нет unit тестов
- Нет integration тестов
- Нет mock-реализаций интерфейсов
- Невозможно проверить корректность кода

**Приоритет**: P1  
**Минимум**: Тесты для service layer и repository

---

## 🔧 Архитектурные улучшения

### 12. Отсутствие async click tracking
**Требование из**: CLAUDE.md "Click tracking must be async"

- Запись клика не должна блокировать redirect response
- Нужен worker pool или channel-based queue
- Критично для latency редиректов

**Приоритет**: P1

### 13. Нет production middleware
**Файл**: `internal/http/router.go`

Отсутствуют:
- Rate limiting (защита от abuse)
- CORS configuration
- Request size limits
- Security headers (X-Frame-Options, CSP, etc.)
- Compression middleware

**Приоритет**: P2 (для production)

### 14. Нет graceful shutdown для background tasks
**Файл**: `cmd/api/main.go`

- Graceful shutdown есть для HTTP server
- Нет механизма для корректной остановки async workers (когда они появятся)
- Могут потеряться click events при shutdown

**Приоритет**: P2

### 15. Отсутствие health check с DB ping
**Файл**: `internal/http/router.go:23-25`

```go
r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
    w.Write([]byte("OK")) // ❌ Не проверяет состояние DB
})
```

- Health endpoint не проверяет доступность БД
- K8s/Docker health checks не обнаружат проблемы с database

**Приоритет**: P2

---

## 🗃️ База данных

### 16. Миграции не применяются автоматически
**Статус**: Файлы миграций есть, но не используются

- В `cmd/api/main.go` нет запуска миграций
- Нет интеграции с `golang-migrate`
- Приходится применять вручную

**Приоритет**: P2  
**Решение**: Добавить auto-migration при старте или отдельный CLI command

### 17. Нет индекса на clicks.created_at
**Файл**: `migrations/000002_clicks.up.sql`

```sql
CREATE INDEX idx_clicks_link_id ON clicks(link_id);
-- Нет индекса на created_at
```

- Для аналитики по временным интервалам нужен индекс
- Запросы типа "клики за последний час" будут медленными

**Приоритет**: P3

---

## 📝 Документация и код-стиль

### 18. Дубликат AGENTS.md и CLAUDE.md
**Файлы**: `AGENTS.md`, `CLAUDE.md`

- Практически идентичный контент
- Упоминания "Codex" в AGENTS.md (copy-paste error)
- Нужен один source of truth

**Приоритет**: P3

### 19. Нет примеров использования API
**Статус**: Отсутствует документация API

- Нет OpenAPI/Swagger спецификации
- Нет примеров curl запросов
- Нет описания expected request/response formats

**Приоритет**: P2

---

## 📊 Приоритизация

### Must have (P0) - Блокируют MVP
1. ✅ Реализовать Service layer с генерацией short_code
2. ✅ Реализовать HTTP handlers (CreateLink, Redirect)
3. ✅ Подключить dependency injection (DB → Repository → Service → Handler)

### Should have (P1) - Критично для production
4. ✅ Создать пакет `internal/apperror/`
5. ✅ Создать HTTP DTO структуры
6. ✅ Async click tracking
7. ✅ Unit тесты для service layer

### Nice to have (P2) - Улучшения
8. ✅ Убрать дублирование middleware
9. ✅ Исправить UpdateLink behavior
10. ✅ Redis cache layer
11. ✅ Production middleware (rate limiting, CORS)
12. ✅ Health check с DB ping

### Low priority (P3) - Можно отложить
13. ✅ Оптимизация NullString → pgtype
14. ✅ Индекс на clicks.created_at
15. ✅ Почистить дублирующую документацию

---

## 🎯 Roadmap для MVP

```
Phase 1: Core functionality (P0)
├─ Service layer implementation
├─ HTTP handlers implementation
└─ Wire up dependencies

Phase 2: Production readiness (P1)
├─ Error handling (apperror package)
├─ DTO layer
├─ Async click tracking
└─ Basic tests

Phase 3: Scale & polish (P2)
├─ Redis cache
├─ Production middleware
├─ Auto migrations
└─ Comprehensive tests
```

**Текущий статус**: Phase 0 (Infrastructure ready, no business logic)  
**Estimated MVP**: ~2-3 дня работы для Phase 1-2

---

# Code Review - Работа с ошибками, логами, Redis

Дата ревью: 2026-06-16

## 🔴 Критические проблемы

### CR-1. Утечка файловых дескрипторов в логгере
**Файл:** `internal/logger/logger.go:26-71`

**Проблема:**
- Функция `New()` возвращает `file.Close` как cleanup-функцию, но `multiHandler` продолжает писать в уже закрытый файл
- При вызове cleanup-функции файл закрывается, но handler продолжает попытки записи
- Это приведет к ошибкам записи логов после shutdown

**Решение:**
- Реализовать правильный graceful shutdown для handler'а
- Добавить sync.Once или atomic flag для безопасного закрытия

---

### CR-2. Игнорирование ошибок при логировании
**Файл:** `internal/logger/logger.go:91-98`

**Проблема:**
```go
func (h *multiHandler) Handle(ctx context.Context, r slog.Record) error {
    var errs []error
    for _, handler := range h.handlers {
        if err := handler.Handle(ctx, r); err != nil {
            errs = append(errs, err)
        }
    }
    return errors.Join(errs...)  // ← возвращается, но никто не проверяет
}
```
- Ошибки собираются через `errors.Join()`, но никогда не проверяются
- При недоступности лог-файла приложение продолжит работу, не зная о проблеме
- Нет алертинга при сбоях логирования

**Решение:**
- Добавить fallback логирование в stderr при ошибках
- Логировать сами ошибки логирования
- Добавить метрики для мониторинга здоровья логирования

---

### CR-3. Дублирование middleware для логирования
**Файл:** `internal/http/router.go:18-22`

**Проблема:**
```go
r.Use(middleware.Logger)                     // Chi's default logger
r.Use(appmiddleware.RequestLogger(log))      // Custom logger
```
- Каждый HTTP-запрос логируется дважды
- Лишняя нагрузка на I/O и производительность
- Избыточность в логах

**Решение:**
- Убрать `middleware.Logger` из цепочки middleware

---

### CR-4. Игнорирование ошибок JSON encoding
**Файл:** `internal/http/handler/helpers.go:11-15`

**Проблема:**
```go
func writeJSON(w http.ResponseWriter, status int, v any) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    json.NewEncoder(w).Encode(v)  // ← ошибка игнорируется
}
```
- Если `Encode()` падает, клиент получит неполный JSON
- HTTP status уже отправлен, изменить его нельзя
- Нет логирования таких ситуаций

**Решение:**
- Проверять ошибку `Encode()` и логировать её
- Рассмотреть pre-encoding в память для критичных ответов

---

## 🟡 Серьезные проблемы

### CR-5. Race condition в асинхронном трекинге кликов
**Файл:** `internal/http/handler/link.go:82-91`

**Проблема:**
```go
go func() {
    if err := h.service.TrackClick(context.Background(), link.ID, r.UserAgent(), r.RemoteAddr); err != nil && h.log != nil {
        h.log.Error("track click failed", ...)
    }
}()
```

Проблемы:
- Используется `context.Background()` — горутина может висеть вечно при проблемах с БД
- Нет механизма graceful shutdown — при остановке сервера горутины будут прерваны
- При высокой нагрузке может создаться неограниченное количество горутин
- Потенциальная утечка горутин

**Решение:**
- Использовать context с таймаутом
- Реализовать worker pool с ограниченным количеством воркеров
- Или использовать очередь (channel) с буфером
- Добавить graceful shutdown для завершения pending задач

---

### CR-6. Неправильная обработка ошибок кэша
**Файл:** `internal/service/link_service.go:70-78`

**Проблема:**
```go
if s.cache != nil {
    link, err := s.cache.GetByShortCode(ctx, shortCode)
    if err == nil && link != nil {  // ← проблема здесь
        if !link.IsActive {
            return nil, ErrInactiveLink
        }
        return link, nil
    }
}
```

Проблемы:
- Любая ошибка Redis (сетевая, таймаут, ошибка десериализации) молча игнорируется
- Нет различия между cache miss и реальной ошибкой
- Нет метрик/логов для мониторинга здоровья кэша
- При падении Redis вся нагрузка полностью уйдет на PostgreSQL без предупреждения

**Решение:**
- Различать `ErrCacheMiss` и другие ошибки
- Логировать неожиданные ошибки Redis
- Добавить метрики cache hit/miss/error
- Рассмотреть circuit breaker pattern

---

### CR-7. Молчаливое игнорирование ошибок кэширования
**Файл:** `internal/service/link_service.go:92-94`

**Проблема:**
```go
if s.cache != nil {
    _ = s.cache.SetLink(ctx, link, s.cacheTTL)  // ← ошибка игнорируется
}
```
- Если Redis недоступен, данные не попадут в кэш
- Следующие запросы опять пойдут в БД
- Нет логирования — невозможно понять, что кэш не работает
- Degradation кэша остается незамеченной

**Решение:**
- Логировать ошибки SetLink
- Добавить метрики для мониторинга
- Рассмотреть алерты при высоком проценте ошибок

---

### CR-8. Отсутствие таймаутов для Redis операций
**Файлы:** `internal/cache/redis.go`, `cmd/api/main.go:63-71`

**Проблема:**
- Redis клиент создается без таймаутов на операции
- Медленный Redis может блокировать HTTP handlers
- Нет `DialTimeout`, `ReadTimeout`, `WriteTimeout` в `redis.Options`

**Решение:**
```go
redisClient := redis.NewClient(&redis.Options{
    Addr:         cfg.Redis.Addr,
    Password:     cfg.Redis.Password,
    DB:           cfg.Redis.DB,
    DialTimeout:  5 * time.Second,
    ReadTimeout:  3 * time.Second,
    WriteTimeout: 3 * time.Second,
})
```

---

### CR-9. Небезопасная проверка ошибки базы данных
**Файл:** `internal/service/link_service.go:150-153`

**Проблема:**
```go
func isUniqueViolation(err error) bool {
    var pgErr *pgconn.PgError
    return errors.As(err, &pgErr) && pgErr.Code == "23505"  // magic string
}
```
- Использование magic string `"23505"` вместо константы
- Может не сработать, если ошибка обернута неправильно

**Решение:**
- Использовать константу: `pgerrcode.UniqueViolation`
- Добавить тесты для проверки обработки wrapped errors

---

## 🟢 Средние проблемы

### CR-10. Отсутствие структурированного логирования ошибок
**Файл:** `internal/http/handler/helpers.go:17-43`

**Проблема:**
```go
log.Error("request failed",
    slog.String("method", r.Method),
    slog.String("path", r.URL.Path),
    slog.Int("status", appErr.Status),
    slog.String("code", appErr.Code),
    slog.String("error", err.Error()),  // ← теряется wrapped error
)
```
- При логировании оригинальной ошибки теряется stack trace
- Wrapped errors логируются как плоская строка
- Невозможно извлечь structured data из wrapped errors

**Решение:**
- Использовать `slog.Any("error", err)` для полной ошибки
- Добавить отдельное поле для wrapped error если есть

---

### CR-11. Inconsistency в error wrapping
**Файлы:** `internal/service/link_service.go`, `internal/repository/*.go`

**Проблема:**
- В сервисе: `fmt.Errorf("create link: %w", err)` — теряется HTTP контекст
- В репозитории: `fmt.Errorf("count clicks: %w", err)` — только техническое описание
- Нет единого стандарта, когда использовать `apperror.Wrap` vs `fmt.Errorf`
- Смешение подходов затрудняет обработку ошибок

**Решение:**
- Документировать стратегию error handling
- Repository слой: только `fmt.Errorf` с техническими деталями
- Service слой: конвертировать в `apperror` с HTTP статусами
- Handler слой: только обработка и логирование

---

### CR-12. Избыточное double-wrapping ошибок
**Файл:** `internal/service/link_service.go:40`

**Проблема:**
```go
var ErrInvalidURL = apperror.New(http.StatusBadRequest, "invalid_url", "invalid url")

// Позже в коде:
return nil, apperror.Wrapf(ErrInvalidURL, http.StatusBadRequest, "invalid_url", "must be http or https")
```
- `ErrInvalidURL` уже содержит status 400 и код "invalid_url"
- Wrapping создает дубликат информации
- Клиент получит тот же статус и код

**Решение:**
- Либо использовать базовую ошибку: `return nil, ErrInvalidURL`
- Либо не создавать предопределенную ошибку, а создавать на месте

---

### CR-13. Отсутствие логирования на уровне сервиса
**Файл:** `internal/service/link_service.go`

**Проблема:**
- Сервисный слой не имеет доступа к логгеру
- Невозможно логировать важные бизнес-события:
  - Количество попыток генерации уникального short code
  - Факт обращения к кэшу vs БД
  - Cache hit/miss statistics
  - Проблемы с Redis
- Все логи только в handlers — слишком поздно для debugging

**Решение:**
- Добавить logger в конструктор `NewLinkService`
- Логировать важные бизнес-метрики и события
- Добавить debug логи для troubleshooting

---

### CR-14. Redis Ping при старте без retry
**Файл:** `cmd/api/main.go:68-71`

**Проблема:**
```go
if err := redisClient.Ping(connectCtx).Err(); err != nil {
    _ = redisClient.Close()
    return fmt.Errorf("ping redis: %w", err)
}
```
- Приложение не стартует, если Redis временно недоступен
- Нет retry механизма
- Для resilience лучше стартовать с degraded режимом

**Решение:**
- Логировать warning вместо fatal при недоступности Redis
- Продолжать работу без кэша (fallback to DB)
- Или добавить retry с exponential backoff

---

### CR-15. Отсутствие метрик для мониторинга
**Общая проблема**

**Проблема:**
- Нет счетчиков cache hit/miss
- Нет метрик latency для Redis vs PostgreSQL
- Нет алертов при degradation кэша
- Невозможно оценить эффективность кэширования

**Решение:**
- Добавить Prometheus metrics или аналог
- Ключевые метрики:
  - `cache_hits_total`, `cache_misses_total`, `cache_errors_total`
  - `redis_latency_seconds`, `db_latency_seconds`
  - `track_click_queue_size`, `track_click_errors_total`
  - `http_request_duration_seconds`

---

### CR-16. Потенциальная проблема с UpdateLink
**Файл:** `internal/repository/postgres_links.go:81-89`

**Проблема:**
```go
ct, err := r.db.Pool.Exec(ctx, query, link.ShortCode, link.OriginalURL, link.IsActive, link.ID)
if err != nil {
    return fmt.Errorf("update link: %w", err)
}
if ct.RowsAffected() == 0 {
    return nil  // ← возвращает nil даже если запись не найдена
}
return nil
```
- Caller не может понять, обновилась запись или нет
- Молчаливый success при попытке обновить несуществующую запись

**Решение:**
- Возвращать `ErrNotFound` если `RowsAffected() == 0`

---

## 📋 Общие рекомендации из Code Review

### По работе с ошибками:
1. Создать централизованный error handler middleware
2. Добавить structured logging для всех ошибок с correlation ID
3. Различать transient (retry-able) и permanent ошибки
4. Документировать error codes в одном месте
5. Стандартизировать подход к error wrapping по слоям

### По работе с логами:
1. ✅ Убрать дублирующий `middleware.Logger` (дубликат проблемы #4)
2. Добавить логгер в сервисный слой
3. Исправить cleanup логгера для graceful shutdown
4. Логировать cache hit/miss для observability
5. Проверять ошибки `json.Encode()`
6. Добавить correlation ID для трейсинга запросов

### По работе с Redis:
1. Добавить таймауты в `redis.Options`
2. Логировать ошибки кэша (не игнорировать молча)
3. Добавить метрики для мониторинга
4. Сделать приложение resilient к недоступности Redis
5. Рассмотреть circuit breaker для Redis
6. Использовать worker pool вместо неограниченных горутин для TrackClick
7. Различать cache miss и error случаи

### По архитектуре:
1. Добавить health check endpoint с проверкой Redis и PostgreSQL
2. Реализовать graceful shutdown для всех компонентов
3. Добавить rate limiting для защиты от DDoS
4. Рассмотреть использование connection pool для горутин трекинга
