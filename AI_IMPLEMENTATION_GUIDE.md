# AI Implementation Guide

This document provides step-by-step instructions for AI-powered implementation of the Link Analytics Service.

---

## Implementation Strategy

Use this specification with AI coding tools (Claude, Cursor, Copilot) in this order:

1. Generate backend structure and database layer
2. Implement redirect hot path with caching
3. Build analytics worker system
4. Create frontend components
5. Setup Docker environment
6. Create load testing suite

---

## Phase 1: Backend Foundation (30 min)

### Task 1.1: Project Setup and Configuration

**Prompt:**
```
Create a Go project structure with:
- main.go as entry point
- config package for environment variables (DATABASE_URL, REDIS_URL, PORT)
- models package with these structs:
  * Link (ID, ShortCode, OriginalURL, UserID, CreatedAt)
  * ClickEvent (ShortCode, Timestamp, IPAddress, UserAgent, Referer, VisitorHash)
  * LinkStats (ShortCode, TotalClicks, UniqueVisitors)
  
Include proper error types and validation.
Use only Go standard library.
```

**Expected Output:**
- `config/config.go` with environment loading
- `models/models.go` with all data structures
- `main.go` skeleton

---

### Task 1.2: Database Layer

**Prompt:**
```
Implement PostgreSQL connection and queries in db/postgres.go:

1. Connection pool setup (max 25 connections)
2. These methods:
   - CreateLink(link *Link) error
   - GetLinkByCode(shortCode string) (*Link, error)
   - GetLinksByUser(userID string) ([]*Link, error)
   - InsertClickEvent(event *ClickEvent) error
   - GetLinkStats(shortCode string) (*LinkStats, error)
   - GetClicksOverTime(shortCode string, period time.Duration) ([]TimePoint, error)
   - GetTopReferrers(shortCode string, limit int) ([]Referrer, error)

3. Database initialization SQL:
   CREATE TABLE links (...according to SPEC.md schema...)
   CREATE TABLE clicks (...according to SPEC.md schema...)
   CREATE TABLE link_stats (...)
   CREATE TABLE top_referrers (...)

Use prepared statements. Handle errors properly.
```

**Expected Output:**
- `db/postgres.go` with all database operations
- `db/init.sql` with schema creation

---

### Task 1.3: Redis Cache Layer

**Prompt:**
```
Implement Redis client in db/redis.go:

1. Connection setup
2. Methods:
   - Get(key string) (string, error)
   - Set(key, value string, ttl time.Duration) error
   - Incr(key string) error
   - Delete(key string) error

3. Connection pool (10 connections)
4. Proper error handling for cache misses vs errors

Use only redis/go-redis library.
```

**Expected Output:**
- `db/redis.go` with cache operations

---

### Task 1.4: Utilities

**Prompt:**
```
Implement utilities:

1. utils/shortcode.go:
   - GenerateShortCode() string
     * 6 characters, alphanumeric
     * Use crypto/rand for security
     * Character set: a-zA-Z0-9

2. utils/hash.go:
   - HashVisitor(ip, userAgent string) string
     * SHA256 hash of IP + UserAgent
     * Return hex string

3. utils/validation.go:
   - IsValidURL(url string) bool
   - ExtractIP(r *http.Request) string (handle X-Forwarded-For)
   - ExtractShortCode(path string) string

Use only Go standard library.
```

**Expected Output:**
- `utils/shortcode.go`
- `utils/hash.go`
- `utils/validation.go`

---

## Phase 2: Core Handlers (30 min)

### Task 2.1: Redirect Handler (Critical Path)

**Prompt:**
```
Implement handlers/redirect.go with HandleRedirect function:

1. Extract short code from URL path
2. Try Redis cache first (key: "link:{short_code}")
3. On cache miss, query PostgreSQL and cache result (TTL: 1 hour)
4. Create ClickEvent and send to analytics channel (non-blocking)
5. Increment Redis counter "clicks:realtime:{short_code}"
6. Return 302 redirect immediately
7. Return 404 if link not found

Performance requirements:
- No blocking operations
- Minimal allocations
- Target: <10ms latency

Use global channel: var analyticsQueue = make(chan ClickEvent, 10000)
```

