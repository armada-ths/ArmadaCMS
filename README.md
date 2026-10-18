# ArmadaCMS

[![Go checks](https://github.com/armada-ths/ArmadaCMS/actions/workflows/go-checks.yml/badge.svg)](https://github.com/armada-ths/ArmadaCMS/actions/workflows/go-checks.yml)
[![Frontend checks](https://github.com/armada-ths/ArmadaCMS/actions/workflows/frontend-checks.yml/badge.svg)](https://github.com/armada-ths/ArmadaCMS/actions/workflows/frontend-checks.yml)
[![Supabase checks](https://github.com/armada-ths/ArmadaCMS/actions/workflows/supabase-checks.yml/badge.svg)](https://github.com/armada-ths/ArmadaCMS/actions/workflows/supabase-checks.yml)
[![Production deployment](https://img.shields.io/github/deployments/armada-ths/ArmadaCMS/Production?label=production&logo=googlecloud)](https://github.com/armada-ths/ArmadaCMS/deployments/Production)
[![API status](https://img.shields.io/website?url=https%3A%2F%2Fcms.armada.nu%2Fhealth&label=CMS%20API)](https://cms.armada.nu/health)
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
- **File storage**: Supabase Storage in local development, staging, and production

## Prerequisites

- [Docker](https://www.docker.com/) and Docker Compose _(required for local development)_
- [Go 1.26+](https://go.dev/dl/) _(optional, for running Go tooling directly)_
- [Node.js 24+](https://nodejs.org/) and pnpm _(required for the pinned local Supabase CLI and frontend tooling)_

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

   The defaults are pre-configured for local Supabase and Docker. See `.env.example` for the full list of variables.

3. **Start the local development stack**

   ```powershell
   ./scripts/dev-up.ps1 -Build
   ```

   This starts a minimal local Supabase profile (Postgres, Storage API, and its Kong gateway) followed by the Go API with Air hot reload. The React-Admin frontend runs separately on the host so Vite can provide reliable HMR with Windows file watching.

   The checked-in migrations also create the guest photo buckets. The local Storage limit permits multi-gigabyte ZIP exports, while the photo bucket still limits individual JPEGs to 25 MB. For local photo testing, Compose supplies a development-only event-token secret, a browser-reachable signed-photo endpoint, a local reCAPTCHA bypass, and a SafeSearch mock. Hosted environments require their real secrets and Google assessments; see [the photo rollout guide](docs/photo-events-rollout.md).

   Only the first run requires `--build`. After that, use:

   ```powershell
   ./scripts/dev-up.ps1
   ```

   In a second terminal, start the admin frontend:

   ```powershell
   cd frontend
   pnpm install
   pnpm dev
   ```

   Local database connection (e.g. for a DB GUI): `postgres:postgres@localhost:54322/postgres`

   To stop the backend and Supabase while preserving local Supabase data:

   ```powershell
   ./scripts/dev-down.ps1
   ```

4. **Optionally clone a remote database**

   `scripts/import-remote-db.ps1` replaces only the `public` schema in local Supabase Postgres, preserving Supabase's system and Storage schemas. Fill in the `SOURCE_DB_*` vars in `.env` (see `.env.example`), start the local stack, then run:

   ```powershell
   ./scripts/import-remote-db.ps1
   ```

   The remote database must be reachable from your machine — for Supabase, allowlist your IP under **Project Settings → Networking → Network restrictions**. Prefer cloning staging over production to avoid handling real data locally.

   Use checked-in Supabase migrations for schema changes; an imported schema is disposable local data and can be replaced by `pnpx supabase@2.118.0 db reset --local`.

5. **Verify the app is running**

   Once the development stack is running, the following URLs are available:
   - **API**: [http://localhost:8080/api/v1/](http://localhost:8080/api/v1/)
   - **Admin UI**: [http://localhost:5173/admin](http://localhost:5173/admin) (host-run Vite frontend)
   - **Health check**: [http://localhost:8080/health](http://localhost:8080/health)

## Database migrations

**Local development, staging, and production** use checked-in SQL migrations. The hosted Supabase project is connected to this GitHub repository, so remote migrations are applied automatically on every push/merge to the tracked branches — no manual CLI commands required.

- `supabase/config.toml` configures the Supabase project link.
- `supabase/seed.sql` bootstraps deterministic roles and feature flags for remote environment resets.
- `supabase/migrations/` holds all checked-in SQL migrations.

To create a new migration, generate a diff against the current remote schema:

```bash
pnpx supabase@2.118.0 db diff -f <migration-name>
```

To validate that all migrations apply cleanly from scratch:

```bash
pnpx supabase@2.118.0 db reset --local
```

Roles and feature flags are seeded from `supabase/seed.sql` alongside the initial admin user (username and password: `admin`). This seed only runs in non-production contexts (new branches, `db reset`) so hardcoded credentials are acceptable.

## VS Code workspace and launches

This repo includes shared VS Code configuration in `.vscode/`:

- `tasks.json` — tasks for starting the backend with local Supabase, rebuilding the backend before start, starting the complete admin development environment, and stopping either the services or the complete local environment. The complete stop task also terminates the associated VS Code background tasks.
- `launch.json` — `Dev: Open Admin UI (Starts Frontend + Backend + Local Supabase)` opens the admin UI after starting the complete development environment. Stopping the debug session runs the complete cleanup task automatically.

If you work across both repos, use the shared workspace file committed in `armada.nu`:

- `../armada.nu/Armada.code-workspace`

That workspace opens both repositories with portable relative paths and includes multi-repo compound launches.

## Project Structure

```text
ArmadaCMS/
├── main.go               # Entry point — routing and server startup
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

- Run the opt-in Storage integration test while the local stack is running:

  ```powershell
  $env:RUN_S3_INTEGRATION = "1"
  $status = pnpx supabase@2.118.0 status -o env
  $env:AWS_ACCESS_KEY_ID = ($status | Select-String '^S3_PROTOCOL_ACCESS_KEY_ID=').Line.Split('=', 2)[1].Trim('"')
  $env:AWS_SECRET_ACCESS_KEY = ($status | Select-String '^S3_PROTOCOL_ACCESS_KEY_SECRET=').Line.Split('=', 2)[1].Trim('"')
  go test -tags=integration -count=1 ./utils
  ```

- Run frontend unit tests:

  ```bash
  cd frontend && pnpm run test
  ```

### PostgreSQL behavior tests

The Go workflow runs two database tests in a separate PostgreSQL 17 service, and the existing `go-checks` check requires them to pass whenever Go changes. No hosted credentials are used; GitHub Actions removes the service and its data after the job.

- `TestDeleteUserWithRefreshTokens`: deletes a user and their tokens, preserves other users, and verifies audit entries and rollback when deletion fails.
- `TestPhotoGuestQuotaLifetimeAndConcurrentUploads`: verifies that deleted/rejected photos still consume quota and concurrent uploads cannot exceed it.

These tests are opt-in locally: ordinary `go test` runs skip them unless their DSNs are set. They create their own limited schema with GORM and do not test the full Supabase migration history.

For a local run, use a disposable PostgreSQL 17 instance with two **fresh, empty** databases named `user_delete_test` and `photo_quota_test`. For example, start a dedicated container (port 55432 avoids the local Supabase ports):

```bash
docker run --detach --rm --name armadacms-database-tests --publish 127.0.0.1:55432:5432 --env POSTGRES_PASSWORD=postgres postgres:17-alpine
```

Once `docker exec armadacms-database-tests pg_isready -U postgres -d postgres` reports ready, run the following from the repository root, with the same Go/libvips prerequisites as other backend tests:

```bash
docker exec armadacms-database-tests psql -U postgres -d postgres -v ON_ERROR_STOP=1 -c 'CREATE DATABASE user_delete_test'
docker exec armadacms-database-tests psql -U postgres -d postgres -v ON_ERROR_STOP=1 -c 'CREATE DATABASE photo_quota_test'
USER_DELETE_TEST_DSN='postgresql://postgres:postgres@localhost:55432/user_delete_test?sslmode=disable' \
PHOTO_QUOTA_TEST_DSN='postgresql://postgres:postgres@localhost:55432/photo_quota_test?sslmode=disable' \
go test -race -count=1 -timeout=5m -v ./Controllers -run '^(TestDeleteUserWithRefreshTokens|TestPhotoGuestQuotaLifetimeAndConcurrentUploads)$'
```

Always remove this dedicated container afterward with `docker rm --force armadacms-database-tests`, including after a failed test. Recreate it for another run; the quota test leaves data in its disposable database. Never point these tests at a shared development, staging or production database.

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

GitHub Actions are pinned to full commit SHAs with version comments. Dependabot checks for version updates every Monday at 08:00 Europe/Stockholm, covering GitHub Actions, pnpm dependencies, Terraform providers, and Go modules. Minor and patch updates are grouped per ecosystem; major updates remain separate PRs and all updates use the existing review and CI requirements. See [the Dependabot configuration](.github/dependabot.yml).

CI and deployments run in GitHub Actions.

- Go, frontend and local Supabase checks run directly on pull requests and are called as reusable workflows by the main/staging push workflow. Internal path filters retain explicit aggregate results when work is intentionally skipped.
- Supabase CI builds and starts the backend container, resets the local database, reports migration status, checks API health, and runs the Storage upload/download/cleanup integration test inside that same container. The test reuses its Go/libvips installation and mounted caches; the runner does not install Go or libvips for this job.
- CodeQL analyzes Actions, JavaScript/TypeScript and Go on PRs, release pushes and its own schedule.
- Release regression tests use Python's standard library and simulate Cloud Run/API failures; the HTTP tests use a local server.
- PR container validation builds the production Dockerfile without GCP credentials, publishing or deployment.

### Verified releases

The shared deployment workflow requires successful Go, frontend, Supabase, CodeQL and release checks for its own source commit. The release job uses the corresponding GitHub Environment, an environment-specific concurrency group, and OIDC Workload Identity Federation to impersonate a narrowly scoped GCP deploy identity. A new push does not interrupt an active release. Repository activation variables control whether each environment deploys; an inactive environment still runs CI.

The image build runs in parallel with all five CI workflows. A branch-specific build identity can publish only to the shared Artifact Registry; it cannot update Cloud Run or impersonate the runtime account. Build connection values are repository variables, while deployment connection values belong to the protected GitHub Environment. Only the release job creates an Environment deployment, and it requires successful CI and image publication before accessing Cloud Run. Newer builds may cancel obsolete builds, but never an ongoing release.

CodeQL caches downloaded Go modules while still compiling source for analysis. Cloud Run API calls reuse an in-memory GCP token until one minute before its reported expiry, refresh on authentication rejection once, and never log or persist credentials. Missing expiry disables token reuse; invalid or expired credentials stop the release.

The production Docker image embeds its commit SHA. Each Actions run/attempt publishes a unique tag, and the service and photo worker use the same immutable registry digest. Buildx registry caches are separate for staging and production. The source branch is checked before building, while preparing recovery, and immediately before activation; an obsolete commit cannot be promoted.

Before changing Cloud Run, the workflow uploads a recovery snapshot as an Actions artifact with 90-day retention. It contains explicit previous revision traffic, the resolved worker digest, previous preview tags, resource etags, configuration fingerprints and release identities. It contains no tokens, environment dumps or raw resource configuration. Failed snapshot publication blocks deployment. A concurrent resource change after preparation also blocks deployment.

The release creates a unique revision with no ordinary traffic and binds the environment's fixed `release-preview` tag to it. Any `LATEST` allocation is pinned to the previously serving explicit revisions in the same update. Staging checks the tagged `run.app` address. Production checks `https://preview.cms.armada.nu` through the existing HTTPS load balancer; production ingress remains restricted to internal traffic and load balancing. The preview host is publicly reachable and uses the application's existing authorization.

Read-only HTTP checks verify `/health`, the database-backed feature flag API and pagination, and admin HTML plus JavaScript/CSS entrypoints. Each check round has a shared two-minute deadline. `/health` remains database-independent and returns `X-Armada-Commit`, `X-Armada-Revision` and `Cache-Control: no-store`. HTTP smoke checks do not execute React or authenticated admin operations.

After the preview passes and the branch is checked again, the worker definition is updated and verified, traffic moves to the exact tested revision, and the normal environment domain is checked. GitHub's Environment deployment succeeds only when live verification and final result publication succeed.

Preview failure restores the previous preview tag without activating worker or ordinary traffic. Activation/live failure restores and verifies previous traffic, worker digest and preview tags. Etag checks and configuration fingerprints prevent recovery from overwriting unexpected external edits. An ambiguous API failure is reconciled against observed and desired state before recovery. Older revisions without identity headers are checked functionally alongside Cloud Run traffic verification.

The result artifact and Actions summary record the phase and recovery outcome; even successful rollback leaves the release failed. Failure to publish the final result marks the workflow failed without rolling back an otherwise verified release. The initial snapshot remains available independently of the final result. Forced runner termination can require operator recovery from that snapshot.

Terraform owns runtime configuration, networking and federation; Actions owns release images, generated revision names, tags and traffic. Preview shares the environment's database and startup seeding. Rollback does not reverse database writes or already-running worker executions. Schema and worker changes remain backward compatible, and hosted database migration remains separate from application release.

The checked-in release targets contain the fixed project, region, service, worker and URLs. Both environments use the production Artifact Registry; staging runs in `europe-north1` and production in `europe-north2`. No Cloud Build deployment trigger, GitHub App credential, release bucket or distributed lock participates in this flow.

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
2. Write a SQL migration in `supabase/migrations/` for the schema change.
3. Validate the migration with a local Supabase reset.
4. Create a controller in `Controllers/` following existing CRUD patterns.
5. Add routes in `main.go` (public for reads, protected for writes).
6. Create `List`, `Create`, `Edit` components in `frontend/src/components/{Resource}/`.
7. Register the `<Resource>` in `frontend/src/App.tsx`.
8. If the resource has file uploads, add it to the multipart list in `frontend/src/dataProvider.ts`.
9. If the resource is displayed on the public site, pass the matching cache tag to the audit helper's `revalidateTags` argument (e.g. `"blog-posts"`) and ensure the same tag is used in the Next.js data hook on `armada.nu`.
