## Plan: AWS exit to Supabase

Migrate ArmadaCMS to Supabase for production PostgreSQL, production/staging/pre-production object storage, and remote environment management while preserving Cloud Run on GCP. Local development continues to use Docker Compose with local Postgres and MinIO — the Supabase CLI is used only as a migration management tool, not as the local runtime. The long-term target is one Terraform-managed Supabase production project imported from the already-created `ArmadaCMS` project, plus a persistent Supabase branch named `staging` for the long-lived staging environment, and ephemeral PR preview branches rolled out together with per-PR GCP services. Until the preview-environment phase is reached, keep the current standalone staging Supabase project in service as the staging database.

**Steps**

1. Phase 0 — Confirm live inventory and freeze assumptions.
   1.1 Capture the current AWS and Supabase estate before touching code or Terraform: production RDS version/settings, production S3 bucket contents/size/object count, staging S3 bucket contents, current staging Supabase project ref/settings, secrets currently stored in GCP Secret Manager, and HCP Terraform workspace/state dependencies. This is blocked only by access to live systems.
   1.2 Record the exact live Supabase project identifiers, region, plan tier, enabled features, database size, branch limits, storage quotas, and whether the already-created project named `ArmadaCMS` is intended to become the production base project. Since terminal execution is unavailable in this planning session, do this via `pnpx supabase` and the dashboard during implementation.
   1.3 Define a maintenance window, rollback window, and success criteria for cutover. Because you accepted a short maintenance window, optimize for correctness over zero-downtime complexity.
   1.4 Snapshot and export all current secrets/config values from GCP/HCP/AWS into an operator checklist so rotations and rollbacks are deterministic.

2. Phase 1 — Lock the target environment model.
   2.1 Adopt the target topology: GCP Cloud Run remains the runtime platform; Supabase becomes the only database and object-storage platform; AWS is retained only until post-cutover validation is complete.
   2.2 Keep the current standalone Supabase staging project as the active staging database until the preview-environment phase. After that, migrate staging to a persistent hosted Supabase branch `staging` if the branching model still fits the operational workflow.
   2.3 Explicitly defer GitHub-integrated hosted preview branches for PRs. Roll them out later together with per-PR GCP preview services so preview infrastructure is introduced as one cohesive feature instead of in half-built layers.
   2.4 First deliver stable production + standalone staging on Supabase-backed workflows, then revisit hosted branching and per-PR preview environments as a later phase.
   2.5 Keep the current standalone Supabase staging project as a safety net through validation, and do not delete it until its replacement topology has been running the staging app successfully for at least one release cycle.

3. Phase 2 — Introduce Supabase-as-code and migration discipline.
   3.1 Add a top-level `supabase/` directory to `ArmadaCMS` with Supabase CLI initialization, config, migrations, and seed files.
   3.2 Link the repo to the existing production Supabase project and immediately pull the current remote schema into versioned migrations using Supabase CLI so the repository, local CLI workflow, and remote production schema all agree.
   3.3 Stop treating GORM `AutoMigrate` as the primary production schema-management mechanism. Keep it temporarily for compatibility if needed, but move the source of truth to checked-in SQL migrations under `supabase/migrations/`.
   3.4 Audit every existing table, index, default, extension, trigger, and permission in production/staging so the initial migration set reflects reality rather than only what GORM knows about.
   3.5 Add `supabase/seed.sql` for remote environment resets and CI. Keep it limited to safe bootstrap data needed for ArmadaCMS to start and be testable: roles, feature flags, optionally a non-production admin account, and minimal relational fixtures.
   3.6 Add CI checks to validate that migrations and seeds apply cleanly in a local Supabase stack before merge.

4. Phase 3 — Prepare the application for Supabase storage.
   4.1 Replace the AWS-specific storage implementation in `utils/aws_s3.go` with a provider-neutral storage abstraction. The immediate backend provider can target Supabase Storage, but the interface should hide provider details from controllers.
   4.2 Preserve existing MIME validation, max-size enforcement, and multipart upload behavior so the admin frontend remains unchanged.
   4.3 Decide how uploaded file locations are represented in the database. Recommended migration path: keep existing URL fields for the initial cutover to minimize API churn, but centralize public-URL generation in one storage service and add a follow-up task to move to provider-neutral object keys later if desired.
   4.4 Create the required Supabase Storage buckets for production and the current standalone staging project. Add hosted-branch bucket strategy later when branching is enabled. Local development continues to use MinIO.
   4.5 Define storage access rules explicitly. Because ArmadaCMS uploads from the backend, prefer server-side credentials/service role access from Cloud Run rather than direct browser uploads for the first migration.
   4.6 Plan the historical object migration: S3 object inventory export, copy to Supabase Storage, checksum/size verification, URL mapping manifest, and idempotent reruns.
   4.7 Plan cleanup/retention of old AWS objects so rollback remains possible until sign-off.