**Expected Output:**
- `handlers/redirect.go` with optimized redirect handler

---

### Task 2.2: Link Management Handlers

**Prompt:**
```
Implement handlers/links.go with these handlers:

1. CreateLink(w http.ResponseWriter, r *http.Request)
   POST /api/links
   Body: {"url": "...", "user_id": "..."}
   - Validate URL
   - Generate short code
   - Check uniqueness (retry if collision)
   - Insert to database
   - Return: {"short_code": "...", "short_url": "...", "original_url": "...", "created_at": "..."}

2. GetLink(w http.ResponseWriter, r *http.Request)
   GET /api/links/{short_code}
   - Get link info
   - Get stats
   - Return combined response

3. ListLinks(w http.ResponseWriter, r *http.Request)
   GET /api/links?user_id=...
   - Query user's links
   - Include basic stats for each
   - Return array

Use proper HTTP status codes. Handle errors gracefully.
```

**Expected Output:**
- `handlers/links.go` with all CRUD operations

---

### Task 2.3: Analytics Handlers

**Prompt:**
```
Implement handlers/analytics.go:

1. GetAnalytics(w http.ResponseWriter, r *http.Request)
   GET /api/analytics/{short_code}?period=24h|7d|30d
   - Get overall stats
   - Get clicks over time (hourly buckets for 24h, daily for 7d/30d)
   - Get top 10 referrers
   - Return JSON response according to API spec

2. StreamAnalytics(w http.ResponseWriter, r *http.Request)
   GET /api/analytics/{short_code}/stream
   - Setup SSE headers
   - Create client channel
   - Send heartbeat every 30s
   - Send updates when clicks occur
   - Cleanup on disconnect

Implement SSEBroker type for managing SSE connections:
- clients map[shortCode][]chan
- AddClient, RemoveClient, Broadcast methods
```

**Expected Output:**
- `handlers/analytics.go` with analytics endpoints
- SSE implementation

---

## Phase 3: Analytics Worker System (30 min)

### Task 3.1: Worker Pool

**Prompt:**
```
Implement workers/analytics_worker.go:

1. StartWorkers(db *sql.DB, redis *redis.Client, broker *SSEBroker, numWorkers int)
   - Start N worker goroutines
   - Each worker processes events from analyticsQueue channel

2. worker(id int, ...)
   - Batch events (max 100 events or 5 seconds, whichever comes first)
   - Flush batch to database:
     * Bulk INSERT into clicks table
     * UPDATE link_stats (increment clicks, update unique visitors)
     * INSERT/UPDATE top_referrers
   - Broadcast click count to SSE clients
   - Handle errors gracefully

3. flushBatch(events []ClickEvent)
   - Use PostgreSQL COPY or batch INSERT
   - Calculate unique visitors using visitor hash
   - Update aggregated stats

Use context for graceful shutdown.
```

**Expected Output:**
- `workers/analytics_worker.go` with worker pool implementation

---

## Phase 4: Middleware and Server Setup (15 min)

### Task 4.1: Middleware

**Prompt:**
```
Implement middleware in middleware/:

1. cors.go - CORS middleware
   - Allow origins: http://localhost:3000
   - Allow methods: GET, POST, OPTIONS
   - Allow headers: Content-Type
   - Handle preflight requests

2. logger.go - Request logging
   - Log: method, path, status, duration
   - Use structured logging (JSON format)
   - Skip logging for redirect endpoint (performance)

3. ratelimit.go - Rate limiting
   - 100 requests per minute per IP
   - Use Redis for distributed rate limiting
   - Return 429 Too Many Requests
   - Skip for redirect endpoint

4. Chain function to compose middleware
```

**Expected Output:**
- `middleware/cors.go`
- `middleware/logger.go`
- `middleware/ratelimit.go`

---

### Task 4.2: Main Server

