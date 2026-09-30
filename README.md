# LinkForge

A URL shortener built with Go and Fiber, using Redis for caching and rate limiting, PostgreSQL for persistent storage, Prometheus and Grafana for monitoring, and k6 for load testing.

```
                Client (Browser / curl)
                        │
                        ▼
                Go Fiber HTTP API
                        │
          ┌─────────────┼─────────────┐
          ▼             ▼             ▼
       Redis        PostgreSQL    Prometheus
     (cache &        (persistent    /metrics
    rate limit)       storage)        │
          │                           ▼
   ┌──────┴──────┐                 Grafana
   ▼             ▼               :3001
Redirect    Rate limit
(cache-     (INCR /
 aside)      EXPIRE)
```

---

## Features

| Feature | Detail |
|---|---|
| Short URL creation | `POST /shorten` — random or custom slug |
| Custom slugs | 3–50 chars, collision-safe via Redis SETNX |
| Expiring URLs | Redis TTL + PostgreSQL `expires_at` |
| Redirects | Cache-aside: Redis HIT → fast; MISS → PostgreSQL fallback |
| Click analytics | Async worker pool writes to PostgreSQL clicks table |
| Rate limiting | Redis INCR/EXPIRE — 10 req/min per IP |
| Prometheus metrics | Request count + duration histograms at `/metrics` |
| Grafana dashboard | Auto-provisioned at `http://localhost:3001` |
| Dashboard UI | `GET /` → `web/index.html` (vanilla HTML/CSS/JS) |
| Unit tests | miniredis — no real Redis needed |
| Load testing | k6 at 10 / 50 / 100 VUs |
| CI | GitHub Actions: `go vet` + `go test` on every push |

---

## Quick Start

**Prerequisites:** Docker, Go 1.21+

```bash
# 1. Start all infrastructure
docker compose up -d

# 2. Start the API
go run ./cmd/api

# 3. Open the dashboard
# http://localhost:3000

# 4. Open Grafana
# http://localhost:3001  (admin / admin)

# 5. Open Prometheus
# http://localhost:9090
```

---

## API Reference

### POST /shorten

```bash
curl -X POST http://localhost:3000/shorten \
  -H "Content-Type: application/json" \
  -d '{"url": "https://github.com", "custom_slug": "mygithub", "expiry_hours": 24}'
```

Request:
```json
{ "url": "https://example.com", "custom_slug": "optional", "expiry_hours": 24 }
```

Response (200):
```json
{ "short_url": "http://localhost:3000/mygithub" }
```

| Status | Reason |
|---|---|
| 400 | Invalid URL or slug (too short/long) |
| 409 | Custom slug already taken |
| 429 | Rate limit exceeded (>10 req/min per IP) |
| 500 | Server error |

### GET /:slug

Redirects to the original URL (HTTP 302).

### GET /links

Returns all saved links with ID, Slug, OriginalURL, CreatedAt, ExpiresAt.

### GET /analytics/:slug

```json
{ "slug": "mygithub", "clicks": 4 }
```

### GET /metrics

Prometheus metrics endpoint.

---

## Redirect Flow (Cache-Aside)

```
GET /:slug
    │
    ▼
Redis GET slug
    │
 ┌──┴──────────────────┐
 │                     │
 HIT                MISS (redis.Nil)
 │                     │
 ▼                     ▼
URL              PostgreSQL SELECT
 │                     │
 │                Not found → 404
 │                     │
 │              Redis SET slug url 24h
 │                     │
 └────────┬────────────┘
          ▼
    RecordClick(slug) → channel
          ▼
      HTTP 302
```

---

## Rate Limiting

- Key: `rate_limit:<client IP>`
- Window: 60 seconds (EXPIRE on first request)
- Limit: 10 requests per window (INCR counter)
- Exceeded: HTTP 429 Too Many Requests

**Why INCR?** Atomic — concurrent requests cannot race on the counter.
**Why EXPIRE only on count == 1?** Window starts at first request and is fixed.
Resetting on every request would allow indefinite bypassing.

---

## Click Tracking

```
Redirect handler
  ▼
RecordClick(slug) → ClickQueue (buffered channel, cap 100)
                         │
            ┌────────────┼────────────┐
            ▼            ▼            ▼
         Worker 1    Worker 2    Worker 3
                         │
                         ▼
            PostgreSQL INSERT INTO clicks
```

The 302 redirect is returned before the INSERT completes.
Click counts are read via `SELECT COUNT(*) FROM clicks WHERE slug = $1`.

---

## Why Redis?

