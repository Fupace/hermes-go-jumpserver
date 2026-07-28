# Hermes JumpServer

A simple, lightweight JumpServer-like bastion host platform written in Go.

## Features

- **Machine CRUD** — Add, edit, delete, and list SSH machines via REST API and web UI
- **Web SSH Terminal** — Connect to machines directly from the browser using xterm.js + WebSocket
- **Web Login** — Simple username/password authentication with session management
- **Self-contained** — SQLite backend, single binary, no external dependencies

## Architecture

```
Browser (xterm.js) ──WebSocket──▶ Go Server ──SSH──▶ Target Machine
       │                              │
       └──HTTP/REST──▶ Go Server ────SQLite (machines, users, sessions)
```

## Quick Start

### Build

```bash
go mod tidy
CGO_ENABLED=1 go build -o jumpserver .
```

### Run

```bash
./jumpserver --addr :8080 --data-dir ./data
```

Default credentials: `admin` / `admin`

Open http://localhost:8080 in your browser.

### Docker

```bash
docker build -t jumpserver .
docker run -p 8080:8080 jumpserver
```

## API

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/login` | Login (returns session cookie) |
| POST | `/api/logout` | Logout |
| GET | `/api/session` | Check session status |
| GET | `/api/machines` | List all machines |
| POST | `/api/machines` | Create machine |
| GET | `/api/machines/:id` | Get machine details |
| PUT | `/api/machines/:id` | Update machine |
| DELETE | `/api/machines/:id` | Delete machine |
| WS | `/api/ssh/:id` | WebSocket SSH terminal |

## Deploy to Kubernetes

```bash
kubectl apply -f k8s/deployment.yaml
kubectl apply -f k8s/service.yaml
```

Set `KUBECONFIG` secret and `K8S_NAMESPACE` variable in GitHub Actions for automated CI/CD deployment.

## License

MIT
