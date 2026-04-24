# Bloope Deployment MVP

Bloope is a local-first deployment platform MVP. It accepts a GitHub repository URL, clones the source, detects the project type, builds an image with Railpack, runs the image with Docker, and exposes the running container through a local Caddy ingress.

The project is intentionally scoped as a one-page deployment pipeline:

- a single UI
- a single API
- a single deployment flow


---

## Links

- **Loom walkthrough:** `https://www.loom.com/share/148185c5153845e9a21d043b38d14d27`
- **Brimble deploy feedback:** `https://nebulamodellingagency.brimble.app/`
- **Sample app:** `https://github.com/sodiqscript111/formora`
- **Deployed app:** `http://nebulamodellingagency.brimble.app/`

## Feedback Deploying on brimble

> All i have to say is this is the fastest have deploy a project ever, extremly fast , i wanted to open another tab to do other stuff and the moment i moved my mouse to open a tab , it's done , the reason the speed is extremly insane to me is cos the project has videos and pictures in it 

---

## Features

- Submit a deployment from a GitHub repository URL
- Validate GitHub repository URLs before execution
- Clone repositories locally into a workspace
- Run lightweight source inspection and readiness hints
- Detect project type and infer small framework-specific startup behavior
- Build container images with Railpack
- Run built images as Docker containers
- Route deployed apps through Caddy as the single ingress point
- Persist deployments, logs, and deployment environment variables in SQLite
- Stream deployment logs live to the UI over SSE
- Show deployment status, image tag, live URL, runtime metadata, and logs in a single-page dashboard

---

## Stack

- **Backend:** Go + Gin
- **Frontend:** Vite + React + TanStack Router + TanStack Query
- **Persistence:** SQLite
- **Build system:** Railpack + BuildKit
- **Runtime:** Docker
- **Ingress:** Caddy
- **Live logs:** Server-Sent Events (SSE)

---

## Architecture

- **Go + Gin backend** serves the deployment API, orchestrates the deployment workflow, persists state, and streams live logs.
- **Vite + React frontend** provides a one-page deployment dashboard for creating deployments and viewing status, details, and logs.
- **SQLite** stores deployments, deployment logs, and deployment environment variables.
- **Railpack** builds container images from cloned repositories.
- **Docker** runs built images as local containers.
- **Caddy** acts as the single local ingress point for all deployed apps.

For common Python web apps, Bloope can infer a runtime start command for:

- FastAPI
- Flask
- Django

The main flow is:

```text
repo URL -> validate -> clone -> source detection -> Railpack build -> Docker run -> Caddy route -> running URL
```

## Deployment Flow

1. A user submits a GitHub repository URL from the frontend.
2. The backend creates a deployment record with an initial `pending` status.
3. The backend validates and normalizes the repository URL.
4. The repository is cloned into a local workspace under `tmp/deployments`.
5. The source is inspected for project type, framework hints, start command hints, and lightweight readiness notes.
6. Railpack builds the application into a container image.
7. The built image is started as a Docker container.
8. Caddy routing is generated and reloaded so traffic reaches the running container.
9. The deployment is marked `running` if the pipeline completes successfully.
10. Deployment logs are persisted and streamed live to the UI during the entire process.

If any stage fails, the deployment is marked `failed`, the error is persisted, and the logs remain visible in the UI.

## Runtime Routing

Each deployment gets a deterministic Docker container name:

```text
bloope-{deploymentID}
```

The runtime maps the app container to a host port. Caddy then routes by host header, so each deployment is served from `/` on its own local hostname:

```text
http://{deploymentID}.localhost:8081
```

to:

```text
host.docker.internal:{hostPort}
```

This host-based routing is important for frontend apps such as Vite builds because generated asset URLs like `/assets/index.js` resolve against the deployment host instead of the platform UI host.

Caddy is started automatically as a Docker container. Its generated Caddyfile lives at:

```text
tmp/caddy/Caddyfile
```

## Persistence

SQLite lives at:

```text
data/bloope.db
```

Override it with:

```powershell
$env:BLOOPE_DB_PATH="data/bloope.db"
```

The persisted tables are:

- `deployments`
- `deployment_logs`
- `deployment_env_vars`

Previously created deployments and logs remain visible after backend restarts.

