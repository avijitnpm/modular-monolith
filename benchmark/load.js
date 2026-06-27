// ============================================================================
// LOAD TEST — Normal Operating Throughput
// ============================================================================
//
// Purpose:
//   Determine the application's throughput under expected production load.
//   Establishes baseline performance metrics at 25, 50, and 100 concurrent users.
//
// Configuration:
//   - Ramp: 0 → 25 → 50 → 100 → 0 VUs
//   - Total duration: ~5 minutes
//   - Each stage holds for enough time to collect stable metrics
//
// Endpoints tested:
//   - GET /health/live   (lightweight — no dependencies)
//   - GET /health/ready  (medium — checks database)
//   - GET /              (heavy — serves embedded frontend)
//
// Metrics collected:
//   - Response time: p50, p95, p99
//   - Requests per second (throughput)
//   - Failed request rate
//
// Usage:
//   k6 run benchmark/load.js
//   k6 run --env BASE_URL=https://your-domain.com benchmark/load.js
//
// ============================================================================

import http from "k6/http";
import { check, sleep } from "k6";

// ---------------------------------------------------------------------------
// Configuration
// ---------------------------------------------------------------------------

const BASE_URL = __ENV.BASE_URL || "http://localhost:8080";

export const options = {
  // Staged ramp-up simulating growing production traffic
  stages: [
    { duration: "30s", target: 25 },   // Ramp to 25 VUs
    { duration: "1m",  target: 25 },   // Hold at 25 VUs
    { duration: "30s", target: 50 },   // Ramp to 50 VUs
    { duration: "1m",  target: 50 },   // Hold at 50 VUs
    { duration: "30s", target: 100 },  // Ramp to 100 VUs
    { duration: "1m",  target: 100 },  // Hold at 100 VUs
    { duration: "30s", target: 0 },    // Ramp down
  ],

  // Thresholds define acceptable performance under normal load
  thresholds: {
    http_req_duration: [
      "p(50)<200",    // Median under 200ms
      "p(95)<500",    // 95th percentile under 500ms
      "p(99)<1000",   // 99th percentile under 1s
    ],
    http_req_failed: ["rate<0.01"],  // Less than 1% failure rate
  },
};

// ---------------------------------------------------------------------------
// Test Execution
// ---------------------------------------------------------------------------

export default function () {
  // Liveness — lightweight check, establishes throughput ceiling
  const live = http.get(`${BASE_URL}/health/live`);
  check(live, {
    "GET /health/live returns 200": (r) => r.status === 200,
  });

  // Readiness — involves database round-trip
  const ready = http.get(`${BASE_URL}/health/ready`);
  check(ready, {
    "GET /health/ready returns 200": (r) => r.status === 200,
  });

  // Root — serves the full embedded frontend
  const root = http.get(`${BASE_URL}/`);
  check(root, {
    "GET / returns 200": (r) => r.status === 200,
  });

  // Simulate realistic user think time between requests
  sleep(1);
}
