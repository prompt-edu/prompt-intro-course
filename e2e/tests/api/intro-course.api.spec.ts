import { request } from '@playwright/test'
import { test, expect } from '../../src/fixtures/api'
import { INTRO_COURSE_API, INTRO_COURSE_API_URL } from '../../src/env'
import {
  FOREIGN_PHASE_ID,
  INTRO_COURSE_PARTICIPANT_COUNT,
  INTRO_COURSE_PHASE_ID,
  SEAT_PLAN,
  SEEDED_TUTOR_COUNT,
  STUDENT_WITHOUT_PROFILE,
  STUDENT_WITH_PROFILE,
} from '../../src/data/constants'
import {
  LECTURER_API_SURFACES,
  MATRIX_ROLES,
  STUDENT_API_SURFACES,
} from '../../src/data/permissionMatrix'

// API-layer tests against the intro-course server. The browser specs cover the UI;
// these cover the server's own auth middleware and phase scoping, addressing the
// server directly rather than through the client-core proxy.

function phasePath(path: string, phaseId = INTRO_COURSE_PHASE_ID): string {
  return `${INTRO_COURSE_API}/course_phase/${phaseId}${path}`
}

test.describe('intro course api: the info endpoint', () => {
  test('is public and reports the service healthy', async () => {
    const anon = await request.newContext({ baseURL: INTRO_COURSE_API_URL })
    const res = await anon.get(`${INTRO_COURSE_API}/info`)
    expect(res.status()).toBe(200)
    expect(await res.json()).toMatchObject({
      serviceName: 'intro-course',
      healthy: true,
      capabilities: { 'phase.copy': true, 'phase.config': true },
    })
    await anon.dispose()
  })
})

test.describe('intro course api: unauthenticated requests', () => {
  for (const surface of [...LECTURER_API_SURFACES, ...STUDENT_API_SURFACES]) {
    test(`${surface.name} without a token is rejected`, async () => {
      const anon = await request.newContext({ baseURL: INTRO_COURSE_API_URL })
      const res = await anon.get(phasePath(surface.path))
      expect(res.status()).toBe(401)
      await anon.dispose()
    })
  }
})

test.describe('intro course api: lecturer endpoints', () => {
  for (const surface of LECTURER_API_SURFACES) {
    for (const role of MATRIX_ROLES) {
      const allowed = surface.allowed.includes(role)

      test(`${role} ${allowed ? 'may' : 'may not'} read ${surface.name}`, async ({
        introCourseAs,
      }) => {
        const api = await introCourseAs(role)
        const res = await api.get(phasePath(surface.path))

        if (allowed) {
          expect(res.status()).toBe(200)
        } else {
          // The SDK middleware rejects a caller without the required role.
          expect([401, 403]).toContain(res.status())
        }
      })
    }
  }

  test('the seeded seat plan, tutors, and profiles are readable', async ({ introCourseAs }) => {
    const api = await introCourseAs('course-lecturer')

    const seats = (await (await api.get(phasePath('/seat_plan'))).json()) as unknown[]
    expect(seats).toHaveLength(SEAT_PLAN.totalSeats)

    const tutors = (await (await api.get(phasePath('/tutor'))).json()) as unknown[]
    expect(tutors).toHaveLength(SEEDED_TUTOR_COUNT)

    const profiles = (await (await api.get(phasePath('/developer_profile'))).json()) as unknown[]
    // Every participant except Stan has a profile.
    expect(profiles).toHaveLength(INTRO_COURSE_PARTICIPANT_COUNT - 1)
  })
})

test.describe('intro course api: student-scoped endpoints', () => {
  for (const surface of STUDENT_API_SURFACES) {
    for (const role of MATRIX_ROLES) {
      const allowed = surface.allowed.includes(role)

      test(`${role} ${allowed ? 'may' : 'may not'} read ${surface.name}`, async ({
        introCourseAs,
      }) => {
        const api = await introCourseAs(role)
        const res = await api.get(phasePath(surface.path))

        if (allowed) {
          expect(res.status()).toBe(200)
        } else {
          expect([401, 403]).toContain(res.status())
        }
      })
    }
  }

  test("a student's own seat assignment is their seeded seat", async ({ introCourseAs }) => {
    const api = await introCourseAs(STUDENT_WITH_PROFILE.role)
    const res = await api.get(phasePath('/seat_plan/own-assignment'))
    expect(res.status()).toBe(200)
    expect(await res.json()).toMatchObject({ seatName: STUDENT_WITH_PROFILE.seatName })
  })

  test("a student's own developer profile is their seeded profile", async ({ introCourseAs }) => {
    const api = await introCourseAs(STUDENT_WITH_PROFILE.role)
    const res = await api.get(phasePath('/developer_profile/self'))
    expect(res.status()).toBe(200)
    expect(await res.json()).toMatchObject({
      appleID: STUDENT_WITH_PROFILE.appleId,
      gitLabUsername: STUDENT_WITH_PROFILE.gitlabUsername,
    })
  })

  test("a student's own peer assignment lists their group", async ({ introCourseAs }) => {
    const api = await introCourseAs(STUDENT_WITH_PROFILE.role)
    const res = await api.get(phasePath('/peer_assignments/own'))
    expect(res.status()).toBe(200)
    const body = (await res.json()) as { peersIReview: { gitlabUsername: string }[] }
    expect(body.peersIReview.map((p) => p.gitlabUsername).sort()).toEqual(
      [...STUDENT_WITH_PROFILE.peerGitlabUsernames].sort(),
    )
  })

  test('a student without a profile gets an empty own profile, not a 500', async ({
    introCourseAs,
  }) => {
    const api = await introCourseAs(STUDENT_WITHOUT_PROFILE.role)
    const res = await api.get(phasePath('/developer_profile/self'))
    // The client relies on this: an empty appleID + gitLabUsername is how the data
    // shell decides the survey has not been filled in yet.
    expect(res.status()).toBe(200)
    expect(await res.json()).toMatchObject({ appleID: '', gitLabUsername: '' })
  })
})

test.describe('intro course api: phase scoping', () => {
  test('a phase of another type holds no intro-course data', async ({ introCourseAs }) => {
    const api = await introCourseAs('course-lecturer')

    // FOREIGN_PHASE_ID is the Application phase on the same course: the caller is
    // authorized, but no seats, tutors, or profiles belong to that phase.
    const seats = (await (
      await api.get(phasePath('/seat_plan', FOREIGN_PHASE_ID))
    ).json()) as unknown[] | null
    expect(seats ?? []).toHaveLength(0)

    const tutors = (await (
      await api.get(phasePath('/tutor', FOREIGN_PHASE_ID))
    ).json()) as unknown[] | null
    expect(tutors ?? []).toHaveLength(0)
  })

  test('a malformed phase id is rejected', async ({ introCourseAs }) => {
    const api = await introCourseAs('course-lecturer')
    const res = await api.get(phasePath('/seat_plan', 'not-a-uuid'))
    expect([400, 401, 403]).toContain(res.status())
  })
})
