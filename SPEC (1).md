# Link Analytics Service - Technical Specification

## Project Overview

**Goal:** Build an MVP URL shortening service with real-time analytics, designed for 1000+ redirects/second.

**Time Constraint:** 4 hours implementation time, 2-3 days deadline

**Tech Stack:**
- Backend: Go (stdlib only, net/http router)
- Frontend: Next.js 14+ (App Router)
- Database: PostgreSQL (primary), Redis (cache/counters)
- Additional: Message queue for async analytics processing

---

## Architecture Decisions

### High-Level Architecture

```
┌─────────────┐
│   Next.js   │
│  Frontend   │
└──────┬──────┘
       │ HTTP/SSE
┌──────▼──────────────────────────┐
│      Go Backend (net/http)      │
│  ┌──────────────────────────┐   │
│  │  Link Management Service │   │
│  │  Redirect Handler        │   │
│  │  Analytics Service       │   │
│  │  SSE/WebSocket Handler   │   │
│  └──────────────────────────┘   │
└───┬────────────┬────────────┬───┘
    │            │            │
┌───▼───┐   ┌───▼───┐   ┌────▼─────┐
│ Redis │   │ Queue │   │PostgreSQL│
│ Cache │   │(Chan) │   │ Database │
└───────┘   └───┬───┘   └──────────┘
                │
           ┌────▼─────┐
           │Analytics │
           │ Worker   │
           └──────────┘
```

### Why This Architecture?

1. **Redis for Hot Data:**
   - Short code → URL mapping (cache)
   - Real-time click counters (INCR is atomic)
   - Rate limiting
   - TTL: 1 hour for mappings, refresh on hit

2. **PostgreSQL for Persistent Data:**
   - Links table (source of truth)
   - Analytics events (time-series data)
   - Aggregated statistics

3. **Go Channels as Queue:**
   - Async analytics processing
   - Buffered channel (10000 capacity)
   - Prevents redirect slowdown
   - Worker pool pattern (10 workers)

4. **No Framework in Go:**
   - Use `net/http` ServeMux
   - Custom middleware chain
   - Manual JSON marshaling

---

## Database Schema

### PostgreSQL Tables

```sql
-- Links table
CREATE TABLE links (
    id SERIAL PRIMARY KEY,
    short_code VARCHAR(10) UNIQUE NOT NULL,
    original_url TEXT NOT NULL,
    user_id VARCHAR(50) NOT NULL,  -- For demo, just a string identifier
    created_at TIMESTAMP DEFAULT NOW(),
    INDEX idx_user_id (user_id),
    INDEX idx_short_code (short_code)
);

-- Analytics events (time-series optimized)
CREATE TABLE clicks (
    id BIGSERIAL PRIMARY KEY,
    short_code VARCHAR(10) NOT NULL,
    clicked_at TIMESTAMP DEFAULT NOW(),
    ip_address INET,
    user_agent TEXT,
    referer TEXT,
    visitor_hash VARCHAR(64),  -- Hash of IP + User-Agent for unique visitors
    INDEX idx_short_code_time (short_code, clicked_at DESC),
    INDEX idx_clicked_at (clicked_at)  -- For time-based queries
);

-- Aggregated statistics (updated by workers)
CREATE TABLE link_stats (
    short_code VARCHAR(10) PRIMARY KEY,
    total_clicks BIGINT DEFAULT 0,
    unique_visitors BIGINT DEFAULT 0,
    last_updated TIMESTAMP DEFAULT NOW()
);

-- Top referrers (materialized for performance)
CREATE TABLE top_referrers (
    short_code VARCHAR(10),
    referer TEXT,
    click_count BIGINT,
    PRIMARY KEY (short_code, referer),
    INDEX idx_short_code_count (short_code, click_count DESC)
);
```

### Redis Keys Structure

```
# URL cache
link:{short_code} → original_url (TTL: 3600s)

# Real-time counters (for SSE updates)
clicks:realtime:{short_code} → count (INCR, TTL: 60s)

# Rate limiting
ratelimit:{ip}:{endpoint} → count (TTL: 60s)
```

---

## API Design

