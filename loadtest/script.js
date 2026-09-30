/**
 * LinkForge k6 Load Test
 *
 * Tests POST /shorten and GET /:slug at three VU levels.
 *
 * IMPORTANT: POST /shorten is rate-limited to 10 req/min per IP.
 * At 100 VUs, most shorten requests will receive HTTP 429.
 * This is expected and correct behaviour — the rate limiter is working.
 *
 * The redirect test (GET /:slug) uses a slug pre-created in the setup()
 * function so every VU can test redirects without being blocked by the limiter.
 *
 * Run:
 *   k6 run loadtest/script.js
 *
 * Override base URL:
 *   BASE_URL=http://myserver:3000 k6 run loadtest/script.js
 */
import http from "k6/http";
import { check, sleep } from "k6";
import { SharedArray } from "k6/data";

const BASE_URL = __ENV.BASE_URL || "http://localhost:3000";

export const options = {
  stages: [
    { duration: "30s", target: 10  },  // ramp to 10 VUs
    { duration: "30s", target: 10  },  // hold at 10 VUs
    { duration: "30s", target: 50  },  // ramp to 50 VUs
    { duration: "30s", target: 50  },  // hold at 50 VUs
    { duration: "30s", target: 100 },  // ramp to 100 VUs
    { duration: "30s", target: 100 },  // hold at 100 VUs
    { duration: "15s", target: 0   },  // ramp down
  ],
  thresholds: {
    // Latency thresholds apply to all requests (including the fast 429s)
    http_req_duration: ["p(95)<500", "p(99)<1000"],
  },
};

// setup() runs once before the test. Creates one slug for all VUs to use
// for redirect testing, so the redirect path is not affected by rate limiting.
export function setup() {
  const payload = JSON.stringify({ url: "https://example.com/load-test" });
  const res = http.post(BASE_URL + "/shorten", payload, {
    headers: { "Content-Type": "application/json" },
  });
  if (res.status !== 200) {
    console.error("setup: could not create slug, status=" + res.status);
    return { shortURL: null };
  }
  const body = JSON.parse(res.body);
  console.log("setup: created slug at " + body.short_url);
  return { shortURL: body.short_url };
}

export default function (data) {
  // ── 1. Attempt to create a short URL ─────────────────────────────────────
  // At high VU counts, most of these will return 429 (rate limiter).
  // That is the correct and expected behaviour.
  const payload = JSON.stringify({ url: "https://example.com/load-test" });
  const createRes = http.post(BASE_URL + "/shorten", payload, {
    headers: { "Content-Type": "application/json" },
  });

  check(createRes, {
    "POST /shorten: 200 or 429": (r) => r.status === 200 || r.status === 429,
  });

  // ── 2. Redirect using the pre-created slug ────────────────────────────────
  // redirects: 0 tells k6 NOT to follow the 302.
  // This measures LinkForge only — not the external destination.
  if (data.shortURL) {
    const redirectRes = http.get(data.shortURL, { redirects: 0 });
    check(redirectRes, {
      "GET /:slug: status 302": (r) => r.status === 302,
    });
  }

  sleep(1);
}
