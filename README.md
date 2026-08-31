# ArmadaCMS

[![Go checks](https://github.com/armada-ths/ArmadaCMS/actions/workflows/go-checks.yml/badge.svg)](https://github.com/armada-ths/ArmadaCMS/actions/workflows/go-checks.yml)
[![Frontend checks](https://github.com/armada-ths/ArmadaCMS/actions/workflows/frontend-checks.yml/badge.svg)](https://github.com/armada-ths/ArmadaCMS/actions/workflows/frontend-checks.yml)
[![Supabase checks](https://github.com/armada-ths/ArmadaCMS/actions/workflows/supabase-checks.yml/badge.svg)](https://github.com/armada-ths/ArmadaCMS/actions/workflows/supabase-checks.yml)
[![Production deployment](https://img.shields.io/github/deployments/armada-ths/ArmadaCMS/Production?label=production&logo=googlecloud)](https://github.com/armada-ths/ArmadaCMS/deployments/Production)
[![API status](https://img.shields.io/website?url=https%3A%2F%2Fcms.armada.nu%2Fhealth&label=CMS%20API)](https://cms.armada.nu/health)
[![OpenSSF Scorecard](https://api.securityscorecards.dev/projects/github.com/armada-ths/ArmadaCMS/badge)](https://securityscorecards.dev/viewer/?uri=github.com/armada-ths/ArmadaCMS)
[![Last commit](https://img.shields.io/github/last-commit/armada-ths/ArmadaCMS)](https://github.com/armada-ths/ArmadaCMS/commits)
[![Open issues](https://img.shields.io/github/issues/armada-ths/ArmadaCMS)](https://github.com/armada-ths/ArmadaCMS/issues)
[![License: MIT](https://img.shields.io/github/license/armada-ths/ArmadaCMS)](https://github.com/armada-ths/ArmadaCMS/blob/main/license.txt)
[![Go 1.26](https://img.shields.io/badge/Go-1.26-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![React 19](https://img.shields.io/badge/React-19-61DAFB?logo=react&logoColor=black)](https://react.dev/)
[![React Admin 5](https://img.shields.io/badge/React_Admin-5-22A6F2?logo=react&logoColor=white)](https://marmelab.com/react-admin/)
[![TypeScript 5.9](https://img.shields.io/badge/TypeScript-5.9-3178C6?logo=typescript&logoColor=white)](https://www.typescriptlang.org/)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-4169E1?logo=postgresql&logoColor=white)](https://www.postgresql.org/)

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

- **Language**: Go 1.26
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
- **Ingress**: External HTTPS load balancing in production; Cloud Run domain mapping in staging
- **Database**: Supabase PostgreSQL (production project with a persistent staging branch)
- **File storage**: Supabase Storage (local dev: MinIO)

## Prerequisites

- [Docker](https://www.docker.com/) and Docker Compose _(required for local development)_
- [Go 1.26+](https://go.dev/dl/) _(optional, for running Go tooling directly)_
- [Node.js 24+](https://nodejs.org/) and pnpm _(optional, for running frontend tooling directly)_

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

   The defaults are pre-configured for the local Docker stack. See `.env.example` for the full list of variables.

3. **Start the local development stack**

   ```bash
   docker compose -f docker-compose.dev.yml up --build
   ```

   This starts the Go API (Air hot reload), the React-Admin frontend (Vite HMR), Postgres, and MinIO in one Docker Compose workflow.

   Only the first run requires `--build`. After that, use:

   ```bash
   docker compose -f docker-compose.dev.yml up
   ```

   Local connection (e.g. for a DB GUI): `postgres:postgres@localhost:5432/armadacms`

   MinIO console: [http://localhost:9001](http://localhost:9001) (login: `minioadmin` / `minioadmin`)

   To stop the stack without deleting data:

   ```bash
   docker compose -f docker-compose.dev.yml stop
   ```

   To remove the containers while keeping named volumes available for reuse:

   ```bash
   docker compose -f docker-compose.dev.yml down
   ```

4. **Optionally clone a remote database**

   `scripts/import-remote-db.ps1` clones a remote PostgreSQL database into the local Postgres container, replacing the local `armadacms` database. Fill in the `SOURCE_DB_*` vars in `.env` (see `.env.example`), then run:

   ```powershell
   ./scripts/import-remote-db.ps1
   ```

   The remote database must be reachable from your machine — for Supabase, allowlist your IP under **Project Settings → Networking → Network restrictions**. Prefer cloning staging over production to avoid handling real data locally.

   After cloning a Supabase database, AutoMigrate needs to be disabled in order to avoid schema conflicts. Set `DB_ENABLE_AUTOMIGRATE=false` in `.env` before starting the server. To apply a local SQL migration file manually, run `cat supabase/migrations/<migration-file>.sql | docker compose -f docker-compose.dev.yml exec -T postgres psql -U postgres -d armadacms`.

5. **Verify the app is running**

   Once the development stack is running, the following URLs are available:
   - **API**: [http://localhost:8080/api/v1/](http://localhost:8080/api/v1/)
   - **Admin UI**: [http://localhost:5173](http://localhost:5173)
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

To validate that all migrations apply cleanly from scratch:

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

ArmadaCMS includes Go unit tests (currently focused on `auth/` and `utils/`) and frontend unit tests using [vitest](https://vitest.dev/) (currently focused on `utils/`).

- Run all Go tests locally:

  ```bash
  go test -race -count=1 ./...
  ```

- Run tests for specific Go packages:

  ```bash
  go test ./auth/... ./utils/...
  ```

- Run frontend unit tests:

  ```bash
  cd frontend && pnpm run test
  ```

## API

All endpoints are under `/api/v1`. Routes are split into:

- **Public** (no auth): `GET` endpoints for resources like exhibitors, events, profiles, teams, dates.
- **Protected** (Bearer JWT): `POST`, `PUT`, `DELETE` and admin-only `GET` endpoints.

Browser cross-origin access is restricted to `armada.nu`, common local development
origins, and any exact origins listed in the optional comma-separated
`CORS_ALLOWED_ORIGINS` environment variable.

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

CI is handled by GitHub Actions and CD by Google Cloud Build.

### GitHub Actions (CI)

Repository checks live in `.github/workflows/` and are path-filtered so unchanged areas are skipped cleanly:

- `go-checks.yml` — for Go files, `go.mod`, `go.sum`, and workflow changes; runs `go vet ./...`, `golangci-lint run`, and `go test -race -count=1 ./...`.
- `frontend-checks.yml` — for `frontend/**` and workflow changes; in `frontend/`, runs `pnpm install --frozen-lockfile`, `pnpm run lint:check`, `pnpm run type-check`, `pnpm run format:check`, and `pnpm run test`.
- `supabase-checks.yml` — for `supabase/**` and workflow changes; starts the local Supabase stack, runs `supabase db reset --local`, and verifies migrations apply cleanly.

All three workflows run on pushes to `main` and `staging` for matching paths, and on pull requests. Each workflow ends with an aggregate status job so checks pass when work is intentionally skipped because no relevant files changed. Superseded runs for the same workflow and branch or pull request are cancelled automatically, and every job has a timeout.

### Google Cloud Build (CD)

Deployments are handled by Google Cloud Build using [`cloudbuild.yaml`](cloudbuild.yaml).

- Cloud Build builds the production container from `Dockerfile.prod` and pushes images to Artifact Registry.
- Branch pushes to `main` and `staging` deploy the resulting image to the corresponding Cloud Run service.
- PR builds use the secret-free `cloudbuild-pr.yaml` configuration with a dedicated unprivileged service account. They validate the container build without publishing or deploying an image; external contributors require an owner or collaborator to comment `/gcbrun` first.
- Trusted branch builds always build the commit SHA, publish that image, and deploy it.
- The pipeline creates and updates GitHub deployment statuses via the configured GitHub App credentials.

The GitHub → Cloud Build trigger wiring is managed in this repository's Terraform configuration, primarily in [`infra/terraform/gcp/prod/cloud_build.tf`](infra/terraform/gcp/prod/cloud_build.tf) and [`infra/terraform/gcp/staging/cloud_build.tf`](infra/terraform/gcp/staging/cloud_build.tf). Those roots provision the branch and PR triggers; `cloudbuild.yaml` defines the trusted branch build/deploy flow and `cloudbuild-pr.yaml` defines unprivileged PR validation.

## Operations notes

- Production runs on Cloud Run with Supabase (PostgreSQL) and Supabase Storage for file uploads.
- The server reads `PORT` from the environment and falls back to `8080`.
- Production database connections use `DB_SSLMODE=require`.
- Tune `DB_MAX_OPEN_CONNS`, `DB_MAX_IDLE_CONNS`, and Cloud Run max instances together to stay within Postgres connection limits.

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
3. Write a SQL migration in `supabase/migrations/` for the schema change.
4. Create a controller in `Controllers/` following existing CRUD patterns.
5. Add routes in `main.go` (public for reads, protected for writes).
6. Create `List`, `Create`, `Edit` components in `frontend/src/components/{Resource}/`.
7. Register the `<Resource>` in `frontend/src/App.tsx`.
8. If the resource has file uploads, add it to the multipart list in `frontend/src/dataProvider.ts`.
9. If the resource is displayed on the public site, pass the matching cache tag to the audit helper's `revalidateTags` argument (e.g. `"blog-posts"`) and ensure the same tag is used in the Next.js data hook on `armada.nu`.
