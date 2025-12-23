# Load Testing Guide

This directory contains the k6 load testing script for the Link Analytics Service.

## Prerequisites

Install k6:
- macOS: `brew install k6`
- Linux: See [k6 installation guide](https://k6.io/docs/getting-started/installation/)
- Windows: Download from [k6 releases](https://github.com/grafana/k6/releases)

## Running the Load Test

1. Make sure the backend is running:
   ```bash
   docker-compose up -d backend
   ```

2. Run the load test:
   ```bash
   k6 run script.js
   ```

3. For JSON output (for reporting):
   ```bash
   k6 run --out json=results.json script.js
   ```

## Test Configuration

The test script:
- Creates 100 test links in the setup phase
- Ramps from 100 to 1000 RPS over 2 minutes
- Holds at 1000 RPS for 30 seconds
- Tests the redirect endpoint (GET /{shortCode})
- Measures latency and error rates

## Thresholds

- p95 latency < 100ms
- p99 latency < 200ms
- Error rate < 1%

## Environment Variables

- `BASE_URL`: Base URL of the backend (default: http://localhost:8080)

Example:
```bash
BASE_URL=http://localhost:8080 k6 run script.js
```

## Generating Reports

After running with JSON output, you can analyze the results:

```bash
# View summary
cat results.json | jq '.'

# Extract key metrics
cat results.json | jq '.metrics.http_req_duration.values'
```

## Expected Results

At 1000 RPS:
- p95 latency should be < 100ms
- p99 latency should be < 200ms
- Error rate should be < 1%
- Cache hit rate should be > 80%