### Base URL: `http://localhost:8080/api`

### Endpoints

#### 1. Create Short Link
```
POST /api/links
Content-Type: application/json

Request:
{
    "url": "https://example.com/very/long/url",
    "user_id": "user123"  // For demo purposes
}

Response (201):
{
    "short_code": "abc123",
    "short_url": "http://localhost:8080/abc123",
    "original_url": "https://example.com/very/long/url",
    "created_at": "2024-01-15T10:30:00Z"
}

Errors:
- 400: Invalid URL format
- 500: Server error
```

#### 2. Get Link Info
```
GET /api/links/{short_code}

Response (200):
{
    "short_code": "abc123",
    "original_url": "https://example.com/very/long/url",
    "created_at": "2024-01-15T10:30:00Z",
    "stats": {
        "total_clicks": 1523,
        "unique_visitors": 892
    }
}

Errors:
- 404: Link not found
```

#### 3. List User Links
```
GET /api/links?user_id=user123

Response (200):
{
    "links": [
        {
            "short_code": "abc123",
            "original_url": "https://example.com/very/long/url",
            "created_at": "2024-01-15T10:30:00Z",
            "total_clicks": 1523
        }
    ]
}
```

#### 4. Redirect (Critical Path - Optimized)
```
GET /{short_code}

Response (302):
Location: https://example.com/very/long/url

Errors:
- 404: Link not found

Implementation Notes:
1. Check Redis cache first
2. If miss, query PostgreSQL and cache
3. Fire-and-forget analytics event to channel
4. Return redirect immediately (< 10ms target)
```

#### 5. Get Analytics
```
GET /api/analytics/{short_code}?period=24h

Parameters:
- period: 24h | 7d | 30d

Response (200):
{
    "short_code": "abc123",
    "total_clicks": 1523,
    "unique_visitors": 892,
    "clicks_over_time": [
        { "timestamp": "2024-01-15T10:00:00Z", "count": 45 },
        { "timestamp": "2024-01-15T11:00:00Z", "count": 67 }
    ],
    "top_referrers": [
        { "referer": "https://twitter.com", "count": 450 },
        { "referer": "https://facebook.com", "count": 320 }
    ]
}
```

#### 6. Real-time Click Stream (SSE)
```
GET /api/analytics/{short_code}/stream

Response (text/event-stream):
data: {"short_code":"abc123","timestamp":"2024-01-15T10:30:45Z","total_clicks":1524}

data: {"short_code":"abc123","timestamp":"2024-01-15T10:30:47Z","total_clicks":1525}

Implementation:
- SSE endpoint with proper headers
- Send heartbeat every 30s
- Broadcast new clicks from worker pool
- Use channels for pub/sub pattern
```

---

## Backend Implementation Guide

### Project Structure

```
backend/
├── main.go                 # Entry point, server setup
├── config/
│   └── config.go          # Configuration management
├── handlers/
│   ├── links.go           # Link CRUD handlers
│   ├── redirect.go        # Redirect handler (hot path)
│   └── analytics.go       # Analytics & SSE handlers
├── services/
│   ├── link_service.go    # Business logic for links
│   ├── analytics_service.go # Analytics processing
│   └── cache_service.go   # Redis operations
├── models/
│   └── models.go          # Data structures
├── middleware/
│   ├── cors.go            # CORS middleware
│   ├── logger.go          # Request logging
│   └── ratelimit.go       # Rate limiting
├── workers/
│   └── analytics_worker.go # Async analytics processor
├── db/
│   ├── postgres.go        # PostgreSQL connection
│   └── redis.go           # Redis connection
└── utils/
    ├── shortcode.go       # Short code generation
    └── hash.go            # Visitor hashing
```

### Key Implementation Details

#### 1. Short Code Generation
```go
// utils/shortcode.go
// Generate 6-character alphanumeric code
// Use crypto/rand for uniqueness
// Retry on collision (rare with 62^6 = 56B combinations)
func GenerateShortCode() string {
    const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
    const length = 6
    // Implementation with crypto/rand
}
```

