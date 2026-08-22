import { test, expect } from '../../src/fixtures/auth'
import { IntroCourseOverviewPage } from '../../src/pages/IntroCourseOverviewPage'
import { SeatAssignmentPage } from '../../src/pages/SeatAssignmentPage'

// Module Federation smoke test: every intro-course page is rendered by the
// intro_course_developer_component REMOTE, which the core shell loads from
// /intro-course-developer/remoteEntry.js through the e2e nginx proxy. If the
// remote fails to load, the shell renders a LoadingError instead of the heading,
// so this one assertion covers the whole MF path — remote build, publicPath,
// nginx prefix stripping, and the 'Intro Course Developer' phase-type mapping.

test.describe('intro course: module federation smoke', () => {
  test.describe('as a lecturer', () => {
    test.use({ role: 'course-lecturer' })

    test('the remote loads and renders inside the core shell', async ({ page }) => {
      const phase = new SeatAssignmentPage(page)
      await phase.goto()
      await phase.expectLoaded()
    })

    test('the sidebar exposes every staff sub-page', async ({ page }) => {
      const overview = new IntroCourseOverviewPage(page)
      await overview.goto()
      await overview.expectLoaded()

      // The sidebar is rendered by the remote's ./sidebar export, so this proves the
      // second of the three module-federation entry points. When the remote's
      // sidebar fails to load, core substitutes a disabled "Intro Course Not
      // Available" item -- exactly what the other modules show in this stack.
      await expect(overview.sidebarEntry).toBeVisible()
      await expect(page.getByText('Intro Course Not Available')).toBeHidden()

      await overview.expandSidebar()

      for (const item of [
        'Participants',
        'Developer Profiles',
        'Tutor Import',
        'Seat Assignments',
        'Peer Assignments',
        'Mailing',
      ]) {
        await expect(overview.sidebarSubItem(item)).toBeVisible()
      }
    })
  })

  test.describe('as a student', () => {
    test.use({ role: 'student' })

    test('the remote loads and renders the student overview', async ({ page }) => {
      const overview = new IntroCourseOverviewPage(page)
      await overview.goto()
      await overview.expectLoaded()
      await expect(overview.surveyStep).toBeVisible()
    })

    test('the sidebar hides the staff sub-pages', async ({ page }) => {
      const overview = new IntroCourseOverviewPage(page)
      await overview.goto()
      await overview.expectLoaded()

      // A student only has the phase's root entry; there are no sub-items to expand.
      await expect(overview.sidebarEntry).toBeVisible()
      await overview.expandSidebar()

      await expect(overview.sidebarSubItem('Seat Assignments')).toBeHidden()
      await expect(overview.sidebarSubItem('Tutor Import')).toBeHidden()
    })
  })
})
