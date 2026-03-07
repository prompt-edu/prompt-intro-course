# PROMPT Intro Course - AI Assistant Guide

This document provides essential context for AI assistants working on the standalone PROMPT Intro Course repository.

## Project Overview

**PROMPT Intro Course** is the extracted intro-course part of the PROMPT platform. It contains:

- **Client**: intro-course developer micro-frontend (Webpack Module Federation remote)
- **Server**: intro-course backend service (Go + Gin + PostgreSQL)
- **Deployment**: dedicated Docker Compose and GitHub Actions workflows for intro-course services only

For full platform context (core app, other course phases, shared architecture), see the main PROMPT repository:

- <https://github.com/prompt-edu/prompt>

## Repository Structure

```text
client/                       # Intro course developer micro-frontend (port 3005)
  routes/                     # Remote route registration for PROMPT core shell
  sidebar/                    # Sidebar registration for PROMPT core shell
  src/introCourse/            # Intro course feature pages, hooks, state, network calls

server/                       # Intro course backend (default localhost:8082)
  developerProfile/           # Student profile survey and device data
  seatPlan/                   # Seat plan creation + assignment
  tutor/                      # Tutor import + GitLab username management
  infrastructureSetup/        # GitLab course/student repo setup
  config/                     # prompt-sdk config endpoint registration
  copy/                       # prompt-sdk copy endpoint registration
  db/
    migration/                # SQL migrations
    query/                    # sqlc query definitions
    sqlc/                     # generated sqlc code

.github/workflows/            # CI/CD for intro-course repo
```

## Quick Start Commands

### Environment

```bash
cp .env.template .env
```

### Database (local)

```bash
docker compose up -d
```

### Client

```bash
cd client
yarn dev
yarn lint
yarn build
```

### Server

```bash
cd server
go run main.go
go test ./...
```

The PROMPT core app (from the main repository) can load this remote at:

- `http://localhost:3005/remoteEntry.js`

## Environment Variables

Commonly relevant variables from `.env.template`:

- Runtime/URLs: `CORE_HOST`, `SERVER_CORE_HOST`, `SERVER_IMAGE_TAG`, `CLIENT_IMAGE_TAG`
- DB: `DB_NAME`, `DB_USER`, `DB_PASSWORD` (plus host/port defaults in server)
- Auth: `KEYCLOAK_HOST`, `KEYCLOAK_REALM_NAME`, `KEYCLOAK_AUTHORIZED_PARTY`, `KEYCLOAK_CLIENT_ID`
- Integrations: `GITLAB_ACCESS_TOKEN`, `SENTRY_DSN`

Server defaults are resolved in `server/utils/getEnv.go` and `server/main.go`.

## Technology Stack

### Frontend (`client/`)

- React + TypeScript
- Webpack 5 + Module Federation
- TanStack React Query
- Zustand
- Tailwind CSS
- Shared PROMPT packages: `@tumaet/prompt-ui-components`, `@tumaet/prompt-shared-state`

### Backend (`server/`)

- Go 1.26
- Gin framework
- PostgreSQL (`pgx`)
- `sqlc` for typed query generation
- `golang-migrate` for DB migrations
- `prompt-sdk` for auth middleware and shared endpoint contracts

## Code Conventions

### Client-Side

**Naming:**

- PascalCase for React components and component folders
- camelCase for utilities, functions, variables
- SCREAMING_SNAKE_CASE for constants

**Types:**

- Prefer `interface` for object structures
- Avoid `any`
- Keep domain interfaces close to feature modules (`src/introCourse/**/interfaces`)

**State and data:**

- Use React Query for server state and request lifecycle
- Use Zustand (`useIntroCourseStore`) for local intro-course state
- Keep API access in `src/introCourse/network/`

**Route + sidebar contract:**

- Maintain exports used by host shell:
  - `client/routes/index.tsx`
  - `client/sidebar/index.tsx`
  - `client/src/provide/index.ts`
- Keep required permissions aligned with backend role checks.

### Server-Side

**Naming:**

- PascalCase: exported types/functions
- camelCase: internal functions/variables
- snake_case: DB schema, table, and column names