- **Speed** — in-memory lookups take microseconds vs milliseconds for PostgreSQL
- **TTL** — native key expiry maps directly to link expiration
- **SETNX** — atomic "set if not exists" prevents duplicate slug creation under concurrency
- **INCR** — atomic increment for rate limiting — no race conditions
- **Simplicity** — a URL shortener is a key→value problem, which is exactly what Redis solves

## Why PostgreSQL?

- **Durability** — ACID, survives restarts. Redis loses data on restart.
- **Source of truth** — if Redis loses data, PostgreSQL still has everything
- **Structured data** — links table with timestamps, expiry, and relational click records
- **SQL aggregations** — `SELECT COUNT(*) FROM clicks WHERE slug = $1`

## Why Both?

They are not alternatives — they do different jobs:

| | Redis | PostgreSQL |
|---|---|---|
| Speed | Microseconds | Milliseconds |
| Durability | None (volatile) | ACID (disk-backed) |
| Role | Cache, rate limit | Source of truth |

## Why SETNX?

Two concurrent requests for the same custom slug would both see "not found" in a read-check-write pattern, and both would write — creating a duplicate.

`SETNX` is a single atomic operation. Only one write succeeds. The other returns `false` and receives a 409 error.

## Why TTL?

Redis native TTL auto-deletes a key after a duration. No cron job, no cleanup query, no extra infrastructure. The rate limit counter and expiring links both use this.

## Why Rate Limiting?

Without a limit, a single script can flood `POST /shorten`, filling the database and consuming Redis memory. 10 req/min per IP is enough for legitimate use and blocks simple automation.

## Why Background Click Workers?

A synchronous PostgreSQL INSERT on every redirect adds latency. Background goroutine workers process click events from a channel asynchronously. The redirect returns 302 immediately without waiting for the database write.

## Why Prometheus?

Logs are strings — querying "requests per second over the last 5 minutes" from logs requires parsing and aggregation infrastructure. Prometheus stores counters and histograms natively and answers that with one PromQL query:

```promql
rate(linkforge_http_requests_total[5m])
```

## Why Grafana?

The Prometheus UI is a query interface, not a monitoring dashboard. Grafana provides panels, time-range selection, and is the standard complement to Prometheus. The dashboard auto-provisions from `grafana/provisioning/` so it is available immediately after `docker compose up`.

## Why Docker?

One command starts PostgreSQL, Redis, Prometheus, and Grafana with pinned versions. No host installation required. Developers get an identical environment regardless of their OS.

## Why Go?

Compiled, fast startup, low memory. Goroutines make the click worker pool trivial. Deployed as a single binary with no runtime VM. Strong standard library — HTTP, JSON, and testing are all built in.

## Why Fiber?

Fiber is built on fasthttp, which avoids standard `net/http` allocations. It is measurably faster under high concurrency. For a URL shortener where the redirect path is the hot path, this matters.

---

## Running Tests

```bash
go test ./...
go vet ./...
```

All Redis tests use `miniredis` — no real Redis server needed.

**Current test suite:**

| Package | Test | What it verifies |
|---|---|---|
| `store` | TestSaveSlug | SETNX prevents duplicate slugs |
| `store` | TestGetURL | Retrieves correct URL |
| `store` | TestGetURLMissing | Returns redis.Nil on miss |
| `store` | TestGetURLRedisError | Returns real error when Redis is down |
| `store` | TestSaveSlugWithExpiry | TTL is set correctly |
| `store` | TestCacheMissAndRepopulate | Miss → repopulate → hit (cache-aside path) |
| `middleware` | TestCheckRateLimitUnderLimit | 10 requests pass, counter increments |
| `middleware` | TestCheckRateLimitOverLimit | 11th request returns count 11 |
| `middleware` | TestCheckRateLimitResetsAfterTTL | FastForward expires key, counter resets |

`TestCacheMissAndRepopulate` is the most important test. It directly mirrors the three steps the redirect handler takes on a cache miss: return redis.Nil → CacheURL → return URL.

---

## Monitoring

### Prometheus

Prometheus scrapes `http://host.docker.internal:3000/metrics` every 15 seconds.

Open: `http://localhost:9090`

Example queries:
```promql
# Requests per second
rate(linkforge_http_requests_total[1m])

# p95 latency
histogram_quantile(0.95, rate(linkforge_http_request_duration_seconds_bucket[1m]))

# Error rate
rate(linkforge_http_requests_total{status=~"4..|5.."}[1m])
```

### Grafana

Open: `http://localhost:3001`
Credentials: `admin` / `admin` (local dev only — change in production)

The Prometheus datasource and a LinkForge dashboard are auto-provisioned.
The dashboard includes:
- HTTP requests/sec
- Request latency (p95 / p99)
- Total requests (5-min window)
- Error rate (4xx / 5xx)

---

## Load Testing

Install k6: https://k6.io/docs/get-started/installation/