## Deployment Environment Variables

The deployment form accepts optional `.env`-style values, one per line:

```text
DATABASE_URL=postgresql://user:password@host/db
REDIS_URL=redis://host:6379/0
```

Bloope stores these values in SQLite and injects them into the Docker container with `docker run -e`.

The API and UI only expose the variable names, not the secret values.

For this MVP, values are stored as plaintext in local SQLite, so only local or test credentials should be used.

## Sample App

To test a real frontend deployment flow quickly, use this public sample app:

- [Formora](https://github.com/sodiqscript111/formora)

It is a good fit for this MVP because it exercises the Git clone, Railpack build, Docker runtime, and Caddy host-based routing path for a frontend app without needing private credentials.

## Project Structure

```text
cmd/                backend entrypoint
frontend/           one-page React UI
internal/           backend application code
data/               SQLite database files
tmp/                cloned repositories and generated runtime artifacts
docker-compose.yml  local stack bootstrap
Dockerfile          backend/frontend container build
```

More specifically:

- `cmd/server` - backend bootstrap, API registration, and frontend static serving
- `internal/services` - deployment creation, orchestration, logging, persistence coordination, and pipeline steps
- `internal/state` - deployment state engine and transition rules
- `internal/handlers` - HTTP API and SSE endpoints
- `internal/validation` - GitHub repository URL normalization and validation
- `internal/repository` - cloning repositories into workspace directories
- `internal/source` - project type detection, Python start-command inference, and readiness hints
- `internal/build` - Railpack integration
- `internal/docker` - Docker runtime integration and health checks
- `internal/caddy` - ingress route generation and reload
- `frontend/src` - deployment dashboard UI

## Local Setup

### Docker Compose

The simplest way to run the platform is:

```powershell
docker compose up --build
```

Then open:

```text
http://localhost:8080
```

Deployed apps are routed through the Compose-managed Caddy ingress:

```text
http://{deploymentID}.localhost:8081
```

Compose starts:

- `bloope` - Go API, React frontend, Railpack integration, and Docker CLI
- `bloope-compose-caddy` - local Caddy ingress for deployed apps
- `bloope-buildkit` - BuildKit daemon used by Railpack

The Bloope container mounts the host Docker socket so it can build and run deployment containers.

If older local containers are already using ports `8080` or `8081`, stop them before running Compose.

### Private GitHub Repositories

Private repositories need a token with repository read access.

Create a local `.env` file next to `docker-compose.yml`:

```text
BLOOPE_GITHUB_TOKEN=github_pat_your_token_here
```

Then restart the main service:

```powershell
docker compose up -d --build bloope
```

### Persistent Volumes

Compose persists data in Docker volumes:

- `bloope-data` - SQLite database
- `bloope-tmp` - cloned repositories and runtime artifacts
- `caddy-config` - generated Caddyfile shared with the Caddy container

### Manual Development

Docker Compose is the recommended cross-platform path, including macOS. The manual commands below are the current Windows-oriented workflow.

Start BuildKit for Railpack:

```powershell
wsl.exe -d Ubuntu -- bash -lc "docker start bloope-buildkit || docker run --privileged -d --name bloope-buildkit moby/buildkit"
```

Run the backend:

```powershell
$env:BUILDKIT_HOST="docker-container://bloope-buildkit"
$env:RAILPACK_USE_WSL="1"
$env:RAILPACK_WSL_DISTRO="Ubuntu"
go run ./cmd/server
```

Run the frontend:

```powershell
cd frontend
npm.cmd run dev
```

On macOS or Linux, use Docker Compose or a native `railpack` binary with `RAILPACK_USE_WSL=0`, then run `npm run dev` inside `frontend/`.

Then open the frontend at the Vite URL and create a deployment.

## Environment Variables

- `BLOOPE_DB_PATH` - SQLite database path. Default: `data/bloope.db`
- `BLOOPE_FRONTEND_DIR` - built frontend directory served by the backend. Default: `frontend/dist`
- `BLOOPE_CADDY_PORT` - local Caddy ingress port. Default: `8081`
- `BLOOPE_DEPLOYMENT_HOST_SUFFIX` - host suffix for deployed apps. Default: `localhost`
- `BLOOPE_PUBLIC_SCHEME` - URL scheme for generated live URLs. Default: `http`
- `BLOOPE_CONTAINER_PORT` - fallback internal app port when the image does not expose a port. Default: `8080`
- `BLOOPE_CADDY_CONTAINER` - Caddy Docker container name. Default: `bloope-caddy`
- `BLOOPE_CADDYFILE_PATH` - generated Caddyfile path. Default: `tmp/caddy/Caddyfile`
- `BLOOPE_GITHUB_TOKEN` - optional GitHub token for cloning private repositories
- `BLOOPE_GIT_BIN` - Git binary. Default: `git`
- `BLOOPE_HEALTHCHECK_TIMEOUT` - container startup health-check timeout. Default: `45s`
- `BLOOPE_HEALTHCHECK_INTERVAL` - container health-check polling interval. Default: `1s`
- `DOCKER_BIN` - Docker CLI binary. Default: `docker`
- `RAILPACK_BIN` - Railpack binary. Default: `railpack`
- `RAILPACK_USE_WSL` - run Railpack through WSL on Windows
- `RAILPACK_WSL_DISTRO` - WSL distro for Railpack. Default: `Ubuntu`
- `BUILDKIT_HOST` - BuildKit endpoint for Railpack. Example: `docker-container://buildkit`
- `VITE_API_BASE_URL` - frontend API base URL. Default: `/api`

## API Overview

The frontend talks to the API under `/api` by default:

- `POST /api/deployments` - create a deployment from a GitHub repository URL
- `GET /api/deployments` - list deployments
- `GET /api/deployments/:id` - fetch a single deployment
- `GET /api/deployments/:id/logs` - fetch historical deployment logs
- `GET /api/deployments/:id/logs/stream` - stream live deployment logs with SSE

The same handlers are also mounted without the `/api` prefix.

## Key Decisions / Tradeoffs

### SSE over WebSocket

I used SSE for live deployment logs because log streaming here is one-way from backend to UI. That kept the implementation smaller and easier to reason about than a WebSocket-based design.

### Explicit state engine

I modeled the deployment lifecycle explicitly and kept valid transitions centralized: `pending -> building -> deploying -> running`, with failure possible at each stage. This makes the workflow easier to reason about and prevents invalid state jumps.

### In-process job execution

For the take-home, deployment execution starts from an in-process worker path rather than a durable queue. That kept the architecture simple enough to get an end-to-end system working quickly.

### Caddy as single ingress

I kept Caddy as the only ingress point so deployed apps are not exposed through raw container ports. That also made frontend asset routing more reliable for Vite-based deployments.

### SQLite for local persistence

SQLite was enough for a local-first MVP and made it easy to persist deployment state and logs without introducing more infrastructure.

## What I'd Do With More Time

- Add direct GitHub integration with OAuth or App-based repository access
- Replace in-process job execution with a durable queue such as Redis Streams
- Add authentication and user-scoped deployments
- Reuse build cache and artifacts to reduce rebuild times
- Add stronger health checks before switching traffic live

## What I'd Rip Out or Simplify

- Replace the current in-process worker path with durable job execution
- Reduce some of the local-only routing and runtime assumptions
- Refactor some of the lightweight source detection logic once supported app types are more explicit

## MVP Assumptions

- The app image either exposes a port or listens on `PORT`
- If no exposed port is found, Bloope uses `BLOOPE_CONTAINER_PORT`, defaulting to `8080`
- Deployment-specific environment variables are only provided when the deployment is created
- Python start-command inference is intentionally small: FastAPI, Flask, and Django only
- If no safe start command is inferred, Bloope uses the image default command
- Host-based local routing uses `{deploymentID}.localhost`, which most browsers resolve to loopback automatically
- Caddy is managed as a local Docker container, not as a system service
- Failed Docker containers are cleaned up best-effort

## Known Limitations

- No authentication
- No multi-host scheduling
- No migrations framework beyond startup SQL
- No rollback or deployment history model yet
- No strong readiness checks beyond the initial startup probe
- Caddy route removal is basic and tied to regenerated config
- SQLite is local and intended for MVP usage only
- Environment variables are stored plaintext in SQLite until a real secrets store is added

## Rough Time Spent

10 - 12 hours
