# End-to-End Tests

Playwright tests that drive the **real** intro-course module inside the **real** PROMPT core
shell, against a real Keycloak and real Postgres databases. Everything runs in Docker Compose,
including the Playwright runner itself, so a local run and a CI run are identical.

## Running it

From the repository root:

```bash
make test-e2e        # build, run the suite, tear down; exit code is the suite result
make test-e2e-ui     # Playwright UI mode, served out of the runner container
make test-e2e-down   # tear down a stack left running
```

`make test-e2e-ui` publishes the UI on **`http://127.0.0.1:8123`** — use `127.0.0.1`, not
`localhost`, which resolves to `::1` first while the published port is IPv4. In UI mode
`e2e/tests` and `e2e/src` are bind-mounted, so edits are picked up without rebuilding.

To run a single file or grep for a test:

```bash
docker compose -f docker-compose.e2e.yml --env-file e2e/.env.e2e run --rm \
  e2e-runner npx playwright test tests/intro-course/mf-smoke.spec.ts
```

To poke at the stack by hand:

```bash
docker compose -f docker-compose.e2e.yml --env-file e2e/.env.e2e up -d
open http://localhost:4100     # core shell; log in as course-lecturer / course-lecturer
curl localhost:18190/api/hello                 # core server
curl localhost:4100/intro-course/api/info      # intro-course server, through the proxy
```

Compose reads its variables from `--env-file e2e/.env.e2e`, but **exported shell variables
still win**. If a stray `DB_HOST` or `CORE_HOST` is exported in your shell, unset it first.

## How it fits together

```
                    e2e-runner  (Chromium, mcr.microsoft.com/playwright)
                              │  BASE_URL=http://client-core
                              ▼
  client-core (:4100)  ── nginx override ──┬─ /intro-course/api/       → server-intro-course:8080
   ghcr prebuilt image                     ├─ /intro-course-developer/ → client-intro-course:80
   env.js: INTRO_COURSE_HOST= (same-origin)├─ /api/                    → server-core:8080
                                           └─ /                        → SPA fallback
                              │
  server-core (:18190)  ghcr prebuilt image
      ├── db                 (core Postgres, seeded from seed/core/)
      ├── keycloak (:18181)  (realm imported from keycloak/realm.json) ── keycloak-db
      └── seaweedfs-{master,volume,filer,s3}
  server-intro-course   built from ../server ── db-intro-course (migrations + e2e_seed.sql)
  client-intro-course   built from ../client
```