```bash
# Make sure the backend is running first
go run ./cmd/api

# Run the load test
k6 run loadtest/script.js
```

The script ramps from 10 → 50 → 100 VUs, holding each level for 30 seconds.
It tests `POST /shorten` and `GET /:slug` (with `redirects: 0` so only LinkForge is measured).

### Load Test Results

Ran on this machine with `k6 run loadtest/script.js` (3m 15s, up to 100 VUs).

```
k6 v2.2.0 — 8860 iterations, 8875 HTTP requests, 3m 15s
```

**Overall (all 8875 requests across the full run):**

| Metric | Value |
|---|---|
| Total requests | 8875 |
| Throughput | 45.3 req/s |
| avg latency | 1.6 ms |
| p90 | 1.7 ms |
| **p95** | **1.86 ms** ✅ (threshold < 500 ms) |
| **p99** | **5 ms** ✅ (threshold < 1000 ms) |
| max latency | 101 ms |

**Check results:**

| Check | Pass rate | Note |
|---|---|---|
| `POST /shorten: status 200` | 0.17% (15/8845) | Rate limiter returns 429 at high VU counts — expected |
| `GET /:slug: status 302` | **100%** (30/30) | All redirect checks passed |

**What the rate limiter failure means:**

`POST /shorten` is limited to **10 requests per minute per IP**. At 100 VUs all running from
the same machine (same IP), the counter hits the limit almost immediately and subsequent requests
get `HTTP 429 Too Many Requests`. This is **correct behaviour** — the rate limiter is working.

The latency thresholds passed because `429` responses are returned instantly from Redis
(< 2 ms), which pulls the overall p95 down. The redirect path (which bypasses the rate limiter)
also confirmed 100% pass rate.

**To measure redirect performance independently**, the updated `loadtest/script.js` uses a
`setup()` function to pre-create one slug, then each VU tests the redirect path without
being blocked by the rate limiter.

**What the numbers mean:**
- **Req/s** — throughput. Higher is better.
- **p95** — 95% of requests finished in under this time. Target: < 500 ms.
- **p99** — 99% of requests finished in under this time. Catches outliers.
- **Rate limiter 429s** — not a bug; intentional protection against request flooding.

---

## CI

GitHub Actions runs on every push and pull request:

```
Push / PR
   ▼
go vet ./...     ← static analysis
   ▼
go test ./...    ← full test suite (miniredis, no external deps)
```

Go version is read from `go.mod` via `go-version-file: go.mod`.

---

## Project Structure

```
linkforge-url-shortener/
├── cmd/api/main.go                  — Fiber setup, routes, startup
├── internal/
│   ├── config/                      — Redis + PostgreSQL config loaders
│   ├── handlers/                    — HTTP handlers (thin layer)
│   │   ├── shorten.go
│   │   ├── redirect.go
│   │   ├── analytics.go
│   │   └── links.go
│   ├── middleware/
│   │   ├── rate_limit.go            — Redis INCR/EXPIRE rate limiter
│   │   ├── rate_limit_test.go
│   │   └── metrics.go               — Prometheus counter + histogram
│   ├── models/                      — Link, ClickEvent, ShortenRequest
│   ├── services/                    — Business logic, click worker pool
│   └── store/
│       ├── postgres.go              — All PostgreSQL queries
│       ├── redis.go                 — All Redis operations
│       └── redis_test.go
├── web/
│   └── index.html                   — Single-page dashboard (HTML/CSS/JS)
├── loadtest/
│   └── script.js                    — k6 load test
├── tests/
│   └── load.js                      — Alternate k6 entry point
├── monitoring/
│   └── prometheus.yml               — Prometheus scrape config
├── grafana/
│   └── provisioning/
│       ├── datasources/
│       │   └── datasource.yml       — Auto-configure Prometheus datasource
│       └── dashboards/
│           ├── dashboard.yml        — Dashboard file provider
│           └── linkforge.json       — Pre-built dashboard panels
├── docs/
│   ├── architecture.md              — Detailed system design
│   └── interview-notes.md           — Interview Q&A cheat sheet
├── .github/workflows/ci.yml
├── docker-compose.yml               — PostgreSQL, Redis, Prometheus, Grafana
├── go.mod
└── README.md
```

---

## Stack

| Component   | Technology          |
|-------------|---------------------|
| Language    | Go                  |
| HTTP        | Fiber v2 (fasthttp) |
| Cache       | Redis 7             |
| Database    | PostgreSQL 16       |
| Containers  | Docker Compose      |
| Monitoring  | Prometheus + Grafana |
| Load tests  | k6                  |
| CI          | GitHub Actions      |
| Frontend    | Vanilla HTML/CSS/JS |
