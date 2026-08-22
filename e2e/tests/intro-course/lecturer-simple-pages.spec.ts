import { test, expect } from '../../src/fixtures/auth'
import {
  IntroCourseParticipantsPage,
  MailingPage,
} from '../../src/pages/IntroCourseSimplePages'
import {
  BACKGROUND_STUDENTS,
  INTRO_COURSE_PARTICIPANT_COUNT,
  INTRO_COURSE_ROUTES,
  STUDENT_WITHOUT_PROFILE,
} from '../../src/data/constants'

// Participants and Mailing are thin wrappers around shared prompt-ui-components
// fed entirely by core. The value here is proving that the remote renders them
// with the right phase context — i.e. that core's participation data reaches the
// remote through the seeded phase.
test.use({ role: 'course-lecturer' })

test.describe('participants', () => {
  test('counts every seeded participant', async ({ page }) => {
    const participants = new IntroCourseParticipantsPage(page)
    await participants.goto()
    await participants.expectLoaded()

    // 56 background students plus the two Keycloak student users.
    await expect(participants.rowCount).toContainText(`${INTRO_COURSE_PARTICIPANT_COUNT} rows`)
  })

  test('finds a background student and a Keycloak student user', async ({ page }) => {
    const participants = new IntroCourseParticipantsPage(page)
    await participants.goto()
    await participants.expectLoaded()

    // The table paginates and sorts by last name, so search rather than scroll.
    await participants.search(BACKGROUND_STUDENTS.first.lastName)
    await expect(
      participants.row(BACKGROUND_STUDENTS.first.firstName, BACKGROUND_STUDENTS.first.lastName),
    ).toBeVisible()

    await participants.search(STUDENT_WITHOUT_PROFILE.lastName)
    await expect(
      participants.row(STUDENT_WITHOUT_PROFILE.firstName, STUDENT_WITHOUT_PROFILE.lastName),
    ).toBeVisible()
  })
})

test.describe('mailing', () => {
  test('shows the reminder and links to the participants page', async ({ page }) => {
    const mailing = new MailingPage(page)
    await mailing.goto()
    await mailing.expectLoaded()

    await expect(mailing.reminder).toBeVisible()
    await expect(mailing.participantsLink).toBeVisible()

    await mailing.participantsLink.click()
    await expect(page).toHaveURL(new RegExp(`${INTRO_COURSE_ROUTES.participants}$`))
    await expect(
      page.getByRole('heading', { level: 1, name: 'Intro Course Participants' }),
    ).toBeVisible({ timeout: 30_000 })
  })
})
