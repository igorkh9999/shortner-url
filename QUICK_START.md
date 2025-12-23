# Quick Start Guide - Gatewise Home Assignment

## What You Have

Two comprehensive documents to guide AI-powered development:

### 1. SPEC.md - Technical Specification
**Purpose:** Complete technical blueprint of the system  
**Use for:** Understanding architecture, design decisions, and system behavior  
**Contains:**
- Architecture diagrams and rationale
- Complete database schema
- Full API design with examples
- Performance optimization strategies
- Docker configuration
- Load testing approach

### 2. AI_IMPLEMENTATION_GUIDE.md - Step-by-Step Implementation
**Purpose:** Concrete prompts for AI code generation  
**Use for:** Actually building the system with Claude/Cursor/Copilot  
**Contains:**
- 6 phases broken into specific tasks
- Ready-to-use AI prompts for each component
- Time estimates per phase
- Verification checklists
- Common pitfalls to avoid

---

## How to Use These Documents

### Recommended Approach

**Choose Option B for submission:** Provide SPEC.md and this implementation guide as your AI workflow documentation.

### Implementation Flow

```
1. Read SPEC.md completely (15 min)
   ↓
2. Set up work environment
   ↓
3. Follow AI_IMPLEMENTATION_GUIDE phase by phase
   ↓
4. Use Claude/Cursor with the specific prompts
   ↓
5. Verify each phase before moving on
   ↓
6. Run load tests and generate report
   ↓
7. Polish and submit
```

---

## Key Design Decisions Explained

### Why Go with stdlib only?
- Shows understanding of fundamentals
- No framework magic - explicit control
- Better performance (no abstractions)
- Easier to review and understand

### Why Redis + PostgreSQL?
- **Redis:** Sub-millisecond cache for hot redirect path
- **PostgreSQL:** Reliable storage with excellent time-series support
- Together: Handles 1000+ RPS easily

### Why Go Channels instead of RabbitMQ/Kafka?
- Simpler for MVP
- No additional service to manage
- Go channels are fast and reliable
- Sufficient for single-server deployment
- Easy to replace with real queue later

### Why Async Analytics?
- **Critical:** Redirect must be <10ms
- Analytics processing takes 50-100ms
- Fire-and-forget pattern keeps redirect fast
- Worker pool handles backpressure

### Why SSE instead of WebSocket?
- Simpler protocol (one-way communication)
- Built on HTTP (no special proxy config)
- Auto-reconnection in browsers
- Sufficient for real-time counter updates

---

## Time Allocation Strategy

**Total time: 4 hours implementation + documentation**

### Suggested Breakdown:
- **Phase 1-3 (Backend):** 1.5 hours
  - Database and core logic: 45 min
  - Redirect hot path: 30 min
  - Analytics workers: 15 min

- **Phase 4 (Server Setup):** 15 minutes
  - Middleware and routing

- **Phase 5 (Frontend):** 1.5 hours
  - Next.js setup: 15 min
  - Create link page: 20 min
  - Links list: 20 min
  - Analytics dashboard: 35 min

- **Phase 6 (DevOps):** 45 minutes
  - Docker setup: 20 min
  - Load testing: 15 min
  - Documentation: 10 min

**Buffer: 30 minutes** for debugging and polish

---

## Using AI Tools Effectively

### With Claude (via API or Web)

**For each task:**
1. Copy the task prompt from AI_IMPLEMENTATION_GUIDE.md
2. Add: "Reference SPEC.md for details"
3. Review output, test, iterate
4. Move to next task

**Example:**
```
I'm implementing Task 1.2: Database Layer from the AI Implementation Guide.

[Paste prompt from guide]

Reference the database schema in SPEC.md section "Database Schema".
Use prepared statements and proper error handling.
```

### With Cursor

**Workflow:**
1. Open SPEC.md and AI_IMPLEMENTATION_GUIDE.md in workspace
2. Use Cmd+K/Ctrl+K with task prompts
3. Reference both documents in chat: "@SPEC.md @AI_IMPLEMENTATION_GUIDE.md"
4. Use Cursor's composer for multi-file changes

### With GitHub Copilot

**Strategy:**
1. Keep SPEC.md open in a tab
2. Use detailed comments above functions
3. Let Copilot autocomplete based on context
4. Verify against SPEC.md requirements

