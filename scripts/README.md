# Development and release scripts

[CMS deployment](../.github/workflows/deploy.yml) builds and releases the API/admin container and photo worker. [`release.py`](release.py) coordinates Cloud Run changes; [`smoke_test.py`](smoke_test.py) verifies HTTP responses. Both use Python's standard library.

## Local development

[`dev-up.ps1`](dev-up.ps1) starts local Supabase and the backend; `-Build` rebuilds the backend image. [`dev-down.ps1`](dev-down.ps1) stops the services while preserving local data. Run the admin frontend separately as described in the [project guide](../README.md#getting-started).

Local Compose configuration supplies development-only photo token credentials, a browser-reachable signed-photo endpoint and a SafeSearch mock. reCAPTCHA uses a dedicated Terraform test key with score 1.0 and real Enterprise assessments. Run `scripts/configure-local-recaptcha.ps1` after the resources in `infra/terraform/gcp/local` have been applied to update the gitignored backend and photo-app environment files. The script preserves other settings. The server-only assessment API key is restricted to the reCAPTCHA API; hosted services retain workload identity and their existing site keys.

Local startup uses the active network adapter's IPv4 address for copied photo event links and QR codes. Set `LOCAL_PHOTO_APP_BASE_URL` before startup to override the detected address, for example when using a VPN or multiple network adapters.

Rejected photos leave the gallery immediately and retain their private file for 24 hours. In Rejected, **Approve selected** cancels deletion and approves the photo again. After the deadline, approval is blocked and the next cleanup permanently removes the file; existing already-deleted files cannot be recovered. Hosted cleanup runs hourly through the existing photo worker schedule; the local API checks expired rejections every minute. Already-issued signed image URLs retain their existing expiry while the file is retained.

### Local database import

[`import-remote-db.ps1`](import-remote-db.ps1) replaces the local `public` schema while preserving Supabase's system and Storage schemas. Fill in `SOURCE_DB_*` in `.env`, start the local stack, then run from the repository root:

```powershell
./scripts/import-remote-db.ps1
```

The remote database must be reachable from your machine; Supabase network restrictions may require allowing your IP. Prefer importing staging data to avoid handling production data locally. The imported schema is disposable: `pnpx supabase@2.118.0 db reset --local` replaces it with checked-in migrations and seed data.

## Release flow

[`release-targets.json`](release-targets.json) defines each branch's project, region, service, worker, registry and URLs. `staging` targets Staging and `main` targets Production. Terraform owns runtime configuration, networking and federation; Actions owns release images, generated revision names, preview tags and traffic.

The image builds alongside CI. Deployment requires successful Go, frontend, Supabase, CodeQL and release checks for the same commit. The branch head is checked before building, during recovery preparation and immediately before activation. Each run/attempt has a unique release identity; API and worker use the same registry digest.

Before Cloud Run changes, Actions persists a recovery snapshot. The release then creates a revision without ordinary traffic and assigns the fixed `release-preview` tag. Existing `LATEST` traffic is pinned to the previously serving explicit revisions. Staging tests the tagged `run.app` URL; production tests `https://preview.cms.armada.nu` through the load balancer.

After preview verification, the release updates and verifies the worker, moves traffic to the tested revision, and checks the live domain. Only the release job uses the GitHub Environment; the parallel build identity cannot modify Cloud Run. Per-environment concurrency serializes releases without cancelling an active release.

## HTTP verification

Each preview/live check round has a shared two-minute deadline and verifies:

- `/health`: HTTP 200, expected commit/revision headers and no-store caching. Liveness remains database-independent.
- `/api/v1/featureflags`: valid JSON, `Content-Range` and pagination.
- `/admin/`: HTML and JavaScript/CSS entrypoints with appropriate content types; SPA fallback cannot mask missing assets.

Production preview DNS/TLS is checked before Cloud Run changes. Identity headers confirm routing to the candidate. These read-only checks do not execute React or authenticated admin operations.

## Failure and recovery

Actions retains two artifacts for 90 days, named with branch, run ID and attempt:

- `cms-restore-…`: previous revision traffic, worker digest, preview tags, resource etags, configuration fingerprints and release identities. Upload must succeed before deployment; subsequent external resource changes block mutation.
- `cms-result-…`: release phase and recovery outcome, also shown in the Actions summary. Metadata excludes tokens, environment dumps and raw resource configuration.

Preview failure restores the previous preview tag without updating the worker or ordinary traffic. Activation/live failure restores and verifies traffic, worker digest and preview tags. Ambiguous API failures are reconciled against observed state; etag and configuration checks prevent overwriting unexpected external edits. Older revisions without identity headers receive functional HTTP checks alongside Cloud Run traffic verification.

A release remains failed after successful rollback. Failure to publish the final result also fails the job, but does not roll back an otherwise verified release. Forced runner termination can require manual recovery using the independent recovery snapshot.

Preview is publicly reachable with existing application authorization and shares the environment's database and startup seeding. Rollback does not reverse database writes or already-started worker executions. Schema and worker changes remain backward compatible; hosted database migrations run separately from application release. Actions concurrency does not prevent manual GCP changes.

## Script tests

From the repository root:

```bash
python -m unittest discover -s scripts/tests -t .
```

Tests simulate external services and release failures; HTTP tests use a local server. Script entrypoints use `python -m scripts.release` and `python -m scripts.smoke_test`.
