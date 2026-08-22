// Centralized environment config. Defaults target the e2e stack's host-published
// ports (CLIENT_PORT / KEYCLOAK_PORT / CORE_API_PORT in e2e/.env.e2e, offset from
// the prompt2 e2e stack's so both can run at once) so running Playwright directly
// on the host (cd e2e && npx playwright test) works against a running stack with
// no env vars; the containerized runner overrides these via docker-compose.e2e.yml.

export const BASE_URL = process.env.BASE_URL ?? 'http://localhost:4100'
export const KEYCLOAK_URL = process.env.KEYCLOAK_URL ?? 'http://localhost:18181'
export const CORE_API_URL = process.env.CORE_API_URL ?? 'http://localhost:18190'
export const KEYCLOAK_REALM = process.env.KEYCLOAK_REALM_NAME ?? 'prompt'

// Public Keycloak client used by the browser app (direct access grants enabled).
export const KEYCLOAK_CLIENT_ID = 'prompt-client'

// The intro-course API prefix on the browser origin (BASE_URL); proxied to
// server-intro-course by e2e/nginx/client-core.conf, mirroring prod Traefik.
export const INTRO_COURSE_API = '/intro-course/api'

// The intro-course server addressed directly, bypassing the proxy. Used by the
// api project so a 401/403 assertion tests the server's auth middleware rather
// than nginx. Not published on the host, so the host default goes through the
// proxy instead.
export const INTRO_COURSE_API_URL =
  process.env.INTRO_COURSE_API_URL ?? `${BASE_URL}`

// Path prefix under which the core shell mounts this repo's Module Federation
// remote (baked into the published core client at build time).
export const INTRO_COURSE_REMOTE_PATH = '/intro-course-developer'

export const tokenEndpoint = () =>
  `${KEYCLOAK_URL}/realms/${KEYCLOAK_REALM}/protocol/openid-connect/token`
