# Backend Project Structure

This document models the backend-only project structure for a Go implementation of the Tramplin platform.

Current constraint: the frontend team is working separately. The backend should therefore be built as an independent service with a stable contract boundary and without frontend-specific coupling.

## Goals

- Keep the HTTP contract stable and centered on `TramplinAPI.yaml`.
- Keep the data model explicit and centered on `Tramplin.dbml` plus SQL migrations.
- Isolate generated code from hand-written business logic.
- Organize code by domain, not by controller or database table only.
- Make it easy for a backend team to work without needing the frontend repo.

## Source Of Truth

Until the repository is split further, these files remain the shared source of truth:

- `TramplinAPI.yaml`: HTTP contract
- `Tramplin.dbml`: documented domain/data model
- `OriginalTechSpec.md`: original product requirements

Runtime truth for the database is the SQL migration history. If a migration and `Tramplin.dbml` ever diverge, fix the DBML in the same change.

## Recommended Repository Layout

```text
.
├── TramplinAPI.yaml
├── Tramplin.dbml
├── OriginalTechSpec.md
├── docs/
│   └── backend-project-structure.md
└── backend/
    ├── go.mod
    ├── go.sum
    ├── Makefile
    ├── README.md
    ├── configs/
    │   ├── local.yaml
    │   └── example.env
    ├── cmd/
    │   ├── api/
    │   │   └── main.go
    │   ├── migrate/
    │   │   └── main.go
    │   └── seed/
    │       └── main.go
    ├── api/
    │   └── openapi/
    │       ├── oapi-codegen.yaml
    │       └── generate.go
    ├── migrations/
    │   └── *.sql
    ├── sql/
    │   ├── queries/
    │   │   ├── auth.sql
    │   │   ├── applicant.sql
    │   │   ├── company.sql
    │   │   ├── opportunity.sql
    │   │   ├── application.sql
    │   │   ├── notification.sql
    │   │   ├── moderation.sql
    │   │   ├── upload.sql
    │   │   ├── location.sql
    │   │   └── tag.sql
    │   └── sqlc.yaml
    ├── internal/
    │   ├── app/
    │   │   ├── bootstrap.go
    │   │   ├── container.go
    │   │   └── routes.go
    │   ├── config/
    │   │   └── config.go
    │   ├── platform/
    │   │   ├── auth/
    │   │   ├── clock/
    │   │   ├── db/
    │   │   ├── logger/
    │   │   ├── objectstorage/
    │   │   ├── pagination/
    │   │   ├── tx/
    │   │   └── validator/
    │   ├── transport/
    │   │   └── http/
    │   │       ├── openapi/
    │   │       │   ├── server.gen.go
    │   │       │   └── types.gen.go
    │   │       ├── middleware/
    │   │       ├── handlers/
    │   │       ├── presenters/
    │   │       └── router.go
    │   ├── domain/
    │   │   ├── auth/
    │   │   ├── applicant/
    │   │   ├── company/
    │   │   ├── opportunity/
    │   │   ├── application/
    │   │   ├── notification/
    │   │   ├── curation/
    │   │   ├── upload/
    │   │   ├── location/
    │   │   └── tag/
    │   └── store/
    │       └── postgres/
    │           ├── sqlc/
    │           ├── repositories/
    │           └── tx.go
    ├── test/
    │   ├── integration/
    │   └── e2e/
    └── scripts/
        ├── generate.sh
        └── test.sh
```

## What Goes Where

### `backend/cmd`

Thin entrypoints only.

- `cmd/api`: starts the HTTP server
- `cmd/migrate`: applies migrations
- `cmd/seed`: optional local/dev seed data

No business logic should live here.

### `backend/internal/app`

Application wiring.

- dependency construction
- service registration
- router assembly
- graceful shutdown

This is the place that knows how the whole system is put together.

### `backend/internal/platform`

Reusable infrastructure code.

Examples:

- JWT and cookie helpers
- CSRF / cross-origin protection
- PostgreSQL connection bootstrap
- object storage adapter
- logging
- validation
- transaction manager
- time abstraction for tests

This layer should not know Tramplin business rules.

### `backend/internal/transport/http`

HTTP adapter layer only.

- request decoding
- auth/context extraction
- response encoding
- mapping domain errors to HTTP status codes
- OpenAPI-generated server interface and DTO types

Recommended split:

- `openapi/`: generated from `../TramplinAPI.yaml`
- `handlers/`: one handler group per domain
- `middleware/`: auth, logging, request ID, CORS, CSRF, panic recovery
- `presenters/`: mapping domain models to API response shapes when needed

Handlers should call domain services. They should not contain SQL and should not directly implement workflow rules.

### `backend/internal/domain`

Business logic layer.

Each domain package should usually contain:

- `model.go`: core entities/value objects
- `service.go`: use cases
- `repository.go`: storage interfaces
- `errors.go`: typed domain errors
- `policy.go` or `rules.go`: workflow/permission rules

Recommended domain boundaries for the current API:

- `auth`: register, login, refresh, logout, current user
- `applicant`: profile, privacy, social links, tags, saved items, connections, recommendations
- `company`: employer profile, companies, memberships, social links, media, verification requests
- `opportunity`: public catalog, employer opportunity CRUD, lifecycle actions
- `application`: apply, withdraw, employer review flow, status history
- `notification`: in-app notifications, preferences, campaigns
- `curation`: moderation cases, curator review flows, curator corrections, admin curator accounts
- `upload`: presign, complete, metadata, download URL, deletion
- `location`: search and normalized location creation
- `tag`: public tags, employer tags, curator tag management

Important rule: domain packages own the lifecycle rules from the spec, for example:

- company membership approval/rejection/revocation
- opportunity status transitions
- application status transitions
- verification request review rules
- curator single-admin invariant

### `backend/internal/store/postgres`

PostgreSQL implementation of repository interfaces.

Suggested split:

- `sqlc/`: generated query code
- `repositories/`: domain-specific repository implementations that wrap sqlc
- `tx.go`: transaction coordination used by services

This layer translates between database rows and domain models.

### `backend/sql`

SQL owned by developers and used by sqlc.

- `sql/queries`: hand-written SQL by domain
- `sqlc.yaml`: generation config

Prefer explicit SQL over a heavy ORM for this project. The API and workflows are already explicit, and the data model is relational and audit-heavy.

### `backend/migrations`

Executable schema history.

This is the runtime database source of truth. Every structural change should land here first, with `Tramplin.dbml` updated in the same change for documentation parity.

### `backend/test`

Two test layers are enough for MVP:

- `integration/`: service + repository tests against a real PostgreSQL instance
- `e2e/`: HTTP tests against the running app

Avoid over-investing in handler-only unit tests when most risk is in lifecycle and permission logic.

## Generated Code Rules

Generated code should be isolated and never edited manually.

Generated:

- `backend/internal/transport/http/openapi/*.gen.go`
- `backend/internal/store/postgres/sqlc/*`

Hand-written:

- domain services
- repositories
- migrations
- SQL queries
- middleware
- handlers
- app wiring

This separation matters because your contract and schema will evolve during MVP.

## Frontend Separation Rules

Because the frontend team is separate for now:

- do not build a frontend-specific BFF layer
- do not put UI state in backend packages
- do not expose database row structs directly as API DTOs
- do not let frontend needs dictate internal package boundaries

The only shared boundary is the HTTP contract in `TramplinAPI.yaml`.

## Suggested Implementation Order

Build the backend in this order:

1. App bootstrap, config, logging, database, auth middleware
2. Auth endpoints and current-user endpoint
3. Upload flow and location/tag helpers
4. Public company/opportunity catalog
5. Employer profile, company, and membership workflow
6. Opportunity management and lifecycle actions
7. Applicant applications and employer review flow
8. Verification requests and curator review flows
9. Notifications and campaigns
10. Remaining curator correction/admin endpoints

This order matches the highest-value dependencies in the current API.

## Practical Naming Guidance

Use package names that reflect business language already present in the API:

- `company`, not `employer_company_management`
- `application`, not `job_application_controller`
- `curation`, not `admin_misc`

Keep packages broad enough to own a workflow, but not so broad that unrelated rules end up mixed together.

## Minimal Initial Backend Deliverables

Before implementing all endpoints, the backend repo should at least contain:

- `backend/go.mod`
- `backend/Makefile`
- OpenAPI generation config
- sqlc config
- migration runner
- app bootstrap
- one implemented vertical slice end-to-end

Recommended first vertical slice:

- `POST /auth/register/employer`
- `POST /auth/login`
- `GET /auth/me`
- `POST /employer/companies`
- `GET /employer/companies`

That slice exercises auth, cookies, DB access, company creation, and membership creation.