#### 2. Redirect Handler (Critical Path)
```go
// handlers/redirect.go
func HandleRedirect(w http.ResponseWriter, r *http.Request) {
    shortCode := extractShortCode(r.URL.Path)
    
    // 1. Try Redis cache (< 1ms)
    originalURL, err := cache.Get(ctx, "link:" + shortCode)
    if err != nil {
        // 2. Cache miss - query PostgreSQL
        originalURL, err = db.GetLinkByCode(ctx, shortCode)
        if err != nil {
            http.NotFound(w, r)
            return
        }
        // 3. Cache for future requests
        cache.Set(ctx, "link:" + shortCode, originalURL, 1*time.Hour)
    }
    
    // 4. Fire async analytics event (non-blocking)
    analyticsQueue <- ClickEvent{
        ShortCode:   shortCode,
        Timestamp:   time.Now(),
        IPAddress:   extractIP(r),
        UserAgent:   r.UserAgent(),
        Referer:     r.Referer(),
        VisitorHash: hashVisitor(extractIP(r), r.UserAgent()),
    }
    
    // 5. Increment Redis counter for real-time updates
    cache.Incr(ctx, "clicks:realtime:" + shortCode)
    
    // 6. Redirect immediately
    http.Redirect(w, r, originalURL, http.StatusFound)
}
```

#### 3. Analytics Worker Pool
```go
// workers/analytics_worker.go
const (
    NumWorkers = 10
    QueueSize  = 10000
)

var analyticsQueue = make(chan ClickEvent, QueueSize)

func StartWorkers(db *sql.DB) {
    for i := 0; i < NumWorkers; i++ {
        go worker(db, i)
    }
}

func worker(db *sql.DB, id int) {
    batch := make([]ClickEvent, 0, 100)
    ticker := time.NewTicker(5 * time.Second)
    
    for {
        select {
        case event := <-analyticsQueue:
            batch = append(batch, event)
            if len(batch) >= 100 {
                flushBatch(db, batch)
                batch = batch[:0]
            }
        case <-ticker.C:
            if len(batch) > 0 {
                flushBatch(db, batch)
                batch = batch[:0]
            }
        }
    }
}

func flushBatch(db *sql.DB, events []ClickEvent) {
    // Batch INSERT into clicks table
    // Update link_stats (total_clicks, unique_visitors)
    // Update top_referrers
    // Broadcast to SSE clients
}
```

#### 4. SSE Implementation
```go
// handlers/analytics.go
type SSEBroker struct {
    clients map[string]map[chan []byte]bool
    mu      sync.RWMutex
}

func (b *SSEBroker) HandleSSE(w http.ResponseWriter, r *http.Request) {
    shortCode := extractShortCode(r.URL.Path)
    
    // Set SSE headers
    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")
    
    // Create client channel
    clientChan := make(chan []byte, 10)
    b.addClient(shortCode, clientChan)
    defer b.removeClient(shortCode, clientChan)
    
    // Send data
    for {
        select {
        case msg := <-clientChan:
            fmt.Fprintf(w, "data: %s\n\n", msg)
            w.(http.Flusher).Flush()
        case <-r.Context().Done():
            return
        }
    }
}
```

#### 5. Middleware Chain
```go
// main.go
func main() {
    mux := http.NewServeMux()
    
    // Redirect endpoint (no rate limit for performance)
    mux.HandleFunc("GET /{shortCode}", handlers.HandleRedirect)
    
    // API endpoints with middleware
    api := http.NewServeMux()
    api.HandleFunc("POST /api/links", handlers.CreateLink)
    api.HandleFunc("GET /api/links/{shortCode}", handlers.GetLink)
    api.HandleFunc("GET /api/links", handlers.ListLinks)
    api.HandleFunc("GET /api/analytics/{shortCode}", handlers.GetAnalytics)
    api.HandleFunc("GET /api/analytics/{shortCode}/stream", handlers.StreamAnalytics)
    
    // Wrap with middleware
    handler := middleware.Chain(
        api,
        middleware.CORS,
        middleware.Logger,
        middleware.RateLimit(100, time.Minute), // 100 req/min per IP
    )
    
    mux.Handle("/api/", handler)
    
    // Start server
    log.Fatal(http.ListenAndServe(":8080", mux))
}
```