5. Phase 4 — Document the Supabase CLI as a migration management tool (not a local runtime).
   5.1 Local development remains on Docker Compose (Postgres + MinIO + backend + frontend). Do not replace this with the Supabase CLI stack.
   5.2 Use the Supabase CLI exclusively for migration authoring (`supabase db diff`), validation (`supabase db reset` against a temporary local stack or CI), and remote pushes (`supabase db push`).
   5.3 Update `.env.example` and README to document the Supabase CLI migration workflow clearly, separated from the day-to-day Docker Compose dev workflow.
   5.4 Replace or complement `scripts/import-remote-db.ps1` with a Supabase-aware restore workflow for cloning remote data into local Postgres for debugging purposes.

6. Phase 5 — Manage Supabase with Terraform.
   6.1 Add a new Terraform root for Supabase platform resources instead of forcing them into existing GCP or AWS roots. Keep one logical root per stack, consistent with repo conventions.
   6.2 Use the Supabase Terraform provider to import the existing production project (`ArmadaCMS`) rather than recreating it. Version-control project settings and storage buckets there now, and add hosted branch configuration in the next phase when the preview-environment rollout begins.
   6.3 Decide whether persistent `staging` branch management belongs in Terraform or is bootstrapped manually then imported; prefer Terraform if provider support is adequate for the exact branch resources you need.
   6.4 Move GCP runtime configuration away from `data.tfe_outputs` coming from `aws/prod` and `aws/staging`. Replace those dependencies with Supabase-derived plain env vars and Secret Manager secrets.
   6.5 Remove the AWS production RDS and S3 outputs as upstream dependencies only after GCP production/staging run exclusively against Supabase.
   6.6 Update HCP Terraform workspace design, remote-state sharing, and documentation to reflect the new provider split. The GCP workspaces should no longer depend on AWS workspaces for DB/storage configuration once cutover is complete.
   6.7 Keep AWS Terraform roots intact but quiescent until final decommission; do not destroy state-managed AWS resources until rollback is no longer needed.

7. Phase 6 — Create the new runtime configuration model.
   7.1 Define the new environment-variable contract for ArmadaCMS backend runtime: database host/port/user/password/name/SSL, Supabase project URL, service-role or storage service credentials, and any storage bucket identifiers.
   7.2 Register new secrets in GCP Secret Manager and wire them into Cloud Run for both production and staging.
   7.3 Keep staging Cloud Run pointed at the current standalone Supabase staging project for now. Update it to use the persistent hosted Supabase `staging` branch only after hosted branching is enabled.
   7.4 Update connection-pool tuning for Supabase limits in each environment and validate that staging/prod values are intentionally different if needed.
   7.5 Document operator procedures for rotating Supabase database credentials, service-role keys, and any GitHub-integration tokens.

8. Phase 7 — Build the preview-environment workflow.
   8.1 Connect the `ArmadaCMS` GitHub repository to Supabase GitHub integration with the correct working directory.
   8.2 Enable automatic preview branches for PRs together with per-PR GCP preview services, and configure required checks so failed hosted previews block merges.
   8.3 Ensure preview branches apply migrations and seed data automatically. Preview branches contain no production data, so seed coverage must be enough to exercise admin flows and smoke tests.
   8.4 Roll out preview app runtimes and hosted preview databases together so service naming, secret injection, and cleanup are deterministic from the start.
   8.5 Update Cloud Build, Terraform, and GitHub automation so preview service naming, secret injection, status checks, and cleanup are deterministic.

9. Phase 8 — Rehearse database migration end to end.
   9.1 Perform a full dress rehearsal from a recent production snapshot into a non-production Supabase target. Measure dump/import time, migration time, application smoke-test time, and rollback time.
   9.2 Verify that GORM, seeds, audit logging, background startup flows, login, CRUD, uploads, and public-site reads work unchanged against the migrated database.
   9.3 Verify all non-schema objects that are easy to miss: extensions, sequences, defaults, collations, indexes, foreign keys, triggers, and ownership/permissions.
   9.4 Compare row counts and selected checksums/table samples between source and target. Record any expected differences.
   9.5 Verify restore and rollback procedures by repointing staging back and forth between old/new backends during rehearsal.

