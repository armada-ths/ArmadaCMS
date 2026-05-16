# ArmadaCMS

Backend API and admin dashboard for [THS Armada](https://armada.nu). Provides REST endpoints consumed by the public website ([armada.nu](https://github.com/armada-ths/armada.nu)) and a React-Admin interface for content management.

## Table of Contents

- [Tech Stack](#tech-stack)
- [Prerequisites](#prerequisites)
- [Getting Started](#getting-started)
- [Database migrations](#database-migrations)
- [VS Code workspace and launches](#vs-code-workspace-and-launches)
- [Project Structure](#project-structure)
- [Testing](#testing)
- [API](#api)
- [Swagger docs](#swagger-docs)
- [Cache Revalidation](#cache-revalidation)
- [CI / CD](#ci--cd)
- [Operations notes](#operations-notes)
- [Infrastructure as code](#infrastructure-as-code)
- [Adding a New Resource](#adding-a-new-resource)

## Tech Stack

### Backend

- **Language**: Go 1.24
- **Router**: [Gorilla Mux](https://github.com/gorilla/mux)
- **ORM**: [GORM](https://gorm.io/) (Postgres)
- **Auth**: JWT (Bearer tokens)
- **Hot reload**: [Air](https://github.com/air-verse/air) (in Docker dev mode)

### Admin Frontend

- **Framework**: [React-Admin v5](https://marmelab.com/react-admin/)
- **Build tool**: [Vite](https://vitejs.dev/)
- **Language**: TypeScript
- **UI**: MUI (Material UI)

### Infrastructure

- **Deployment**: Google Cloud Run (containerized Go API + bundled React-Admin frontend)
- **Ingress**: HTTPS load balancing in front of Cloud Run
- **Database**: Supabase (PostgreSQL)
- **File storage**: Supabase Storage (local dev: MinIO)

## Prerequisites

- [Docker](https://www.docker.com/) and Docker Compose _(required for local development)_
- [Go 1.24+](https://go.dev/dl/) _(optional, for running Go tooling directly)_
- [Node.js 20+](https://nodejs.org/) and npm _(optional, for running frontend tooling directly)_

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

   For the Docker development stack, the defaults in `.env.example` already point to the bundled Postgres and MinIO services:

   ```env
   DB_HOST=postgres
   DB_PORT=5432
   DB_USER=postgres
   DB_PASSWORD=postgres
   DB_NAME=armadacms
   DB_SSLMODE=disable
   ```

   See `.env.example` for the full list of variables, including MinIO and Supabase Storage settings.

3. **Start the local development stack**

   ```bash
   docker compose -f docker-compose.dev.yml up --build
   ```

   This starts the Go API (Air hot reload), the React-Admin frontend (Vite HMR), Postgres, and MinIO in one Docker Compose workflow.

   Only the first run requires `--build`. After that, use:

   ```bash
   docker compose -f docker-compose.dev.yml up
   ```

   Postgres defaults:
   - Host (from containers): `postgres`
   - Host (from your machine): `localhost`
   - Port: `5432`
   - Database: `armadacms`
   - User: `postgres`
   - Password: `postgres`

   MinIO defaults:
   - API: `http://localhost:9000`
   - Console: `http://localhost:9001`
   - Login: `minioadmin` / `minioadmin`

   To stop the stack later without deleting data:

   ```bash
   docker compose -f docker-compose.dev.yml stop
   ```

   To remove the containers while keeping named volumes available for reuse:

   ```bash
   docker compose -f docker-compose.dev.yml down
   ```

4. **Optionally clone a remote database into your local Postgres**

   If you want realistic local data, the repo includes a PowerShell import script that can clone any reachable PostgreSQL database, for example staging or a temporarily allowlisted production instance.

   Start the Postgres container from the dev stack first if it is not already running:

   ```bash
   docker compose -f docker-compose.dev.yml up -d postgres
   ```

   Add these values to your local `.env` first:

   ```env
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

   The script:
   - dumps the remote PostgreSQL database using a Dockerized `pg_dump`
   - drops and recreates your local `armadacms` database
   - imports the dump into the local Docker Postgres container

   Notes:
   - The `pg_dump` client must be the same major version as the source database, or newer. Since both staging and production use PostgreSQL 17 (via Supabase), the default clone tooling uses `postgres:17`.
   - The remote database has to be reachable from your machine. For Supabase, you can temporarily allowlist your current IP under **Project Settings → Networking → Network restrictions** in the Supabase dashboard.
   - The script replaces your local database completely.
   - Cloning production means copying real data locally, so handle that dump carefully and prefer staging where possible.

5. **Optionally verify the production-style container locally**

   This is slower than the development stack above, but closer to what runs in production:

   ```bash
   docker compose up --build
   ```

   The Go app is served on `http://localhost:8080`, with the admin frontend bundled at `/admin/`.

   This workflow is mainly for production verification, not day-to-day local development. If you want it to talk to locally started Postgres and MinIO on Docker Desktop, use `host.docker.internal` instead of `localhost`, for example:

   ```env
   DB_HOST=host.docker.internal
   S3_ENDPOINT=http://host.docker.internal:9000
   S3_PUBLIC_URL=http://localhost:9000
   ```

6. **Set up file uploads**

   File uploads (profile photos, exhibitor logos, event images) use an S3-compatible storage service configured through generic `S3_*` env vars. Both MinIO (local dev) and Supabase Storage (staging/production) use the same vars.
   - The Docker development stack starts MinIO automatically with the defaults already in `.env.example`.
   - MinIO listens on port `9000`, and the console is available at [http://localhost:9001](http://localhost:9001) with login `minioadmin` / `minioadmin`.

   For Docker-based workflows, make sure your `.env` has the MinIO block active (it is enabled by default in `.env.example`):

   ```env
   S3_BUCKET=armada-dev
   S3_ENDPOINT=http://minio:9000
   S3_PUBLIC_URL=http://localhost:9000
   AWS_ACCESS_KEY_ID=minioadmin
   AWS_SECRET_ACCESS_KEY=minioadmin
   ```

   `S3_ENDPOINT` is the address the Go server uses to reach MinIO. `S3_PUBLIC_URL` is the address the browser uses to load uploaded files. In Docker-based development, those values differ because the backend reaches MinIO at `minio:9000` while the browser uses `localhost:9000`.

   For staging/production (Supabase Storage), Terraform injects the same `S3_*` vars pointing at Supabase's S3-compatible endpoint.

7. **Verify the app is running**

   Once the development stack is running, the following URLs are available:
   - **API**: [http://localhost:8080/api/v1/](http://localhost:8080/api/v1/)
   - **Admin UI (Vite dev)**: [http://localhost:5173](http://localhost:5173)
   - **Admin UI (production build)**: [http://localhost:8080/admin/](http://localhost:8080/admin/) _(production-style local verification only)_
   - **Health check**: [http://localhost:8080/health](http://localhost:8080/health)

## Database migrations

**Local development** uses GORM AutoMigrate, which runs automatically on every server startup (controlled by `DB_ENABLE_AUTOMIGRATE`, default `true`). No extra steps are needed.

**Remote environments (staging, production)** use checked-in SQL migrations. The Supabase project is connected to this GitHub repository, so migrations are applied automatically on every push/merge to the tracked branches — no manual CLI commands required.

- `supabase/config.toml` configures the Supabase project link.
- `supabase/seed.sql` bootstraps deterministic roles and feature flags for remote environment resets.
- `supabase/migrations/` holds all checked-in SQL migrations.

To create a new migration, generate a diff against the current remote schema:

```bash
pnpx supabase db diff -f <migration-name>
```

You can also validate that all migrations apply cleanly from scratch:

```bash
pnpx supabase db reset
```

Roles and feature flags are seeded from `supabase/seed.sql` alongside the initial admin user (username and password: `admin`). This seed only runs in non-production contexts (new branches, `db reset`) so hardcoded credentials are acceptable.

## VS Code workspace and launches

This repo includes shared VS Code configuration in `.vscode/`:

- `tasks.json` — shared Docker tasks for `docker dev up`, `docker dev up --build`, `docker dev stop`, and `docker dev down`
- `launch.json` — a `Docker` launch that starts the dev stack via the shared task and opens the admin UI

If you work across both repos, use the shared workspace file committed in `armada.nu`:

- `../armada.nu/Armada.code-workspace`

That workspace opens both repositories with portable relative paths and includes multi-repo compound launches.

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
├── infra/
│   └── terraform/         # Terraform layout, shared conventions, and provider-specific roots
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

## Testing

ArmadaCMS includes Go unit tests (currently focused on `auth/` and `utils/`).

- Run all tests locally:

  ```bash
  go test -race -count=1 ./...
  ```

- Run tests for specific packages:

  ```bash
  go test ./auth/... ./utils/...
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

## Swagger docs

The API is documented with [Swagger / OpenAPI 2.0](https://swagger.io/) using [swaggo/swag](https://github.com/swaggo/swag).

**Swagger UI** is served at `/swagger/index.html`:

| Environment | URL                                                |
| ----------- | -------------------------------------------------- |
| Local dev   | <http://localhost:8080/swagger/index.html>         |
| Staging     | <https://staging.cms.armada.nu/swagger/index.html> |
| Production  | <https://cms.armada.nu/swagger/index.html>         |

Click **Authorize** in the UI and enter `Bearer <token>` (token obtained from `POST /api/v1/login`) to test protected endpoints.

### Regenerating the spec

Run this command from the repo root whenever you add or change routes or annotations:

```bash
swag init --generalInfo main.go --output docs --parseInternal
```

This overwrites `docs/docs.go`, `docs/swagger.json`, and `docs/swagger.yaml`. Commit these generated files alongside your code changes.

> **Prerequisites:** install the `swag` CLI once with `go install github.com/swaggo/swag/cmd/swag@latest`.

## Cache Revalidation

Write operations automatically purge the public site's ISR cache via `utils.RevalidateTag(tag)`, which POSTs to armada.nu's `/api/revalidate` endpoint. Tags are passed as the trailing `revalidateTags ...string` argument to the audit helpers. Requires `REVALIDATION_URL` and `REVALIDATION_SECRET` env vars (silently skipped if unset). See [`armada.nu/.github/copilot-instructions.md`](https://github.com/armada-ths/armada.nu/blob/main/.github/copilot-instructions.md) for the full tag inventory.

## CI / CD

CI is split between GitHub Actions for repository checks and Google Cloud Build for container build/deploy automation.

### GitHub Actions (CI)

Repository checks live in `.github/workflows/` and are path-filtered so unchanged areas are skipped cleanly:

- `go-checks.yml` — for Go files, `go.mod`, `go.sum`, and workflow changes; runs `go vet ./...`, `golangci-lint run`, and `go test -race -count=1 ./...`.
- `frontend-checks.yml` — for `frontend/**` and workflow changes; in `frontend/`, runs `npm ci`, `npm run lint:check`, `npm run type-check`, and `npm run format:check`.
- `supabase-checks.yml` — for `supabase/**` and workflow changes; starts the local Supabase stack, runs `supabase db reset --local`, and verifies migrations apply cleanly.

All three workflows run on pushes to `main` and `staging` for matching paths, and on pull requests. Each workflow ends with an aggregate status job so checks pass when work is intentionally skipped because no relevant files changed.

### Google Cloud Build / Cloud Run (CD)

Deployments are handled by Google Cloud Build using [`cloudbuild.yaml`](cloudbuild.yaml), not by GitHub Actions.

- Cloud Build builds the production container from `Dockerfile.prod`.
- Images are pushed to Artifact Registry.
- Non-PR builds can deploy the resulting image to the configured Cloud Run service.
- PR builds can build and push preview-tagged images without deploying them.
- For merged changes, the pipeline can reuse an already-built PR image when available instead of rebuilding from scratch.
- When GitHub App credentials are configured in the Cloud Build trigger environment, the pipeline also creates and updates GitHub deployment statuses.

The GitHub → Cloud Build trigger wiring is managed in this repository's Terraform configuration, primarily in [`infra/terraform/gcp/prod/cloud_build.tf`](infra/terraform/gcp/prod/cloud_build.tf) and [`infra/terraform/gcp/staging/cloud_build.tf`](infra/terraform/gcp/staging/cloud_build.tf). Those roots provision the branch and PR triggers, while `cloudbuild.yaml` remains the source of truth for the build, image-promotion, and deployment steps the triggers execute.

## Operations notes

- Production traffic is served through Cloud Run. PostgreSQL uses Supabase; file uploads use Supabase Storage.
- The backend expects Cloud Run to provide `PORT` in production and falls back to `8080` locally.
- Production database connections should use `DB_SSLMODE=require`.
- Cloud Run instance scaling should stay aligned with PostgreSQL connection limits; tune `DB_MAX_OPEN_CONNS`, `DB_MAX_IDLE_CONNS`, and Cloud Run max instances together.

## Infrastructure as code

Terraform documentation is split by scope:

- [`infra/terraform/README.md`](infra/terraform/README.md) — shared layout, conventions, workspace naming, and cross-workspace wiring
- [`infra/terraform/gcp/prod/README.md`](infra/terraform/gcp/prod/README.md) — GCP production root details and workspace setup
- [`infra/terraform/gcp/staging/README.md`](infra/terraform/gcp/staging/README.md) — GCP staging root details and workspace setup
- [`infra/terraform/supabase/prod/README.md`](infra/terraform/supabase/prod/README.md) — Supabase production root details and workspace setup

Use those documents as the canonical source for infrastructure specifics rather than duplicating them here.

## Adding a New Resource

1. Create a model in `models/` with GORM struct tags and camelCase JSON tags.
2. Register the model in `db.DB.AutoMigrate(...)` in `main.go`.
3. Create a controller in `Controllers/` following existing CRUD patterns.
4. Add routes in `main.go` (public for reads, protected for writes).
5. Create `List`, `Create`, `Edit` components in `frontend/src/components/{Resource}/`.
6. Register the `<Resource>` in `frontend/src/App.tsx`.
7. If the resource has file uploads, add it to the multipart list in `frontend/src/dataProvider.ts`.
8. If the resource is displayed on the public site, pass the matching cache tag to the audit helper's `revalidateTags` argument (e.g. `"blog-posts"`) and ensure the same tag is used in the Next.js data hook on `armada.nu`.
