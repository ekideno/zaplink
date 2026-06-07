# ZapLink Architecture

## Overview

ZapLink is a high-performance URL shortener service built with Go.

It provides:
- URL shortening
- Fast redirection with caching
- Click tracking
- Scalable layered architecture

---

## System architecture

The system follows a layered architecture:

Handler → Service → Repository → Database / Cache

### Responsibilities:

- **Handler**: HTTP layer, request validation, response formatting
- **Service**: business logic (short code generation, validation, flow control)
- **Repository**: data access layer (PostgreSQL / Redis)
- **Cache (Redis)**: fast lookup for hot URLs

---

## Request flow

### Create link

1. Client sends request:
   `POST /api/v1/links`

2. Handler:
   - validates input URL
   - parses request body

3. Service:
   - generates unique short code
   - validates business rules

4. Repository:
   - saves link in PostgreSQL

5. Response:
   - returns generated short URL

---

### Redirect flow

1. Client requests:
   `GET /{shortCode}`

2. Handler:
   - extracts short code

3. Cache layer (Redis):
   - checks if URL exists in cache

4. If cache miss:
   - query PostgreSQL via repository
   - store result in Redis

5. Service:
   - returns original URL

6. Response:
   - HTTP 302 redirect

---

## Data model

### Links

Stores shortened URLs.

- id (BIGINT identity primary key)
- short_code (VARCHAR(8), unique)
- original_url (TEXT)
- is_active (BOOLEAN, default true)
- created_at (TIMESTAMPTZ, default now())

---

### Clicks

Stores analytics for each redirect.

- id (BIGINT identity primary key)
- link_id (foreign key to `links.id`, cascade on delete)
- created_at (TIMESTAMPTZ, default now())
- user_agent (TEXT, optional)
- ip_address (INET, optional)

---

## Tech stack

- Go (Chi router)
- PostgreSQL (persistent storage)
- Redis (caching layer)
- Docker / Docker Compose
- golang-migrate-compatible SQL migrations in `migrations/`
