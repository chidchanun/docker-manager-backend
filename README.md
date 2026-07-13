# Docker Manager Backend

Go REST API สำหรับควบคุม Docker Engine ผ่าน Docker Socket Proxy

## Development

```powershell
cd D:\DockerManager\backend
Invoke-Expression (Get-Content .env -Raw)
go run .\cmd\api\main.go
```

Local default คือ `127.0.0.1:8080`; ไฟล์ development ปัจจุบันสามารถกำหนด `APP_ADDR=127.0.0.1:10000` เพื่อให้ Angular proxy เชื่อมต่อ

## Environment variables

| Name | Required | Description |
|---|---|---|
| `ADMIN_EMAIL` | Yes | Admin login email |
| `ADMIN_PASSWORD_HASH` | Yes | bcrypt hash, cost validated |
| `APP_ADDR` | No | API address; default `127.0.0.1:8080` |
| `SESSION_TTL` | No | Session TTL; default `12h`, minimum `5m` |
| `SESSION_COOKIE_SECURE` | No | Use `true` behind HTTPS |
| `REDIS_ADDR` | No | Redis address; blank uses in-memory sessions |
| `DOCKER_HOST` | No | Docker daemon/socket proxy address |
| `METRICS_ADDR` | No | Internal Prometheus address; default `127.0.0.1:9090` |

## Commands

```powershell
go run .\cmd\hash-password\main.go
go test ./...
go build -o bin\docker-manager-api.exe .\cmd\api
```

## API

Public:

| Method | Path | Description |
|---|---|---|
| GET | `/api/health` | Health check |
| POST | `/api/auth/login` | Login |

Authenticated:

| Method | Path | Description |
|---|---|---|
| GET | `/api/auth/me` | Current user |
| POST | `/api/auth/logout` | Logout |
| GET | `/api/docker/info` | Docker host info |
| GET | `/api/containers?all=true` | Containers |
| GET | `/api/containers/stats` | Runtime stats and totals |
| GET | `/api/containers/{id}` | Container detail |
| GET | `/api/containers/{id}/logs` | Log snapshot |
| GET | `/api/containers/{id}/logs/stream` | SSE Live Logs |
| POST | `/api/containers/{id}/start` | Start |
| POST | `/api/containers/{id}/stop?timeout=10` | Stop |
| POST | `/api/containers/{id}/restart?timeout=10` | Restart |
| POST | `/api/containers/{id}/pause` | Pause |
| POST | `/api/containers/{id}/unpause` | Unpause |
| POST | `/api/containers/{id}/kill` | SIGKILL |
| DELETE | `/api/containers/{id}` | Remove without volumes/force |
| PATCH | `/api/containers/{id}/policy` | Restart/resource policy |
| GET | `/api/audit?limit=100` | Redis Stream audit entries |

Metrics server exposes `/metrics` on `METRICS_ADDR`; it is not mounted on the public API mux

## Container policy payload

```json
{
  "restart_policy": "unless-stopped",
  "maximum_retry_count": 0,
  "cpus": 2,
  "memory_bytes": 4294967296,
  "pids_limit": 500
}
```

MemorySwap is updated with Memory to satisfy Docker and prevent additional swap allowance

## Security

- Same-origin protection for unsafe methods
- Secure/HttpOnly/SameSite session cookie
- Login rate limiting
- Secret redaction from container command arguments
- Persistent audit through Redis Stream
- Security headers and bounded request bodies
- Docker access restricted through Socket Proxy
