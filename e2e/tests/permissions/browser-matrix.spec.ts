import { Page } from '@playwright/test'
import { test, expect } from '../../src/fixtures/auth'
import { INTRO_COURSE_PHASE_ID, SEEDED_COURSE } from '../../src/data/constants'
import { BROWSER_SURFACES, MATRIX_ROLES } from '../../src/data/permissionMatrix'

// Data-driven access control over every route the remote exposes, for every
// seeded role. The interesting negative is course-editor: it holds a course-scoped
// role on the seeded course, so it gets past core's course gate, but the intro
// routes require PROMPT_ADMIN or COURSE_LECTURER.
//
// Assert the DENIAL, never merely the absence of content: a blank page caused by
// a broken remote would otherwise pass as "correctly blocked".

async function expectBlocked(page: Page, heading: string) {
  await expect(page.getByText('Access Denied')).toBeVisible({ timeout: 30_000 })
  await expect(page.getByText('You do not have permission to access this page.')).toBeVisible()
  await expect(page.getByRole('heading', { level: 1, name: heading })).toBeHidden()
}

async function expectAllowed(page: Page, heading: string) {
  await expect(page.getByRole('heading', { level: 1, name: heading })).toBeVisible({
    timeout: 30_000,
  })
  await expect(page.getByText('Access Denied')).toBeHidden()
}

for (const role of MATRIX_ROLES) {
  test.describe(`as ${role}`, () => {
    test.use({ role })

    for (const surface of BROWSER_SURFACES) {
      const allowed = surface.allowed.includes(role)

      test(`${allowed ? 'sees' : 'is blocked from'} ${surface.name}`, async ({ page }) => {
        await page.goto(
          `/management/course/${SEEDED_COURSE.id}/${INTRO_COURSE_PHASE_ID}${surface.route}`,
        )

        if (allowed) {
          await expectAllowed(page, surface.heading)
        } else {
          await expectBlocked(page, surface.heading)
        }
      })
    }
  })
}
