# ArmadaCMS

Backend API and admin dashboard for [THS Armada](https://armada.nu). Provides REST endpoints consumed by the public website ([armada.nu](https://github.com/armada-ths/armada.nu)) and a React-Admin interface for content management.

## Table of Contents

- [Tech Stack](#tech-stack)
- [Prerequisites](#prerequisites)
- [Getting Started](#getting-started)
- [Project Structure](#project-structure)
- [API](#api)
- [Operations notes](#operations-notes)
- [Adding a New Resource](#adding-a-new-resource)

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

- **Deployment**: Google Cloud Run (containerized Go API + bundled React-Admin frontend)
- **Ingress**: HTTPS load balancing in front of Cloud Run
- **Database**: PostgreSQL (AWS RDS)
- **File storage**: AWS S3

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

   The repo includes a small standalone Postgres Compose file:

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
   SOURCE_DB_USER=
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
   - The `pg_dump` client must be the same major version as the source database, or newer. Since production is PostgreSQL 17, the default clone tooling uses `postgres:17`.
   - The remote database has to be reachable from your machine. Since production only allows the Cloud Run NAT IP, you must temporarily allowlist your current IP or clone from staging instead.
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
- **Health check**: [http://localhost:8080/health](http://localhost:8080/health)

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
├── Dockerfile.prod        # Production multi-stage build for Cloud Run
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

## Operations notes

- Production traffic is served through Cloud Run, while PostgreSQL and file uploads remain on AWS-managed services.
- The backend expects Cloud Run to provide `PORT` in production and falls back to `8080` locally.
- Production database connections should use `DB_SSLMODE=require`.
- Cloud Run instance scaling should stay aligned with PostgreSQL connection limits; tune `DB_MAX_OPEN_CONNS`, `DB_MAX_IDLE_CONNS`, and Cloud Run max instances together.

## Adding a New Resource

1. Create a model in `models/` with GORM struct tags and camelCase JSON tags.
2. Register the model in `db.DB.AutoMigrate(...)` in `main.go`.
3. Create a controller in `Controllers/` following existing CRUD patterns.
4. Add routes in `main.go` (public for reads, protected for writes).
5. Create `List`, `Create`, `Edit` components in `frontend/src/components/{Resource}/`.
6. Register the `<Resource>` in `frontend/src/App.tsx`.
7. If the resource has file uploads, add it to the multipart list in `frontend/src/dataProvider.ts`.
