# TaskQueue ⚡

> A reliable, distributed background job queue in Go — Redis-backed, with automatic retries (exponential backoff), a dead-letter queue, and an HTTP API for enqueueing tasks and monitoring queue health. Built from scratch as a smaller, transparent alternative to Celery/Sidekiq.

![CI](https://github.com/YOUR_USERNAME/taskqueue/actions/workflows/ci.yml/badge.svg)
![Go](https://img.shields.io/badge/Go-1.22-00ADD8)
![License](https://img.shields.io/badge/license-MIT-green)

## Why build this?

Most apps eventually need to run work outside the request/response cycle — sending emails, resizing images, generating reports. This implements the core patterns a production task queue needs, using nothing but Go's standard library + a Redis client:

- **Reliable queue pattern** — tasks move atomically from `pending` → a worker-specific `processing` list (via `RPOPLPUSH`), so a crashed worker never silently loses a task mid-flight.
- **Exponential backoff retries** — failed tasks are rescheduled with increasing delay (1s, 2s, 4s... capped at 60s) instead of hammering a struggling downstream service.
- **Dead-letter queue** — tasks that exhaust their retry budget are moved aside for inspection instead of retrying forever.
- **Horizontal scalability** — run as many worker processes as you want against the same Redis instance; they coordinate purely through Redis, no shared state.

## Architecture

```
                 ┌─────────────┐
   HTTP POST     │             │   BRPOPLPUSH    ┌──────────┐
  /api/tasks ───▶│    Redis    │◀───────────────▶│ Worker 1 │
                 │             │                  └──────────┘
                 │  pending    │   BRPOPLPUSH    ┌──────────┐
                 │  processing │◀───────────────▶│ Worker 2 │
                 │  delayed    │                  └──────────┘
                 │  dead_letter│
                 └─────────────┘
                        ▲
                        │ GET /api/stats
                 ┌─────────────┐
                 │  HTTP API   │
                 └─────────────┘
```

```
taskqueue/
├── cmd/server/main.go          # entrypoint: wires API + worker pool together
├── internal/queue/queue.go     # Redis-backed reliable queue (the core logic)
├── internal/worker/worker.go   # concurrent worker pool + handler dispatch
├── internal/api/api.go         # HTTP API (enqueue, health, stats)
├── internal/queue/queue_test.go # 7 tests using miniredis (no real Redis needed)
├── Dockerfile
├── docker-compose.yml           # app + redis, one command to run
└── .github/workflows/ci.yml
```

## Quick Start

### With Docker Compose (easiest)

```bash
docker compose up --build
```

### Locally

```bash
# 1. Start Redis (or use docker: docker run -p 6379:6379 redis:7-alpine)
redis-server

# 2. Run the server
go run ./cmd/server
```

## Usage

**Enqueue a task:**

```bash
curl -X POST http://localhost:8080/api/tasks \
  -H "Content-Type: application/json" \
  -d '{"type": "send_email", "payload": {"to": "user@example.com", "subject": "Hi"}, "max_retries": 3}'
```

**Check queue health:**

```bash
curl http://localhost:8080/api/stats
```

```json
{
  "enqueued": 42,
  "completed": 38,
  "failed": 3,
  "dead_letter": 1,
  "pending_depth": 0,
  "delayed_depth": 1
}
```

**Register your own task handlers** (in `cmd/server/main.go`):

```go
pool.RegisterHandler("generate_report", func(ctx context.Context, payload []byte) error {
    var p struct{ ReportID string `json:"report_id"` }
    json.Unmarshal(payload, &p)
    return generateReport(ctx, p.ReportID)
})
```

## Running Tests

```bash
go test ./... -v
```

Tests use [miniredis](https://github.com/alicebob/miniredis) (an in-memory Redis implementation), so no real Redis instance is needed to run the suite.

## Design Notes

- **Why `RPOPLPUSH` instead of plain `RPOP`?** Plain pop-and-process has a window where a task is removed from the queue but the worker crashes before finishing it — the task is lost forever. `RPOPLPUSH` atomically moves it to a per-worker processing list, so an "orphaned" processing list is detectable and recoverable (a production hardening would add a reaper that requeues stale processing-list entries after a timeout).
- **Why a sorted set for delayed retries?** A `ZSET` scored by "ready-at" timestamp lets us cheaply query "which retries are due now" (`ZRangeByScore`) without scanning the whole retry backlog.

## Roadmap

- [ ] Reaper for orphaned processing-list entries (worker crash recovery)
- [ ] Priority queues (multiple pending lists with weighted consumption)
- [ ] Web dashboard (currently JSON-only `/api/stats`)
- [ ] Prometheus metrics endpoint

## License

MIT — see [LICENSE](LICENSE).
