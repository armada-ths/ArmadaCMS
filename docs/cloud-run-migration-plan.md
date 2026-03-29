# ArmadaCMS Cloud Run migration plan

## Goal

Migrate `ArmadaCMS` from the current AWS ECS/Fargate setup to **Google Cloud Run**, while initially keeping:

- **AWS RDS** for PostgreSQL
- **AWS S3** for file uploads

This minimizes migration risk while removing the most expensive parts of the current AWS hosting stack.

---

## Recommended target architecture

```text
Cloud Run (Go API + bundled admin frontend)
        |
        +--> AWS RDS (existing PostgreSQL)
        |
        +--> AWS S3 (existing file storage)
```

### Why this target

This approach preserves the existing application architecture and avoids an unnecessary database migration in phase 1.

Benefits:

- removes ECS/Fargate costs
- removes ELB costs
- reduces surrounding AWS app-hosting overhead
- keeps the already working DB and file storage
- requires fewer code changes than a full platform migration

---

## Current repo facts that make Cloud Run feasible

The current codebase is already close to Cloud Run-compatible:

- `Dockerfile.prod` already builds a deployable production container
- `main.go` already serves:
  - `/api/v1/*`
  - `/admin/`
  - `/health`
- `frontend/src/context/globalApi.ts` already uses same-origin API routing in production:
  - `${window.location.origin}/api/v1`

This means the current monolithic deployment model can be reused on Cloud Run.

---

## Migration strategy

### Phase 1 — Move compute only

Migrate the app runtime from AWS ECS to Cloud Run, while keeping:

- RDS unchanged
- S3 unchanged

### Phase 2 — Observe and stabilize

After cutover, measure:

- Cloud Run cost
- DB latency
- DB connection usage
- operational complexity of cross-cloud DB access

### Phase 3 — Reevaluate DB hosting later

Only if needed, consider moving PostgreSQL later to:

- Supabase
- Cloud SQL
- another hosted Postgres option

This keeps phase 1 focused and low-risk.

---

## Required repo changes

## 1. Make the server listen on Cloud Run's `PORT`

### Why

Cloud Run injects a `PORT` environment variable and expects the container to bind to it.

### Current state

In `main.go`, the server currently uses a hardcoded port:

- `const port = 8080`

### Required change

Update `main.go` to:

- read `PORT` from environment
- default to `8080` locally if not set

### File

- `main.go`

---

## 2. Enable production DB SSL

### Why

The current example env file uses:

- `DB_SSLMODE=disable`

That is fine for local development but should not be used for production RDS access from Cloud Run.

### Required change

Use a production value such as:

- `DB_SSLMODE=require`

### Files / config touched

- `.env.example`
- Cloud Run environment variables or secrets

---

## 3. Move deployment secrets/env vars to Cloud Run configuration

### Why

The current ECS deployment injects env/secrets using AWS task definitions. Cloud Run needs its own env + secret configuration.

### Environment variables currently needed

Database:

- `DB_HOST`
- `DB_PORT`
- `DB_USER`
- `DB_PASSWORD`
- `DB_NAME`
- `DB_SSLMODE`

Eventro integration:

- `EVENTRO_API`
- `EVENTRO_FAIR_ID`
- `EVENTRO_ORG`

S3 / AWS:

- `S3_BUCKET`
- `AWS_REGION`
- `AWS_ACCESS_KEY_ID` _(if using static credentials)_
- `AWS_SECRET_ACCESS_KEY` _(if using static credentials)_

JWT:

- `jwtsecret_laganda`

### Important note

`jwtsecret_laganda` is used in `utils/tokenHelper.go` but is currently not documented in `.env.example`.

### Files / config touched

- `.env.example`
- Cloud Run deployment configuration
- GitHub Actions deploy workflow

---

## 4. Replace ECS deployment workflow with Cloud Run deployment

### Current AWS-specific deployment files

- `.github/workflows/build.yml`
- `.github/workflows/deploy.yml`
- `task-definition.json`

These are tightly coupled to:

- ECR
- ECS
- AWS task definitions

### Required change

Introduce a Cloud Run deployment workflow.

### Suggested new file

- `.github/workflows/deploy-cloud-run.yml`

### Workflow responsibilities

- authenticate to GCP
- select project + region
- build/deploy the app to Cloud Run
- inject required env vars / secrets references

### Possible deployment methods

#### Option A — Source deploy

Simpler initial path:

- deploy from repo source
- let Google build the image

#### Option B — Image deploy

More explicit path:

- build Docker image
- push to Artifact Registry
- deploy image to Cloud Run