---

## Frontend Implementation Guide

### Project Structure

```
frontend/
├── app/
│   ├── layout.tsx              # Root layout
│   ├── page.tsx                # Home/Create link page
│   ├── links/
│   │   └── page.tsx            # My links page
│   └── analytics/
│       └── [shortCode]/
│           └── page.tsx        # Analytics detail page
├── components/
│   ├── CreateLinkForm.tsx      # Link creation form
│   ├── LinksTable.tsx          # Links list table
│   ├── AnalyticsDashboard.tsx  # Analytics overview
│   ├── ClicksChart.tsx         # Time-series chart
│   ├── ReferrersTable.tsx      # Top referrers
│   └── RealtimeCounter.tsx     # SSE-powered counter
├── lib/
│   ├── api.ts                  # API client functions
│   └── utils.ts                # Helper functions
└── types/
    └── index.ts                # TypeScript interfaces
```

### Key Components

#### 1. Create Link Form
```typescript
// components/CreateLinkForm.tsx
'use client'

import { useState } from 'react'
import { createLink } from '@/lib/api'

export function CreateLinkForm() {
  const [url, setUrl] = useState('')
  const [shortUrl, setShortUrl] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError(null)
    setLoading(true)

    try {
      // Validate URL
      new URL(url)
      
      const result = await createLink(url, 'demo-user')
      setShortUrl(result.short_url)
    } catch (err) {
      setError('Invalid URL or server error')
    } finally {
      setLoading(false)
    }
  }

  const copyToClipboard = () => {
    if (shortUrl) {
      navigator.clipboard.writeText(shortUrl)
      // Show success toast
    }
  }

  return (
    <div className="max-w-2xl mx-auto p-6">
      <form onSubmit={handleSubmit} className="space-y-4">
        <div>
          <label htmlFor="url" className="block text-sm font-medium mb-2">
            Enter URL to shorten
          </label>
          <input
            type="text"
            id="url"
            value={url}
            onChange={(e) => setUrl(e.target.value)}
            placeholder="https://example.com/very/long/url"
            className="w-full px-4 py-2 border rounded-lg"
            required
          />
        </div>
        
        <button
          type="submit"
          disabled={loading}
          className="w-full bg-blue-600 text-white py-2 rounded-lg hover:bg-blue-700 disabled:opacity-50"
        >
          {loading ? 'Creating...' : 'Shorten URL'}
        </button>
      </form>

      {error && (
        <div className="mt-4 p-4 bg-red-50 text-red-700 rounded-lg">
          {error}
        </div>
      )}

      {shortUrl && (
        <div className="mt-6 p-4 bg-green-50 rounded-lg">
          <p className="text-sm text-gray-600 mb-2">Your short URL:</p>
          <div className="flex items-center gap-2">
            <input
              type="text"
              value={shortUrl}
              readOnly
              className="flex-1 px-4 py-2 bg-white border rounded-lg"
            />
            <button
              onClick={copyToClipboard}
              className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700"
            >
              Copy
            </button>
          </div>
        </div>
      )}
    </div>
  )
}
```

#### 2. Real-time Counter (SSE)
```typescript
// components/RealtimeCounter.tsx
'use client'

import { useEffect, useState } from 'react'

interface Props {
  shortCode: string
  initialCount: number
}

export function RealtimeCounter({ shortCode, initialCount }: Props) {
  const [count, setCount] = useState(initialCount)
  const [isConnected, setIsConnected] = useState(false)

  useEffect(() => {
    const eventSource = new EventSource(
      `http://localhost:8080/api/analytics/${shortCode}/stream`
    )

    eventSource.onopen = () => {
      setIsConnected(true)
    }

    eventSource.onmessage = (event) => {
      const data = JSON.parse(event.data)
      setCount(data.total_clicks)
    }

    eventSource.onerror = () => {
      setIsConnected(false)
      eventSource.close()
    }

    return () => {
      eventSource.close()
    }
  }, [shortCode])

  return (
    <div className="bg-white rounded-lg shadow p-6">
      <div className="flex items-center justify-between mb-2">
        <h3 className="text-lg font-semibold">Total Clicks</h3>
        <div className={`w-2 h-2 rounded-full ${isConnected ? 'bg-green-500' : 'bg-gray-300'}`} />
      </div>
      <p className="text-4xl font-bold">{count.toLocaleString()}</p>
      <p className="text-sm text-gray-500 mt-1">
        {isConnected ? 'Live updates' : 'Reconnecting...'}
      </p>
    </div>
  )
}
```

#### 3. Clicks Chart
```typescript
// components/ClicksChart.tsx
'use client'

