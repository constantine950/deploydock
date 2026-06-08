# DeployDock — Product Requirements Document

**Version:** 1.0  
**Author:** Adebesin Omotoyosi James  
**Status:** Draft  
**Last Updated:** 2026-06-08

---

## 1. Overview

DeployDock is a self-hosted Platform-as-a-Service (PaaS) that lets developers push code and have it automatically containerized, deployed, and served — with zero-downtime rolling updates, automatic HTTPS, and a live log streaming dashboard.

It is, in essence, your own Heroku — built from scratch, running on infrastructure you own.

---

## 2. Problem Statement

Managed PaaS platforms (Heroku, Render, Railway) abstract away too much, cost too much at scale, and lock you into their infrastructure. Developers who want full control over their deployment pipeline — while still enjoying a smooth push-to-deploy experience — have no lightweight, self-hostable alternative that is easy to understand end-to-end.

DeployDock solves this by providing a minimal, auditable PaaS layer on top of Docker and Nginx, deployable on any Linux VPS.

---

## 3. Goals

- Git push → running container in under 2 minutes
- Zero-downtime rolling deploys with automatic rollback on failure
- Automatic HTTPS provisioning via Let's Encrypt
- Secure environment variable management
- Live log streaming per deployment
- Resource quota controls per app
- A clean dashboard UI that reflects real deployment state in real time

---

## 4. Non-Goals (v1)

- Multi-server / cluster deployments (Kubernetes backend)
- Built-in database provisioning
- Team collaboration / multi-user roles
- Billing or usage metering
- Support for monorepo multi-service deployments
- Windows host support

---

## 5. Supported Runtimes

| Runtime | Detection Signal        | Base Image                                     |
| ------- | ----------------------- | ---------------------------------------------- |
| Node.js | `package.json`          | `node:20-alpine`                               |
| Python  | `requirements.txt`      | `python:3.12-slim`                             |
| Go      | `go.mod`                | `golang:1.23-alpine` (build) + `scratch` (run) |
| Static  | `index.html` (no above) | `nginx:alpine`                                 |

---

## 6. Deployment Lifecycle

```
Git Push
   │
   ▼
Webhook Received (POST /webhooks/git)
   │  Validate signature
   │  Create deployment record — status: queued
   │  Enqueue build job in Redis
   ▼
Build Worker (goroutine pool)
   │  Clone repo to temp dir
   │  Detect runtime
   │  Select Dockerfile template
   │  Build Docker image → tag: deploydock/{app-id}:{deployment-id}
   │  Stream build output → build_logs table
   │  On failure: status = failed, store error
   ▼
Deploy Engine
   │  Start new container from image
   │  Assign internal port, connect to app network
   │  Poll health check (max 30s)
   │  On health pass:
   │    Update Nginx upstream → new container
   │    Reload Nginx (nginx -s reload)
   │    Stop and remove old container
   │    status = live
   │  On health fail:
   │    Stop new container
   │    status = failed (previous container remains live)
   ▼
Live
   │  App reachable via subdomain: {app-slug}.deploydock.{domain}
   │  Logs streaming via WebSocket
   │  Env vars injected at container start
```

---

## 7. Resource Model

### App

| Field      | Type      | Notes                                   |
| ---------- | --------- | --------------------------------------- |
| id         | UUID      | Primary key                             |
| user_id    | UUID      | Owner                                   |
| name       | string    | Human-readable                          |
| slug       | string    | URL-safe, unique — used in subdomain    |
| repo_url   | string    | Git repo to clone                       |
| branch     | string    | Default: `main`                         |
| runtime    | enum      | node, python, go, static                |
| status     | enum      | idle, building, deploying, live, failed |
| created_at | timestamp |                                         |

### Deployment

| Field          | Type      | Notes                                                  |
| -------------- | --------- | ------------------------------------------------------ |
| id             | UUID      |                                                        |
| app_id         | UUID      |                                                        |
| commit_sha     | string    |                                                        |
| commit_message | string    |                                                        |
| status         | enum      | queued, building, deploying, live, failed, rolled_back |
| container_id   | string    | Docker container ID                                    |
| image_tag      | string    | deploydock/{app-id}:{deployment-id}                    |
| port           | int       | Internal container port                                |
| started_at     | timestamp |                                                        |
| finished_at    | timestamp |                                                        |

