# Translation review development

The authenticated **Website content > Translations** page reviews English-to-Swedish content. It covers the 27 stored field pairs in `internal/translations/registry.go` (15 resource types), including unpublished and historical records. Exhibitor-owned descriptions support manual review only; neither the API nor worker permits AI generation for them. The public content endpoints and original English fields keep their existing formats.

## Configuration

Generation is disabled unless `TRANSLATION_AI_ENABLED=true`. Manual drafting, rejection and approval work without AI. The database migration is required for the review page even when generation is disabled. `TRANSLATION_WORKER_MODE=1` runs the queue worker instead of the HTTP server.

| Variable                         | Default                 | Purpose                                                 |
| -------------------------------- | ----------------------- | ------------------------------------------------------- |
| `GOOGLE_CLOUD_PROJECT`           | unset                   | Vertex AI project                                       |
| `TRANSLATION_REGION`             | `eu`                    | EU Vertex endpoint/location                             |
| `TRANSLATION_MODEL`              | `gemini-3.5-flash-lite` | Model for new jobs                                      |
| `TRANSLATION_WORKER_JOB_NAME`    | unset                   | Full Cloud Run Job resource name for immediate dispatch |
| `TRANSLATION_DAILY_TOKEN_BUDGET` | `1000000`               | Shared Stockholm calendar-day token cap                 |
| `TRANSLATION_MAX_RECORDS`        | `50`                    | Maximum records per request, hard upper limit 50        |
| `TRANSLATION_MAX_OUTPUT_TOKENS`  | `8192`                  | Per-call output limit, including thinking               |

Cloud Run obtains access tokens from its runtime service account through the metadata server. No browser AI credentials or model tools are used. The optional Terraform worker is controlled by `enable_translation_worker`; `translation_ai_enabled` defaults to false and its recovery schedule remains paused. The worker and API use the same image; release preflight, image updates and rollback include the optional translation job when its full resource name is configured on the API. Terraform does not manage subsequent release image changes.

The worker claims only explicitly requested jobs. Database leases, two unique running slots and an advisory lock enforce concurrency across executions. Each attempt reserves a conservative UTF-8 byte upper bound for request tokens plus protocol headroom and the entire output allowance. Provider-reported total usage settles that reservation, including thinking. Unknown usage and expired leases charge the complete reservation. Retryable failures have at most three attempts; retries reserve budget separately. Budget wait resumes at the next Stockholm midnight. The recovery schedule creates no new requests.

## Rules and drafts

Edit `internal/translations/prompt.md` and `glossary.json` through normal code review. They are embedded in the binary. New jobs store immutable text snapshots and content hashes, and queued jobs use their saved instructions even after a deployment. The initial glossary and examples were reviewed against the public site's English and Swedish catalogs; no runtime dependency on that repository exists. Terminology warnings assist the reviewer; they never perform string replacements.

Generation writes proposals only. Regeneration retains previous proposal rows, marking current drafts superseded. Explicit regeneration can propose a replacement for approved AI-assisted text while keeping its published value until approval; manually authored translations remain excluded. Approval checks source fingerprints, target values and draft versions inside the same transaction as the content update and audit entry. It publishes only selected fields on one record, with normal cache invalidation after commit. Conflict responses are HTTP 409 and do not publish any selected field. Approved text stays published when its original changes; review statuses are computed against current source/target values. Deleted content removes its proposals through database triggers; running reservations settle before orphan jobs are removed.

Local unsaved edits survive resource/page changes while the CMS remains open; the browser warns before unloading. Save draft persists selected fields without publication. Discard local edits reloads the latest baseline after a conflict. Direct table access is revoked from public Supabase roles and RLS is enabled. Existing resource view/edit permissions protect every review operation.

## Verification

- `go test -race -count=1 ./...` runs mocked provider, policy, registry and backend regression tests.
- Set `TRANSLATION_REVIEW_TEST_DSN` to a disposable **local** PostgreSQL database named `translation_review_test` to run the review migration, approval, rejection, version, concurrency and budget tests. The fixture replaces its translation tables; never point it at application data.
- In `frontend/`, run `pnpm run lint:check`, `pnpm run type-check`, `pnpm run format:check`, `pnpm run test` and `pnpm run build`.
- Install Chromium with `pnpm exec playwright install chromium`, then run `pnpm exec vitest run --config vitest.browser.config.ts src/components/Translations/Translations.browser.tsx` for the paired-field review flow. Browser tests mock AI/API writes.
- `python -m unittest discover -s scripts/tests -t .` verifies image release and rollback with mocked external writes.
- `terraform fmt -check -recursive infra/terraform/gcp` and `terraform validate` in each changed root validate infrastructure without applying it.

A manually reviewed live evaluation is required before activation. Automated tests validate structure, preservation and workflow, not unrestricted translation quality. Keep generation disabled until real labels, prose, Markdown, HTML and Armada terminology have been reviewed.