10. Phase 9 — Rehearse storage migration end to end.
    10.1 Export an inventory of all production and staging S3 objects, grouped by bucket/prefix and linked back to database rows that reference them.
    10.2 Copy objects into the target Supabase buckets using a repeatable sync job. Produce a manifest containing source key, target key, size, checksum if available, and resulting public URL.
    10.3 Run a database rewrite or mapping step so all stored URLs now point to Supabase Storage for the rehearsal environment.
    10.4 Smoke-test every upload path and every common read path, including old objects, newly uploaded objects, and images consumed by `armada.nu`.
    10.5 Decide whether production cutover uses a short write freeze plus one final incremental sync, or a temporary dual-write period. Recommended with your downtime tolerance: short write freeze plus final incremental sync for simpler correctness.

11. Phase 10 — Execute staging cutover first.
    11.1 Migrate staging file storage from AWS S3 to Supabase Storage while continuing to use the standalone staging Supabase project. Move staging runtime to a persistent hosted `staging` branch later, once branching is enabled.
    11.2 Validate admin CRUD, uploads, public-site reads, Eventro integration, auth token refresh, audit logs, and startup seeding behaviors.
    11.3 Run one full staging release cycle on the new topology before touching production.
    11.4 Keep the old standalone staging project and AWS staging bucket intact during this proving period.

12. Phase 11 — Execute production cutover.
    12.1 Announce and enter the maintenance window.
    12.2 Freeze writes to ArmadaCMS admin and any automated jobs that mutate data or upload files.
    12.3 Take final production database dump and final incremental storage sync from AWS S3.
    12.4 Import the final database state into the Supabase production project and run any final migrations or grants.
    12.5 Rewrite database-stored file URLs if that has not already been done in a controlled pre-step.
    12.6 Update GCP Secret Manager and Cloud Run production to point to Supabase database/storage credentials.
    12.7 Run smoke tests immediately after deploy: health endpoint, login, list endpoints, create/update/delete on a low-risk resource, and one upload path end to end.
    12.8 Reopen writes only after the smoke-test checklist passes.

13. Phase 12 — Stabilize, observe, and only then remove AWS.
    13.1 Monitor application logs, database connections, storage errors, auth flows, and image delivery for a defined observation window.
    13.2 Keep RDS and S3 read-only or otherwise rollback-capable until the observation window closes.
    13.3 Remove GCP references to AWS credentials, delete AWS IAM upload users, retire staging and production S3 buckets, and decommission RDS only after explicit sign-off.
    13.4 Delete or archive the old standalone staging Supabase project once its replacement staging topology has fully replaced it.
    13.5 Update all runbooks and architecture docs so the repo documents the new reality rather than the archaeological layer beneath it.

**Relevant files**

- `c:\Users\einar\Programmeringsprojekt\Armada\ArmadaCMS\db\connect.go` — DB DSN and pool tuning; verify compatibility and environment contract.
- `c:\Users\einar\Programmeringsprojekt\Armada\ArmadaCMS\main.go` — current `AutoMigrate` and seed startup behavior; decide transitional vs long-term role once Supabase migrations are introduced.
- `c:\Users\einar\Programmeringsprojekt\Armada\ArmadaCMS\utils\aws_s3.go` — current AWS/MinIO-specific upload implementation to replace with a storage abstraction targeting Supabase Storage.
- `c:\Users\einar\Programmeringsprojekt\Armada\ArmadaCMS\Controllers\ExhibitorController.go` — multipart upload consumer.
- `c:\Users\einar\Programmeringsprojekt\Armada\ArmadaCMS\Controllers\EventController.go` — multipart upload consumer.
- `c:\Users\einar\Programmeringsprojekt\Armada\ArmadaCMS\Controllers\ProfileController.go` — multipart upload consumer.
- `c:\Users\einar\Programmeringsprojekt\Armada\ArmadaCMS\Controllers\BlogpostController.go` — multipart upload consumer.
- `c:\Users\einar\Programmeringsprojekt\Armada\ArmadaCMS\frontend\src\dataProvider.ts` — frontend multipart behavior expected to remain stable.
- `c:\Users\einar\Programmeringsprojekt\Armada\ArmadaCMS\.env.example` — local/dev/runtime environment contract to redesign for Supabase.
- `c:\Users\einar\Programmeringsprojekt\Armada\ArmadaCMS\docker-compose.dev.yml` — local dev topology; Postgres and MinIO stay here permanently.
- `c:\Users\einar\Programmeringsprojekt\Armada\ArmadaCMS\scripts\import-remote-db.ps1` — current remote clone workflow to replace/complement with Supabase-aware, cross-platform restore/import tooling.
- `c:\Users\einar\Programmeringsprojekt\Armada\ArmadaCMS\infra\terraform\README.md` — current provider/environment layout to update.
- `c:\Users\einar\Programmeringsprojekt\Armada\ArmadaCMS\infra\terraform\gcp\prod\locals.tf` — currently injects AWS-derived env vars into Cloud Run production.
- `c:\Users\einar\Programmeringsprojekt\Armada\ArmadaCMS\infra\terraform\gcp\staging\locals.tf` — currently mixes Supabase DB with AWS S3 env vars; will move fully to Supabase.
- `c:\Users\einar\Programmeringsprojekt\Armada\ArmadaCMS\infra\terraform\gcp\prod\README.md` — production runtime contract and secret handling docs to revise.
- `c:\Users\einar\Programmeringsprojekt\Armada\ArmadaCMS\infra\terraform\gcp\staging\README.md` — staging runtime contract to revise around persistent Supabase branch.
- `c:\Users\einar\Programmeringsprojekt\Armada\ArmadaCMS\infra\terraform\aws\prod\README.md` — production AWS resources currently providing DB/storage.
- `c:\Users\einar\Programmeringsprojekt\Armada\ArmadaCMS\infra\terraform\aws\staging\README.md` — staging AWS storage root to retire.
- `c:\Users\einar\Programmeringsprojekt\Armada\ArmadaCMS\.github\workflows\keep-staging-alive.yml` — likely removable once the old free-tier staging project is retired.
- `c:\Users\einar\Programmeringsprojekt\Armada\ArmadaCMS\README.md` — developer workflow documentation; update to clarify that local dev stays on Docker Compose and Supabase CLI is a migration management tool only.
- `c:\Users\einar\Programmeringsprojekt\Armada\armada.nu\src\env.ts` — if the public site ever needs new CMS/storage-facing env vars or preview awareness.

