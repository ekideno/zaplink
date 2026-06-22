# Архитектура ZapLink

[English](ARCHITECTURE.md) | [Русский](#русский)

---

## Русский

Комплексная архитектурная документация сервиса сокращения URL ZapLink.

## Содержание

- [Обзор](#обзор)
- [Системная архитектура](#системная-архитектура)
- [Выбор технологий](#выбор-технологий)
- [Модель данных](#модель-данных)
- [Потоки запросов](#потоки-запросов)
- [Стратегия кеширования](#стратегия-кеширования)
- [Обработка ошибок](#обработка-ошибок)
- [Оптимизации производительности](#оптимизации-производительности)
- [Архитектурные решения](#архитектурные-решения)

---

## Обзор

ZapLink — это **высокопроизводительный сервис сокращения URL**, построенный на Go, спроектированный для обработки тысяч редиректов в секунду с латентностью менее 20мс.

**Основные возможности:**
- Сокращение URL с генерацией уникальных кодов
- Быстрое перенаправление с кешированием Redis
- Отслеживание аналитики кликов
- Интеграция с Prometheus метриками
- Структурированное логирование с slog

---

## Системная архитектура

ZapLink следует паттерну **слоёной архитектуры** с чётким разделением ответственности:

![Системная архитектура](docs/images/system_architecture.png)

### Ответственность слоёв

| Слой       | Ответственность | НЕ обрабатывает |
|------------|-----------------|-----------------|
| **Handler** | HTTP запрос/ответ, JSON маршалинг, валидация ввода | Бизнес-логика, доступ к БД |
| **Service** | Бизнес-логика, генерация кодов, оркестрация | Прямые запросы к БД, HTTP concerns |
| **Repository** | Доступ к PostgreSQL, SQL запросы | Кеширование, бизнес-правила |
| **Cache** | Redis операции, управление TTL | Бизнес-логика, валидация |

**Критическое правило дизайна:** Repository слой НЕ управляет кешированием. Service оркеструет Repository и Cache независимо.

---

## Выбор технологий

### Go 1.25
**Почему Go:**
- Отличная производительность HTTP (нативный `net/http`)
- Простая модель конкурентности (goroutines для async трекинга кликов)
- Мощная стандартная библиотека (context, slog, sql)
- Быстрая компиляция и деплоймент

### Chi Router
**Почему Chi:**
- Лёгкий и идиоматичный
- Отличная поддержка middleware
- Роутинг на основе context
- Совместим с `net/http.Handler`

**Рассмотренные альтернативы:**
- Gin (больше функций, тяжелее)
- Echo (схожая производительность, другой API)

### PostgreSQL 17
**Почему PostgreSQL:**
- ACID гарантии для создания ссылок
- Богатое индексирование (unique constraint на `short_code`)
- Поддержка JSONB для будущей расширяемости
- Надёжные инструменты миграций (golang-migrate)

**Дизайн схемы:**
- Таблица `links`: основное хранилище с уникальным индексом `short_code`
- Таблица `clicks`: аналитика с foreign key на `links`

### Redis 7
**Почему Redis:**
- Субмиллисекундные lookup (p99 < 1мс)
- Простая модель key-value (short_code → сериализованный JSON)
- Поддержка TTL для автоматического истечения
- Опциональный (приложение работает без Redis если недоступен)

**Стратегия кеширования:** Write-through с TTL 1 час (настраивается через `REDIS_TTL_SECONDS`)

### Prometheus + Grafana
**Почему Prometheus:**
- Индустриальный стандарт для метрик
- Pull-based scraping (нет overhead инструментации)
- Мощный язык запросов PromQL
- Нативная поддержка histogram для латентностей

**Экспонируемые метрики:**
- HTTP request counters (по method, route, status)
- Request duration histograms
- Бизнес-метрики (созданные ссылки, обслуженные редиректы, отслеженные клики)

---

## Модель данных

### Таблица Links

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

| Колонка      | Тип          | Ограничения              | Назначение                        |
|--------------|--------------|--------------------------|-----------------------------------|
| id           | BIGSERIAL    | PRIMARY KEY              | Внутренний идентификатор          |
| short_code   | VARCHAR(8)   | UNIQUE NOT NULL          | Пользовательский короткий код (base62) |
| original_url | TEXT         | NOT NULL                 | Оригинальный длинный URL          |
| is_active    | BOOLEAN      | DEFAULT true NOT NULL    | Флаг soft delete                  |
| created_at   | TIMESTAMPTZ  | DEFAULT now() NOT NULL   | Временная метка создания          |

**Стратегия индексирования:**
- `idx_links_short_code`: Быстрый lookup при редиректе (самый частый запрос)
- `idx_links_created_at`: Аналитические запросы (недавние ссылки)

### Таблица Clicks

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

| Колонка     | Тип        | Ограничения                            | Назначение               |
|-------------|------------|----------------------------------------|--------------------------|
| id          | BIGSERIAL  | PRIMARY KEY                            | Внутренний идентификатор |
| link_id     | BIGINT     | FOREIGN KEY → links(id) ON DELETE CASCADE | Ссылка на link        |
| created_at  | TIMESTAMPTZ | DEFAULT now() NOT NULL                | Временная метка клика    |
| user_agent  | TEXT       | NULL                                   | User-agent браузера      |
| ip_address  | INET       | NULL                                   | IP адрес клиента         |

**Стратегия индексирования:**
- `idx_clicks_link_id`: Быстрая агрегация подсчёта кликов по ссылке
- `idx_clicks_created_at`: Временные аналитические запросы

---

## Потоки запросов

### 1. Создание короткой ссылки

![Создание короткой ссылки](docs/images/create_short_link.png)

**Ключевые шаги:**
1. Handler валидирует тело запроса (JSON unmarshaling)
2. Service валидирует формат URL (regex проверка)
3. Service генерирует уникальный короткий код (base62 кодирование timestamp + случайные байты)
4. Repository вставляет в PostgreSQL с unique constraint
5. При конфликте (редко): повторная попытка с новым кодом (макс. 3 попытки)

**Инкрементируемые метрики:** `zaplink_links_created_total`

---

### 2. Редирект (горячий путь)

![Редирект](docs/images/redirect.png)

**Cache Hit (95% запросов):**
1. Handler извлекает `short_code` из URL path
2. Service запрашивает Redis через Cache слой
3. **Cache HIT** (p99 < 1мс)
4. Handler возвращает 302 редирект немедленно
5. **Async**: Порождает goroutine для трекинга клика в фоне (не блокирует ответ)

**Cache Miss (5% запросов):**
1. Redis возвращает пустой результат (ключ не найден)
2. Service запрашивает PostgreSQL через Repository
3. Если найдено: Service заполняет Redis с TTL 1 час
4. Handler возвращает 302 редирект
5. **Async**: Трекинг клика в фоне


**Инкрементируемые метрики:** 
- `zaplink_redirects_total` (синхронно, всегда)
- `zaplink_clicks_tracked_total` (async, может отставать или fail молча)

---

## Стратегия кеширования

### Паттерн Write-Through Cache

**При чтении (Cache Hit):**
```
Service → Cache.Get(short_code) → Redis GET
          └─ HIT → вернуть данные ссылки
```

**При чтении (Cache Miss):**
```
Service → Cache.Get(short_code) → Redis GET
          └─ MISS → Repository.GetByShortCode() → PostgreSQL SELECT
                 → Cache.Set(short_code, link, TTL=3600) → Redis SET
                 → вернуть данные ссылки
```

**При записи (создание ссылки):**
```
Service → Repository.CreateLink() → PostgreSQL INSERT
       → (НЕ заполнять кеш - пусть первое чтение заполнит)
```

**При обновлении (модификация ссылки):**
```
Service → Repository.UpdateLink() → PostgreSQL UPDATE
       → Cache.Delete(short_code) → Redis DEL (инвалидация)
```

### Конфигурация кеша

| Параметр           | По умолчанию | Переменная окружения  | Назначение                       |
|--------------------|--------------|-----------------------|----------------------------------|
| TTL                | 3600s        | `REDIS_TTL_SECONDS`   | Как долго кешировать ссылки      |
| Connection Timeout | 5s           | -                     | Таймаут подключения Redis        |
| Operation Timeout  | 2s           | -                     | Таймаут Redis GET/SET            |

### Graceful Degradation

Если Redis **недоступен при старте**:
- Приложение логирует предупреждение и продолжает без кеша
- Все запросы попадают напрямую в PostgreSQL
- Производительность ухудшается, но сервис остаётся доступным

Если Redis **падает во время работы**:
- Операции кеша возвращают ошибки
- Service откатывается к Repository (PostgreSQL)
- Ошибки логируются, но не передаются клиенту

---

## Обработка ошибок

### Типизированные ошибки (пакет apperror)

ZapLink использует **доменные ошибки** для различия бизнес-ошибок от технических:

```go
// internal/apperror/error.go
type Error struct {
    StatusCode int    // HTTP статус код
    Code       string // Машиночитаемый код ошибки
    Message    string // Человекочитаемое сообщение
    Err        error  // Базовая ошибка (опционально)
}
```

### Поток ошибок

```
Repository → Возвращает apperror.Error
    ↓
Service → Проверяет тип ошибки, может добавить контекст
    ↓
Handler → Вызывает writeError() helper
    ↓
Client ← Получает JSON ответ с ошибкой
```

### Распространённые коды ошибок

| HTTP Status | Код ошибки         | Слой       | Причина                         |
|-------------|--------------------|-----------|---------------------------------|
| 400         | `invalid_url`      | Service   | Валидация URL не прошла         |
| 400         | `invalid_request`  | Handler   | JSON unmarshal не удался        |
| 404         | `not_found`        | Repository| Short code не существует        |
| 404         | `link_inactive`    | Service   | Ссылка существует, но is_active=false |
| 500         | `db_error`         | Repository| Запрос PostgreSQL не удался     |
| 500         | `internal_error`   | Handler   | Неожиданная panic или ошибка    |

---

## Оптимизации производительности

### 1. Кеширование Redis
- **Влияние:** 95% cache hit rate снижает нагрузку на БД в 20 раз
- **Латентность:** p99 < 20мс vs p99 < 100мс без кеша

### 2. Асинхронный трекинг кликов
- **Влияние:** Трекинг кликов НЕ блокирует ответ редиректа
- **Реализация:** Goroutine с отдельным context и таймаутом 5с

```go
go func() {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    
    if err := h.service.TrackClick(ctx, short_code, userAgent, ipAddress); err != nil {
        log.Error("failed to track click", slog.String("short_code", short_code), slog.Any("error", err))
    }
}()
```

**Компромисс:** Счётчик кликов может быть слегка неточным при сбое трекинга (приемлемо для аналитики).

### 3. Индексы базы данных
- `idx_links_short_code`: B-tree индекс на `short_code` (критично)
- `idx_clicks_link_id`: Foreign key индекс для агрегации подсчёта кликов

### 4. Connection Pooling
```go
db.SetMaxOpenConns(25)      // Макс. одновременных соединений
db.SetMaxIdleConns(5)       // Пул idle соединений
db.SetConnMaxLifetime(5m)   // Рециклирование соединений
```

---

## Архитектурные решения

### 1. Почему Repository НЕ управляет кешированием

**Проблема:** Смешивание логики кеширования в Repository нарушает Single Responsibility Principle.

**Решение:** Service оркеструет Repository (постоянное хранилище) и Cache (эфемерное хранилище) независимо.

**Преимущества:**
- Repository остаётся чистым data access слоем (тестируется без Redis)
- Cache может быть заменён (например, Memcached) без изменения Repository
- Чёткое разделение: Repository = источник истины, Cache = оптимизация

### 2. Почему асинхронный трекинг кликов

**Проблема:** Запись click records в PostgreSQL добавляет ~10-20мс латентности к ответу редиректа.

**Решение:** Порождать goroutine для трекинга клика в фоне.

**Компромиссы:**
- ✅ Латентность редиректа снижена на 20мс (p99 < 20мс vs < 40мс)
- ✅ Улучшен пользовательский опыт (быстрее редиректы)
- ❌ Счётчик кликов может быть слегка неточным при сбое трекинга
- ❌ Нет backpressure если БД медленная (goroutines могут накапливаться)

**Приемлемо потому что:** Аналитические данные не критичны. 99.5% точность трекинга достаточна.

### 3. Почему Graceful Cache Degradation

**Проблема:** Жёсткая зависимость от Redis означает single point of failure.

**Решение:** Если Redis недоступен, работать без кеша (режим только PostgreSQL).

**Преимущества:**
- Сервис остаётся доступным даже при падении Redis
- Упрощённая локальная разработка (Redis не обязателен)
- Снижена операционная сложность

**Компромисс:** Производительность ухудшается (латентность в 10 раз выше), но сервис онлайн.

---

## Будущие улучшения

- [ ] Добавить distributed tracing (OpenTelemetry)
- [ ] Реализовать rate limiting (token bucket)
- [ ] Добавить поддержку кастомных коротких кодов
- [ ] Реализовать истечение ссылок (TTL)
- [ ] Добавить analytics API (статистика кликов по временному диапазону)
- [ ] Geo-location трекинг (MaxMind GeoIP2)
- [ ] Admin UI дашборд
- [ ] GraphQL API

---

**Для документации API см. [API.ru.md](API.ru.md)**  
**Для настройки observability см. [OBSERVABILITY.ru.md](OBSERVABILITY.ru.md)**
