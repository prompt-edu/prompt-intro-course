# PROMPT Intro Course

Standalone repository for the PROMPT intro course services.

## Structure

- `server/` Go intro course backend
- `client/` Intro course developer micro-frontend
- `docker-compose.yml` Local development database
- `docker-compose.prod.yml` Production deployment for intro course services only

## Local Development

1. Copy `.env.template` to `.env` and adapt values if needed.
   Intro-course-specific runtime variables previously kept in `prompt2` now live in this repository.
2. Link the shared library from the main PROMPT repository:
   ```bash
   ln -sfn ../prompt2/clients/shared_library shared_library
   ```
3. Start the intro-course database:
   ```bash
   docker compose up -d
   ```
4. Run server:
   ```bash
   cd server
   go run main.go
   ```
5. Install client dependencies and run the client:
   ```bash
   cd client
   yarn install
   yarn dev
   ```

Use a Node LTS release (recommended: Node 22) for local client tooling.

The core app in the main PROMPT repository can load this client via Module Federation at `http://localhost:3005`.

## Production Deployment

CI/CD workflows are in `.github/workflows/`:

- `build-and-push.yml` builds and pushes server + client images
- `dev.yml` runs tests, builds, and deploys to dev VM
- `prod.yml` runs release builds and deploys to prod VM
- `deploy.yml` deploys only intro-course containers using `docker-compose.prod.yml`
