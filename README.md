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
