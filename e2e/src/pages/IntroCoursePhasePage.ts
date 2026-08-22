import { Locator, Page, expect } from '@playwright/test'
import {
  INTRO_COURSE_PHASE_ID,
  INTRO_COURSE_PHASE_NAME,
  SEEDED_COURSE,
} from '../data/constants'

// Base for every page of the intro-course Module Federation remote. The core
// shell mounts the remote's routes under /management/course/:courseId/:phaseId,
// and the remote's own paths (client/routes/index.tsx) hang off that.
export abstract class IntroCoursePhasePage {
  // The remote's route, relative to the phase root ('' for the overview).
  protected abstract readonly route: string

  // The <h1> rendered by ManagementPageHeader on this page.
  protected abstract readonly headingName: string

  constructor(protected readonly page: Page) {}

  get heading(): Locator {
    return this.page.getByRole('heading', { level: 1, name: this.headingName })
  }

  async goto(courseId = SEEDED_COURSE.id, phaseId = INTRO_COURSE_PHASE_ID) {
    await this.page.goto(`/management/course/${courseId}/${phaseId}${this.route}`)
  }

  // The remote is lazily imported by the shell, so the first assertion after a
  // navigation needs a longer timeout than the 10s expect default.
  async expectLoaded() {
    await expect(this.heading).toBeVisible({ timeout: 30_000 })
  }

  // Access Denied card rendered by the shell's PermissionRestriction when the
  // role lacks the route's requiredPermissions.
  get accessDenied(): Locator {
    return this.page.getByText('Access Denied')
  }

  async expectAccessDenied() {
    await expect(this.accessDenied).toBeVisible({ timeout: 30_000 })
    await expect(this.heading).toBeHidden()
  }

  // The shell's sidebar entry for this phase. Entries are buttons, not links, and
  // the sub-items sit behind a sibling Toggle.
  get sidebarEntry(): Locator {
    return this.page.getByRole('button', { name: INTRO_COURSE_PHASE_NAME, exact: true })
  }

  get sidebarToggle(): Locator {
    return this.page
      .getByRole('listitem')
      .filter({ has: this.sidebarEntry })
      .getByRole('button', { name: 'Toggle' })
  }

  // Expands the phase's sidebar sub-items if they are collapsed.
  async expandSidebar() {
    await expect(this.sidebarEntry).toBeVisible({ timeout: 30_000 })
    const toggle = this.sidebarToggle
    if ((await toggle.count()) > 0 && (await toggle.getAttribute('data-state')) !== 'open') {
      await toggle.click()
    }
  }

  // Sub-items render as plain spans inside the collapsible sub-menu, not as links
  // or buttons. Call expandSidebar() first.
  sidebarSubItem(name: string): Locator {
    return this.page
      .locator('[data-sidebar="menu-sub"]')
      .getByText(name, { exact: true })
  }
}
