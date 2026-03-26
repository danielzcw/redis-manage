# Redis Manage

A lightweight Redis management web UI built with Go. Single binary, zero external dependencies.

## Features

- **Key Browser** — Scan, view, edit, and delete keys with content-aware parsing (auto-detects JSON, text, binary)
- **Queue Management** — First-class views for List and Stream types with push/pop operations and consumer group monitoring
- **Big Key Analysis** — Full keyspace scan with MEMORY USAGE for accurate byte-level sizing
- **Hot Key Detection** — Sampling-based frequency analysis using OBJECT FREQ (LFU) with fallback

## Quick Start

```bash
# Build
go build -o bin/redis-manage ./cmd/server

# Run (connects to localhost:6379 by default)
./bin/redis-manage

# With custom Redis
REDIS_ADDR=redis.example.com:6380 REDIS_PASSWORD=secret ./bin/redis-manage
```

Open http://localhost:9528 in your browser.

## Configuration

| Env Variable   | Default          | Description          |
|----------------|------------------|----------------------|
| PORT           | 9528             | HTTP server port     |
| REDIS_ADDR     | localhost:6379   | Redis server address |
| REDIS_PASSWORD |                  | Redis password       |
| REDIS_DB       | 0                | Redis database index |

## API

| Method | Path                     | Description              |
|--------|--------------------------|--------------------------|
| GET    | /api/info                | Redis server info        |
| GET    | /api/keys                | Scan keys                |
| GET    | /api/keys/detail         | Get key value            |
| DELETE | /api/keys/detail         | Delete key               |
| PUT    | /api/keys/ttl            | Set key TTL              |
| PUT    | /api/keys/value          | Set string value         |
| GET    | /api/queues              | List queue-type keys     |
| GET    | /api/queues/detail       | Queue detail             |
| POST   | /api/queues/push         | Push to queue            |
| POST   | /api/queues/pop          | Pop from queue           |
| GET    | /api/analysis/bigkeys    | Scan for big keys        |
| GET    | /api/analysis/hotkeys    | Detect hot keys          |

## Tech Stack

- Go + Gin (HTTP)
- go-redis/v9 (Redis client)
- Embedded SPA (single binary deployment)
- zap (structured logging)
