# ArmadaCMS

Backend API and admin dashboard for [THS Armada](https://armada.nu). Provides REST endpoints consumed by the public website ([armada.nu](https://github.com/armada-ths/armada.nu)) and a React-Admin interface for content management.

## Table of Contents

- [Tech Stack](#tech-stack)
- [Prerequisites](#prerequisites)
- [Getting Started](#getting-started)
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
- **File storage**: provider-neutral upload service backed by MinIO/AWS S3 today, with Supabase Storage support via the S3-compatible endpoint
- **Hot reload**: [Air](https://github.com/air-verse/air) (in Docker dev mode)

### Admin Frontend

- **Framework**: [React-Admin v5](https://marmelab.com/react-admin/)
- **Build tool**: [Vite](https://vitejs.dev/)
- **Language**: TypeScript
- **UI**: MUI (Material UI)

### Infrastructure

- **Deployment**: Google Cloud Run (containerized Go API + bundled React-Admin frontend)
- **Ingress**: HTTPS load balancing in front of Cloud Run
- **Database**: PostgreSQL — migrating from AWS RDS to Supabase (rehearsal complete; cutover pending)
- **File storage**: AWS S3 (local dev: MinIO) — storage migration pending

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

   See `.env.example` for the full list of variables, including MinIO and AWS S3 settings.

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
   - The `pg_dump` client must be the same major version as the source database, or newer. Since production is PostgreSQL 17, the default clone tooling uses `postgres:17`.
   - The remote database has to be reachable from your machine. Since production only allows the Cloud Run NAT IP, you must temporarily allowlist your current IP or clone from staging instead.
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

6. **Set up file uploads (MinIO, AWS S3, or Supabase Storage)**

   File uploads (profile photos, exhibitor logos, event images) now go through a provider-neutral storage service. **For local development, MinIO is the permanent default and will not be replaced.**
   - The Docker development stack already starts MinIO automatically and uses the `.env.example` default `S3_ENDPOINT=http://minio:9000`.
   - The production-style local verification flow also needs container-reachable endpoints such as `host.docker.internal` if you want to use locally started services.
   - Supabase Storage is also supported for backend uploads through its S3-compatible endpoint once you generate storage access keys and configure the `SUPABASE_STORAGE_*` variables.

   MinIO listens on port `9000`, and the console is available at [http://localhost:9001](http://localhost:9001) with login `minioadmin` / `minioadmin`.

   For Docker-based workflows, make sure your `.env` has the MinIO block active (it is enabled by default in `.env.example`):

   ```env
   S3_BUCKET=armada-dev
   S3_ENDPOINT=http://minio:9000
   S3_PUBLIC_URL=http://localhost:9000
   AWS_ACCESS_KEY_ID=minioadmin
   AWS_SECRET_ACCESS_KEY=minioadmin
   ```

   `S3_ENDPOINT` is the address the Go server uses to reach MinIO. `S3_PUBLIC_URL` is the address the browser uses to load uploaded files. In Docker-based development, those values differ because the backend reaches MinIO at `minio:9000` while the browser uses `localhost:9000`.

   If you want to use AWS S3 instead, keep `STORAGE_PROVIDER=s3`, comment out the MinIO block in `.env`, and enable the AWS S3 settings from `.env.example`.

   If you want to use Supabase Storage for backend uploads, set `STORAGE_PROVIDER=supabase` and configure:

   ```env
   SUPABASE_URL=https://<project-ref>.supabase.co
   SUPABASE_STORAGE_S3_ENDPOINT=https://<project-ref>.storage.supabase.co/storage/v1/s3
   SUPABASE_STORAGE_BUCKET=armadacms-files
   SUPABASE_STORAGE_REGION=eu-north-1
   SUPABASE_STORAGE_ACCESS_KEY_ID=
   SUPABASE_STORAGE_SECRET_ACCESS_KEY=
   ```

   The backend will then upload through the S3-compatible endpoint (prefer the direct `*.storage.supabase.co` hostname for performance) and build public URLs using `https://<project-ref>.supabase.co/storage/v1/object/public/...`.

7. **Verify the app is running**

   Once the development stack is running, the following URLs are available:
   - **API**: [http://localhost:8080/api/v1/](http://localhost:8080/api/v1/)
   - **Admin UI (Vite dev)**: [http://localhost:5173](http://localhost:5173)
   - **Admin UI (production build)**: [http://localhost:8080/admin/](http://localhost:8080/admin/) _(production-style local verification only)_
   - **Health check**: [http://localhost:8080/health](http://localhost:8080/health)

## Supabase migration groundwork

The repository includes a `supabase/` scaffold for the AWS → Supabase migration. **Local development continues to use Docker Compose (Postgres + MinIO) and is not changing.** The Supabase CLI is used only as a migration management tool — for authoring, validating, and pushing schema changes to remote environments (production, staging, pre-production).

- `supabase/config.toml` configures the Supabase CLI project for migration management.
- `supabase/seed.sql` bootstraps deterministic roles and feature flags for remote environment resets and CI.
- `supabase/migrations/` holds all checked-in SQL migrations applied to remote Supabase environments.
- `docs/supabase-migration-inventory.md` is the operator checklist for the migration phases.
- `docs/supabase-app-schema-inventory.md` captures the ArmadaCMS application tables and bootstrap data in the migration.

### What is intentionally deferred

- Hosted Supabase branching (including a persistent hosted `staging` branch)
- Supabase PR preview branches
- Per-PR GCP preview services

Those features will be introduced in the next phase, as one coordinated preview-environment rollout.

### Supabase CLI migration workflow

The Supabase CLI is used to manage schema migrations for remote environments. It is **not** used for local development.

1. Push pending migrations to a linked remote environment:
   - `pnpx supabase db push`
2. Validate that all migrations apply cleanly from scratch (uses a temporary local Supabase stack for verification only):
   - `pnpx supabase db reset`
3. Generate a new migration from schema changes:
   - `pnpx supabase db diff -f <migration-name>`
4. Link the CLI to a remote project:
   - `pnpx supabase link --project-ref <project-ref>`

The schema migration set is complete and validated:

- `20260504083246_remote_schema.sql` — extensions and schema grants baseline
- `20260504085048_armadacms_application_schema_snapshot.sql` — full ArmadaCMS application schema DDL
- `20260504090108_harden_public_schema_access_and_rls.sql` — RLS + privilege hardening
- `20260504092802_create_public_storage_bucket.sql` — Supabase Storage bucket
- `20260513000000_fix_schema_gaps.sql` — schema gap fix (columns/table missing from snapshot vs RDS)

A full end-to-end DB restore rehearsal was completed on 2026-05-13. Both the staging Supabase project (`yfybmnqzclpmpncyfmdc`) and the production project (`rsdjnixgxqauonaofrwr`) now contain a validated restore of the current RDS production data. See `docs/supabase-migration-inventory.md` for the validated restore procedure and expected row counts.

`DB_ENABLE_AUTOMIGRATE` is the transition switch for GORM runtime schema management. It defaults to enabled for local Docker Compose. Supabase-managed environments (staging, production) should set `DB_ENABLE_AUTOMIGRATE=false` at cutover time, since schema changes will be applied exclusively through `supabase db push`.

Bootstrap responsibilities are split intentionally: deterministic roles and feature flags are seeded from `supabase/seed.sql`, while the optional initial admin user remains in Go startup because it depends on environment variables.

The next steps before production cutover are the storage migration rehearsal (S3 → Supabase Storage) and updating the GCP Cloud Run production environment to point at Supabase.

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

GitHub Actions workflows in `.github/workflows/`:

| Workflow                 | Trigger                                                  | What it does                                                       |
| ------------------------ | -------------------------------------------------------- | ------------------------------------------------------------------ |
| `go-checks.yml`          | Push to `main`/`staging` when Go files change, PRs       | `go vet`, `golangci-lint`, `go test -race`                         |
| `frontend-checks.yml`    | Push to `main`/`staging` when frontend files change, PRs | `npm run lint:check`, `npm run type-check`, `npm run format:check` |
| `keep-staging-alive.yml` | Weekly schedule                                          | `curl` to staging `/health` to prevent Supabase free-tier pause    |

## Operations notes

- Production traffic is served through Cloud Run. PostgreSQL and file uploads currently use AWS-managed services; the cutover to Supabase DB and Supabase Storage is in progress (DB rehearsal complete, storage migration and production cutover pending).
- The backend expects Cloud Run to provide `PORT` in production and falls back to `8080` locally.
- Production database connections should use `DB_SSLMODE=require`.
- Cloud Run instance scaling should stay aligned with PostgreSQL connection limits; tune `DB_MAX_OPEN_CONNS`, `DB_MAX_IDLE_CONNS`, and Cloud Run max instances together.
- The standalone staging Supabase project (`yfybmnqzclpmpncyfmdc`) and production project (`rsdjnixgxqauonaofrwr`) both contain a validated restore of the current RDS data as of 2026-05-13.

## Infrastructure as code

Terraform documentation is split by scope:

- [`infra/terraform/README.md`](infra/terraform/README.md) — shared layout, conventions, workspace naming, and cross-workspace wiring
- [`infra/terraform/gcp/prod/README.md`](infra/terraform/gcp/prod/README.md) — GCP production root details and workspace setup
- [`infra/terraform/gcp/staging/README.md`](infra/terraform/gcp/staging/README.md) — GCP staging root details and workspace setup
- [`infra/terraform/aws/prod/README.md`](infra/terraform/aws/prod/README.md) — AWS production root details and workspace setup
- [`infra/terraform/aws/staging/README.md`](infra/terraform/aws/staging/README.md) — AWS staging root details and workspace setup

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
