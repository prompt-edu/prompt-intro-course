import { test as base, request, APIRequestContext, expect } from '@playwright/test'
import {
  CORE_API_URL,
  INTRO_COURSE_API_URL,
  KEYCLOAK_CLIENT_ID,
  tokenEndpoint,
} from '../env'
import { Role, ROLES } from '../data/roles'

// Mint a bearer token via Keycloak's direct access grant (password grant).
// `prompt-client` is a public client with this grant enabled, so no secret.
export async function tokenFor(role: Role): Promise<string> {
  const account = ROLES[role]
  const ctx = await request.newContext()
  try {
    const res = await ctx.post(tokenEndpoint(), {
      form: {
        grant_type: 'password',
        client_id: KEYCLOAK_CLIENT_ID,
        username: account.username,
        password: account.password,
        scope: 'openid',
      },
    })
    if (!res.ok()) {
      throw new Error(
        `token request for "${role}" failed: ${res.status()} ${await res.text()}`,
      )
    }
    const body = (await res.json()) as { access_token: string }
    return body.access_token
  } finally {
    await ctx.dispose()
  }
}

async function contextFor(baseURL: string, role: Role): Promise<APIRequestContext> {
  const token = await tokenFor(role)
  return request.newContext({
    baseURL,
    extraHTTPHeaders: { Authorization: `Bearer ${token}` },
  })
}

// An APIRequestContext targeting the core API, authenticated as the given role.
export function apiContextFor(role: Role): Promise<APIRequestContext> {
  return contextFor(CORE_API_URL, role)
}

// An APIRequestContext targeting the intro-course server, authenticated as the
// given role.
export function introCourseContextFor(role: Role): Promise<APIRequestContext> {
  return contextFor(INTRO_COURSE_API_URL, role)
}

// API-layer test with `apiAs(role)` / `introCourseAs(role)` helpers. Created
// contexts are disposed automatically at the end of the test.
export const test = base.extend<{
  apiAs: (role: Role) => Promise<APIRequestContext>
  introCourseAs: (role: Role) => Promise<APIRequestContext>
}>({
  apiAs: async ({}, use) => {
    const created: APIRequestContext[] = []
    await use(async (role: Role) => {
      const ctx = await apiContextFor(role)
      created.push(ctx)
      return ctx
    })
    await Promise.all(created.map((c) => c.dispose()))
  },
  introCourseAs: async ({}, use) => {
    const created: APIRequestContext[] = []
    await use(async (role: Role) => {
      const ctx = await introCourseContextFor(role)
      created.push(ctx)
      return ctx
    })
    await Promise.all(created.map((c) => c.dispose()))
  },
})

export { expect }