**Verification**

1. Live inventory verified: database versions, bucket inventories, secret inventories, Terraform workspace dependencies, and current Supabase project details captured before any changes.
2. Migration tooling verified: `supabase db reset` applies all migrations and seeds cleanly in a temporary local Supabase stack or CI; `supabase db push` applies pending migrations to remote environments.
3. Migration discipline verified: CI fails when migrations or seeds do not apply cleanly; hosted preview-branch checks are added later when the preview-environment phase begins.
4. Storage abstraction verified: all upload endpoints (`profiles`, `events`, `exhibitors`, `blogposts`) can create and update records against Supabase Storage; public URLs resolve correctly from both admin and `armada.nu`.
5. Staging verified on the active staging database topology: health endpoint, auth/login/refresh, CRUD, uploads, public-site reads, Eventro sync paths, and audit logs all pass.
6. Dress rehearsal verified: database import, storage sync, URL rewrite, and rollback are executed end to end on non-production infrastructure with timings recorded.
7. Production cutover verified during maintenance window: smoke tests pass before writes reopen; post-cutover monitoring shows no elevated DB/storage/auth errors.
8. AWS decommission verified only after observation-window sign-off and documented rollback retirement.

**Decisions**

- Included scope: production database migration, production/staging/pre-production storage migration, Supabase migrations/seeds as remote schema management, Terraform-managed Supabase adoption, Cloud Run/GCP integration updates, and cutover/rollback/decommission planning.
- Excluded from all phases: replacing local Docker Compose development with Supabase CLI. Local dev permanently uses Docker Compose (Postgres + MinIO). Supabase CLI is a migration authoring and push tool only.
- Excluded from first migration wave: adopting Supabase Auth, Realtime, Edge Functions, or direct browser uploads. These are not needed to eliminate AWS and would add unnecessary risk.
- Recommended environment model for the current phase: one imported production Supabase project, the existing standalone staging Supabase project, and no hosted preview branches yet. Proceed with a persistent hosted `staging` branch and ephemeral PR preview branches in the next phase, together with per-PR GCP services.
- Recommended cutover model: short production maintenance window with final DB dump + final incremental object sync rather than near-zero-downtime dual-write.
- Recommended storage migration model: backend-driven uploads to Supabase Storage, keep existing API/frontend upload contract stable for the first wave.

**Further Considerations**

1. If Supabase Terraform branch resource support is incomplete for the exact branch workflow you want, bootstrap the persistent `staging` branch manually, then import/manage the supported subset in Terraform while documenting the gap clearly.
2. Enable hosted Supabase preview branches together with per-PR Cloud Run preview services rather than separately, so preview infrastructure is introduced as one coherent system.
3. After AWS exit is complete, consider a second hardening project to replace absolute file URLs stored in database rows with provider-neutral object keys to reduce future storage lock-in.
