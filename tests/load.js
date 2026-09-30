import http from "k6/http";
import { check, sleep } from "k6";
import { Rate } from "k6/metrics";

// ── Custom metric: tracks error rate ──────────────────────────────────────────
const errorRate = new Rate("errors");

// ── Test configuration ────────────────────────────────────────────────────────
// Three sequential stages: ramp up → sustain → ramp down.
// Adjust BASE_URL if the backend is on a different address.
const BASE_URL = __ENV.BASE_URL || "http://localhost:3000";

export const options = {
  stages: [
    { duration: "30s", target: 10  },   // ramp to 10 VUs
    { duration: "30s", target: 50  },   // ramp to 50 VUs
    { duration: "30s", target: 100 },   // ramp to 100 VUs
    { duration: "30s", target: 0   },   // ramp down
  ],
  thresholds: {
    // p95 response time must be under 500 ms
    http_req_duration: ["p(95)<500", "p(99)<1000"],
    // error rate must stay below 5 %
    errors: ["rate<0.05"],
  },
};

// ── Scenario ──────────────────────────────────────────────────────────────────
// Each virtual user runs this function in a loop for the test duration.
// We exercise both the shorten endpoint and the redirect endpoint so that
// the Redis cache and PostgreSQL are both exercised.
export default function () {
  // 1. Create a short URL (exercises POST /shorten, rate limiter, Redis, PostgreSQL)
  const payload = JSON.stringify({
    url: "https://example.com/load-test",
  });

  const createRes = http.post(`${BASE_URL}/shorten`, payload, {
    headers: { "Content-Type": "application/json" },
  });

  const createOK = check(createRes, {
    "create: status is 200 or 201": (r) =>
      r.status === 200 || r.status === 201,
    "create: has short_url field": (r) => {
      try {
        return JSON.parse(r.body).short_url !== undefined;
      } catch {
        return false;
      }
    },
  });

  errorRate.add(!createOK);

  // 2. Follow the short URL to test the redirect path (Redis cache hit after
  //    the first request, Redis miss + PostgreSQL fallback on a fresh key).
  if (createOK) {
    let shortURL;
    try {
      shortURL = JSON.parse(createRes.body).short_url;
    } catch {
      shortURL = null;
    }

    if (shortURL) {
      // redirects: false so k6 does not auto-follow; we just check the 302
      const redirectRes = http.get(shortURL, { redirects: 0 });
      const redirectOK = check(redirectRes, {
        "redirect: status is 302": (r) => r.status === 302,
      });
      errorRate.add(!redirectOK);
    }
  }

  // 3. Fetch the links list (exercises GET /links → PostgreSQL)
  const linksRes = http.get(`${BASE_URL}/links`);
  const linksOK = check(linksRes, {
    "links: status is 200": (r) => r.status === 200,
  });
  errorRate.add(!linksOK);

  sleep(1);
}
