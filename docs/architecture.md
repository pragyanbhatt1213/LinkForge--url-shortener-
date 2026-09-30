# Architecture – LinkForge URL Shortener

## System Overview

```
                    Client (Browser / curl)
                            │
                            ▼
                    Go Fiber HTTP API
                    (cmd/api/main.go)
                            │
              ┌─────────────┼────────────────┐
              ▼             ▼                ▼
           Redis        PostgreSQL       Prometheus
         (cache &        (persistent      /metrics
        rate limit)       storage)
              │
       ┌──────┴──────┐
       ▼             ▼
  Redirect        Rate limit
  (cache-aside)   (INCR / EXPIRE)
```

---

## Request Flow

### POST /shorten

```
Client
  │  POST /shorten  { url, custom_slug, expiry_hours }
  ▼
RateLimit middleware  →  Redis INCR rate_limit:<IP>
  │  count ≤ 10: pass
  │  count > 10: 429 Too Many Requests
  ▼
Shorten handler
  ▼
services.CreateShortURL
  ├─ ValidateURL (must be http/https with a host)
  ├─ If no custom slug → GenerateSlug() (random 6 chars)
  ├─ If custom slug   → validate length (3–50 chars)
  ├─ store.SaveSlug   → Redis SETNX slug url TTL
  │    SETNX returns false if slug already taken → "slug already exists"
  └─ store.SaveLinkWithExpiry → PostgreSQL INSERT (slug, url, expires_at)
  ▼
{ "short_url": "http://localhost:3000/<slug>" }
```

### GET /:slug  (redirect)

```
Client
  │  GET /abc123
  ▼
Redirect handler
  ▼
store.GetURL("abc123")  ──▶  Redis GET
                               │
                    ┌──────────┴──────────┐
                    ▼                     ▼
                 HIT                    MISS (redis.Nil)
                    │                     │
                    │            store.GetLink → PostgreSQL SELECT
                    │                     │
                    │               Not found → 404
                    │                     │
                    │            store.CacheURL → Redis SET (24h TTL)
                    │                     │
                    └──────────┬──────────┘
                               ▼
                     services.RecordClick(slug)
                     (sends to buffered channel)
                               ▼
                         HTTP 302 Redirect
```

### GET /analytics/:slug

```
Client
  │  GET /analytics/abc123
  ▼
Analytics handler
  ▼
store.GetClickCount("abc123")
  ▼
PostgreSQL: SELECT COUNT(*) FROM clicks WHERE slug = $1
  ▼
{ "slug": "abc123", "clicks": 7 }
```

---

## Database Schema

### `links` table

| Column       | Type        | Notes                        |
|------------- |-------------|------------------------------|
| id           | SERIAL PK   | Auto-increment               |
| slug         | TEXT UNIQUE | Short identifier             |
| original_url | TEXT        | Destination URL              |
| created_at   | TIMESTAMPTZ | Default NOW()                |
| expires_at   | TIMESTAMPTZ | NULL = never expires         |

### `clicks` table

| Column     | Type        | Notes                    |
|------------|-------------|--------------------------|
| id         | SERIAL PK   | Auto-increment           |
| slug       | TEXT        | Foreign key (logical)    |
| clicked_at | TIMESTAMPTZ | When the visit happened  |

---

## Redis Usage

| Purpose         | Key pattern          | Command  | TTL                     |
|-----------------|----------------------|----------|-------------------------|
| URL cache       | `<slug>`             | SET / GET | `expiry_hours` or 24 h |
| Duplicate guard | `<slug>`             | SETNX    | Same as above           |
| Rate limiting   | `rate_limit:<IP>`    | INCR     | 60 s (set on first req) |

---

## Caching Strategy – Cache-Aside

1. **Read:** Check Redis first. If found (HIT), return immediately.
2. **Miss:** Read from PostgreSQL. Write the result back into Redis with a TTL.
3. **Write:** On `POST /shorten`, write to Redis (SETNX) **and** PostgreSQL together.

The application manages the cache explicitly — Redis does not observe PostgreSQL writes automatically.

---

## Rate Limiting

Implemented in `internal/middleware/rate_limit.go`.

- Key: `rate_limit:<client IP>`
- On each request: `INCR key` → atomic increment
- On first request (count == 1): `EXPIRE key 60` → sets 60-second window
- If count > 10: return `HTTP 429 Too Many Requests`
- After 60 s: key expires, counter resets automatically

**Why INCR?** It is atomic — concurrent requests cannot both read the same count and incorrectly decide they are within the limit.

