// ============================================================================
// SMOKE TEST — Deployment Verification
// ============================================================================
//
// Purpose:
//   Verify the application is deployed correctly and responding to requests.
//   This is the first test to run after any deployment.
//
// Configuration:
//   - 1 virtual user
//   - 30 second duration
//
// Endpoints tested:
//   - GET /health/live   (liveness probe — process is running)
//   - GET /health/ready  (readiness probe — dependencies connected)
//   - GET /             (root — embedded frontend serves)
//
// Expected results:
//   - 100% success rate (zero failed requests)
//   - All responses return HTTP 200
//
// Usage:
//   k6 run benchmark/smoke.js
//   k6 run --env BASE_URL=https://your-domain.com benchmark/smoke.js
//
// ============================================================================

import http from "k6/http";
import { check, sleep } from "k6";

// ---------------------------------------------------------------------------
// Configuration
// ---------------------------------------------------------------------------

const BASE_URL = __ENV.BASE_URL || "http://localhost:8080";

export const options = {
  // Single virtual user for 30 seconds
  vus: 1,
  duration: "30s",

  // Strict thresholds — any failure means deployment is broken
  thresholds: {
    http_req_failed: ["rate==0"],          // Zero failed requests
    http_req_duration: ["p(99)<1000"],     // All requests under 1s
  },
};

// ---------------------------------------------------------------------------
// Test Execution
// ---------------------------------------------------------------------------

export default function () {
  // Liveness check — confirms process is running
  const live = http.get(`${BASE_URL}/health/live`);
  check(live, {
    "GET /health/live returns 200": (r) => r.status === 200,
  });

  // Readiness check — confirms database connection is healthy
  const ready = http.get(`${BASE_URL}/health/ready`);
  check(ready, {
    "GET /health/ready returns 200": (r) => r.status === 200,
  });

  // Root endpoint — confirms embedded frontend is served
  const root = http.get(`${BASE_URL}/`);
  check(root, {
    "GET / returns 200": (r) => r.status === 200,
  });

  // Small pause between iterations to avoid tight-looping
  sleep(1);
}