### Recommendation

Start with **Option A** unless there is a strong reason to control the container registry path explicitly.

---

## Recommended production hardening

These are not strictly required to get the app running, but are strongly recommended.

## 5. Add DB connection pool tuning

### Why

`db/connect.go` currently opens a GORM connection but does not tune the underlying SQL pool.

That is risky in autoscaling/serverless environments because:

- each Cloud Run instance can open multiple DB connections
- instance bursts can overwhelm the database if left unchecked

### Recommended additions

After opening the DB connection, configure:

- maximum open connections
- maximum idle connections
- connection max lifetime
- connection max idle time

### Suggested env vars

- `DB_MAX_OPEN_CONNS`
- `DB_MAX_IDLE_CONNS`
- `DB_CONN_MAX_LIFETIME_MINUTES`
- `DB_CONN_MAX_IDLE_TIME_MINUTES`

### File

- `db/connect.go`

---

## 6. Add Cloud Run scaling guardrails

### Why

Cloud Run can scale quickly. Since this app talks to a relational database, it is best to start conservatively.

### Recommended initial Cloud Run settings

- **CPU:** `1`
- **Memory:** `512Mi` or `1Gi`
- **Min instances:** `0`
- **Max instances:** `2` or `3`
- **Concurrency:** `10` to `20`
- **Timeout:** `120s`

### Why this matters

These settings help avoid:

- DB connection spikes
- accidental overscaling
- unnecessary cost growth

---

## 7. Add a `.dockerignore`

### Why

There is currently no `.dockerignore` file in the repo.

Without one, Docker build context may include:

- `.git`
- `.env`
- temp files
- local build output
- `node_modules`

This slows builds and may increase risk of including unnecessary files.

### Suggested new file

- `.dockerignore`

### Recommended exclusions

- `.git`
- `.github`
- `.env`
- `tmp`
- `frontend/node_modules`
- `node_modules`
- local caches / artifacts

---

## 8. Make AWS region configurable in S3 helper

### Why

`utils/aws_s3.go` currently hardcodes:

- region: `eu-north-1`
- S3 URL format with `s3.eu-north-1.amazonaws.com`

This works today, but it is better to respect `AWS_REGION` from the environment.

### File

- `utils/aws_s3.go`

### Recommendation

Use:

- `AWS_REGION` from env if present
- fallback to `eu-north-1`

This is not required for Cloud Run but is a good cleanup.

---

## 9. Update docs for local vs production usage

### Why

The current docs are still oriented around local Docker and AWS ECS assumptions.

The repo should document the new Cloud Run production setup clearly.

### Files

- `.env.example`
- `README.md`

### Recommended doc additions

- required Cloud Run env vars
- production SSL guidance for RDS
- how admin frontend is served on Cloud Run
- deployment steps for staging/production

---

## Optional improvements

## 10. Tighten production CORS handling

### Why

`main.go` currently reflects the request origin dynamically into `Access-Control-Allow-Origin`.

That is permissive and convenient, but can be improved in production.

### File

- `main.go`

### Recommendation

Either:

- leave it unchanged for migration speed
- or add an allowlist-based origin check via env var

This is optional for phase 1.

---

## 11. Split admin frontend from API later (optional)

### Why

The current container bundles:

- Go backend
- React admin frontend

This is acceptable for the first Cloud Run migration.

### Recommendation

Do **not** split this during phase 1.

If desired later, move the admin frontend to a static hosting platform such as:

- Vercel
- Firebase Hosting
- Cloud Storage + CDN

This can reduce backend container size and simplify frontend hosting, but is not necessary for the initial migration.

---

## Concrete migration phases

## Phase 0 — Prepare the repo

Make the required code/config updates without changing production hosting yet.

### Changes to prepare

- update `main.go` to support `PORT`
- update `db/connect.go` with pool tuning
- update `.env.example`
- add `.dockerignore`
- optionally improve `utils/aws_s3.go`
- update deployment docs in `README.md`

### Validation

Build and run the production image locally, then verify:

- `/health`
- `/admin/`
- `/api/v1/login`
- a few protected CRUD flows
- file upload to S3

---

## Phase 1 — Deploy to Cloud Run using existing RDS and S3

### Infrastructure tasks

1. Create or select a GCP project
2. Enable required GCP services:
   - Cloud Run
   - Cloud Build
   - Artifact Registry _(if using image-based deploys)_
   - Secret Manager _(recommended)_
3. Deploy the app from the repo root
4. Set env vars / secrets in Cloud Run
5. Configure DB connectivity to the existing RDS instance

### Networking choice