---

## Critical Success Factors

### Must-Haves:

1. **Performance:**
   - Redirect latency < 100ms @ 1000 RPS
   - Error rate < 1%
   - Load test report with graphs

2. **Functionality:**
   - All CRUD operations work
   - Real-time updates via SSE
   - Analytics are accurate

3. **Code Quality:**
   - No Go frameworks (stdlib only)
   - Clean architecture
   - Proper error handling

4. **Documentation:**
   - docker-compose up works first try
   - README explains everything
   - SPEC.md shows design thinking

### Nice-to-Haves:

- Beautiful UI
- Extra features
- Over-optimization

**Focus on the must-haves first!**

---

## Common Questions

### Q: Can I use a Go framework like Gin or Echo?
**A:** No. Assignment explicitly requires stdlib only. Use `net/http.ServeMux`.

### Q: Should I implement user authentication?
**A:** No. Use a simple user_id string for demo purposes. Focus on the core functionality.

### Q: What if I can't hit 1000 RPS?
**A:** Document what you achieved and why. Analyze the bottleneck. Show you understand the problem.

### Q: How important is the load test?
**A:** Very important. It demonstrates your understanding of high-load systems. The report is part of the deliverable.

### Q: Can I skip the real-time updates?
**A:** No. SSE/real-time is a core requirement. It's a key differentiator of the product.

---

## Submission Checklist

Before submitting:

### Structure:
```
link-analytics-service/
├── backend/
├── frontend/
├── load-test/
│   ├── script.js
│   └── report.md
├── docker-compose.yml
├── SPEC.md (this document)
├── AI_IMPLEMENTATION_GUIDE.md
└── README.md
```

### Functionality:
- [ ] docker-compose up starts everything
- [ ] Can create links via frontend
- [ ] Redirect works
- [ ] Can view link list
- [ ] Analytics dashboard shows data
- [ ] Real-time counter updates

### Performance:
- [ ] Load test ran successfully
- [ ] Achieved at least 500 RPS
- [ ] Report has graphs and analysis
- [ ] Bottleneck identified

### Documentation:
- [ ] README has clear instructions
- [ ] SPEC.md explains design decisions
- [ ] Load test report is complete
- [ ] Code has reasonable comments

---

## Architecture Quick Reference

```
User Request
     ↓
Next.js Frontend (Port 3000)
     ↓
Go Backend (Port 8080)
     ↓
   ┌─────┴─────┐
   ↓           ↓
Redis      PostgreSQL
(Cache)    (Persistent)
```

**Redirect Flow:**
1. User clicks short link
2. Go checks Redis cache (1ms)
3. If miss, query PostgreSQL (5ms)
4. Return 302 redirect (total: <10ms)
5. Fire analytics event to channel (async)
6. Worker processes event (50ms, async)

**Real-time Flow:**
1. Frontend opens SSE connection
2. Worker processes click event
3. Worker broadcasts to SSE broker
4. Broker sends to all connected clients
5. Frontend updates counter

---

## Performance Targets

| Metric | Target | Measurement |
|--------|--------|-------------|
| Redirect latency (p50) | <10ms | k6 |
| Redirect latency (p95) | <50ms | k6 |
| Redirect latency (p99) | <100ms | k6 |
| Throughput | 1000+ RPS | k6 |
| Error rate | <1% | k6 |
| Cache hit rate | >80% | Redis logs |

---

## Next Steps

1. **Review both documents** (30 min)
2. **Set up your workspace** (15 min)
3. **Start Phase 1** using AI_IMPLEMENTATION_GUIDE.md
4. **Build iteratively** - test each phase
5. **Run load tests** as soon as backend is ready
6. **Polish and document** your results

---

## Good Luck! 🚀

Remember:
- **Quality over features** - working basics beat broken extras
- **Use AI effectively** - structured prompts get better results
- **Test as you go** - catch issues early
- **Document your thinking** - show your understanding

This assignment tests your ability to:
1. Design for high performance
2. Use AI tools effectively
3. Make pragmatic technical decisions
4. Deliver working software quickly

You've got this! The specifications are comprehensive and the prompts are ready. Just follow the guide step by step.
