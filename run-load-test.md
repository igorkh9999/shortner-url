# How to Run the Load Test

## Prerequisites Check

1. ✅ Backend is running (already confirmed)
2. ⚠️ k6 must be in PATH - **restart your terminal** after adding to PATH

## Steps to Run

### Option 1: Basic Load Test

```cmd
cd C:\playground\shortner-url\load-test
k6 run script.js
```

### Option 2: With JSON Output (for analysis)

```cmd
cd C:\playground\shortner-url\load-test
k6 run --out json=results.json script.js
```

### Option 3: If k6 is not in PATH, use full path

```cmd
cd C:\playground\shortner-url\load-test
C:\k6\k6.exe run script.js
```

(Replace `C:\k6\k6.exe` with your actual k6 path)

### Option 4: Custom BASE_URL

If your backend runs on a different URL:

**In Command Prompt:**

```cmd
cd C:\playground\shortner-url\load-test
set BASE_URL=http://localhost:8080
k6 run script.js
```

**In PowerShell:**

```powershell
cd C:\playground\shortner-url\load-test
$env:BASE_URL="http://localhost:8080"
k6 run script.js
```

## What to Expect

The test will:

1. **Setup phase**: Create 100 test links (takes a few seconds)
2. **Ramp up**: 100 → 500 → 1000 RPS over 90 seconds
3. **Sustain**: Hold at 1000 RPS for 30 seconds
4. **Total duration**: ~2 minutes

## Expected Results

At 1000 RPS:

-   ✅ p95 latency < 100ms
-   ✅ p99 latency < 200ms
-   ✅ Error rate < 1%
-   ✅ Cache hit rate > 80%

## Troubleshooting

-   **"k6 is not recognized"**: Close and reopen your terminal, or use full path to k6.exe
-   **Connection errors**: Make sure backend is running: `docker-compose ps`
-   **Test fails**: Check backend logs: `docker-compose logs backend`
