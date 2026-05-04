# Supabase application schema inventory

This document captures the gap between the checked-in Supabase platform baseline in `supabase/migrations/20260504083246_remote_schema.sql` and the actual ArmadaCMS application schema that is still created implicitly by GORM `AutoMigrate`.

## Current source of truth

At the moment, the application schema is derived from:

1. `main.go` calling `db.DB.AutoMigrate(...)`
2. the GORM model structs in `models/`
3. startup seeders in `Controllers/RoleController.go`, `Controllers/FeatureFlagController.go`, and `Controllers/UserController.go`
4. runtime audit-log writes in `audit/audit.go`

The pulled Supabase migration currently captures the hosted project baseline only. It does **not** yet create the ArmadaCMS application tables.

The first generated application snapshot now exists in `supabase/migrations/20260504085048_armadacms_application_schema_snapshot.sql`.

## Tables still missing from checked-in SQL migrations

### Core auth and audit

| Table            | Source model          | Notes                                                                                                      |
| ---------------- | --------------------- | ---------------------------------------------------------------------------------------------------------- |
| `roles`          | `models.Role`         | `name` is unique; `permissions` is stored as `text` with JSON-encoded content via a custom Go type.        |
| `users`          | `models.User`         | optional `role_id` FK to `roles.id`; `created_at` and `updated_at` default to `now()`.                     |
| `refresh_tokens` | `models.RefreshToken` | references `users.id`; currently no explicit `ON DELETE` rule in the model.                                |
| `audit_logs`     | `models.AuditLog`     | `old_data` and `new_data` use `jsonb`; indexes exist on actor, action, resource type/id, and `created_at`. |

### Public/content models

| Table                 | Source model               | Notes                                                                                                                         |
| --------------------- | -------------------------- | ----------------------------------------------------------------------------------------------------------------------------- |
| `teams`               | `models.Team`              | lookup/config table used by profiles and recruitment roles.                                                                   |
| `profiles`            | `models.Profile`           | optional `team_id` FK with `ON DELETE SET NULL`; `eventro_key` is unique.                                                     |
| `industries`          | `models.Industry`          | unique `name`.                                                                                                                |
| `programs`            | `models.Program`           | unique `name`; unlike most models, `id` is declared as a primary key without an explicit `autoIncrement` tag.                 |
| `employments`         | `models.Employment`        | unique `name`.                                                                                                                |
| `exhibitors`          | `models.Exhibitor`         | optional `eventro_id` unique; several nullable text fields; many-to-many relations to industries, programs, employments.      |
| `events`              | `models.Event`             | unique `eventro_id`; several timestamps; `show` defaults to `true`.                                                           |
| `recruitment_periods` | `models.RecruitmentPeriod` | optional `eventro_id` unique; parent table for recruitment roles.                                                             |
| `recruitment_roles`   | `models.RecruitmentRole`   | FK to `recruitment_periods` with `ON DELETE CASCADE`; optional `team_id` with `ON DELETE SET NULL`; unique `eventro_role_id`. |
| `fair_date_configs`   | `models.FairDateConfig`    | stores fair-date values as flat strings, including comma-separated `fair_days`.                                               |
| `feature_flags`       | `models.FeatureFlag`       | unique `key`; seeded on startup.                                                                                              |
| `highlight_cards`     | `models.HighlightCard`     | `description` uses `text`; nullable `brand` defaults to `'ARMADA'`.                                                           |
| `blogposts`           | `models.Blogpost`          | includes `user_id`, `text`, title/author/image fields, and `created_at`.                                                      |

### Join tables created implicitly by GORM

| Table                   | Relationship                 | Notes                                                                                     |
| ----------------------- | ---------------------------- | ----------------------------------------------------------------------------------------- |
| `exhibitor_industries`  | `exhibitors` ↔ `industries`  | created via `many2many:exhibitor_industries`; cascade behavior should match GORM output.  |
| `exhibitor_programs`    | `exhibitors` ↔ `programs`    | created via `many2many:exhibitor_programs`; cascade behavior should match GORM output.    |
| `exhibitor_employments` | `exhibitors` ↔ `employments` | created via `many2many:exhibitor_employments`; cascade behavior should match GORM output. |

## Bootstrap data that must survive the migration

### Required startup seed data

- `roles`
  - `admin` with permissions `[*]`
  - `member` with the current limited profile/team permissions
- `feature_flags`
  - all entries in `models.DefaultFeatureFlags`

