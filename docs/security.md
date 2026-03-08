# Security Guide

## Secure by Default

Cocopilot ships with safe defaults suitable for local development:

| Setting | Default | Security Implication |
|---------|---------|---------------------|
| Listen address | `127.0.0.1:8080` | Binds localhost only — safe default for local dev |
| API key auth | Disabled | No auth on mutations — enable for shared/production use |
| DB path | `./tasks.db` | Local file — ensure proper file permissions |

## Enabling Authentication

For any shared or production deployment, enable API key authentication:

```bash
export COCO_REQUIRE_API_KEY=true
export COCO_API_KEY=$(openssl rand -hex 32)
go run .
```

With auth enabled:
- All v2 mutation endpoints require `Authorization: Bearer <key>`
- Read-only endpoints (GET) remain accessible
- The kanban UI remains accessible

## Reverse Proxy Setup

Never expose the server directly to the internet. Use a reverse proxy with TLS.

### nginx

```nginx
server {
    listen 443 ssl;
    server_name cocopilot.example.com;

    ssl_certificate     /etc/ssl/certs/cocopilot.pem;
    ssl_certificate_key /etc/ssl/private/cocopilot.key;

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

        # SSE support
        proxy_set_header Connection '';
        proxy_buffering off;
        proxy_cache off;
        proxy_read_timeout 3600s;
    }
}
```

### Caddy

```
cocopilot.example.com {
    reverse_proxy 127.0.0.1:8080
}
```

Caddy automatically provisions TLS certificates.

## Workdir Guardrails

When agents execute tasks that modify the filesystem:
- Tasks are scoped to a project's configured working directory
- The scanner respects `.gitignore` patterns
- File operations go through the project file API, which validates paths

## API Key Management

- Use a strong, random key (at least 32 hex characters)
- Rotate keys by updating the `COCO_API_KEY` environment variable and restarting
- Never commit API keys to source control
- Use environment variables or a secrets manager in CI/CD

## Database Security

- SQLite stores all data in a single file — protect it with filesystem permissions
- Run regular backups via the `/api/v2/backup` endpoint
- The backup/restore API validates schema versions to prevent data corruption
- Delete `tasks.db` to reset all data (migrations recreate the schema on restart)

## Network Security

- The default listen address is `127.0.0.1:8080` (localhost only). To expose on all interfaces:
  ```bash
  COCO_HTTP_ADDR=0.0.0.0:8080 go run .
  ```
- Use firewall rules to restrict access to the server port
- All SSE connections are server-to-client (no websocket upgrade needed)

## Reporting Vulnerabilities

If you discover a security vulnerability, please report it responsibly. See [SECURITY.md](../SECURITY.md) for reporting instructions.
