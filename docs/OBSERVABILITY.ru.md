# Руководство по наблюдаемости

[English](OBSERVABILITY.md) | [Русский](#русский)

---

## Русский

Полное руководство по мониторингу, логированию и наблюдаемости в ZapLink.

## Содержание

- [Обзор](#обзор)
- [Архитектура](#архитектура)
- [Метрики Prometheus](#метрики-prometheus)
- [Дашборды Grafana](#дашборды-grafana)
- [Логирование Loki](#логирование-loki)
- [Инструкции по настройке](#инструкции-по-настройке)
- [Запросы и алертинг](#запросы-и-алертинг)
- [Устранение неполадок](#устранение-неполадок)

---

## Обзор

ZapLink включает **полный стек наблюдаемости** для мониторинга здоровья, производительности и поведения приложения:

- **Prometheus** - Сбор и хранение метрик
- **Grafana** - Визуализация и дашборды
- **Loki** - Агрегация логов
- **Promtail** - Отправка логов из Docker-контейнеров

**Архитектура стека:**

```
┌─────────────┐
│  ZapLink    │ (Go приложение на :8080)
│   App       │
└──┬────────┬─┘
   │        │
   │ /metrics │ stdout логи
   │        │
   ▼        ▼
┌──────┐  ┌───────────┐
│Prome-│  │ Promtail  │ (Сборщик Docker логов)
│theus │  └─────┬─────┘
│:9090 │        │
└──┬───┘        │
   │            ▼
   │       ┌────────┐
   │       │  Loki  │ (Агрегация логов)
   │       │ :3100  │
   │       └────┬───┘
   │            │
   └────────────┼─────────┐
                │         │
                ▼         ▼
           ┌──────────────────┐
           │     Grafana      │ (Визуализация)
           │      :3000       │
           └──────────────────┘
```

---

## Архитектура

### Компоненты

| Компонент  | Назначение                       | Порт | URL                        |
|------------|----------------------------------|------|----------------------------|
| Prometheus | Сбор и хранение метрик           | 9090 | http://localhost:9090      |
| Grafana    | Дашборды и визуализация          | 3000 | http://localhost:3000      |
| Loki       | Агрегация и запросы логов        | 3100 | http://localhost:3100      |
| Promtail   | Сбор Docker логов                | 9080 | (внутренний)               |
| ZapLink    | Метрики и логи приложения        | 8080 | http://localhost:8080/metrics |

### Поток данных

**Метрики:**
1. ZapLink предоставляет endpoint `/metrics` (формат Prometheus)
2. Prometheus собирает данные каждые 15 секунд
3. Prometheus хранит данные временных рядов
4. Grafana запрашивает Prometheus для визуализации на дашбордах

**Логи:**
1. ZapLink пишет структурированные логи в stdout (slog)
2. Docker захватывает stdout как логи контейнера
3. Promtail читает Docker логи и отправляет в Loki
4. Loki хранит логи с метками (service, container, level)
5. Grafana запрашивает Loki для исследования логов

---

## Метрики Prometheus

### Доступные метрики

ZapLink предоставляет 5 основных бизнес-метрик и метрик инфраструктуры:

#### 1. Счетчик HTTP-запросов

**Название:** `zaplink_http_requests_total`  
**Тип:** Counter  
**Метки:** `method`, `route`, `status`  
**Описание:** Общее количество обработанных HTTP-запросов

**Примеры значений:**
```
zaplink_http_requests_total{method="GET",route="/{short_code}",status="302"} 15234
zaplink_http_requests_total{method="POST",route="/links",status="201"} 428
zaplink_http_requests_total{method="GET",route="/health",status="200"} 892
```

**Варианты использования:**
- Частота запросов по endpoint
- Частота ошибок (status >= 400)
- Распределение трафика

---

#### 2. Длительность HTTP-запросов

**Название:** `zaplink_http_request_duration_seconds`  
**Тип:** Histogram  
**Метки:** `method`, `route`  
**Бакеты:** `[0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10]` (секунды)  
**Описание:** Распределение задержки HTTP-запросов

**Пример гистограммы:**
```
zaplink_http_request_duration_seconds_bucket{method="GET",route="/{short_code}",le="0.005"} 1234
zaplink_http_request_duration_seconds_bucket{method="GET",route="/{short_code}",le="0.01"} 1456
zaplink_http_request_duration_seconds_bucket{method="GET",route="/{short_code}",le="0.025"} 1489
zaplink_http_request_duration_seconds_sum{method="GET",route="/{short_code}"} 18.45
zaplink_http_request_duration_seconds_count{method="GET",route="/{short_code}"} 1523
```

**Варианты использования:**
- Расчет p50, p95, p99 задержки
- Мониторинг SLO (например, "95% запросов < 100ms")
- Обнаружение деградации производительности

---

#### 3. Счетчик созданных ссылок

**Название:** `zaplink_links_created_total`  
**Тип:** Counter  
**Метки:** (нет)  
**Описание:** Общее количество успешно созданных коротких ссылок

**Пример значения:**
```
zaplink_links_created_total 428
```

**Варианты использования:**
- Скорость создания ссылок (ссылок/секунду)
- Отслеживание роста бизнеса
- Планирование мощностей

---

#### 4. Счетчик выполненных редиректов

**Название:** `zaplink_redirects_total`  
**Тип:** Counter  
**Метки:** (нет)  
**Описание:** Общее количество выполненных редиректов (успешных запросов /{short_code})

**Пример значения:**
```
zaplink_redirects_total 15234
```

**Варианты использования:**
- Частота редиректов (редиректов/секунду)
- Эффективность кеша (сравнение с запросами к БД)
- Отслеживание вовлеченности пользователей

---

#### 5. Счетчик отслеженных кликов

**Название:** `zaplink_clicks_tracked_total`  
**Тип:** Counter  
**Метки:** (нет)  
**Описание:** Общее количество кликов, успешно сохраненных в базу данных

**Пример значения:**
```
zaplink_clicks_tracked_total 15198
```

**Варианты использования:**
- Успешность отслеживания кликов (сравнение с `redirects_total`)
- Обнаружение потери данных (если отслеживание не работает)
- Здоровье pipeline аналитики

**Примечание:** Этот счетчик увеличивается асинхронно. Небольшая задержка или потеря данных ожидаемы.

---

### Конфигурация Prometheus

**Файл:** `prometheus.yml`

```yaml
global:
  scrape_interval: 15s      # Сбор каждые 15 секунд
  scrape_timeout: 10s       # Таймаут, если сбор занимает > 10s

scrape_configs:
  - job_name: "zaplink"
    metrics_path: /metrics
    static_configs:
      - targets:
          - "host.docker.internal:8080"  # ZapLink приложение
```

**Ключевые настройки:**
- `scrape_interval: 15s` - Баланс между свежестью данных и стоимостью хранения
- `host.docker.internal` - DNS Docker Desktop для хост-машины

---

## Дашборды Grafana

### Настройка

1. **Доступ к Grafana:**  
   Откройте http://localhost:3000  
   Логин: `admin` / `admin` (измените при первом входе)

2. **Добавьте источник данных Prometheus:**
   - Перейдите в **Configuration** → **Data Sources** → **Add data source**
   - Выберите **Prometheus**
   - URL: `http://prometheus:9090`
   - Нажмите **Save & Test**

3. **Добавьте источник данных Loki:**
   - Перейдите в **Configuration** → **Data Sources** → **Add data source**
   - Выберите **Loki**
   - URL: `http://loki:3100`
   - Нажмите **Save & Test**

---

### Рекомендуемые панели дашборда

#### Панель 1: Частота запросов

**Тип:** Graph (Time series)  
**Запрос:**
```promql
rate(zaplink_http_requests_total[1m])
```
**Легенда:** `{{method}} {{route}} {{status}}`

Показывает запросы в секунду с разбивкой по endpoint.

---

#### Панель 2: Частота ошибок

**Тип:** Graph (Time series)  
**Запрос:**
```promql
sum(rate(zaplink_http_requests_total{status=~"5.."}[1m]))
```
**Легенда:** `5xx ошибок/сек`

Отслеживает серверные ошибки (500, 503 и т.д.)

---

#### Панель 3: p99 задержка

**Тип:** Graph (Time series)  
**Запрос:**
```promql
histogram_quantile(0.99, 
  rate(zaplink_http_request_duration_seconds_bucket[1m])
)
```
**Легенда:** `p99 задержка`

Показывает 99-й процентиль длительности запроса (мониторинг SLO).

---

#### Панель 4: Производительность редиректов

**Тип:** Stat (Single value)  
**Запрос:**
```promql
histogram_quantile(0.99, 
  rate(zaplink_http_request_duration_seconds_bucket{route="/{short_code}"}[5m])
)
```
**Единица измерения:** секунды  
**Пороги:** Зеленый < 0.05, Желтый < 0.1, Красный >= 0.1

Выделяет задержку редиректа (основная метрика производительности).

---

#### Панель 5: Скорость создания ссылок

**Тип:** Graph (Time series)  
**Запрос:**
```promql
rate(zaplink_links_created_total[5m])
```
**Легенда:** `ссылок/сек`

Бизнес-метрика: скорость создания ссылок.

---

#### Панель 6: Потеря отслеживания кликов

**Тип:** Stat (Single value)  
**Запрос:**
```promql
(zaplink_redirects_total - zaplink_clicks_tracked_total) / zaplink_redirects_total * 100
```
**Единица измерения:** процент  
**Пороги:** Зеленый < 1%, Желтый < 5%, Красный >= 5%

Отслеживает потерю данных в асинхронном отслеживании кликов.

---

#### Панель 7: Трафик по кодам статуса

**Тип:** Pie chart  
**Запрос:**
```promql
sum by (status) (zaplink_http_requests_total)
```

Визуальная разбивка ответов 2xx, 3xx, 4xx, 5xx.

---

### Пример JSON дашборда

Сохраните это для импорта готового дашборда:

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

## Логирование Loki

### Формат логов

ZapLink использует **структурированное логирование** с пакетом Go `slog`:

**Пример вывода лога:**
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

**Уровни логирования:**
- `DEBUG` - Подробная диагностическая информация
- `INFO` - События нормальной работы (HTTP-запросы)
- `WARN` - Сбои кеша, деградированный режим
- `ERROR` - Ошибки БД, сбои отслеживания кликов

---

### Конфигурация Promtail

**Файл:** `promtail.yml`

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
      - cri: {}  # Парсинг логов в формате CRI
```

**Ключевые возможности:**
- Автоматическое обнаружение Docker-контейнеров
- Извлечение меток: `container`, `service`, `project`
- Отправка логов в Loki каждые 5 секунд

---

### Запросы логов в Grafana

**Доступ:** Grafana → Explore → Выберите источник данных "Loki"

#### Примеры запросов

**1. Все логи из приложения ZapLink:**
```logql
{service="app"}
```

**2. Только логи уровня ERROR:**
```logql
{service="app"} |= "ERROR"
```

**3. Неудачные операции кеша:**
```logql
{service="app"} |= "cache" |= "failed"
```

**4. Запросы к конкретному короткому коду:**
```logql
{service="app"} |= "abc123"
```

**5. Медленные запросы (> 100ms):**
```logql
{service="app"} | json | duration_ms > 100
```

**6. Ошибки 5xx:**
```logql
{service="app"} | json | status >= 500
```

---

### Метки логов

Promtail автоматически добавляет эти метки:

| Метка     | Пример значения    | Источник                         |
|-----------|--------------------|----------------------------------|
| container | `zaplink-app`      | Имя Docker-контейнера            |
| service   | `app`              | Имя сервиса Docker Compose       |
| project   | `zaplink`          | Имя проекта Docker Compose       |
| level     | `INFO`, `ERROR`    | Распарсено из JSON лога          |

**Фильтрация по метке:**
```logql
{service="app", level="ERROR"}
```

---

## Инструкции по настройке

### Запуск полного стека

```bash
# Запустите все сервисы наблюдаемости
docker compose up -d prometheus grafana loki promtail

# Проверьте здоровье сервисов
docker compose ps

# Просмотр логов
docker compose logs -f grafana
```

### Проверка endpoint метрик

```bash
curl http://localhost:8080/metrics
```

Ожидаемый вывод:
```
# HELP zaplink_http_requests_total Total number of HTTP requests
# TYPE zaplink_http_requests_total counter
zaplink_http_requests_total{method="GET",route="/health",status="200"} 5
...
```

### Проверка сбора данных Prometheus

1. Откройте http://localhost:9090
2. Перейдите в **Status** → **Targets**
3. Проверьте, что цель `zaplink` имеет статус **UP**

### Проверка приема данных Loki

1. Откройте Grafana → Explore
2. Выберите источник данных Loki
3. Запрос: `{service="app"}`
4. Должны отобразиться последние логи

---

## Запросы и алертинг

### Полезные запросы PromQL

**Частота запросов (за последние 5 минут):**
```promql
rate(zaplink_http_requests_total[5m])
```

**Средняя задержка:**
```promql
rate(zaplink_http_request_duration_seconds_sum[5m]) 
/ 
rate(zaplink_http_request_duration_seconds_count[5m])
```

**Частота ошибок (процент):**
```promql
sum(rate(zaplink_http_requests_total{status=~"5.."}[5m])) 
/ 
sum(rate(zaplink_http_requests_total[5m])) * 100
```

**Процент попаданий в кеш (оценка):**
```promql
rate(zaplink_redirects_total[5m]) / 
(rate(zaplink_redirects_total[5m]) + rate(postgres_queries[5m]))
```

---

### Правила алертинга (пример)

Создайте `alerts.yml` и смонтируйте в Prometheus:

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
          summary: "Высокая частота ошибок (> 5%)"
        
      - alert: HighLatency
        expr: |
          histogram_quantile(0.99, 
            rate(zaplink_http_request_duration_seconds_bucket[1m])
          ) > 0.5
        for: 5m
        annotations:
          summary: "p99 задержка > 500ms"
      
      - alert: ClickTrackingDegraded
        expr: |
          (zaplink_redirects_total - zaplink_clicks_tracked_total) 
          / zaplink_redirects_total > 0.1
        for: 5m
        annotations:
          summary: "Потеря отслеживания кликов > 10%"
```

---

## Устранение неполадок

### Prometheus не собирает данные

**Симптом:** Цели отображаются как DOWN в UI Prometheus

**Проверка:**
```bash
# Из контейнера Prometheus
docker exec zaplink-prometheus wget -O- http://host.docker.internal:8080/metrics
```

**Решение:** Убедитесь, что приложение ZapLink запущено на порту 8080

---

### Grafana не может подключиться к Prometheus

**Симптом:** Ошибка "Bad Gateway" в источнике данных

**Решение:** Используйте `http://prometheus:9090` (имя в Docker-сети), а не `localhost`

---

### Нет логов в Loki

**Симптом:** Пустые результаты в Grafana Explore

**Проверка Promtail:**
```bash
docker logs zaplink-promtail
```

**Распространенные проблемы:**
- Promtail не имеет доступа к Docker socket → смонтируйте `/var/run/docker.sock`
- Несоответствие меток → проверьте, существует ли `{service="app"}`

---

### Метрики не увеличиваются

**Симптом:** Счетчики застряли на 0

**Проверка:**
1. Зарегистрированы ли метрики? (`metrics.Register()` вызван в `main.go`)
2. Увеличивают ли обработчики метрики? (например, `metrics.LinksCreatedTotal.Inc()`)
3. Проверьте опечатки в именах метрик

---

## Лучшие практики

1. **Настройте алерты рано** - Не ждите инцидентов
2. **Мониторьте SLO** - Определите целевые значения p99 задержки и частоты ошибок
3. **Используйте метки разумно** - Избегайте меток с высокой кардинальностью (например, ID пользователей)
4. **Гигиена дашбордов** - Удаляйте неиспользуемые панели, организуйте по приоритету
5. **Сэмплирование логов** - В production с высоким трафиком сэмплируйте подробные логи (уровень DEBUG)
6. **Политики хранения** - Настройте хранение Prometheus (по умолчанию: 15 дней)

---

**Для деталей архитектуры см. [ARCHITECTURE.md](ARCHITECTURE.md)**  
**Для документации API см. [API.md](API.md)**
