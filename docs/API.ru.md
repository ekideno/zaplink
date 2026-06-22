# API Reference

[English](API.md) | [Русский](#русский)

---

## Русский

Документация REST API ZapLink с детальной спецификацией эндпоинтов, примерами запросов/ответов и кодами ошибок.

**Base URL:** `http://localhost:8080`

**Content-Type:** Все запросы и ответы используют `application/json`

---

## Содержание

- [Эндпоинты](#эндпоинты)
  - [Создание короткой ссылки](#создание-короткой-ссылки)
  - [Получение информации о ссылке](#получение-информации-о-ссылке)
  - [Редирект на оригинальный URL](#редирект-на-оригинальный-url)
  - [Проверка здоровья](#проверка-здоровья)
  - [Prometheus метрики](#prometheus-метрики)
- [Ответы с ошибками](#ответы-с-ошибками)
- [Примеры](#примеры)

---

## Эндпоинты

### Создание короткой ссылки

Генерирует сокращённый URL для заданного оригинального URL.

**Эндпоинт:** `POST /links`

**Тело запроса:**

```json
{
  "url": "https://example.com/very/long/url/path"
}
```

| Поле | Тип    | Обязательно | Описание                           |
|------|--------|-------------|------------------------------------|
| url  | string | Да          | Оригинальный URL для сокращения (должен быть валидным HTTP/HTTPS) |

**Успешный ответ:** `201 Created`

```json
{
  "short_code": "a1b2c3d",
  "short_url": "http://localhost:8080/a1b2c3d"
}
```

| Поле       | Тип    | Описание                              |
|------------|--------|---------------------------------------|
| short_code | string | Сгенерированный уникальный код (7-8 символов) |
| short_url  | string | Полный сокращённый URL                |

**Ответы с ошибками:**

- `400 Bad Request` - Неверный формат URL
- `500 Internal Server Error` - Ошибка базы данных или сервера

**Пример:**

```bash
curl -X POST http://localhost:8080/links \
  -H "Content-Type: application/json" \
  -d '{
    "url": "https://github.com/ekideno/zaplink"
  }'
```

---

### Получение информации о ссылке

Получить информацию о сокращённой ссылке, включая количество кликов.

**Эндпоинт:** `GET /links/{short_code}`

**Параметры пути:**

| Параметр   | Тип    | Описание              |
|------------|--------|-----------------------|
| short_code | string | Короткий код ссылки   |

**Успешный ответ:** `200 OK`

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

| Поле         | Тип     | Описание                                  |
|--------------|---------|-------------------------------------------|
| id           | integer | Внутренний ID в базе данных               |
| short_code   | string  | Уникальный короткий код                   |
| original_url | string  | Оригинальный длинный URL                  |
| is_active    | boolean | Активна ли ссылка (true/false)            |
| created_at   | string  | Временная метка создания (ISO 8601 UTC)   |
| click_count  | integer | Общее количество редиректов/кликов        |

**Ответы с ошибками:**

- `404 Not Found` - Короткий код не существует
- `500 Internal Server Error` - Ошибка базы данных

**Пример:**

```bash
curl http://localhost:8080/links/a1b2c3d
```

---

### Редирект на оригинальный URL

Редирект на оригинальный URL, связанный с коротким кодом. **Это основной эндпоинт редиректа.**

**Эндпоинт:** `GET /{short_code}`

**Параметры пути:**

| Параметр   | Тип    | Описание            |
|------------|--------|---------------------|
| short_code | string | Короткий код ссылки |

**Успешный ответ:** `302 Found`

```
HTTP/1.1 302 Found
Location: https://github.com/ekideno/zaplink
```

Браузер автоматически следует редиректу на оригинальный URL.

**Побочные эффекты:**
- Клик отслеживается асинхронно (user-agent, IP адрес, timestamp)
- НЕ блокирует ответ редиректа
- Кеш обновляется, если Redis доступен

**Ответы с ошибками:**

- `404 Not Found` - Короткий код не существует или ссылка неактивна
- `500 Internal Server Error` - Ошибка базы данных

**Пример:**

```bash
# Следовать редиректу
curl -L http://localhost:8080/a1b2c3d

# Просмотреть только заголовок редиректа
curl -I http://localhost:8080/a1b2c3d
```

**Производительность:**
- Кешированные редиректы: **p99 < 20мс**
- Редиректы из БД: **p99 < 100мс**

---

### Проверка здоровья

Проверить статус здоровья сервиса, включая подключение к PostgreSQL и Redis.

**Эндпоинт:** `GET /health`

**Успешный ответ:** `200 OK`

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

| Поле      | Тип    | Описание                                     |
|-----------|--------|----------------------------------------------|
| status    | string | Общее здоровье: "healthy" или "unhealthy"    |
| timestamp | string | Временная метка проверки (ISO 8601 UTC)      |
| checks    | object | Статусы отдельных компонентов                |

**Деградированный ответ:** `200 OK` (когда Redis недоступен)

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

**Примечание:** Сервис продолжает работать без Redis (кеш gracefully деградирует до работы только с БД).

**Ответ с ошибкой:** `503 Service Unavailable` (когда PostgreSQL недоступен)

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

**Пример:**

```bash
curl http://localhost:8080/health
```

---

### Prometheus метрики

Экспонирует Prometheus-совместимые метрики для мониторинга.

**Эндпоинт:** `GET /metrics`

**Успешный ответ:** `200 OK` (text/plain)

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

**Доступные метрики:**

| Имя метрики                            | Тип       | Лейблы                  | Описание                         |
|----------------------------------------|-----------|-------------------------|----------------------------------|
| `zaplink_http_requests_total`          | Counter   | method, route, status   | Всего HTTP запросов              |
| `zaplink_http_request_duration_seconds`| Histogram | -                       | Распределение длительности запросов |
| `zaplink_links_created_total`          | Counter   | -                       | Всего создано ссылок             |
| `zaplink_redirects_total`              | Counter   | -                       | Всего обслужено редиректов       |
| `zaplink_clicks_tracked_total`         | Counter   | -                       | Всего сохранено кликов в БД      |

**Пример:**

```bash
curl http://localhost:8080/metrics
```

**Примечание:** Этот эндпоинт публичный (без аутентификации). Рассмотрите ограничение доступа в продакшене.

---

## Ответы с ошибками

Все ответы с ошибками следуют этому формату:

```json
{
  "error": {
    "code": "error_code",
    "message": "Человеко-читаемое сообщение об ошибке"
  }
}
```

### Распространённые коды ошибок

| HTTP Status | Код ошибки         | Описание                                    |
|-------------|--------------------|---------------------------------------------|
| 400         | `invalid_url`      | Формат URL неверен или пуст                 |
| 400         | `invalid_request`  | Тело запроса некорректно или отсутствуют поля |
| 404         | `not_found`        | Короткий код не существует                  |
| 404         | `link_inactive`    | Ссылка существует, но помечена как неактивная |
| 500         | `db_error`         | Операция с базой данных не удалась          |
| 500         | `internal_error`   | Неожиданная ошибка сервера                  |

**Пример ответа с ошибкой:**

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

## Примеры

### JavaScript (fetch)

```javascript
// Создать короткую ссылку
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

// Использование
createShortLink('https://example.com/long/url')
  .then(data => console.log('Short URL:', data.short_url))
  .catch(err => console.error('Ошибка:', err));
```

### Python (requests)

```python
import requests

# Создать короткую ссылку
def create_short_link(original_url):
    response = requests.post(
        'http://localhost:8080/links',
        json={'url': original_url}
    )
    response.raise_for_status()
    return response.json()

# Получить информацию о ссылке
def get_link_info(short_code):
    response = requests.get(f'http://localhost:8080/links/{short_code}')
    response.raise_for_status()
    return response.json()

# Использование
result = create_short_link('https://example.com/long/url')
print(f"Short URL: {result['short_url']}")

info = get_link_info(result['short_code'])
print(f"Клики: {info['click_count']}")
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

## Аутентификация

⚠️ **Не реализована.** Все эндпоинты публичные.

Будущие соображения:
- API key аутентификация для создания ссылок
- Admin эндпоинты с JWT
- Публичный read-only доступ для редиректов

---

## CORS

Cross-Origin Resource Sharing (CORS) **не настроен** по умолчанию.

Для включения CORS для frontend приложений, добавьте middleware в `internal/http/router.go`:

```go
import "github.com/go-chi/cors"

r.Use(cors.Handler(cors.Options{
    AllowedOrigins: []string{"https://your-frontend.com"},
    AllowedMethods: []string{"GET", "POST", "OPTIONS"},
    AllowedHeaders: []string{"Content-Type"},
}))
```

---

**Для получения дополнительной информации об архитектуре и реализации см. [ARCHITECTURE.ru.md](ARCHITECTURE.ru.md)**
