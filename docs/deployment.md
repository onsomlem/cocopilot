# Deployment Guide

Production-ready deployment patterns for Cocopilot.

> For initial setup and development, see [full-setup-guide.md](full-setup-guide.md).  
> For security hardening, see [security.md](security.md).

---

## Quick Start (Binary)

```bash
# Build
go build -o cocopilot ./cmd/cocopilot

# Run with production settings
COCO_DB_PATH=/var/lib/cocopilot/tasks.db \
COCO_HTTP_ADDR=127.0.0.1:8080 \
COCO_REQUIRE_API_KEY=true \
COCO_API_KEY=your-secret-key \
./cocopilot
```

---

## Systemd Service

Create `/etc/systemd/system/cocopilot.service`:

```ini
[Unit]
Description=Cocopilot Agentic Task Queue
After=network.target

[Service]
Type=simple
User=cocopilot
Group=cocopilot
WorkingDirectory=/opt/cocopilot
ExecStart=/opt/cocopilot/cocopilot
Restart=on-failure
RestartSec=5

# Environment
Environment=COCO_DB_PATH=/var/lib/cocopilot/tasks.db
Environment=COCO_HTTP_ADDR=127.0.0.1:8080
Environment=COCO_REQUIRE_API_KEY=true
EnvironmentFile=-/etc/cocopilot/env

# Security hardening
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/lib/cocopilot
PrivateTmp=true

[Install]
WantedBy=multi-user.target
```

Store secrets in `/etc/cocopilot/env`:

```bash
COCO_API_KEY=your-secret-key
```

Enable and start:

```bash
sudo useradd -r -s /usr/sbin/nologin cocopilot
sudo mkdir -p /var/lib/cocopilot /etc/cocopilot
sudo chown cocopilot:cocopilot /var/lib/cocopilot
sudo chmod 600 /etc/cocopilot/env

sudo systemctl daemon-reload
sudo systemctl enable --now cocopilot
sudo journalctl -u cocopilot -f
```

---

## Docker

### Dockerfile

```dockerfile
FROM golang:1.23-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /cocopilot ./cmd/cocopilot

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=builder /cocopilot /usr/local/bin/cocopilot
COPY migrations/ /migrations/
VOLUME /data
ENV COCO_DB_PATH=/data/tasks.db
ENV COCO_HTTP_ADDR=0.0.0.0:8080
EXPOSE 8080
ENTRYPOINT ["cocopilot"]
```

### docker-compose.yml

```yaml
services:
  cocopilot:
    build: .
    ports:
      - "8080:8080"
    volumes:
      - cocopilot-data:/data
    environment:
      COCO_DB_PATH: /data/tasks.db
      COCO_HTTP_ADDR: "0.0.0.0:8080"
      COCO_REQUIRE_API_KEY: "true"
      COCO_API_KEY: "${COCO_API_KEY}"
    restart: unless-stopped

volumes:
  cocopilot-data:
```

```bash
COCO_API_KEY=your-secret docker compose up -d
```

---

## Reverse Proxy (nginx)

```nginx
upstream cocopilot {
    server 127.0.0.1:8080;
}

server {
    listen 443 ssl http2;
    server_name tasks.example.com;

    ssl_certificate     /etc/letsencrypt/live/tasks.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/tasks.example.com/privkey.pem;

    location / {
        proxy_pass http://cocopilot;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    # SSE endpoints need long timeouts
    location ~ ^/(events|api/v2/events/stream|api/v2/projects/.*/events/stream) {
        proxy_pass http://cocopilot;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_http_version 1.1;
        proxy_set_header Connection "";
        proxy_buffering off;
        proxy_read_timeout 3600s;
    }
}
```

---

## Backup & Restore

### API-based backup

```bash
# Backup
curl -s http://127.0.0.1:8080/api/v2/backup \
  -H "Authorization: Bearer $COCO_API_KEY" \
  -o backup-$(date +%Y%m%d).db

# Restore
curl -s -X POST http://127.0.0.1:8080/api/v2/restore \
  -H "Authorization: Bearer $COCO_API_KEY" \
  --data-binary @backup-20260310.db
```

### Automated daily backup (cron)

```bash
# /etc/cron.d/cocopilot-backup
0 2 * * * cocopilot curl -sf http://127.0.0.1:8080/api/v2/backup -H "Authorization: Bearer $(cat /etc/cocopilot/api-key)" -o /var/backups/cocopilot/tasks-$(date +\%Y\%m\%d).db && find /var/backups/cocopilot -name '*.db' -mtime +30 -delete
```

---

## Monitoring

### Health check

```bash
curl -sf http://127.0.0.1:8080/api/v2/health | jq .ok
# Returns: true
```

### Metrics endpoint

```bash
curl -s http://127.0.0.1:8080/api/v2/metrics | jq
```

Returns task counts by status, active leases, agent count, automation circuit state, and schema version.

### Status endpoint

```bash
curl -s http://127.0.0.1:8080/api/v2/status | jq
```

Returns uptime, active projects, active runs, and schema version.

### Docker health check

Add to your Dockerfile or compose:

```yaml
healthcheck:
  test: ["CMD", "wget", "-q", "--spider", "http://localhost:8080/api/v2/health"]
  interval: 30s
  timeout: 5s
  retries: 3
```

---

## Environment Variables Reference

| Variable | Default | Description |
|----------|---------|-------------|
| `COCO_DB_PATH` | `./tasks.db` | SQLite database file path |
| `COCO_HTTP_ADDR` | `127.0.0.1:8080` | Listen address (`host:port`) |
| `COCO_REQUIRE_API_KEY` | `false` | Enable API key authentication |
| `COCO_API_KEY` | — | Shared API key value |
| `COCO_AUTOMATION_RULES` | — | JSON array of automation rules |
| `COCO_MAX_AUTOMATION_DEPTH` | `5` | Max automation recursion depth |
| `COCO_AUTOMATION_RATE_LIMIT` | `100` | Max automation executions/hour |
| `COCO_AUTOMATION_BURST_LIMIT` | `10` | Max automation executions/minute |
