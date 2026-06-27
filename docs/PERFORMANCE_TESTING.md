# Performance Testing

Complete guide to the k6 benchmarking framework for the modular-monolith.

---

## Purpose

This framework provides repeatable performance benchmarks for the infrastructure layer. It does not test business logic. It validates that the HTTP server, database connections, and embedded frontend serve correctly under load.

Every SaaS built from this template inherits this framework. Run benchmarks after infrastructure changes, dependency upgrades, or deployment configuration modifications.

---

## Prerequisites

| Requirement | Purpose |
|-------------|---------|
| k6 | Load testing tool |
| Running application | Target for benchmarks |
| Docker (optional) | Run k6 without local install |

---

## Installing k6

### macOS

```bash
brew install k6
```

### Debian / Ubuntu

```bash
sudo gpg -k
sudo gpg --no-default-keyring --keyring /usr/share/keyrings/k6-archive-keyring.gpg \
  --keyserver hkp://keyserver.ubuntu.com:80 \
  --recv-keys C5AD17C747E3415A3642D57D77C6C491D6AC1D69
echo "deb [signed-by=/usr/share/keyrings/k6-archive-keyring.gpg] https://dl.k6.io/deb stable main" \
  | sudo tee /etc/apt/sources.list.d/k6.list
sudo apt-get update && sudo apt-get install k6
```

### Windows

```powershell
choco install k6
# or
winget install k6 --source winget
```

### Docker (no install)

```bash
docker run --rm -i --network host grafana/k6 run - <benchmark/smoke.js
```

### Verify installation

```bash
k6 version
```

---

## Executing Benchmarks

### Start the application

```bash
make docker-up
# Wait for health check to pass:
curl http://localhost:8080/health/ready
```

### Run via Makefile

```bash
make benchmark-smoke     # 30 seconds — verify deployment
make benchmark-load      # ~5 minutes — normal throughput
make benchmark-stress    # ~10 minutes — find breaking point
make benchmark-soak      # ~60 minutes — detect leaks
```

### Run directly

```bash
k6 run benchmark/smoke.js
k6 run benchmark/load.js
k6 run benchmark/stress.js
k6 run benchmark/soak.js
```

### Custom target URL

```bash
k6 run --env BASE_URL=https://staging.example.com benchmark/load.js
```

### Custom soak duration

```bash
k6 run --env DURATION=30m benchmark/soak.js
```

---

## Execution Order

Always run in this sequence:

1. **Smoke** — If this fails, stop. The deployment is broken.
2. **Load** — Establishes baseline performance numbers.
3. **Stress** — Identifies the capacity ceiling.
4. **Soak** — Detects slow leaks (run during low-activity windows).

---

## Interpreting Results

### k6 Output Metrics

| Metric | Meaning |
|--------|---------|
| `http_req_duration` | Time from request start to response received |
| `http_req_failed` | Percentage of non-2xx responses |
| `http_reqs` | Total requests made (divide by duration for req/s) |
| `vus` | Current number of active virtual users |
| `iterations` | Complete script executions |

### Percentiles

| Percentile | Interpretation |
|------------|---------------|
| p(50) | Median — what most users experience |
| p(95) | Tail latency — what 1 in 20 users experiences |
| p(99) | Worst case — what 1 in 100 users experiences |

### Smoke Test

| Result | Meaning |
|--------|---------|
| All checks pass | Deployment is healthy |
| Any check fails | Deployment is broken — investigate immediately |

### Load Test

| Result | Meaning |
|--------|---------|
| All thresholds pass | Application handles expected production load |
| p95 > 500ms | Response times are high — investigate bottleneck |
| Failures > 1% | Errors under normal load — critical issue |

### Stress Test

| Result | Meaning |
|--------|---------|
| Degrades at 250 VUs | Low capacity — likely connection pool or CPU bound |
| Degrades at 500 VUs | Moderate capacity — suitable for most workloads |
| Degrades at 1000 VUs | High capacity — infrastructure scales well |
| Recovers after ramp-down | Application is resilient |
| Does not recover | Resource leak or broken state — investigate |

### Soak Test

| Symptom | Likely Cause |
|---------|-------------|
| Memory grows linearly | Memory leak (unclosed buffers, retained references) |
| Latency increases over time | GC pressure from memory leak |
| Connection count grows | Database connection pool leak |
| Goroutine count grows | Goroutine leak (missing context cancellation) |
| Stable metrics throughout | No leaks — application is healthy |

---

## Expected Outputs

### Healthy Smoke

```
✓ GET /health/live returns 200
✓ GET /health/ready returns 200
✓ GET / returns 200

checks.........................: 100.00%
http_req_duration..............: avg=5ms  p(99)=20ms
http_req_failed................: 0.00%
```

### Healthy Load (100 VUs)

```
http_req_duration..............: avg=15ms  p(50)=10ms  p(95)=45ms  p(99)=120ms
http_req_failed................: 0.00%
http_reqs......................: 18000  ~60/s per VU
```

### Stress Degradation Point

```
# At 500 VUs — still healthy:
http_req_duration..............: p(95)=200ms

# At 750 VUs — degradation begins:
http_req_duration..............: p(95)=1500ms
http_req_failed................: 3.2%
```

---

## Monitoring During Tests

### Docker container resources

```bash
docker stats
```

### PostgreSQL connections

```sql
SELECT count(*) FROM pg_stat_activity;
```

### Application logs

```bash
docker compose logs -f app
```

---

## Common Failures

| Failure | Cause | Fix |
|---------|-------|-----|
| `connection refused` | Application not running | Start with `make docker-up` |
| `dial tcp: connection reset` | Connection pool exhausted | Increase pool size or reduce VUs |
| `context deadline exceeded` | Request timeout | Check database queries, increase timeout |
| All checks fail at start | Wrong BASE_URL | Verify URL with `curl` first |
| OOM kill during stress | Memory limits too low | Increase container memory limit |
| Gradual timeout increase in soak | Connection leak | Check `pg_stat_activity` for idle connections |

---

## Adding New Benchmarks

When adding API endpoints to the application, extend the benchmark scripts:

1. Add the endpoint to the appropriate test script
2. Add a `check()` call to verify the response
3. Update thresholds if the endpoint has different performance characteristics
4. Document the expected behavior in this file

---

## File Reference

```
benchmark/
├── smoke.js       1 VU, 30s — deployment verification
├── load.js        25→50→100 VUs, 5 min — throughput baseline
├── stress.js      100→1000 VUs, 10 min — capacity ceiling
├── soak.js        100 VUs, 60 min — leak detection
└── README.md      Quick-start guide
```