There are two main connectivity styles for RDS:

#### Simpler first version

- expose RDS publicly
- enforce SSL
- restrict access as much as possible

#### Hardened version

- give Cloud Run stable egress
- allowlist that egress IP in the RDS security group

### Recommendation

Decide explicitly which connectivity model to use before production cutover.

---

## Phase 2 — Replace CI/CD

### Goal

Retire the AWS ECS/ECR deployment path and replace it with Cloud Run deployment automation.

### Files to retire or stop using

- `.github/workflows/build.yml`
- `.github/workflows/deploy.yml`
- `task-definition.json`

### New workflow

Create:

- `.github/workflows/deploy-cloud-run.yml`

### Workflow responsibilities

- checkout code
- authenticate to GCP
- deploy to Cloud Run
- provide env/secrets references
- target the correct region and service

---

## Phase 3 — Cutover and validation

### Before cutover

Verify the Cloud Run deployment supports:

- `GET /health`
- admin login
- key CRUD flows
- public endpoints used by `armada.nu`
- Eventro endpoints
- S3 uploads

### Cutover steps

- point production domain / traffic to Cloud Run
- verify `armada.nu` can reach the new backend
- monitor logs, error rates, and DB connectivity

---

## Exact repo touchpoints

## Files that should change

- `main.go`
  - read `PORT`
  - optional CORS improvements
- `db/connect.go`
  - DB pool tuning
- `.env.example`
  - add production env docs and missing vars
- `utils/aws_s3.go`
  - optional region cleanup
- `README.md`
  - add Cloud Run deployment docs
- `.github/workflows/deploy-cloud-run.yml`
  - new deploy workflow
- `.dockerignore`
  - new file

## Files that may become obsolete

- `task-definition.json`
- `.github/workflows/build.yml`
- `.github/workflows/deploy.yml`

## Files that can likely stay unchanged

- `Dockerfile.prod`
- `frontend/src/context/globalApi.ts`
- admin static serving logic in `main.go`

---

## Suggested production environment variables

For `Cloud Run + existing RDS + existing S3`, the production service will likely need:

Application / runtime:

- `PORT`

Database:

- `DB_HOST`
- `DB_PORT`
- `DB_USER`
- `DB_PASSWORD`
- `DB_NAME`
- `DB_SSLMODE=require`
- `DB_MAX_OPEN_CONNS`
- `DB_MAX_IDLE_CONNS`
- `DB_CONN_MAX_LIFETIME_MINUTES`
- `DB_CONN_MAX_IDLE_TIME_MINUTES`

Eventro:

- `EVENTRO_API`
- `EVENTRO_FAIR_ID`
- `EVENTRO_ORG`

S3 / AWS:

- `S3_BUCKET`
- `AWS_REGION`
- `AWS_ACCESS_KEY_ID` / `AWS_SECRET_ACCESS_KEY` _(if using static credentials)_

JWT:

- `jwtsecret_laganda`

---

## Suggested initial Cloud Run settings

Start conservatively and tune later:

- **CPU:** `1`
- **Memory:** `512Mi` or `1Gi`
- **Min instances:** `0`
- **Max instances:** `2`
- **Concurrency:** `10`
- **Timeout:** `120s`

These values are intended to reduce risk during the initial migration.

---

## Recommended implementation order

1. Update `main.go` to support `PORT`
2. Add pool tuning in `db/connect.go`
3. Update `.env.example`
4. Add `.dockerignore`
5. Update `README.md`
6. Add Cloud Run GitHub Actions workflow
7. Deploy to staging Cloud Run service
8. Validate endpoints and uploads
9. Cut over production traffic

---

## Risks to watch during migration

### 1. DB connectivity

Likely the main migration risk if keeping AWS RDS.

### 2. Connection storms

Mitigate with:

- DB pool tuning
- low max instances
- modest concurrency

### 3. Upload flow regressions

Uploads should continue to work because they already go through S3.

### 4. Admin UI routing

Should remain stable because production API requests are same-origin.

---

## Recommended phase-1 conclusion

For the initial migration, the best practical path is:

- keep the current monolithic container
- make the app Cloud Run-friendly
- deploy it to Cloud Run
- keep existing RDS and S3
- replace ECS deployment automation with GCP deployment automation

This maximizes savings while minimizing migration risk.

---

## Follow-up work after successful migration

After the app is stable on Cloud Run, revisit these optional improvements:

- move DB off RDS if cross-cloud ops become annoying
- split admin frontend from API if desired
- tighten CORS policy
- improve secrets handling / credential strategy
- optimize memory/CPU/concurrency based on observed traffic