**Module structure:**

```text
module/
  moduleDTO/        # request/response DTOs
  router.go         # Gin routes + auth middleware
  service.go        # business logic
  validation.go     # input validation
  *_test.go         # tests
```

**Database access with sqlc:**

1. Update SQL in `server/db/query/*.sql`
2. Run `sqlc generate` in `server/`
3. Use generated methods from `server/db/sqlc`

**Migrations:**

- Migrations are in `server/db/migration/*.sql`
- Server startup runs migrations via:

```bash
migrate -path ./db/migration -database $DATABASE_URL up
```

## Intro-Course Specific Functional Areas

### Client pages

Primary intro-course pages include:

- Student flow: developer profile survey + own seat assignment
- Lecturer/admin pages:
  - Participants
  - Developer Profiles
  - Tutor Import
  - Seat Assignments
  - Mailing

### Server modules

- `developerProfile`: student profile CRUD and exported device lists
- `seatPlan`: seat plan creation, update, delete, own assignment lookup
- `tutor`: tutor import and GitLab username updates
- `infrastructureSetup`: GitLab course setup and per-student repo setup/status
- `copy`: copy endpoint registration for course-phase duplication
- `config`: config endpoint registration for phase-level configuration

## API and Auth Patterns

### Base route groups

- Course-phase scoped APIs:
  - `intro-course/api/course_phase/:coursePhaseID/*`
- Copy endpoint group:
  - `intro-course/api/*`

### Auth middleware

Initialize once in `main.go`:

```go
promptSDK.InitAuthenticationMiddleware(keycloakURL, realm, coreURL)
```

Apply per route with explicit roles, e.g.:

```go
authMiddleware(promptSDK.PromptAdmin, promptSDK.CourseLecturer)
```

Roles used in this repo:

- `PromptAdmin`
- `CourseLecturer`
- `CourseStudent`

## Module Federation Pattern (Client)

In `client/webpack.config.ts`:

- Remote name: `intro_course_developer_component`
- Dev port: `3005`
- Exposes:
  - `./routes`
  - `./sidebar`
  - `./provide`

When changing exposed modules or remote name, update host integration in main PROMPT repo accordingly.

## CI/CD

Relevant workflows in `.github/workflows/`:

- `lint-server.yml`: server linting
- `dev.yml`: PR/push pipeline (lint, test, build, deploy to dev on push)
- `build-and-push.yml`: builds/pushes server and client images
- `deploy.yml`: deploys intro-course services via `docker-compose.prod.yml`
- `prod.yml`: production release/deploy flow

Container images built by CI:

- `ghcr.io/prompt-edu/prompt-intro-course/prompt-server-intro-course`
- `ghcr.io/prompt-edu/prompt-intro-course/prompt-clients-intro-course-developer`

## Testing

### Server

```bash
cd server
go test ./...
```

Notes:

- Tests use `testcontainers-go` for DB-backed integration tests.
- Test helpers live in `server/testutils/`.
- `database_dumps/intro_course.sql` is used for seeded scenarios.

### Client

- Lint before merging client changes:

```bash
cd client
yarn lint
```

## Reusable Libraries (Prefer Over Custom Implementations)

Use shared PROMPT libraries whenever possible before introducing new primitives.

### `@tumaet/prompt-ui-components`

- Shared UI component set used throughout intro-course pages.
- Already included via imports and Tailwind config.

### `@tumaet/prompt-shared-state`

- Shared domain types and role constants.
- Used for permission checks and course-phase participation models.

### `prompt-sdk` (`github.com/prompt-edu/prompt-sdk`)

- Authentication middleware
- Shared endpoint registration (`promptTypes.RegisterCopyEndpoint`, `promptTypes.RegisterConfigEndpoint`)
- Utility helpers used across services

## Important Notes

- Keep client route permissions and server role checks consistent.
- Keep generated sqlc code in sync with query files and migrations.
- Do not change API base paths unless host-shell and deployment routing are updated together.
- Do not commit secrets; `.env.template` only contains placeholders.
- For broader architectural decisions, refer to the main PROMPT repository:
  - <https://github.com/prompt-edu/prompt>