Three properties of the published core client make this work without touching the
[prompt](https://github.com/prompt-edu/prompt) repository:

1. It is a **production** build, so its Module Federation remote for this repo is baked in as
   the relative URL `/intro-course-developer/remoteEntry.js`. Serving our remote under that
   prefix is all that is required — there is no env var for it.
2. `client/rspack.config.mjs` already sets `output.publicPath: 'auto'` and exposes `./routes`,
   `./sidebar`, and `./provide` under the MF name `intro_course_developer_component` — exactly
   the contract core expects.
3. `INTRO_COURSE_HOST` is already in the core client's `env.template.js`. Left **empty**,
   `parseURL('')` returns `window.location.origin`, so the remote's axios calls are same-origin
   and get proxied — no CORS involved.

Everything addresses everything else by **Docker service name**, never `localhost`: Chromium
hard-codes `localhost` to loopback and ignores `/etc/hosts`, so a containerized browser cannot
reach the stack through a host-gateway remap. Service names also make the realm's redirect URIs
and the Keycloak token issuer line up (`hostname-strict=false` derives the issuer from the
request host).

Host ports are published (client `4100`, core API `18190`, Keycloak `18181` — see
`CLIENT_PORT` / `CORE_API_PORT` / `KEYCLOAK_PORT` in `.env.e2e`) so the stack can be inspected by
hand and driven from a host-side Playwright. They are deliberately offset from the prompt2 e2e
stack's `4000` / `18090` / `18081`, so both stacks can run at the same time. No database has a
host volume, so every run re-seeds from scratch.

The `/api/` location exists because the intro-course component reaches exactly one **core**
endpoint — `PUT /api/keycloak/:courseID/group`, used by the tutor page — through its *own* axios
instance, whose baseURL is the browser origin rather than `CORE_HOST`. In prod that works because
Traefik routes `/api` on the core host to core; the override mirrors it. `client-core` therefore
`depends_on` `server-core`: nginx resolves every proxy upstream at startup and refuses to boot if
one is missing.

SeaweedFS is not optional: the core server calls `log.Fatal` at boot if it cannot reach the
`prompt-files` and `prompt-privacy-exports` buckets. The intro course itself never uses S3.

## Serial execution

The suite runs with `workers: 1`. There is only **one** intro-course phase, so seats, peer groups,
tutors, and developer profiles are shared mutable state across every spec — unlike the platform
repo, which can afford a dedicated fixture phase per spec file. Mutating specs still snapshot and
restore their slice of the fixture, so any single test can be run on its own and a Playwright retry
starts from a clean state. A cold-start full run takes about 2.5 minutes.

## Authentication

Three layers, all real.

**The realm** (`keycloak/realm.json`) is a copy of `prompt2/e2e/keycloak/realm.json`. Keeping this
repo's *service names* identical to that stack is what makes the copy work — its redirect URIs
already list `http://client-core/*`. Two additions on top of the upstream file, both of which must
be re-applied when refreshing it:

1. This repo's host ports (`4100` / `18190`) are appended to `prompt-client`'s redirect URIs and
   web origins and to `prompt-server`'s redirect URIs, so a host-side Playwright run can complete
   the login redirect.
2. The Keycloak group tree `/Prompt/ios2425-iPraktikumFull` (with `Lecturer` and `Editor`
   sub-groups) is pre-created. Core normally creates that tree when a course is created through
   the UI; the seeded course is inserted straight into the database, so nothing ever created it.
   Without it the tutor page's `PUT /api/keycloak/:courseID/group` fails with
   *"Group path does not exist"* and the tutor table stays gated forever.

Username equals password for every user. On the seeded course (`iPraktikumFull`):

| user              | intro-course access                                      |
| ----------------- | -------------------------------------------------------- |
| `admin`           | `PROMPT_Admin` — everything                               |
| `lecturer`        | `ios2425-iPraktikumFull-Lecturer` — everything            |
| `course-lecturer` | `ios2425-iPraktikumFull-Lecturer` — everything            |
| `course-editor`   | `…-Editor` — **blocked** from every intro-course route     |
| `student`         | Stan: participates, **no** developer profile or seat      |
| `student2`        | Selma: participates, has a profile, a seat, and peers     |

`course-editor` is the interesting negative: it clears core's course-level gate but not the
routes' `requiredPermissions`, so it exercises the remote's own authorization.

**Browser auth** — `src/global-setup.ts` waits for the stack, then drives the real Keycloak login
form once per role and writes `.auth/<role>.json`. Specs pick a session with
`test.use({ role: 'course-lecturer' })`; the `role` fixture maps it to a `storageState`. The
intro-course component reads `localStorage.jwt_token` on every request, so `LoginPage` waits for
that key before the state is captured.

**API auth** — `src/fixtures/api.ts` mints tokens through Keycloak's password grant. `apiAs(role)`
targets the core API; `introCourseAs(role)` targets the intro-course server directly, so a
401/403 assertion tests the server's middleware rather than nginx.

## Readiness

`global-setup` gates on four things before any spec runs, in order:

1. `${CORE_API_URL}/api/hello` — a 200 here means core got all the way through migrations,
   Keycloak OIDC discovery, phase-type bootstrap, and both S3 buckets.
2. `${BASE_URL}` — the core client's nginx is serving.
3. `${BASE_URL}/intro-course/api/info` — asserts the JSON (`serviceName`, `healthy`), not just a
   status: the SPA fallback answers 200 with `index.html` for a misrouted path, so this doubles
   as a check that the `/intro-course/api/` proxy is wired correctly.
4. `${BASE_URL}/intro-course-developer/remoteEntry.js` — asserts the body is JavaScript, not
   HTML. Without it a broken prefix-stripping rule shows up much later as an opaque
   `LoadingError` in every browser spec.

The intro-course and core server images are distroless, so they have no container healthcheck;
these polls are the readiness gate.

## Test data

Two seeds, deliberately kept separate.

**Core** (`seed/core/`) is loaded by the `db` container's `/docker-entrypoint-initdb.d` in
filename order:

- `10_core_seed.sql` — a **verbatim copy** of `prompt2/e2e/seed/e2e_seed.sql`, so it can be
  refreshed with a plain file copy. It pins `schema_migrations = 24`; the core image is further
  along, so core's own `migrate up` applies the remainder on boot.
- `20_intro_course.sql` — everything intro-course specific: the `Intro Course Developer` phase
  type (with a **fixed** UUID, because core's startup init matches by name and would otherwise
  mint a random one), its `devices` output DTO, the phase itself, a graph edge appending it to
  the tail of the course's phase chain, and 56 background students with their participations.

**Intro course** (`server/database_dumps/e2e_seed.sql`, plus `seed/intro-course/`) is loaded by
`db-intro-course`. The compose file mounts the six migrations from `server/db/migration/`, then
the fixture, then `90_schema_migrations.sql` which pins the applied version so the server's
boot-time `migrate up` is a no-op for `0001..0006`. Mounting the migrations rather than shipping
a schema dump keeps `server/db/migration/` the single source of truth.

**When you add a migration**, add a mount line to `db-intro-course` in `docker-compose.e2e.yml`
and bump the version in `seed/intro-course/90_schema_migrations.sql`. Leaving both alone also
works — the server applies the new migration on top of the pinned version — but then the fixture
data is inserted before it.

The fixture is 6 tutors, 89 Rechnerhalle seats (12 chair Macs, 6 tutor seats), 56 developer
profiles, and 17 peer groups, plus profile/seat/peer rows for Selma. Every UUID a spec asserts on
is pinned in `src/data/constants.ts` — assert identities from there, never row counts that shift
when the seed grows.

**Regenerating the core seed** (only needed if a core migration makes the current dump
unloadable): start a throwaway Postgres, let the core server migrate and seed it, then
`pg_dump --inserts --no-owner` it back into `10_core_seed.sql` and re-add the `schema_migrations`
pin. Upstream `prompt2/e2e/README.md` documents the same recipe.

## Writing tests

Browser specs use the page objects in `src/pages/`. They all extend `IntroCoursePhasePage`, which
knows the core shell's URL shape (`/management/course/:courseId/:phaseId<route>`) and the
`Access Denied` card, so a new page needs only its route and its `<h1>`.

```ts
import { test, expect } from '../../src/fixtures/auth'
import { SeatAssignmentPage } from '../../src/pages/SeatAssignmentPage'

test.use({ role: 'course-lecturer' })

test('the seeded seat plan renders', async ({ page }) => {
  const seats = new SeatAssignmentPage(page)
  await seats.goto()
  await seats.expectSeatPlanLoaded()
})
```

API specs use the `api` project (`*.api.spec.ts`, no browser) and the `introCourseAs` fixture.

Conventions worth keeping:

- **Mutating specs restore the fixture.** `tests/intro-course/helpers.ts` has typed primitives for
  snapshotting and replaying the seat plan, peer assignments, and developer profiles. Snapshot in
  `beforeAll`, replay in `afterEach`, and mint the teardown context inside the helper so it works
  even when the test failed early.
- **Teardown through the API where possible, `src/db.ts` where not.** The one case that needs
  direct SQL today is the developer-profile survey: `POST /developer_profile` is a create rather
  than an upsert and there is no DELETE endpoint, so only a real row delete lets the spec survive a
  retry. Keep that list short — assertions belong against the API or the UI.
- **Assert denials, not absences.** A blank page from a broken remote must not pass as "correctly
  blocked" — check for the `Access Denied` card explicitly.
- **Assert identities, not counts**, wherever the seed might grow.
- **Add a `data-testid`** rather than relying on a `title` attribute or an ambiguous label. Seat
  cells, peer group members, table rows, and the four Yes/No pairs on the survey all carry one.
- **New route?** Add it to `src/data/permissionMatrix.ts` and every role is covered automatically
  by `tests/permissions/browser-matrix.spec.ts`.

## What is deliberately not covered

`GITLAB_ACCESS_TOKEN` is unset, so the server's GitLab client is nil and repo setup, peer
sync/unsync, and student GitLab provisioning fail by construction. The specs assert the surfaced
**error** instead of talking to a real GitLab.

## Debugging a failure

- The HTML report is at `e2e/playwright-report/` (`npx playwright show-report` inside `e2e/`), and
  CI uploads it as the `playwright-report` artifact.
- Traces (on first retry), videos, and screenshots land in `e2e/test-results/`, uploaded as
  `playwright-test-results`.
- If `global-setup` fails to log a role in, it writes `test-results/global-setup-<role>.png` plus
  the page title and body to the log.
- A readiness timeout names the exact URL it gave up on; `docker compose … logs server-core` or
  `… logs server-intro-course` usually explains why.

## Files

| Path                                     | Purpose                                          |
| ---------------------------------------- | ------------------------------------------------ |
| `../docker-compose.e2e.yml`              | The whole stack                                  |
| `../Makefile`                            | `test-e2e`, `test-e2e-ui`, `test-e2e-down`       |
| `.env.e2e`                               | Committed test-only credentials and image tags   |
| `Dockerfile`                             | Runner image; also typechecks the suite          |
| `playwright.config.ts`                   | Projects (`api`, `chromium`), retries, reporters |
| `keycloak/realm.json`                    | Verbatim copy of the prompt2 e2e realm           |
| `nginx/client-core.conf`                 | Stands in for prod Traefik routing               |
| `seed/core/`, `seed/intro-course/`       | Database seeds                                   |
| `src/env.ts`                             | Every URL, overridable by env var                |
| `src/db.ts`                              | Direct DB access, for teardowns the API can't do  |
| `src/global-setup.ts`                    | Readiness gates + per-role login                 |
| `src/data/`                              | Roles, seeded constants, permission matrix       |
| `src/fixtures/`                          | `role`/`storageState` and API-token fixtures     |
| `src/pages/`                             | Page objects                                     |
| `tests/`                                 | The specs                                        |
