# ArmadaCMS

Backend API and admin dashboard for [THS Armada](https://armada.nu). Provides REST endpoints consumed by the public website ([armada.nu](https://github.com/armada-ths/armada.nu)) and a React-Admin interface for content management.

## Tech Stack

### Backend

- **Language**: Go 1.24
- **Router**: [Gorilla Mux](https://github.com/gorilla/mux)
- **ORM**: [GORM](https://gorm.io/) (Postgres)
- **Auth**: JWT (Bearer tokens)
- **File storage**: AWS S3
- **Hot reload**: [Air](https://github.com/air-verse/air) (in Docker dev mode)

### Admin Frontend

- **Framework**: [React-Admin v5](https://marmelab.com/react-admin/)
- **Build tool**: [Vite](https://vitejs.dev/)
- **Language**: TypeScript
- **UI**: MUI (Material UI)

### Infrastructure

- **Deployment**: Docker (currently deployed on AWS ECS, prepared for Cloud Run migration)
- **Database**: PostgreSQL (AWS RDS)

## Prerequisites

- [Go 1.24+](https://go.dev/dl/)
- [Node.js 20+](https://nodejs.org/) and npm (for the admin frontend)
- [Docker](https://www.docker.com/) and Docker Compose (optional, for containerized setup)
- PostgreSQL instance (local or remote)

## Getting Started

1. **Clone the repo**

   ```bash
   git clone https://github.com/armada-ths/ArmadaCMS.git
   cd ArmadaCMS
   ```

2. **Set up environment variables**

   ```bash
   cp .env.example .env
   ```

   Edit `.env` with your Postgres credentials. See `.env.example` for all available variables and descriptions.

3. **Start a local PostgreSQL instance (recommended for local backend development)**

   The repo now includes a small standalone Postgres Compose file:

   ```bash
   docker compose -f docker-compose.db.yml up -d
   ```

   This creates a local database with the following defaults:
   - Host: `localhost`
   - Port: `5432`
   - Database: `armadacms`
   - User: `postgres`
   - Password: `postgres`
   - Image: `postgres:17`

   For local use, set your `.env` database section to:

   ```bash
   DB_HOST=localhost
   DB_PORT=5432
   DB_USER=postgres
   DB_PASSWORD=postgres
   DB_NAME=armadacms
   DB_SSLMODE=disable
   ```

   To stop the database later without deleting its data:

   ```bash
   docker compose -f docker-compose.db.yml stop
   ```

   To stop it and remove the container while keeping the named volume available for reuse:

   ```bash
   docker compose -f docker-compose.db.yml down
   ```

4. **Optionally clone a remote database into your local Postgres**

   If you want realistic local data, the repo includes a PowerShell import script that can clone any reachable PostgreSQL database (for example staging, or prod if your current IP is temporarily allowlisted).

   Add these values to your local `.env` first:

   ```bash
   SOURCE_DB_HOST=
   SOURCE_DB_PORT=5432
   SOURCE_DB_USER=postgres
   SOURCE_DB_PASSWORD=
   SOURCE_DB_NAME=
   SOURCE_DB_SSLMODE=require
   SOURCE_DB_TOOLS_IMAGE=postgres:17
   ```

   Then run:

   ```powershell
   ./scripts/import-remote-db.ps1
   ```

   What it does:
   - dumps the remote PostgreSQL database using a Dockerized `pg_dump`
   - drops and recreates your local `armadacms` database
   - imports the dump into the local Docker Postgres container

   Notes:
   - The `pg_dump` client must be the same major version as the source database, or newer. Since production is PostgreSQL 17, the default clone tooling now uses `postgres:17`.
   - The remote database still has to be reachable from your machine. If production only allows the Cloud Run NAT IP, you must temporarily allowlist your current IP or clone from staging instead.
   - The script replaces your local database completely.
   - Cloning production means copying real data locally, so handle that dump carefully and prefer staging where possible.

### Option A: Docker production-style (slow — full rebuild)

Builds the frontend and backend in one step:

```bash
docker compose up --build
```

### Option B: Docker with hot reload (recommended for Docker users)

Runs Go (Air hot-reload) and Vite (HMR) in separate containers with volume mounts — no image rebuild needed on code changes:

```bash
docker compose -f docker-compose.dev.yml up --build
```

Only the first run requires `--build`. After that, just `docker compose -f docker-compose.dev.yml up`.

If you are using the standalone local Postgres from `docker-compose.db.yml`, the backend container will automatically connect to it through `host.docker.internal` while your regular local `.env` can keep `DB_HOST=localhost` for non-Docker runs.

If you need a different hostname for Docker-based backend development, set this in `.env`:

```bash
DB_HOST_DOCKER=host.docker.internal
```

### Option C: Run locally without Docker (fastest)

Run the Go backend and Vite frontend in **two separate terminals**:

**Terminal 1 — Go backend with hot reload:**

```bash
# Install Air (one-time)
go install github.com/air-verse/air@latest

# Start backend with auto-rebuild on .go changes
air
```

Or without Air: `go run main.go` (manual restart on changes).

**Terminal 2 — Vite frontend with HMR:**

```bash
cd frontend
npm install
npm run dev
```

Once running (any option), the server is available at:

- **API**: [http://localhost:8080/api/v1/](http://localhost:8080/api/v1/)
- **Admin UI (production build)**: [http://localhost:8080/admin/](http://localhost:8080/admin/) _(Option A only)_
- **Admin UI (Vite dev)**: [http://localhost:5173](http://localhost:5173) _(Options B & C)_
- **Health check**: [http://localhost:8080/health](http://localhost:8080/health) _(recommended startup/liveness endpoint for Cloud Run)_

## Deploying to Cloud Run

The application is now prepared for a Cloud Run deployment while keeping the current AWS RDS database and AWS S3 file storage.

### Why this works well

- `Dockerfile.prod` already builds a single production image containing both the Go API and the React-Admin frontend
- `main.go` now respects Cloud Run's `PORT` environment variable automatically
- the admin frontend uses same-origin API requests in production, so `/admin/` and `/api/v1` can stay on the same Cloud Run service
- `/health` is a lightweight endpoint that is suitable for smoke checks after deployment

### Required production environment variables

At minimum, configure these in Cloud Run:

- `DB_HOST`
- `DB_PORT`
- `DB_USER`
- `DB_PASSWORD`
- `DB_NAME`
- `DB_SSLMODE=require`
- `jwtsecret_laganda`
- `S3_BUCKET`
- `AWS_REGION`
- `EVENTRO_API`
- `EVENTRO_FAIR_ID`
- `EVENTRO_ORG`

Optional but recommended for Cloud Run:

- `DB_MAX_OPEN_CONNS`
- `DB_MAX_IDLE_CONNS`
- `DB_CONN_MAX_LIFETIME_MINUTES`
- `DB_CONN_MAX_IDLE_TIME_MINUTES`

If you are not using workload-based AWS credentials, also set:

- `AWS_ACCESS_KEY_ID`
- `AWS_SECRET_ACCESS_KEY`

### Recommended first Cloud Run settings

Use conservative settings first and tune later based on real traffic:

- **CPU**: `1`
- **Memory**: `512Mi` or `1Gi`
- **Min instances**: `0`
- **Max instances**: `2`
- **Concurrency**: `10`
- **Timeout**: `120s`

These settings help prevent your application from opening too many PostgreSQL connections if Cloud Run scales up.

### Deployment approach

You can deploy from the Google Cloud console directly by connecting the GitHub repository to Cloud Run, which is a good fit if you want the built-in auto-deploy flow instead of maintaining a custom CI workflow.

Suggested deployment path:

1. Connect the repository in the Cloud Run console.
2. Point the build at the `ArmadaCMS/` directory.
3. Use `Dockerfile.prod` as the production container build.
4. Configure the environment variables and secrets listed above.
5. Verify the deployed service with `GET /health` before switching production traffic.

For a longer step-by-step reference, see:

- [`docs/cloud-run-migration-plan.md`](./docs/cloud-run-migration-plan.md)

## Project Structure

```text
ArmadaCMS/
├── main.go               # Entry point — routing, auto-migration, server startup
├── auth/
│   └── middleware.go      # JWT Bearer token auth middleware
├── Controllers/           # HTTP handlers (one per resource)
├── models/                # GORM model structs
├── db/
│   └── connect.go         # Postgres connection setup
├── utils/                 # Helpers (S3 upload, JWT, password hashing)
├── frontend/              # React-Admin SPA (Vite)
│   └── src/
│       ├── App.tsx            # Resource registrations
│       ├── dataProvider.ts    # Custom ra-data-simple-rest provider
│       ├── components/        # List, Create, Edit per resource
│       └── context/           # Auth provider, API endpoint config
├── Dockerfile.dev         # Lightweight dev image (Go + Air only)
├── Dockerfile.prod        # Production multi-stage build
├── docker-compose.yml     # Docker Compose (production-style)
└── docker-compose.dev.yml # Docker Compose (hot-reload dev)
```

## API

All endpoints are under `/api/v1`. Routes are split into:

- **Public** (no auth): `GET` endpoints for resources like exhibitors, events, profiles, teams, dates.
- **Protected** (Bearer JWT): `POST`, `PUT`, `DELETE` and admin-only `GET` endpoints.

### Example endpoints

| Method   | Endpoint                  | Auth     | Description         |
| -------- | ------------------------- | -------- | ------------------- |
| `POST`   | `/api/v1/login`           | No       | Get JWT tokens      |
| `GET`    | `/api/v1/exhibitors`      | No       | List all exhibitors |
| `POST`   | `/api/v1/exhibitors`      | Required | Create exhibitor    |
| `PUT`    | `/api/v1/exhibitors/{id}` | Required | Update exhibitor    |
| `DELETE` | `/api/v1/exhibitors/{id}` | Required | Delete exhibitor    |
| `GET`    | `/api/v1/dates`           | No       | Get fair dates      |
| `GET`    | `/health`                 | No       | Health check        |

## Adding a New Resource

1. Create a model in `models/` with GORM struct tags and camelCase JSON tags.
2. Register the model in `db.DB.AutoMigrate(...)` in `main.go`.
3. Create a controller in `Controllers/` following existing CRUD patterns.
4. Add routes in `main.go` (public for reads, protected for writes).
5. Create `List`, `Create`, `Edit` components in `frontend/src/components/{Resource}/`.
6. Register the `<Resource>` in `frontend/src/App.tsx`.
7. If the resource has file uploads, add it to the multipart list in `frontend/src/dataProvider.ts`.
