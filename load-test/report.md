# Load Test Report

## Test Configuration

- **Duration:** 2 minutes (ramp-up) + 30 seconds (sustained)
- **Load Profile:**
  - 0-30s: Ramp to 100 RPS
  - 30-60s: Ramp to 500 RPS
  - 60-90s: Ramp to 1000 RPS
  - 90-120s: Hold at 1000 RPS
- **Test Environment:**
  - Backend: Go service on port 8080
  - Database: PostgreSQL 16
  - Cache: Redis 7
  - Test Links: 100

## Results Summary

### Maximum RPS Achieved
- **Target:** 1000 RPS
- **Achieved:** [To be filled after test run]

### Latency Percentiles

| Percentile | Target | Achieved | Status |
|------------|--------|----------|--------|
| p50        | < 10ms | [TBD]    | [TBD]  |
| p95        | < 100ms| [TBD]    | [TBD]  |
| p99        | < 200ms| [TBD]    | [TBD]  |

### Error Rate
- **Target:** < 1%
- **Achieved:** [To be filled after test run]

### Throughput
- **Peak RPS:** [To be filled]
- **Average RPS:** [To be filled]

## Performance Analysis

### Where did system start degrading?
[To be filled after analysis]

### What was the bottleneck?
- [ ] CPU
- [ ] Memory
- [ ] Database I/O
- [ ] Redis I/O
- [ ] Network
- [ ] Other: [specify]

### Resource Usage

#### CPU Usage
- **Average:** [TBD]%
- **Peak:** [TBD]%

#### Memory Usage
- **Average:** [TBD] MB
- **Peak:** [TBD] MB

#### Database Connections
- **Active:** [TBD]
- **Idle:** [TBD]
- **Max:** 25

#### Redis Connections
- **Active:** [TBD]
- **Max:** 10

### Cache Hit Rate
- **Target:** > 80%
- **Achieved:** [TBD]%

## Performance Graphs

### Latency Over Time
[Insert graph showing latency percentiles over time]

### Throughput Over Time
[Insert graph showing RPS over time]

### Error Rate Over Time
[Insert graph showing error rate over time]

## Conclusions

### System Capacity
[To be filled after analysis]

### Recommendations
1. [To be filled]
2. [To be filled]
3. [To be filled]

## Test Run Details

- **Date:** [Date of test]
- **k6 Version:** [Version]
- **Test Duration:** [Actual duration]
- **Total Requests:** [Total]
- **Successful Requests:** [Count]
- **Failed Requests:** [Count]