**Why EXPIRE only on count == 1?** The window starts with the first request. Setting EXPIRE on every request would keep resetting the window and the counter would never expire.

---

## Click Tracking

Synchronous database writes on every redirect would add latency.
Instead, a channel-based worker pool is used:

```
Redirect handler
  │
  ▼
services.RecordClick(slug)
  │
  ▼
ClickQueue (buffered channel, capacity 100)
  │
  ├─ Worker 1 ──▶ store.SaveClick → PostgreSQL INSERT
  ├─ Worker 2 ──▶ store.SaveClick → PostgreSQL INSERT
  └─ Worker 3 ──▶ store.SaveClick → PostgreSQL INSERT
```

- 3 workers started at startup (`services.StartClickWorkers(3)`)
- The redirect handler returns the `302` immediately without waiting for the INSERT
- Click count is read by `GET /analytics/:slug` via `SELECT COUNT(*)`

---

## Concurrency

| Pattern                  | Where                             |
|--------------------------|-----------------------------------|
| Goroutine worker pool    | `services.StartClickWorkers(3)`   |
| Buffered channel         | `ClickQueue` (capacity 100)       |
| Redis atomic INCR        | Rate limiter (no race condition)  |
| Redis SETNX              | Prevent duplicate slugs           |

---

## URL Expiration

- **Redis TTL:** The slug key expires automatically after `expiry_hours` (or 24 h default). Expired keys return redis.Nil, which triggers the PostgreSQL fallback.
- **PostgreSQL `expires_at`:** Stored as a timestamp. Available for future enforcement (e.g. filtering expired rows from `GET /links`).
- **Negative or zero expiry:** Rejected in the service layer with a validation error.

---

## Error Handling

| Scenario               | Response                          |
|------------------------|-----------------------------------|
| Invalid URL            | 400 `{ "error": "invalid URL" }`  |
| Slug < 3 or > 50 chars | 400 validation error              |
| Slug already taken     | 409 Conflict                      |
| Rate limit exceeded    | 429 Too Many Requests             |
| Slug not found         | 404 Not Found                     |
| Redis down             | 500 Internal Server Error         |
| PostgreSQL error       | 500 Internal Server Error         |

---

## Observability

The `/metrics` endpoint exposes Prometheus-format data.

**Custom metrics (registered in `internal/middleware/metrics.go`):**

| Metric                                  | Type      | Labels                     |
|-----------------------------------------|-----------|----------------------------|
| `linkforge_http_requests_total`         | Counter   | method, path, status       |
| `linkforge_http_request_duration_seconds` | Histogram | method, path             |

**Also available at `/metrics`:**
Go runtime metrics (goroutines, GC, heap) and process metrics (CPU, file descriptors) — provided automatically by the Prometheus Go client library.

---

## Load Testing

Run with [k6](https://k6.io):

```bash
# Simple ramp-up script (tests/load.js)
k6 run tests/load.js

# Advanced staged script with setup() and SharedArray (loadtest/script.js)
k6 run loadtest/script.js
```

Both scripts exercise `POST /shorten` and `GET /:slug`. `loadtest/script.js` additionally
pre-creates a slug in `setup()` so redirect VUs bypass rate limiting.

---

## Project Structure

```
linkforge-url-shortener/
├── cmd/api/main.go              — entry point: Fiber setup, routes
├── internal/
│   ├── config/                  — Redis and PostgreSQL config loaders
│   ├── handlers/                — HTTP handlers (thin: parse → call service → respond)
│   ├── middleware/              — rate limiter, Prometheus metrics middleware
│   ├── models/                  — shared structs (Link, ClickEvent, ShortenRequest)
│   ├── services/                — business logic, click worker pool
│   └── store/                   — all database calls (Redis + PostgreSQL)
├── web/index.html               — single-page dashboard (HTML/CSS/vanilla JS)
├── tests/load.js                — k6 load test (simple ramp-up)
├── loadtest/script.js           — k6 load test (staged with setup())
├── docs/
│   ├── architecture.md          — this file
│   └── interview-notes.md       — interview Q&A cheat sheet (gitignored)
├── monitoring/prometheus.yml    — Prometheus scrape config
├── grafana/provisioning/        — Grafana datasource & dashboard provisioning
├── .github/workflows/ci.yml     — GitHub Actions: go vet + go test
├── .env.example                 — environment variable template
├── docker-compose.yml           — PostgreSQL + Redis + Prometheus + Grafana
└── README.md
```
