# PROMPT Intro Course

Standalone repository for the PROMPT intro course services.

## Structure

- `server/` Go intro course backend
- `client/` Intro course developer micro-frontend
- `e2e/` Playwright end-to-end tests (see [`e2e/README.md`](e2e/README.md))
- `docker-compose.yml` Local development database
- `docker-compose.e2e.yml` Full end-to-end stack (core shell + this repo + Keycloak)
- `docker-compose.prod.yml` Production deployment for intro course services only

## Local Development

1. Copy `.env.template` to `.env` and adapt values if needed.
   Intro-course-specific runtime variables previously kept in `prompt2` now live in this repository.
2. Start the intro-course database:
   ```bash
   docker compose up -d
   ```
3. Run server:
   ```bash
   cd server
   go run main.go
   ```
4. Install client dependencies and run the client:
   ```bash
   cd client
   yarn install
   yarn dev
   ```

The client consumes the published `@tumaet/prompt-shared-state` and
`@tumaet/prompt-ui-components` packages directly.

Use a Node LTS release (recommended: Node 22) for local client tooling.

The core app in the main PROMPT repository can load this client via Module Federation at `http://localhost:3005`.

## Testing

Server unit and integration tests (testcontainers-backed):

```bash
cd server
go test ./...
```

End-to-end tests. These stand the real PROMPT core shell up around this repository's server and
micro-frontend, with a real Keycloak, and drive it in Chromium — all in Docker Compose, so no
local setup beyond Docker is needed:

```bash
make test-e2e        # build, run the suite, tear down
make test-e2e-ui     # Playwright UI mode on http://127.0.0.1:8123
make test-e2e-down   # tear down a stack left running
```

See [`e2e/README.md`](e2e/README.md) for the stack layout, the seeded test users and data, and how
to add tests.

## Production Deployment

CI/CD workflows are in `.github/workflows/`:

- `build-and-push.yml` builds and pushes server + client images
- `test-e2e.yml` runs the Playwright suite against the full Compose stack
- `dev.yml` runs tests, builds, and deploys to dev VM
- `prod.yml` runs release builds and deploys to prod VM
- `deploy.yml` deploys only intro-course containers using `docker-compose.prod.yml`

### Keycloak 26.8.2: required admin step

The tutor page asks the core server to create the tutor group and to sync tutors and students
into Keycloak groups. Core does that through the Keycloak Admin REST API with the `prompt-server`
service account. Since Keycloak 26.8.2 the Admin REST API answers `401 Unauthorized` unless the
token carries the built-in `kc-admin-api` client scope, and upgrading Keycloak does not assign
that scope to `prompt-server`.

After upgrading a Keycloak instance to 26.8.2 or later, a Keycloak admin has to do this once per
realm:

1. Open the Keycloak admin console and select the PROMPT realm.
2. Go to `Clients > prompt-server > Client scopes`.
3. Click **Add client scope**, select `kc-admin-api`, and add it as **Default**.
4. The scope only applies to newly issued tokens. If calls still fail a few minutes later,
   restart the core server so it fetches a fresh service-account token.

To check it, open the tutor page of an intro-course phase as a lecturer: it shows
"Tutor group configured in PROMPT" instead of staying gated. If the step is missing, the core
server logs `failed to get group from Keycloak ... 401 Unauthorized`.

Nothing in this repository needs to change for a deployment; the scope lives in the Keycloak
realm. The full Keycloak setup is documented in the main repository under
[`docs/admin/keycloak-prod.md`](https://github.com/prompt-edu/prompt/blob/main/docs/admin/keycloak-prod.md).

For the 2026–27 student-repository rollout and provisional three-room seat plan, see
[`docs/ios-2026-27-setup.md`](docs/ios-2026-27-setup.md).
