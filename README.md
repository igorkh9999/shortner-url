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

```
┌─────────────┐
│   Next.js   │
│  Frontend   │
└──────┬──────┘
       │ HTTP/SSE
┌──────▼──────────────────────────┐
│      Go Backend (net/http)       │
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

## Tech Stack

- **Backend:** Go (stdlib only, net/http)
- **Frontend:** Next.js 14 (App Router)
- **Database:** PostgreSQL 16
- **Cache:** Redis 7
- **Message Queue:** Go channels

## Performance

- **Target:** 1000+ redirects/second
- **Redirect latency:** <50ms (p95)
- **Cache hit rate:** >90%

## Development

### Backend

```bash
cd backend
go mod download
go run main.go
```

Environment variables:
- `DATABASE_URL`: PostgreSQL connection string
- `REDIS_URL`: Redis connection string (default: localhost:6379)
- `PORT`: Server port (default: 8080)

### Frontend

```bash
cd frontend
npm install
npm run dev
```

Environment variables:
- `NEXT_PUBLIC_API_URL`: Backend API URL (default: http://localhost:8080)

## API Documentation

### Create Short Link
```
POST /api/links
Content-Type: application/json

{
  "url": "https://example.com/very/long/url",
  "user_id": "user123"
}

Response: {
  "short_code": "abc123",
  "short_url": "http://localhost:8080/abc123",
  "original_url": "https://example.com/very/long/url",
  "created_at": "2024-01-15T10:30:00Z"
}
```

### Get Link Info
```
GET /api/links/{short_code}

Response: {
  "short_code": "abc123",
  "original_url": "https://example.com/very/long/url",
  "created_at": "2024-01-15T10:30:00Z",
  "stats": {
    "total_clicks": 1523,
    "unique_visitors": 892
  }
}
```

### List User Links
```
GET /api/links?user_id=user123

Response: {
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

### Redirect
```
GET /{short_code}

Response: 302 Redirect
Location: https://example.com/very/long/url
```

### Get Analytics
```
GET /api/analytics/{short_code}?period=24h|7d|30d

Response: {
  "short_code": "abc123",
  "total_clicks": 1523,
  "unique_visitors": 892,
  "clicks_over_time": [
    { "timestamp": "2024-01-15T10:00:00Z", "count": 45 }
  ],
  "top_referrers": [
    { "referer": "https://twitter.com", "count": 450 }
  ]
}
```

### Real-time Click Stream (SSE)
```
GET /api/analytics/{short_code}/stream

Response: text/event-stream
data: {"short_code":"abc123","timestamp":"2024-01-15T10:30:45Z","total_clicks":1524}
```

## Load Testing

See [load-test/README.md](load-test/README.md) for detailed instructions.

Quick start:
```bash
cd load-test
k6 run script.js
```

## Performance Features

### Redis Caching Strategy
- Short code → URL mapping cached for 1 hour
- Real-time click counters with 60s TTL
- Cache refresh on hit

### Async Analytics Processing
- Fire-and-forget pattern for redirect handler
- Worker pool (10 workers) processes events in batches
- Batch size: 100 events or 5 seconds, whichever comes first

### Worker Pool Design
- Buffered channel (10K capacity)
- Prevents redirect slowdown
- Graceful shutdown with context cancellation

## Project Structure

```
.
├── backend/
│   ├── main.go              # Entry point
│   ├── config/              # Configuration
│   ├── handlers/            # HTTP handlers
│   ├── middleware/          # Middleware (CORS, logging, rate limiting)
│   ├── models/              # Data models
│   ├── db/                  # Database connections (PostgreSQL, Redis)
│   ├── workers/             # Analytics worker pool
│   └── utils/               # Utilities (shortcode, hash, validation)
├── frontend/
│   ├── app/                 # Next.js app router pages
│   ├── components/          # React components
│   ├── lib/                 # API client
│   └── types/               # TypeScript types
├── load-test/               # k6 load testing scripts
├── docker-compose.yml       # Docker Compose configuration
└── README.md               # This file
```

## Database Schema

See [backend/db/init.sql](backend/db/init.sql) for the complete schema.

Key tables:
- `links`: Shortened URLs
- `clicks`: Click events (time-series)
- `link_stats`: Aggregated statistics
- `top_referrers`: Top referrer statistics

## License

MIT

