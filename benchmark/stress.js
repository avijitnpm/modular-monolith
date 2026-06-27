// ============================================================================
// STRESS TEST — Find Failure Point
// ============================================================================
//
// Purpose:
//   Identify the point at which the application begins to degrade.
//   Ramps VUs aggressively until failures, high latency, or resource
//   exhaustion occur.
//
// Configuration:
//   - Ramp: 100 → 250 → 500 → 750 → 1000 VUs
//   - Each stage holds briefly to measure stability at that level
//   - Total duration: ~10 minutes
//
// What to watch for:
//   - HTTP response times exceeding 2s (p95)
//   - Error rate climbing above 5%
//   - Container restarts (check `docker ps` after run)
//   - OOM kills (check `dmesg` or `docker inspect`)
//   - Connection timeouts or refused connections
//
// Interpreting results:
//   The VU count where p95 latency exceeds 2s or error rate exceeds 5%
//   marks the degradation point. Document this value for capacity planning.
//
// Endpoints tested:
//   - GET /health/live   (lightweight)
//   - GET /health/ready  (database dependency)
//   - GET /              (embedded frontend)
//
// Usage:
//   k6 run benchmark/stress.js
//   k6 run --env BASE_URL=https://your-domain.com benchmark/stress.js
//
// ============================================================================

import http from "k6/http";
import { check, sleep } from "k6";

// ---------------------------------------------------------------------------
// Configuration
// ---------------------------------------------------------------------------

const BASE_URL = __ENV.BASE_URL || "http://localhost:8080";

export const options = {
  // Aggressive ramp — each stage increases load significantly
  stages: [
    { duration: "30s", target: 100 },   // Warm-up to 100 VUs
    { duration: "1m",  target: 100 },   // Hold — baseline
    { duration: "30s", target: 250 },   // Ramp to 250 VUs
    { duration: "1m",  target: 250 },   // Hold — observe
    { duration: "30s", target: 500 },   // Ramp to 500 VUs
    { duration: "1m",  target: 500 },   // Hold — observe
    { duration: "30s", target: 750 },   // Ramp to 750 VUs
    { duration: "1m",  target: 750 },   // Hold — observe
    { duration: "30s", target: 1000 },  // Ramp to 1000 VUs
    { duration: "1m",  target: 1000 },  // Hold — observe breaking point
    { duration: "1m",  target: 0 },     // Ramp down — observe recovery
  ],

  // Relaxed thresholds — this test expects degradation
  // Thresholds here document acceptable limits, not pass/fail
  thresholds: {
    http_req_duration: ["p(95)<5000"],   // Flag if p95 exceeds 5s
    http_req_failed: ["rate<0.50"],      // Flag if >50% fail (total collapse)
  },
};

// ---------------------------------------------------------------------------
// Test Execution
// ---------------------------------------------------------------------------

export default function () {
  // Liveness — should remain fast even under stress
  const live = http.get(`${BASE_URL}/health/live`);
  check(live, {
    "GET /health/live returns 200": (r) => r.status === 200,
  });

  // Readiness — database pool pressure shows here first
  const ready = http.get(`${BASE_URL}/health/ready`);
  check(ready, {
    "GET /health/ready returns 200": (r) => r.status === 200,
  });

  // Root — full response, memory/GC pressure
  const root = http.get(`${BASE_URL}/`);
  check(root, {
    "GET / returns 200": (r) => r.status === 200,
  });

  // Minimal sleep — maximize pressure
  sleep(0.5);
}