**Prompt:**
```
Update main.go:

1. Load configuration from environment
2. Connect to PostgreSQL and Redis
3. Initialize SSE broker
4. Start analytics workers (10 workers)
5. Setup routes:
   - GET /{shortCode} -> redirect handler (no middleware)
   - POST /api/links -> create link (with middleware)
   - GET /api/links/{shortCode} -> get link (with middleware)
   - GET /api/links -> list links (with middleware)
   - GET /api/analytics/{shortCode} -> get analytics (with middleware)
   - GET /api/analytics/{shortCode}/stream -> SSE stream (no rate limit)

6. Graceful shutdown:
   - Handle SIGINT/SIGTERM
   - Close database connections
   - Wait for worker queues to drain

Use http.ServeMux. No frameworks.
```

**Expected Output:**
- Complete `main.go` with server setup

---

## Phase 5: Frontend Implementation (45 min)

### Task 5.1: Next.js Setup and API Client

**Prompt:**
```
Create Next.js 14 project with App Router:

1. Setup TypeScript configuration
2. Install dependencies: react-chartjs-2, chart.js, chartjs-adapter-date-fns
3. Create types in types/index.ts:
   - Link interface
   - Analytics interface
   - ClickData interface
   - Referrer interface

4. Create API client in lib/api.ts:
   - createLink(url: string, userId: string)
   - getLinks(userId: string)
   - getLink(shortCode: string)
   - getAnalytics(shortCode: string, period: string)

Use fetch API. Handle errors properly.
```

**Expected Output:**
- Next.js project structure
- `types/index.ts`
- `lib/api.ts`

---

### Task 5.2: Create Link Page

**Prompt:**
```
Implement app/page.tsx (home page):

1. CreateLinkForm component:
   - Input field for URL
   - Validate URL on client side
   - Submit button with loading state
   - Display created short URL
   - Copy to clipboard button
   - Show success/error messages

2. Styling with Tailwind CSS
3. Use "demo-user" as hardcoded user ID
4. Form validation with proper error messages

Make it look professional and clean.
```

**Expected Output:**
- `app/page.tsx` with link creation form

---

### Task 5.3: Links List Page

**Prompt:**
```
Implement app/links/page.tsx:

1. Fetch all links for user on mount
2. Display in a table:
   - Short code (with copy button)
   - Original URL (truncated if long, show full on hover)
   - Total clicks
   - Created date (formatted)
   - Actions: View Analytics button

3. Loading skeleton while fetching
4. Empty state if no links
5. Responsive design

Use Tailwind CSS for styling.
```

**Expected Output:**
- `app/links/page.tsx` with links table

---

### Task 5.4: Analytics Dashboard

**Prompt:**
```
Implement app/analytics/[shortCode]/page.tsx:

1. Fetch analytics data on mount
2. Display:
   - Link info (short code, original URL)
   - Real-time counter (using RealtimeCounter component)
   - Overall stats cards (total clicks, unique visitors)
   - Clicks chart (last 24h by default, with period selector)
   - Top 10 referrers table

3. Period selector: 24h, 7d, 30d
4. Responsive layout
5. Loading states

Create separate components for reusability:
- components/RealtimeCounter.tsx (SSE)
- components/ClicksChart.tsx (Chart.js)
- components/ReferrersTable.tsx
- components/StatsCard.tsx
```

**Expected Output:**
- `app/analytics/[shortCode]/page.tsx`
- All component files

---

### Task 5.5: Real-time Counter Component

**Prompt:**
```
Implement components/RealtimeCounter.tsx:

1. Accept shortCode and initialCount props
2. Setup SSE connection to /api/analytics/{shortCode}/stream
3. Update count on message receive
4. Show connection status indicator (green = connected, gray = disconnected)
5. Auto-reconnect on disconnect (exponential backoff)
6. Cleanup on unmount

Use useEffect hook. Handle errors gracefully.
```

**Expected Output:**
- `components/RealtimeCounter.tsx` with SSE implementation

---

### Task 5.6: Clicks Chart Component