import { Line } from 'react-chartjs-2'
import {
  Chart as ChartJS,
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  Title,
  Tooltip,
  Legend,
  TimeScale
} from 'chart.js'
import 'chartjs-adapter-date-fns'

ChartJS.register(
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  Title,
  Tooltip,
  Legend,
  TimeScale
)

interface Props {
  data: Array<{ timestamp: string; count: number }>
}

export function ClicksChart({ data }: Props) {
  const chartData = {
    labels: data.map(d => new Date(d.timestamp)),
    datasets: [
      {
        label: 'Clicks',
        data: data.map(d => d.count),
        borderColor: 'rgb(59, 130, 246)',
        backgroundColor: 'rgba(59, 130, 246, 0.1)',
        tension: 0.4,
      },
    ],
  }

  const options = {
    responsive: true,
    maintainAspectRatio: false,
    scales: {
      x: {
        type: 'time' as const,
        time: {
          unit: 'hour' as const,
        },
      },
      y: {
        beginAtZero: true,
      },
    },
    plugins: {
      legend: {
        display: false,
      },
    },
  }

  return (
    <div className="bg-white rounded-lg shadow p-6">
      <h3 className="text-lg font-semibold mb-4">Clicks Over Time</h3>
      <div style={{ height: '300px' }}>
        <Line data={chartData} options={options} />
      </div>
    </div>
  )
}
```

#### 4. API Client
```typescript
// lib/api.ts
const API_BASE = 'http://localhost:8080/api'

export async function createLink(url: string, userId: string) {
  const response = await fetch(`${API_BASE}/links`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ url, user_id: userId }),
  })
  
  if (!response.ok) {
    throw new Error('Failed to create link')
  }
  
  return response.json()
}

export async function getLinks(userId: string) {
  const response = await fetch(`${API_BASE}/links?user_id=${userId}`)
  
  if (!response.ok) {
    throw new Error('Failed to fetch links')
  }
  
  return response.json()
}

export async function getAnalytics(shortCode: string, period: '24h' | '7d' | '30d' = '24h') {
  const response = await fetch(`${API_BASE}/analytics/${shortCode}?period=${period}`)
  
  if (!response.ok) {
    throw new Error('Failed to fetch analytics')
  }
  
  return response.json()
}
```

---

## Load Testing Strategy

### Tool: k6

### Test Script Structure
```javascript
// load-test/script.js
import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  stages: [
    { duration: '30s', target: 100 },   // Ramp to 100 RPS
    { duration: '30s', target: 500 },   // Ramp to 500 RPS
    { duration: '30s', target: 1000 },  // Ramp to 1000 RPS
    { duration: '30s', target: 1000 },  // Hold at 1000 RPS
  ],
  thresholds: {
    http_req_duration: ['p(95)<100', 'p(99)<200'], // 95% < 100ms, 99% < 200ms
    http_req_failed: ['rate<0.01'], // Error rate < 1%
  },
};

// Prepare test data: create 100 links before test
export function setup() {
  const links = [];
  for (let i = 0; i < 100; i++) {
    const res = http.post('http://localhost:8080/api/links', JSON.stringify({
      url: `https://example.com/page${i}`,
      user_id: 'loadtest',
    }), {
      headers: { 'Content-Type': 'application/json' },
    });
    links.push(JSON.parse(res.body).short_code);
  }
  return { links };
}

