// ============================================================================
// SOAK TEST — Resource Leak Detection
// ============================================================================
//
// Purpose:
//   Detect resource leaks by running sustained load over an extended period.
//   Memory leaks, connection pool exhaustion, goroutine leaks, and file
//   descriptor leaks will manifest over time under constant pressure.
//
// Configuration:
//   - 100 virtual users
//   - 60 minute duration
//   - Constant load (no ramp — leak detection requires steady state)
//
// What to monitor during execution:
//   - Memory growth:     docker stats (RSS should remain stable)
//   - CPU growth:        docker stats (should not trend upward)
//   - Goroutine count:   /debug/pprof/goroutine (if enabled)
//   - Connection count:  SELECT count(*) FROM pg_stat_activity
//   - Error count:       k6 output (should remain at 0%)
//
// How to interpret results:
//   - Stable memory + stable latency = no leaks
//   - Growing memory over time = memory leak (likely unclosed resources)
//   - Growing goroutine count = goroutine leak
//   - Growing connection count = connection pool leak
//   - Increasing latency over time = GC pressure or resource contention
//
// Endpoints tested:
//   - GET /health/live   (lightweight — goroutine/memory baseline)
//   - GET /health/ready  (database — connection pool stability)
//   - GET /              (full response — GC pressure)
//
// Usage:
//   k6 run benchmark/soak.js
//   k6 run --env BASE_URL=https://your-domain.com benchmark/soak.js
//   k6 run --env DURATION=30m benchmark/soak.js   # shorter soak
//
// ============================================================================

import http from "k6/http";
import { check, sleep } from "k6";

// ---------------------------------------------------------------------------
// Configuration
// ---------------------------------------------------------------------------

const BASE_URL = __ENV.BASE_URL || "http://localhost:8080";
const DURATION = __ENV.DURATION || "60m";

export const options = {
  // Constant load — ramps quickly then holds for the entire duration
  stages: [
    { duration: "1m",    target: 100 },  // Ramp up to 100 VUs
    { duration: DURATION, target: 100 }, // Hold at 100 VUs for full duration
    { duration: "30s",   target: 0 },    // Ramp down
  ],

  // Thresholds for a healthy soak — same as load test
  // If these fail after 60 minutes but pass initially, there is a leak
  thresholds: {
    http_req_duration: [
      "p(50)<200",    // Median should stay stable
      "p(95)<500",    // p95 should not drift upward
      "p(99)<1000",   // p99 should not drift upward
    ],
    http_req_failed: ["rate<0.01"],  // Less than 1% failure rate
  },
};

// ---------------------------------------------------------------------------
// Test Execution
// ---------------------------------------------------------------------------

export default function () {
  // Liveness — baseline operation, tests goroutine creation/cleanup
  const live = http.get(`${BASE_URL}/health/live`);
  check(live, {
    "GET /health/live returns 200": (r) => r.status === 200,
  });

  // Readiness — exercises database connection pool over time
  const ready = http.get(`${BASE_URL}/health/ready`);
  check(ready, {
    "GET /health/ready returns 200": (r) => r.status === 200,
  });

  // Root — exercises memory allocation for full page responses
  const root = http.get(`${BASE_URL}/`);
  check(root, {
    "GET / returns 200": (r) => r.status === 200,
  });

  // Standard think time — keeps load consistent
  sleep(1);
}
