# Backend

This directory is reserved for the Go backend implementation.

For now, the contract and data-model source files remain at repository root:

- `../TramplinAPI.yaml`
- `../Tramplin.dbml`

The proposed backend layout, package boundaries, and ownership rules are documented in [../docs/backend-project-structure.md](../docs/backend-project-structure.md).

Current implementation status:

- startup now opens a PostgreSQL connection pool and fails fast if the database is unreachable
- working stdlib Go HTTP server
- working auth cookie flow
- working applicant profile, privacy, tags, social links, connections, and recommendation endpoints
- working authenticated applicant profile visibility endpoints, including employer review access for applicants who applied
- working applicant saved opportunities and saved companies endpoints
- working employer profile, companies, company social links, company media, and membership workflow endpoints
- working employer company verification-request endpoints and curator verification review/list endpoints
- working employer custom-tag creation plus curator tag-management endpoints
- working curator moderation-case list/create/update endpoints
- working curator user/applicant/employer/company/opportunity correction endpoints
- working admin curator-account list/create/update endpoints with single-administrator protection
- working applicant and employer application workflow endpoints
- working employer and public opportunity endpoints backed by PostgreSQL, including explicit activate/close/archive lifecycle actions and public map geo filters
- working employer notification campaign draft/update/send/cancel endpoints with in-app notification fanout
- working public companies, public tags, locations, notification preferences, and notification list/read endpoints
- working upload/media metadata/download-url flow with PostgreSQL-backed `media_files` records
- no remaining route-level `501 not_implemented` stubs in the current API surface

Database configuration:

- The backend accepts either `TRAMPLIN_DATABASE_URL` or granular PostgreSQL env vars.
- If `TRAMPLIN_DATABASE_URL` is not set, the backend builds one from:
  `TRAMPLIN_DATABASE_HOST`, `TRAMPLIN_DATABASE_PORT`, `TRAMPLIN_DATABASE_USER`,
  `TRAMPLIN_DATABASE_PASSWORD`, `TRAMPLIN_DATABASE_NAME`, and `TRAMPLIN_DATABASE_SSLMODE`.
- Default local fallback is:
  `postgres://postgres@127.0.0.1:5432/tramplin?sslmode=disable`

Local PostgreSQL bootstrap example:

```bash
cd backend
make db-init
```

`make db-init` initializes the local cluster if needed, starts PostgreSQL, creates the
`tramplin` role/database, and applies migrations.

On later runs you usually only need:

```bash
cd backend
make db-start
```

Run locally:

```bash
cd backend
make infra-start
make db-migrate
make seed-dev
make run
```

Run the fully containerized stack:

```bash
cd backend
make docker-bootstrap
```

This starts:

- `api` on `http://127.0.0.1:8080`
- containerized PostgreSQL on `127.0.0.1:5433`
- containerized MinIO API on `http://127.0.0.1:9100`
- containerized MinIO console on `http://127.0.0.1:9101`

Useful commands:

```bash
cd backend
make docker-build
make docker-up
make docker-seed
make docker-bootstrap
make docker-logs
make docker-ps
make docker-down
```

The Compose stack uses separate host ports for PostgreSQL and MinIO, while the API stays on `8080`.
`make docker-bootstrap` is the quickest way to start the stack and load the seeded demo dataset.

The container image:

- builds the API, migration, and seeder binaries;
- runs embedded SQL migrations automatically before starting the API;
- rewrites client-facing MinIO presigned URLs to the host-visible endpoint in Compose mode;
- can also be used as a one-shot migration or seed container.

Docker files:

- `Dockerfile`
- `compose.yml`
- `docker/entrypoint.sh`

GitHub Actions container publishing:

- `.github/workflows/backend-container.yml` runs Go tests, PostgreSQL-backed integration tests, and the real MinIO media roundtrip tests before the image job.
- Pull requests build the image for validation only after the test job passes.
- Pushes to the default branch publish `latest`, branch, and `sha-*` tags to
  `ghcr.io/<repo-owner>/tramplin-api` only if the test job passes.
- Git tags matching `v*` also publish a matching version tag.
- The workflow builds both `linux/amd64` and `linux/arm64`.

Development seeder:

- `make seed-dev` inserts a repeatable local dataset for manual QA and frontend integration.
- The seeder is idempotent for its own records and can be re-run after migrations or code changes.
- If object storage is configured and reachable, it also uploads a small set of sample media assets.
- Shared seeded password: `password123`
- Seeded users:
  `seed.admin.curator@tramplin.local`,
  `seed.curator@tramplin.local`,
  `seed.owner@tramplin.local`,
  `seed.recruiter@tramplin.local`,
  `seed.applicant.anna@tramplin.local`,
  `seed.applicant.boris@tramplin.local`,
  `seed.applicant.clara@tramplin.local`
- Seeded content includes:
  one verified company (`seed-labs`),
  two public opportunities (`junior-go-backend-internship`, `spring-career-meetup`),
  one draft mentor program (`backend-mentor-circle`),
  applications, saved entities, applicant social links, connections, recommendations,
  notifications, a notification campaign, a verification request, and a moderation case.

Object storage configuration:

- Upload/download routes use an S3-compatible object store.
- The backend reads:
  `TRAMPLIN_OBJECT_STORAGE_ENDPOINT`, `TRAMPLIN_OBJECT_STORAGE_ACCESS_KEY`,
  `TRAMPLIN_OBJECT_STORAGE_SECRET_KEY`, `TRAMPLIN_OBJECT_STORAGE_BUCKET`,
  `TRAMPLIN_OBJECT_STORAGE_USE_SSL`, `TRAMPLIN_OBJECT_STORAGE_REGION`,
  `TRAMPLIN_OBJECT_STORAGE_PUBLIC_URL`, `TRAMPLIN_OBJECT_STORAGE_PUBLIC_USE_SSL`,
  `TRAMPLIN_UPLOAD_URL_TTL`, and `TRAMPLIN_DOWNLOAD_URL_TTL`.
- `TRAMPLIN_OBJECT_STORAGE_ENDPOINT` is the backend-to-storage endpoint.
- `TRAMPLIN_OBJECT_STORAGE_PUBLIC_URL` is optional and rewrites presigned upload/download URLs
  to a host-visible address for clients. This is what makes the Compose MinIO setup usable
  from Postman or a browser on the host machine.
- If none of the object-storage vars are set, the backend still boots, but
  upload/download-url endpoints return `503 service_unavailable`.
- If object-storage config is partially set, startup fails fast.
- For local development, MinIO can be started with:

```bash
cd backend
make minio-start
```

- Stop or inspect it with:

```bash
cd backend
make minio-status
make minio-stop
```

- Local MinIO defaults:
  `TRAMPLIN_OBJECT_STORAGE_ENDPOINT=127.0.0.1:9000`
  `TRAMPLIN_MINIO_CONSOLE_ADDRESS=127.0.0.1:9001`
  `TRAMPLIN_OBJECT_STORAGE_ACCESS_KEY=minioadmin`
  `TRAMPLIN_OBJECT_STORAGE_SECRET_KEY=minioadmin`
  `TRAMPLIN_OBJECT_STORAGE_BUCKET=tramplin-media`
- The backend creates the bucket automatically on startup when the credentials are valid.

Run integration tests:

```bash
cd backend
make test-integration
```

Run the real-MinIO media roundtrip tests:

```bash
cd backend
make infra-start
make test-integration-minio
```

`make test-integration` uses the fake object-store harness for fast contract coverage.
`make test-integration-minio` runs the upload/download media tests against the real local MinIO server.

Health checks:

```bash
curl -sS http://127.0.0.1:8080/health/live
curl -sS http://127.0.0.1:8080/health/ready
```

Current persistence note:

- The implemented auth, uploads/media metadata access, applicant profile/privacy/visibility/tags/social links/connections/recommendations/saved entities, application workflow, employer profile/company/company-social/company-media/membership/verification-request, curator verification review/list, curator user/applicant/employer/company/opportunity correction flows, admin curator-account flows, employer custom-tag creation, curator tag management, curator moderation-case flows, opportunity and lifecycle-action flows, employer notification campaign flows, public company/tag/opportunity, location, and notification preference/list/read flows are backed by PostgreSQL.

Local dev database files:

- cluster data lives under `backend/.local/postgres/data`
- runtime log lives at `backend/.local/postgres/postgres.log`
- local environment lives in `backend/.env`
- local MinIO data lives under `backend/.local/minio/data`
- local MinIO log lives at `backend/.local/minio/minio.log`