### Environment

| Field      | Type      | Notes                     |
| ---------- | --------- | ------------------------- |
| id         | UUID      |                           |
| app_id     | UUID      |                           |
| key        | string    |                           |
| value      | string    | AES-256 encrypted at rest |
| created_at | timestamp |                           |

### Domain

| Field      | Type      | Notes                   |
| ---------- | --------- | ----------------------- |
| id         | UUID      |                         |
| app_id     | UUID      |                         |
| hostname   | string    | e.g. myapp.example.com  |
| ssl_status | enum      | pending, active, failed |
| created_at | timestamp |                         |

---

## 8. API Surface (v1)

### Auth

| Method | Path           | Description    |
| ------ | -------------- | -------------- |
| POST   | /auth/register | Create account |
| POST   | /auth/login    | Get JWT token  |

### Apps

| Method | Path      | Description      |
| ------ | --------- | ---------------- |
| GET    | /apps     | List user's apps |
| POST   | /apps     | Create app       |
| GET    | /apps/:id | Get app detail   |
| DELETE | /apps/:id | Delete app       |

### Deployments

| Method | Path                      | Description             |
| ------ | ------------------------- | ----------------------- |
| GET    | /apps/:id/deployments     | List deployments        |
| POST   | /apps/:id/deploy          | Trigger manual deploy   |
| GET    | /deployments/:id          | Get deployment detail   |
| POST   | /deployments/:id/rollback | Rollback to this deploy |
| GET    | /deployments/:id/logs     | WebSocket — live logs   |

### Env Vars

| Method | Path               | Description               |
| ------ | ------------------ | ------------------------- |
| GET    | /apps/:id/env      | List keys (values masked) |
| POST   | /apps/:id/env      | Set env var               |
| DELETE | /apps/:id/env/:key | Delete env var            |

### Domains

| Method | Path                  | Description       |
| ------ | --------------------- | ----------------- |
| GET    | /apps/:id/domains     | List domains      |
| POST   | /apps/:id/domains     | Add custom domain |
| DELETE | /apps/:id/domains/:id | Remove domain     |

### Webhooks

| Method | Path          | Description        |
| ------ | ------------- | ------------------ |
| POST   | /webhooks/git | Receive push event |

---

## 9. Architecture Decisions

### Why Go for the backend?

Go's concurrency model (goroutines) maps cleanly to the build worker pool pattern. The Docker SDK for Go is first-class. Binaries are small and fast to start — important when the backend itself is containerized.

### Why Nginx over Traefik?

Nginx config is explicit, readable, and battle-tested. Traefik's dynamic config via labels is elegant but adds a layer of abstraction we don't need for v1. Nginx gives us full control over upstream definitions and reload behaviour.

### Why Redis for the build queue?

Redis list-based queues (LPUSH/BRPOP) are simple, fast, and already required for log pub/sub. No need for a separate message broker.

### Why Postgres for everything else?

Structured relational data with strong consistency guarantees. Deployment state transitions are critical — a relational model with proper constraints prevents invalid state.

---

## 10. Security Considerations

- Webhook payloads validated by HMAC-SHA256 signature
- JWT tokens for API authentication (RS256, 24h expiry)
- Env var values AES-256 encrypted at rest, never returned in API responses after creation
- Docker containers run with no-new-privileges, non-root user where possible
- Nginx configured to strip internal headers before proxying

---

## 11. Success Criteria

- [ ] `git push` to a real repo triggers a build and deploy with no manual intervention
- [ ] Zero-downtime verified by a curl loop running during deploy
- [ ] HTTPS working on a custom domain with auto-renewing cert
- [ ] Live log stream visible in browser during build and deploy
- [ ] Rollback restores previous live deployment in under 10 seconds
- [ ] Full stack runs with a single `docker-compose up`
