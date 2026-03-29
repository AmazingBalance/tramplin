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
- working applicant profile, privacy, and tags endpoints
- working employer profile, companies, and membership workflow endpoints
- working employer and public opportunity endpoints backed by PostgreSQL
- working public companies, public tags, locations, and notification preferences endpoints
- explicit `501 not_implemented` responses for the remaining API surface

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
make run
```

Run integration tests:

```bash
cd backend
make test-integration
```

Health checks:

```bash
curl -sS http://127.0.0.1:8080/health/live
curl -sS http://127.0.0.1:8080/health/ready
```

Current persistence note:

- The implemented auth, applicant profile/privacy/tags, employer profile/company/membership, opportunity, public company/tag/opportunity, location, and notification-preference flows are backed by PostgreSQL.
- The remaining unimplemented routes still return `501 not_implemented`.

Local dev database files:

- cluster data lives under `backend/.local/postgres/data`
- runtime log lives at `backend/.local/postgres/postgres.log`
- local environment lives in `backend/.env`