### Conditional startup seed data

- initial admin user from `INITIAL_ADMIN_USERNAME` / `INITIAL_ADMIN_PASSWORD`
  - created only when the `users` table is empty
  - currently created **without** assigning a role in code

### Runtime-managed data

- `audit_logs`
  - inserted during audited create/update/delete requests
  - pruning is performed opportunistically in `audit/audit.go` during writes, based on `AUDIT_LOG_RETENTION_DAYS`

## Important migration gotchas

1. `permissions` is logically JSON but declared as `type:text` in the GORM model. The SQL migration should preserve the current behavior unless we intentionally choose to normalize it later.
2. `programs.id` does not carry an explicit `autoIncrement` tag in the model, so we should verify the existing live schema before encoding the SQL definition.
3. the three exhibitor join tables are currently implicit GORM artifacts; we should inspect the live schema before freezing their exact PK/index/FK layout in SQL.
4. `SeedInitialAdminUser` only ensures a user exists; it does not currently assign the seeded user to the `admin` role.
5. the application still calls `AutoMigrate`, but it is now gated by `DB_ENABLE_AUTOMIGRATE` so checked-in SQL migrations can gradually take over as the source of truth.
6. the generated snapshot inherits broad `public` grants from the current Supabase baseline defaults. Before anything reaches production, we should review whether those grants are acceptable and whether ArmadaCMS needs tighter privilege/RLS posture for `public` tables.

## Snapshot status

- Generated migration snapshot: `supabase/migrations/20260504085048_armadacms_application_schema_snapshot.sql`
- Hardening migration: `supabase/migrations/20260504090108_harden_public_schema_access_and_rls.sql`
- Cleanup pass completed: the generated snapshot no longer needs to own access control because grants and RLS now live in the dedicated hardening migration.
- Validation run completed successfully with:
  - `pnpx supabase db reset --local`
  - `pnpx supabase db diff --local` → `No schema changes found`
  - `pnpx supabase db lint -s public --fail-on error`
- Snapshot caveat: it is a generated first cut and still deserves manual review for grants, defaults, and whether any data seeding should move from Go startup into SQL.

## RLS decision

Yes — the ArmadaCMS application tables in `public` should have RLS enabled.

Reasoning:

- Supabase treats `public` as an exposed schema.
- ArmadaCMS currently uses its own Go API and direct Postgres connections rather than relying on Supabase's Data API for these tables.
- That means enabling RLS and revoking broad privileges from `anon` / `authenticated` / `service_role` gives us a safer deny-by-default posture without changing the current application architecture.

Important nuance:

- RLS is **not** the only protection we need. If the backend continues using highly privileged direct database credentials, that backend can still bypass the spirit of end-user row-level rules.
- For the current migration phase, enabling RLS is still worth doing as defense in depth and to avoid accidentally exposing the app tables through Supabase APIs.
- If ArmadaCMS later wants to use Supabase's Data API directly for any of these tables, we should then add explicit grants and table-specific policies instead of relaxing everything globally.

## First migration cut to implement next

The first real ArmadaCMS schema migration should focus on application tables only and leave Supabase-managed schemas alone.

Recommended order:

1. create lookup tables: `roles`, `teams`, `industries`, `programs`, `employments`
2. create auth tables: `users`, `refresh_tokens`
3. create content tables without dependent joins: `profiles`, `exhibitors`, `events`, `recruitment_periods`, `recruitment_roles`, `fair_date_configs`, `feature_flags`, `highlight_cards`, `blogposts`, `audit_logs`
4. create join tables: `exhibitor_industries`, `exhibitor_programs`, `exhibitor_employments`
5. seed bootstrap rows that are environment-safe in `supabase/seed.sql` or a dedicated data migration

## Verification checklist before writing SQL

- inspect the live schema of the current production database and/or a GORM-created local database for exact column types and join-table details
- verify table naming, especially `fair_date_configs` and the three `exhibitor_*` join tables
- confirm whether `roles.permissions` is stored as `text` or was manually migrated to `jsonb` in an existing environment
- decide which bootstrap data belongs in SQL seeds vs. Go startup code during the transition period

## Next deliverable

The next implementation step is to decide where and when `DB_ENABLE_AUTOMIGRATE` should be disabled by default across local, staging, and production environments, then finish the storage rollout details: object path conventions, bucket/policy review, generated access keys per environment, and the actual blob copy from AWS.
