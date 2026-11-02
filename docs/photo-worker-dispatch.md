# Photo worker dispatch

The recovery schedule checks for work every five minutes through `POST /api/v1/internal/photos/dispatch` on the existing CMS API. Empty checks return 204 without creating a Cloud Run Job execution. The same read-only check is used for direct starts after an export is requested.

A worker starts only when there are queued exports for a non-deleting event, expired running export leases, expired completed exports, rejected photos due for deletion, pending event deletion that is not blocked by a running export, or original photos missing thumbnails that the worker can process. The worker keeps the existing claim, retention, image-quality and deletion logic. A concurrent worker can consume work after the dispatch snapshot; the check does not replace worker locking.

## Authentication and configuration

The endpoint accepts Google-signed OIDC tokens for the exact `PHOTO_DISPATCH_AUDIENCE` and verified `PHOTO_DISPATCH_SERVICE_ACCOUNT` email. CMS user JWTs cannot call it. Terraform sets these identifiers from `photo_dispatch_url` and the configured Scheduler service account. No shared secret is introduced.

Local developers can run the photo worker with its existing environment settings. Hosted scheduling requires the dispatcher API and matching infrastructure configuration; the default URL is the normal CMS hostname for each environment. Use the environment's `photo_dispatch_url` variable when using another hostname. All settings describe recurring configuration, not a requirement to activate cloud resources during development.

## Verification

Set `PHOTO_DISPATCH_TEST_DSN` to a disposable local PostgreSQL database named `photo_dispatch_test` and run `go test -race -count=1 ./Controllers -run TestPhotoWorkerDispatchQueueCheck`. The fixture replaces its photo tables. Never point it at application data.

`go test -race -count=1 ./...` also covers OIDC signature/audience/account validation and rejection of CMS user tokens. External worker starts are mocked in tests. Run `terraform fmt -check -recursive infra/terraform/gcp` and `terraform validate` in both changed roots; no infrastructure apply is needed for local validation.
