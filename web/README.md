# LinkForge – Frontend Dashboard

This directory contains the frontend for the LinkForge URL shortener.
It is a **single HTML file** with no external dependencies, no build step, and no frameworks.

---

## Files

```
web/
├── index.html        — The complete single-page dashboard (HTML + CSS + JS)
├── README.md         — This file
└── INTERVIEW_NOTES.md — Technical cheat sheet for interviews
```

---

## APIs Used

| Frontend Action      | HTTP Method | Endpoint              |
|----------------------|-------------|-----------------------|
| Create short URL     | POST        | `/shorten`            |
| Load all links       | GET         | `/links`              |
| Get click analytics  | GET         | `/analytics/:slug`    |

---

## Request / Response Flow

### Creating a Short URL

```
User fills form and clicks "Create Short URL"
        ↓
JavaScript reads form values
        ↓
fetch() POST /shorten   { url, custom_slug, expiry_hours }
        ↓
Go Fiber receives JSON body
        ↓
Shorten handler → services.CreateShortURL()
        ↓
Service validates URL and slug, generates slug if none provided
        ↓
Redis stores slug → URL  (with TTL)
        ↓
PostgreSQL stores persistent record (slug, original_url, expires_at)
        ↓
JSON response: { "short_url": "http://localhost:3000/abc123" }
        ↓
Dashboard displays clickable short URL
Dashboard refreshes the links table
```

### Loading All Links

```
Page loads  (or new link is created)
        ↓
fetch() GET /links
        ↓
Go Fiber → ListLinks handler → store.GetLinks()
        ↓
PostgreSQL SELECT * FROM links
        ↓
JSON array: [{ ID, Slug, OriginalURL, CreatedAt, ExpiresAt }, ...]
        ↓
JavaScript renders a table row for each link
```

---

## How Analytics Works

Each time a user visits a short URL (`GET /:slug`), the backend calls
`services.RecordClick(slug)`, which asynchronously writes a row to the
`clicks` table in PostgreSQL.

When the Analytics button is clicked in the dashboard:

```
User clicks "Analytics" button for a slug
        ↓
fetch() GET /analytics/:slug
        ↓
Go Fiber → Analytics handler → store.GetClickCount(slug)
        ↓
PostgreSQL SELECT COUNT(*) FROM clicks WHERE slug = ?
        ↓
JSON response: { "slug": "mygithub", "clicks": 4 }
        ↓
Dashboard displays "Clicks: 4" in the table row
```

The analytics request does **not** reload the page.
Only the one table cell is updated in place.

---

## Why fetch() Is Used

`fetch()` is the modern browser-native way to make HTTP requests from JavaScript.
It returns a Promise, so we can use `async/await` for clean, readable code.
No libraries (such as axios) are needed for simple GET and POST requests.

---

## Why Vanilla JavaScript Was Chosen

- The dashboard has three simple operations: create, list, and analytics.
- A framework like React adds complexity that is not justified for a project this size.
- Vanilla JS is easier for a student to read, debug, and explain in an interview.
- There is no build step, so the file can be opened directly in a browser or served as a static file.

---

## How the Frontend Is Served by Fiber

At the moment the frontend is a standalone file.
To serve it from the Go Fiber application, a static file route must be added to `main.go`:

```go
app.Static("/dashboard", "./web")
```

Once that route exists, the dashboard will be available at:

```
http://localhost:3000/dashboard
```

> **Note:** This static route does not currently exist in the backend.
> The frontend developer or backend developer must add it when ready.
> Do not open `index.html` as a `file://` URL when testing API calls,
> because the browser will block the requests. Use a local server instead.

---

## How to Run

**1. Start PostgreSQL and Redis using Docker Compose:**

```bash
docker compose up -d
```

**2. Start the Go backend:**

```bash
go run ./cmd/api
```

**3. Open the dashboard:**

Once a Fiber static route pointing to the `web/` directory has been configured,
open a browser and visit:

```
http://localhost:3000/dashboard
```

Until the static route is added, you can serve the file with any static server,
for example:

```bash
# Python (built into most systems)
python -m http.server 8080 --directory web
```

Then visit `http://localhost:8080` and the API calls will go to `http://localhost:3000`.

> If you use a separate server for the HTML file, set `API_BASE_URL` inside
> `index.html` to `"http://localhost:3000"` so fetch() calls use the correct address.