export default function(data) {
  // Randomly select a short code
  const shortCode = data.links[Math.floor(Math.random() * data.links.length)];
  
  // Test redirect endpoint
  const res = http.get(`http://localhost:8080/${shortCode}`, {
    redirects: 0, // Don't follow redirects
  });
  
  check(res, {
    'status is 302': (r) => r.status === 302,
    'redirect latency < 50ms': (r) => r.timings.duration < 50,
  });
  
  sleep(0.1); // Small delay between requests
}
```

### Expected Metrics to Capture

1. **Latency Distribution:**
   - p50 (median): < 10ms
   - p95: < 50ms
   - p99: < 100ms
   - p99.9: < 200ms

2. **Throughput:**
   - Successful requests per second at each load level
   - Target: 1000+ RPS with <1% errors

3. **Resource Usage:**
   - CPU usage
   - Memory usage
   - Redis connections
   - PostgreSQL connections

4. **Degradation Points:**
   - At what RPS does latency spike?
   - At what RPS do errors start occurring?
   - What's the bottleneck? (CPU, I/O, database, Redis)

---

## Docker Compose Setup

```yaml
# docker-compose.yml
version: '3.8'

services:
  postgres:
    image: postgres:16-alpine
    environment:
      POSTGRES_DB: linkanalytics
      POSTGRES_USER: linkuser
      POSTGRES_PASSWORD: linkpass
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - ./backend/db/init.sql:/docker-entrypoint-initdb.d/init.sql
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U linkuser"]
      interval: 5s
      timeout: 5s
      retries: 5

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
    volumes:
      - redis_data:/data
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 5s
      timeout: 3s
      retries: 5

  backend:
    build:
      context: ./backend
      dockerfile: Dockerfile
    ports:
      - "8080:8080"
    environment:
      DATABASE_URL: postgres://linkuser:linkpass@postgres:5432/linkanalytics?sslmode=disable
      REDIS_URL: redis:6379
      PORT: 8080
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy
    restart: unless-stopped

  frontend:
    build:
      context: ./frontend
      dockerfile: Dockerfile
    ports:
      - "3000:3000"
    environment:
      NEXT_PUBLIC_API_URL: http://localhost:8080
    depends_on:
      - backend
    restart: unless-stopped

volumes:
  postgres_data:
  redis_data:
```

### Backend Dockerfile
```dockerfile
# backend/Dockerfile
FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o main .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/main .
EXPOSE 8080
CMD ["./main"]
```

### Frontend Dockerfile
```dockerfile
# frontend/Dockerfile
FROM node:20-alpine AS builder

WORKDIR /app
COPY package*.json ./
RUN npm ci

COPY . .
RUN npm run build

FROM node:20-alpine
WORKDIR /app
COPY --from=builder /app/.next ./.next
COPY --from=builder /app/node_modules ./node_modules
COPY --from=builder /app/package.json ./package.json
COPY --from=builder /app/public ./public