**Prompt:**
```
Implement components/ClicksChart.tsx using Chart.js:

1. Accept data prop: Array<{timestamp: string, count: number}>
2. Create line chart showing clicks over time
3. X-axis: time (formatted nicely)
4. Y-axis: click count
5. Responsive design
6. Smooth line (tension: 0.4)
7. Blue color scheme

Use react-chartjs-2. Register required Chart.js components.
```

**Expected Output:**
- `components/ClicksChart.tsx` with chart implementation

---

## Phase 6: DevOps and Testing (30 min)

### Task 6.1: Docker Setup

**Prompt:**
```
Create Docker configuration:

1. docker-compose.yml:
   - PostgreSQL 16 with init.sql volume
   - Redis 7
   - Backend (Go) with health checks
   - Frontend (Next.js)
   - Proper depends_on with health checks
   - Volume persistence for data

2. backend/Dockerfile:
   - Multi-stage build
   - Alpine base image
   - Copy only necessary files
   - Optimize for size

3. frontend/Dockerfile:
   - Multi-stage build
   - Node 20 Alpine
   - Production build
   - Only copy build artifacts

Ensure single-command startup: docker-compose up -d
```

**Expected Output:**
- `docker-compose.yml`
- `backend/Dockerfile`
- `frontend/Dockerfile`

---

### Task 6.2: Load Testing

**Prompt:**
```
Create load-test/script.js using k6:

1. Setup phase:
   - Create 100 test links
   - Store short codes in array

2. Test stages:
   - 30s: ramp to 100 RPS
   - 30s: ramp to 500 RPS
   - 30s: ramp to 1000 RPS
   - 30s: hold at 1000 RPS

3. Test scenario:
   - Random short code from array
   - GET /{shortCode}
   - Don't follow redirects
   - Assert status 302

4. Thresholds:
   - p95 < 100ms
   - p99 < 200ms
   - Error rate < 1%

5. Generate HTML report with --out json=results.json

Include instructions to run: k6 run --out json=results.json script.js
```

**Expected Output:**
- `load-test/script.js`
- Instructions in `load-test/README.md`

---

### Task 6.3: Load Test Report Template

**Prompt:**
```
Create load-test/report.md template:

Structure:
1. Test Configuration
   - Duration
   - Load profile
   - Test environment specs

2. Results Summary
   - Max RPS achieved
   - Latency percentiles at each load level
   - Error rate
   - Throughput graph

3. Performance Analysis
   - Where did system start degrading?
   - What was the bottleneck?
   - CPU/Memory usage
   - Cache hit rate

4. Conclusions
   - System capacity
   - Recommendations

Include placeholders for graphs and metrics.
```

**Expected Output:**
- `load-test/report.md` template

---

### Task 6.4: Documentation

**Prompt:**
```
Create comprehensive README.md:

1. Project Overview
   - What it does
   - Tech stack
   - Architecture diagram (ASCII)

2. Quick Start
   - Prerequisites
   - docker-compose up command
   - Access URLs

3. Development Setup
   - Backend: go run main.go
   - Frontend: npm run dev
   - Database migrations

4. API Documentation
   - All endpoints with examples
   - Request/response formats

5. Load Testing
   - How to run tests
   - Expected results
   - Link to report

6. Performance Features
   - Redis caching strategy
   - Async analytics processing
   - Worker pool design

7. Project Structure
   - Directory tree with explanations

Make it professional and comprehensive.
```

**Expected Output:**
- `README.md` with complete documentation

---

## AI Code Generation Tips

### When Using Claude/Cursor/Copilot:

1. **Generate in Order:**
   - Start with Phase 1, complete it fully before Phase 2
   - Each phase builds on previous ones

2. **Reference SPEC.md:**
   - Include "According to SPEC.md..." in prompts
   - Copy relevant sections into prompt context

3. **Iterative Refinement:**
   - Generate base implementation first
   - Then optimize for performance
   - Then add error handling

4. **Testing as You Go:**
   - Test each component before moving on
   - Use curl for API testing
   - Use browser for frontend testing

5. **Don't Over-Engineer:**
   - Stick to requirements
   - No premature optimization
   - Keep it simple and working

---

## Verification Checklist

After each phase, verify:

### Phase 1:
- [ ] Go project compiles without errors
- [ ] Database connection works
- [ ] Redis connection works
- [ ] Short code generation produces unique codes

### Phase 2:
- [ ] Create link endpoint works
- [ ] Redirect endpoint returns 302
- [ ] List links returns data
- [ ] Analytics endpoint returns stats

### Phase 3:
- [ ] Workers start without crashing
- [ ] Click events are processed
- [ ] Stats are updated in database
- [ ] SSE sends updates

### Phase 4:
- [ ] CORS allows frontend requests
- [ ] Rate limiting works
- [ ] Server starts successfully
- [ ] Graceful shutdown works

### Phase 5:
- [ ] Frontend builds without errors
- [ ] All pages render correctly
- [ ] API calls succeed
- [ ] SSE connection works
- [ ] Charts display data

### Phase 6:
- [ ] Docker Compose starts all services
- [ ] Services can communicate
- [ ] Data persists across restarts
- [ ] Load test runs successfully

---

## Time Management

Target times per phase:
- Phase 1: 30 minutes
- Phase 2: 30 minutes
- Phase 3: 30 minutes
- Phase 4: 15 minutes
- Phase 5: 45 minutes
- Phase 6: 30 minutes
- **Buffer:** 30 minutes

**Total: 3.5 hours**

This leaves 30 minutes buffer for:
- Debugging
- Polish
- Documentation review
- Final testing

---

## Common Pitfalls to Avoid

1. **Don't use Go frameworks** - Assignment requires stdlib only
2. **Don't block redirect handler** - Analytics must be async
3. **Don't forget indexes** - Performance depends on proper indexing
4. **Don't ignore cache TTL** - Stale data can cause issues
5. **Don't hardcode URLs** - Use environment variables
6. **Don't skip error handling** - Production-ready code handles errors
7. **Don't forget health checks** - Docker Compose needs them
8. **Don't skip SSE heartbeat** - Prevents connection timeouts

---

## Final Quality Checks

Before submission:

1. **Functionality:**
   - Run through all user flows manually
   - Create link → List links → View analytics → Real-time updates

2. **Performance:**
   - Run load test
   - Verify latency < 100ms at 1000 RPS
   - Check error rate < 1%

3. **Code Quality:**
   - No obvious bugs
   - Proper error handling
   - Clean code structure
   - Comments where needed

4. **Documentation:**
   - README is complete
   - SPEC.md explains decisions
   - Load test report has data
   - docker-compose.yml works

5. **Presentation:**
   - Code is formatted
   - Consistent naming
   - No debug logs in production
   - Clean Git history (if using Git)

---

## Success Metrics

Your implementation should achieve:

✅ **Functional Requirements:**
- All API endpoints working
- Frontend pages rendering correctly
- Real-time updates via SSE
- Data persists across restarts

✅ **Performance Requirements:**
- 1000+ redirects/second
- p95 latency < 100ms
- Error rate < 1%
- Cache hit rate > 80%

✅ **Code Quality:**
- No frameworks in Go (stdlib only)
- Clean separation of concerns
- Proper error handling
- Production-ready code

✅ **Documentation:**
- Clear SPEC.md showing design decisions
- Comprehensive README
- Load test report with analysis
- Architecture diagrams

---

## Example AI Workflow

Here's how to use this guide with an AI tool:

1. **Start with Context:**
   ```
   I'm building a URL shortening service with real-time analytics.
   Read SPEC.md for full requirements.
   I'll implement this in phases following the AI Implementation Guide.
   ```

2. **Phase-by-Phase:**
   ```
   Let's start Phase 1, Task 1.1: Project Setup and Configuration
   [Copy task prompt from guide]
   ```

3. **Review and Iterate:**
   ```
   The config looks good, but add validation for required env vars.
   Also add a default PORT value of 8080 if not specified.
   ```

4. **Move to Next Task:**
   ```
   Great! Now let's do Phase 1, Task 1.2: Database Layer
   [Copy task prompt from guide]
   ```

5. **Continue Until Complete**

---

Good luck! This specification should help you leverage AI tools effectively while maintaining full understanding of the system design.
