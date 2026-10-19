# ArmadaCMS

[![CI main](https://img.shields.io/github/actions/workflow/status/armada-ths/ArmadaCMS/deploy.yml?branch=main&event=push&label=CI%20main&logo=githubactions)](https://github.com/armada-ths/ArmadaCMS/actions/workflows/deploy.yml?query=branch%3Amain)
[![CI staging](https://img.shields.io/github/actions/workflow/status/armada-ths/ArmadaCMS/deploy.yml?branch=staging&event=push&label=CI%20staging&logo=githubactions)](https://github.com/armada-ths/ArmadaCMS/actions/workflows/deploy.yml?query=branch%3Astaging)
[![CodeQL](https://img.shields.io/github/actions/workflow/status/armada-ths/ArmadaCMS/codeql.yml?label=CodeQL&logo=github)](https://github.com/armada-ths/ArmadaCMS/actions/workflows/codeql.yml)
[![Container build](https://img.shields.io/github/actions/workflow/status/armada-ths/ArmadaCMS/container-build.yml?event=pull_request&label=container%20build&logo=docker)](https://github.com/armada-ths/ArmadaCMS/actions/workflows/container-build.yml)
[![Production deployment](https://img.shields.io/github/deployments/armada-ths/ArmadaCMS/Production?label=production&logo=googlecloud)](https://github.com/armada-ths/ArmadaCMS/deployments/Production)
[![API status](https://img.shields.io/website?url=https%3A%2F%2Fcms.armada.nu%2Fhealth&label=CMS%20API)](https://cms.armada.nu/health)

[![License: MIT](https://img.shields.io/github/license/armada-ths/ArmadaCMS)](https://github.com/armada-ths/ArmadaCMS/blob/main/license.txt)
[![Go 1.26](https://img.shields.io/badge/Go-1.26-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![React 19](https://img.shields.io/badge/React-19-61DAFB?logo=react&logoColor=black)](https://react.dev/)
[![React Admin 5](https://img.shields.io/badge/React_Admin-5-22A6F2?logo=react&logoColor=white)](https://marmelab.com/react-admin/)
[![TypeScript 6](https://img.shields.io/badge/TypeScript-6-3178C6?logo=typescript&logoColor=white)](https://www.typescriptlang.org/)
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

- **Backend**: Go 1.26, [Gorilla Mux](https://github.com/gorilla/mux), [GORM](https://gorm.io/) and JWT authentication.
- **Admin**: [React-Admin v5](https://marmelab.com/react-admin/), TypeScript, MUI and Vite.
- **Infrastructure**: Google Cloud Run, Supabase PostgreSQL and Supabase Storage. Production uses an HTTPS load balancer; staging uses Cloud Run domain mapping.

## Prerequisites

- [Docker](https://www.docker.com/) and Docker Compose _(required for local development)_
- [Go 1.26+](https://go.dev/dl/) _(optional, for running Go tooling directly)_
- [Node.js 24+](https://nodejs.org/) and pnpm _(required for the pinned local Supabase CLI and frontend tooling)_

## Getting Started

1. **Clone and configure**

   ```bash
   git clone https://github.com/armada-ths/ArmadaCMS.git
   cd ArmadaCMS
   cp .env.example .env
   ```

   `.env.example` contains defaults for local Supabase and Docker.

2. **Start the backend and local Supabase**

   ```powershell
   ./scripts/dev-up.ps1 -Build
   ```

   This starts Postgres, Storage and the Go API with hot reload. Omit `-Build` on subsequent starts.

3. **Start the admin frontend in a second terminal**

   ```powershell
   cd frontend
   pnpm install --frozen-lockfile
   pnpm dev
   ```

   The frontend runs on the host for reliable Vite hot reload.

| Service  | Local address                                |
| -------- | -------------------------------------------- |
| Admin UI | <http://localhost:5173/admin>                |
| API      | <http://localhost:8080/api/v1/>              |
| Health   | <http://localhost:8080/health>               |
| Database | `postgres:postgres@localhost:54322/postgres` |

Stop the backend and Supabase with `./scripts/dev-down.ps1`; local data is preserved. See [development scripts](scripts/README.md#local-development) for local photo behavior and optional database import.

## Database migrations

Schema changes live in [`supabase/migrations/`](supabase/migrations/). The hosted Supabase GitHub integration applies migrations to its tracked branches; local changes are validated with the pinned CLI:

```bash
pnpx supabase@2.118.0 db diff -f <migration-name>
pnpx supabase@2.118.0 db reset --local
```

`db reset` replaces local database data with the migrations and [`supabase/seed.sql`](supabase/seed.sql). The non-production seed provides roles, feature flags and an `admin` / `admin` account; never use these credentials in production.

## VS Code workspace and launches

[`.vscode/tasks.json`](.vscode/tasks.json) provides start, rebuild and stop tasks. The admin launch in [`launch.json`](.vscode/launch.json) starts the full environment and stops it when debugging ends.

For both Armada repositories, open [`../armada.nu/Armada.code-workspace`](../armada.nu/Armada.code-workspace). It includes shared launches and portable paths.

## Project Structure

```text
ArmadaCMS/
├── main.go               # Routing and startup
├── Controllers/          # HTTP handlers
├── models/               # GORM models
├── auth/                 # JWT authentication
├── db/                   # Database connection
├── utils/                # Storage, tokens and other helpers
├── frontend/src/         # React-Admin resources and data/auth providers
├── supabase/             # Migrations, seed and local configuration
├── scripts/              # Local development and release tooling
├── infra/terraform/      # Infrastructure configuration
└── Dockerfile.{dev,prod}  # Development and production images
```

## Testing

Run backend tests from the repository root (requires Go and libvips):

```bash
go test -race -count=1 ./...
```

Run frontend tests from `frontend/`:

```bash
pnpm run test
```

[Integration test instructions](docs/testing/README.md) cover local Storage and the two opt-in PostgreSQL behavior tests. These require explicit test credentials; ordinary Go test runs skip them. See the [script guide](scripts/README.md#script-tests) for release-script tests and the [frontend guide](frontend/README.md#validate-changes) for lint/type checks.

## API

Endpoints use `/api/v1`. Public reads need no authentication; writes and admin-only reads require a Bearer JWT obtained from `POST /api/v1/login`. See [Swagger](#swagger-docs) for available endpoints.

Cross-origin requests allow `armada.nu`, local development origins and any additional exact origins listed in `CORS_ALLOWED_ORIGINS`.

## Swagger docs

Interactive API docs are available in [local development](http://localhost:8080/swagger/index.html), [staging](https://staging.cms.armada.nu/swagger/index.html) and [production](https://cms.armada.nu/swagger/index.html). Use **Authorize** with `Bearer <token>` for protected endpoints.

After changing routes or annotations, regenerate and commit the files in `docs/`:

```bash
swag init --generalInfo main.go --output docs --parseInternal
```

Install the generator if needed with `go install github.com/swaggo/swag/cmd/swag@latest`.

## Cache Revalidation

Write operations invalidate the public site's cache through audit-helper revalidation tags. This requires `REVALIDATION_URL` and `REVALIDATION_SECRET`; it is skipped when unset. See the [shared tag conventions](https://github.com/armada-ths/armada.nu/blob/main/.github/copilot-instructions.md).

## CI / CD

GitHub Actions runs Go linting/tests, frontend checks, local Supabase migration/Storage tests, CodeQL, and release-script regression tests. Pull requests also build the production container without publishing or deploying it. Path filters skip unaffected checks while retaining summary results.

Pushes to `staging` deploy to **Staging**; `main` deploys to **Production**, when enabled. Image builds run alongside CI, but deployment requires all checks to pass for the same commit:

**Build → save recovery snapshot → deploy without traffic → preview test → update worker and move traffic → live test.**

Releases use short-lived GCP credentials and the same immutable image digest for the API and photo worker. Releases are serialized per environment; new pushes do not interrupt an active release. Failed activation or live verification restores the previous traffic and worker image. Rollback does not undo database writes or running worker executions, so schema and worker changes must remain backward compatible.

Recovery snapshots and release results are retained as Actions artifacts for 90 days. See [release verification and recovery](scripts/README.md) for checks, failure behavior, and limitations.

Actions are pinned to commit SHAs. [Dependabot](.github/dependabot.yml) opens weekly dependency updates; minor/patch updates are grouped, and major updates remain separate PRs.

## Operations notes

- Production runs on Cloud Run with Supabase (PostgreSQL) and Supabase Storage for file uploads.
- The server reads `PORT` from the environment and falls back to `8080`.
- Production database connections use `DB_SSLMODE=require`.
- Tune `DB_MAX_OPEN_CONNS`, `DB_MAX_IDLE_CONNS`, and Cloud Run max instances together to stay within Postgres connection limits.

## Infrastructure as code

See the [Terraform guide](infra/terraform/README.md) for layout and shared conventions, with environment details in [GCP production](infra/terraform/gcp/prod/README.md), [GCP staging](infra/terraform/gcp/staging/README.md) and [Supabase production](infra/terraform/supabase/prod/README.md).

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