EXPOSE 3000
CMD ["npm", "start"]
```

---

## Performance Optimization Checklist

### Backend Optimizations

1. **Redirect Handler:**
   - ✅ Redis cache for hot links (1-hour TTL)
   - ✅ Async analytics processing (fire-and-forget)
   - ✅ Connection pooling (PostgreSQL: 25 max, Redis: 10)
   - ✅ No unnecessary logging on hot path
   - ✅ Minimal allocations

2. **Database:**
   - ✅ Proper indexes on queries
   - ✅ Batch inserts for analytics (100 events/batch)
   - ✅ Read replicas consideration (if needed)
   - ✅ Connection pooling

3. **Redis:**
   - ✅ Pipeline commands when possible
   - ✅ Appropriate TTLs
   - ✅ Memory limits and eviction policy (allkeys-lru)

4. **Worker Pool:**
   - ✅ Buffered channels (10K capacity)
   - ✅ Batch processing
   - ✅ Graceful shutdown

### Frontend Optimizations

1. **Next.js:**
   - ✅ Server components where possible
   - ✅ Proper loading states
   - ✅ Error boundaries
   - ✅ SWR for data fetching with revalidation

2. **Real-time:**
   - ✅ SSE reconnection logic
   - ✅ Heartbeat for connection health
   - ✅ Graceful degradation if offline

---

## Implementation Order (AI Generation)

### Phase 1: Backend Core (1 hour)
1. Project structure and config
2. Database models and migrations
3. Redis connection setup
4. Short code generation utility
5. Basic link CRUD handlers
6. Redirect handler (without analytics)

### Phase 2: Analytics System (1 hour)
1. Analytics worker pool
2. Click event processing
3. Aggregation queries
4. Analytics API endpoints
5. SSE implementation

### Phase 3: Frontend (1.5 hours)
1. Next.js project setup
2. Create link form
3. Links list page
4. Analytics dashboard
5. Real-time counter component
6. Charts integration

### Phase 4: Testing & Docker (0.5 hours)
1. Docker Compose setup
2. Load test script
3. Basic integration tests
4. README documentation

---

## Testing Checklist

### Functional Tests
- [ ] Create link with valid URL
- [ ] Create link with invalid URL (error)
- [ ] Redirect works for existing link
- [ ] Redirect returns 404 for non-existent link
- [ ] List links returns all user links
- [ ] Analytics shows correct metrics
- [ ] SSE stream sends updates
- [ ] SSE reconnects on disconnect

### Performance Tests
- [ ] Redirect latency < 50ms @ 100 RPS
- [ ] Redirect latency < 100ms @ 1000 RPS
- [ ] No memory leaks during sustained load
- [ ] Redis cache hit rate > 90%
- [ ] Worker queue doesn't overflow

### Edge Cases
- [ ] Duplicate short codes handled
- [ ] Very long URLs supported (up to 2048 chars)
- [ ] Special characters in URLs
- [ ] High concurrent link creation
- [ ] Multiple SSE clients for same link

---

## README Template

```markdown
# Link Analytics Service

URL shortening service with real-time analytics, built with Go and Next.js.

## Quick Start

1. Start all services:
   ```bash
   docker-compose up -d
   ```

2. Access the application:
   - Frontend: http://localhost:3000
   - Backend API: http://localhost:8080

3. Run load tests:
   ```bash
   cd load-test
   k6 run script.js
   ```

## Architecture

- Backend: Go (net/http only)
- Frontend: Next.js 14 (App Router)
- Database: PostgreSQL 16
- Cache: Redis 7
- Message Queue: Go channels

## Performance

- Target: 1000+ redirects/second
- Redirect latency: <50ms (p95)
- Cache hit rate: >90%

## Development

### Backend
```bash
cd backend
go run main.go
```

### Frontend
```bash
cd frontend
npm install
npm run dev
```

## Load Test Results

See [load-test/report.md](load-test/report.md)
```

---

## AI Generation Prompts

When using AI tools to implement this specification, use these targeted prompts:

### Backend Generation
```
Using only Go standard library (net/http), implement:
1. A link management service with PostgreSQL
2. A high-performance redirect handler with Redis caching
3. An async analytics worker pool using Go channels
4. SSE endpoint for real-time updates

Follow the exact database schema and API design in SPEC.md.
Focus on performance: cache hot data, async processing, connection pooling.
```

### Frontend Generation
```
Using Next.js 14 App Router, implement:
1. Link creation form with URL validation
2. Links list with navigation to analytics
3. Analytics dashboard with real-time counter (SSE)
4. Time-series chart using Chart.js
5. Top referrers table

Use TypeScript and Tailwind CSS. Focus on clean component design.
```

### Load Test Generation
```
Create a k6 load test script that:
1. Sets up 100 test links
2. Ramps from 100 to 1000 RPS over 2 minutes
3. Tests the redirect endpoint
4. Measures p50/p95/p99 latency and error rate
5. Generates a report with graphs
```

---

## Success Criteria

✅ **Functionality:**
- All CRUD operations work
- Redirects work correctly
- Analytics show accurate data
- Real-time updates via SSE

✅ **Performance:**
- Redirect latency < 100ms @ 1000 RPS
- Error rate < 1%
- System stable under load

✅ **Code Quality:**
- Clean architecture
- Proper error handling
- Well-documented
- Docker Compose works on first try

✅ **AI Documentation:**
- Clear specification showing design decisions
- Implementation order defined
- Performance trade-offs explained
