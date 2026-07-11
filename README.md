# Docker Manager Backend

REST API ภาษา Go สำหรับตรวจสอบและควบคุม Docker Engine พร้อม session-cookie authentication และ resource statistics

## ความสามารถ

- Admin login ด้วย bcrypt hash และ HttpOnly session cookie
- Docker host, container list, detail และ logs
- Start, Stop และ Restart container
- CPU, memory, block I/O และ network I/O จาก Docker Stats API

## ความต้องการ

- Go 1.24+
- Docker Engine หรือ Docker Desktop
- user ที่รัน API มีสิทธิ์เชื่อมต่อ Docker daemon

## สร้าง Password Hash

```powershell
cd D:\DockerManager\backend
go run .\cmd\hash-password\main.go
```

รหัสผ่านต้องยาว 12–72 bytes จากนั้นนำ bcrypt hash ไปใส่ `ADMIN_PASSWORD_HASH`

## Environment

สร้าง `.env` สำหรับ PowerShell:

```powershell
$env:APP_ADDR = "127.0.0.1:10000"
$env:ADMIN_EMAIL = "admin@example.com"
$env:ADMIN_PASSWORD_HASH = '<bcrypt-hash>'
$env:SESSION_TTL = "12h"
$env:SESSION_COOKIE_SECURE = "false"
```

`ADMIN_EMAIL` และ `ADMIN_PASSWORD_HASH` จำเป็นต้องกำหนด `SESSION_TTL` ต้องไม่น้อยกว่า 5 นาที และ production ผ่าน HTTPS ควรใช้ `SESSION_COOKIE_SECURE=true`

## รัน

```powershell
cd D:\DockerManager\backend
Invoke-Expression (Get-Content .env -Raw)
go run .\cmd\api\main.go
```

## API

| Method | Path | รายละเอียด |
|---|---|---|
| GET | `/api/health` | Health check (public) |
| POST | `/api/auth/login` | Login (public) |
| GET | `/api/auth/me` | Current session |
| POST | `/api/auth/logout` | Logout |
| GET | `/api/docker/info` | Docker host information |
| GET | `/api/containers?all=true` | Container list |
| GET | `/api/containers/stats` | Running container stats และค่ารวม |
| GET | `/api/containers/{id}` | Container detail |
| GET | `/api/containers/{id}/logs` | Logs |
| POST | `/api/containers/{id}/start` | Start |
| POST | `/api/containers/{id}/stop?timeout=10` | Stop |
| POST | `/api/containers/{id}/restart?timeout=10` | Restart |

นอกจาก health และ login ทุก endpoint ต้องส่ง session cookie

## ทดสอบและ Build

```powershell
go test ./...
go build -o bin\docker-manager-api.exe .\cmd\api
```

Session เก็บใน memory ดังนั้นเมื่อ restart API ผู้ใช้ต้อง login ใหม่ สำหรับ production ควรใช้ HTTPS reverse proxy และจำกัดสิทธิ์ Docker daemon

เมื่อกำหนด `REDIS_ADDR` ระบบจะเก็บ session ใน Redis พร้อม TTL; ถ้าไม่กำหนดจะ fallback เป็น in-memory สำหรับ local development
