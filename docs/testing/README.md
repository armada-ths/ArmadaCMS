# Integration tests

Run commands from the repository root. Native Go tests require Go and libvips, matching the backend test environment in CI. For ordinary Go, frontend and release-script tests, see the [project guide](../../README.md#testing).

## Storage

With the local development stack running, this opt-in test uploads, downloads and removes a test object from local Supabase Storage:

```powershell
$env:RUN_S3_INTEGRATION = "1"
$status = pnpx supabase@2.118.0 status -o env
$env:AWS_ACCESS_KEY_ID = ($status | Select-String '^S3_PROTOCOL_ACCESS_KEY_ID=').Line.Split('=', 2)[1].Trim('"')
$env:AWS_SECRET_ACCESS_KEY = ($status | Select-String '^S3_PROTOCOL_ACCESS_KEY_SECRET=').Line.Split('=', 2)[1].Trim('"')
go test -tags=integration -count=1 ./utils
```

CI runs the test inside the existing backend container; it does not require Go or libvips on the runner. Local native runs use the credentials above and the endpoints from the local environment.

## PostgreSQL behavior tests

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
