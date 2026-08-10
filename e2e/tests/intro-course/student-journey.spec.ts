import { test, expect } from '../../src/fixtures/auth'
import { IntroCourseOverviewPage } from '../../src/pages/IntroCourseOverviewPage'
import {
  INTRO_COURSE_PHASE_ID,
  STUDENT_WITHOUT_PROFILE,
  STUDENT_WITH_PROFILE,
} from '../../src/data/constants'
import { deleteDeveloperProfile } from '../../src/db'

// The two halves of the student flow, split across the two seeded Keycloak users:
// Stan has no developer profile (so he can submit the survey), Selma has a
// profile, a seat, and peers (so the seat display has something to render).

test.describe('student journey: submitting the developer profile survey', () => {
  test.use({ role: 'student' })

  const SUBMITTED = {
    appleId: 'stan.e2e@icloud.com',
    gitlabUsername: 'stane2e',
    hasMacBook: true,
  }

  // Reset Stan to the pre-survey state. This has to delete the row, not blank it:
  // POST /developer_profile is a create, not an upsert, so a blanked row would make
  // the next submission fail with a duplicate key — including on a Playwright retry.
  test.afterEach(async () => {
    await deleteDeveloperProfile(
      INTRO_COURSE_PHASE_ID,
      STUDENT_WITHOUT_PROFILE.courseParticipationId,
    )
  })

  test('step 1 is open and step 2 is locked before the survey', async ({ page }) => {
    const overview = new IntroCourseOverviewPage(page)
    await overview.goto()
    await overview.expectLoaded()

    await expect(overview.surveyStep).toBeVisible()
    // No seat assignment yet, so step 2 is disabled and titled accordingly.
    await expect(overview.seatStep(false)).toBeVisible()
    await expect(overview.notAStudentAlert).toBeHidden()

    // The survey form is open by default when there is no profile.
    await expect(overview.appleIdInput).toBeVisible()
  })

  test('submitting the survey succeeds and offers the next step', async ({ page }) => {
    const overview = new IntroCourseOverviewPage(page)
    await overview.goto()
    await overview.expectLoaded()

    await overview.submitSurvey(SUBMITTED)
    await expect(overview.continueButton).toBeVisible()

    // Persisted: after a reload the survey step is completed rather than open.
    await page.reload()
    await overview.expectLoaded()
    await overview.openStep('Developer Profile Survey')
    await expect(overview.successHeading).toBeVisible({ timeout: 30_000 })
  })

  test('the survey rejects an empty submission', async ({ page }) => {
    const overview = new IntroCourseOverviewPage(page)
    await overview.goto()
    await overview.expectLoaded()

    await overview.submitButton.click()

    // zod validation keeps the form open instead of reporting success.
    await expect(overview.successHeading).toBeHidden()
    await expect(overview.appleIdInput).toBeVisible()
  })

  test('the helper dialogs open', async ({ page }) => {
    const overview = new IntroCourseOverviewPage(page)
    await overview.goto()
    await overview.expectLoaded()

    // Three buttons labelled "Help" share the form, hence the testids.
    for (const testId of ['apple-id-help', 'gitlab-help']) {
      await page.getByTestId(testId).click()
      const dialog = page.getByRole('dialog')
      await expect(dialog).toBeVisible()
      // Radix renders its own sr-only "Close" in the corner, so two buttons carry
      // that name; the footer one is last in the DOM.
      await dialog.getByRole('button', { name: 'Close' }).last().click()
      await expect(dialog).toBeHidden()
    }
  })

  test('Stan has no seat, so the seat display shows the empty state', async ({ page }) => {
    const overview = new IntroCourseOverviewPage(page)
    await overview.goto()
    await overview.expectLoaded()

    // Step 2 is disabled while there is no assignment, so the display is asserted
    // through its title rather than by opening the collapsible.
    await expect(overview.seatStep(false)).toBeVisible()
  })
})

test.describe('student journey: the seat assignment display', () => {
  test.use({ role: 'student2' })

  test('shows the assigned seat, the tutor, and the review peers', async ({ page }) => {
    const overview = new IntroCourseOverviewPage(page)
    await overview.goto()
    await overview.expectLoaded()

    // Selma has both a profile and a seat, so step 2 is enabled and open.
    await expect(overview.seatStep(true)).toBeVisible()

    await expect(overview.seatInformationCard).toBeVisible()
    await expect(page.getByText(STUDENT_WITH_PROFILE.seatName)).toBeVisible()
    // Her seat is an ordinary chair, and she brought her own MacBook.
    await expect(page.getByText('Own MacBook')).toBeVisible()

    // Alice supervises row 1.
    await expect(overview.tutorCard).toBeVisible()
    await expect(page.getByText('Alice Mueller')).toBeVisible()
    await expect(page.getByText('alice.mueller@example.com')).toBeVisible()

    await expect(overview.reviewPeersCard).toBeVisible()
    for (const peer of STUDENT_WITH_PROFILE.peerGitlabUsernames) {
      await expect(page.getByText(peer, { exact: true })).toBeVisible()
    }
    await expect(overview.gitlabRepoButtons).toHaveCount(
      STUDENT_WITH_PROFILE.peerGitlabUsernames.length,
    )
  })

  test('the survey step shows her existing profile as completed', async ({ page }) => {
    const overview = new IntroCourseOverviewPage(page)
    await overview.goto()
    await overview.expectLoaded()

    await overview.openStep('Developer Profile Survey')
    await expect(overview.successHeading).toBeVisible({ timeout: 30_000 })
  })
})

test.describe('student journey: staff see the disabled student view', () => {
  test.use({ role: 'course-lecturer' })

  test('the overview warns that the viewer is not a student', async ({ page }) => {
    const overview = new IntroCourseOverviewPage(page)
    await overview.goto()
    await overview.expectLoaded()

    await expect(overview.notAStudentAlert).toBeVisible()
    await expect(overview.surveyStep).toBeVisible()
  })
})
