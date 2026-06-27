# Benchmark Suite

k6 performance benchmark scripts for the modular-monolith infrastructure.

## Scripts

| Script | Purpose | VUs | Duration |
|--------|---------|-----|----------|
| `smoke.js` | Verify deployment correctness | 1 | 30s |
| `load.js` | Measure normal throughput | 25→50→100 | ~5 min |
| `stress.js` | Find failure point | 100→250→500→750→1000 | ~10 min |
| `soak.js` | Detect resource leaks | 100 | 60 min |

## Prerequisites

Install k6: https://k6.io/docs/get-started/installation/

```bash
# macOS
brew install k6

# Debian/Ubuntu
sudo gpg -k
sudo gpg --no-default-keyring --keyring /usr/share/keyrings/k6-archive-keyring.gpg --keyserver hkp://keyserver.ubuntu.com:80 --recv-keys C5AD17C747E3415A3642D57D77C6C491D6AC1D69
echo "deb [signed-by=/usr/share/keyrings/k6-archive-keyring.gpg] https://dl.k6.io/deb stable main" | sudo tee /etc/apt/sources.list.d/k6.list
sudo apt-get update && sudo apt-get install k6

# Docker (no install needed)
docker run --rm -i grafana/k6 run - <benchmark/smoke.js
```

## Quick Start

```bash
# Start the application
make docker-up

# Run benchmarks via Makefile
make benchmark-smoke
make benchmark-load
make benchmark-stress
make benchmark-soak
```

## Custom Base URL

All scripts accept a `BASE_URL` environment variable:

```bash
k6 run --env BASE_URL=https://staging.example.com benchmark/load.js
```

## Execution Order

Run in this order for a complete assessment:

1. **Smoke** — confirms deployment is healthy
2. **Load** — establishes baseline metrics
3. **Stress** — finds breaking point
4. **Soak** — detects leaks (run overnight or during low-activity periods)

## Full Documentation

See [docs/PERFORMANCE_TESTING.md](../docs/PERFORMANCE_TESTING.md) for detailed interpretation guidance.
