import { test, expect } from '../../src/fixtures/auth'
import { DeveloperProfilesPage } from '../../src/pages/DeveloperProfilesPage'
import {
  BACKGROUND_STUDENTS,
  INTRO_COURSE_PARTICIPANT_COUNT,
  STUDENT_WITHOUT_PROFILE,
} from '../../src/data/constants'
import { putDeveloperProfile, withIntroCourseApi } from './helpers'

test.use({ role: 'course-lecturer' })

test.describe('developer profiles: the lecturer table', () => {
  test('lists every participant with their survey status', async ({ page }) => {
    const profiles = new DeveloperProfilesPage(page)
    await profiles.goto()
    await profiles.expectTableLoaded()

    expect(await profiles.totalCount()).toBe(INTRO_COURSE_PARTICIPANT_COUNT)
    expect(await profiles.shownCount()).toBe(INTRO_COURSE_PARTICIPANT_COUNT)

    // A background student with a profile, and Stan, who has none.
    const withProfile = profiles.row(BACKGROUND_STUDENTS.first.courseParticipationId)
    await expect(withProfile).toContainText(BACKGROUND_STUDENTS.first.lastName)
    await expect(withProfile).toContainText(BACKGROUND_STUDENTS.first.gitlabUsername)
    await expect(withProfile).toContainText(BACKGROUND_STUDENTS.first.appleId)

    const withoutProfile = profiles.row(STUDENT_WITHOUT_PROFILE.courseParticipationId)
    await expect(withoutProfile).toContainText(STUDENT_WITHOUT_PROFILE.lastName)
    // 'Not set' is rendered for both the GitLab username and the Apple ID.
    await expect(withoutProfile.getByText('Not set')).toHaveCount(2)
  })

  test('the GitLab status column shows repositories are not created', async ({ page }) => {
    const profiles = new DeveloperProfilesPage(page)
    await profiles.goto()
    await profiles.expectTableLoaded()

    // The seed has no student_gitlab_processes rows and the stack has no GitLab
    // token, so every row reports "Repository not yet created" on hover.
    const row = profiles.row(BACKGROUND_STUDENTS.first.courseParticipationId)
    await row.getByRole('cell').nth(6).hover()
    await expect(page.getByText('Repository not yet created').first()).toBeVisible()
  })

  test('sorting by Name reorders the rows', async ({ page }) => {
    const profiles = new DeveloperProfilesPage(page)
    await profiles.goto()
    await profiles.expectTableLoaded()

    const unsorted = await profiles.visibleNames()

    await profiles.columnHeader('Name').click()
    const ascending = await profiles.visibleNames()
    expect(ascending).not.toEqual(unsorted)
    expect(ascending).toEqual([...ascending].sort((a, b) => a.localeCompare(b)))

    await profiles.columnHeader('Name').click()
    const descending = await profiles.visibleNames()
    expect(descending).toEqual([...ascending].reverse())
  })

  test('sorting by Survey groups the completed profiles together', async ({ page }) => {
    const profiles = new DeveloperProfilesPage(page)
    await profiles.goto()
    await profiles.expectTableLoaded()

    await profiles.columnHeader('Survey').click()
    // Stan is the only participant without a profile, so he ends up at one end.
    const rows = profiles.rows
    const first = await rows.first().getAttribute('data-testid')
    const last = await rows.last().getAttribute('data-testid')
    expect([first, last]).toContain(
      `developer-profile-row-${STUDENT_WITHOUT_PROFILE.courseParticipationId}`,
    )
  })

  test('filtering by survey status narrows the table', async ({ page }) => {
    const profiles = new DeveloperProfilesPage(page)
    await profiles.goto()
    await profiles.expectTableLoaded()

    await profiles.toggleFilter('Not Completed')
    // Only Stan has no developer profile.
    await expect(profiles.rows).toHaveCount(1)
    await expect(profiles.row(STUDENT_WITHOUT_PROFILE.courseParticipationId)).toBeVisible()
    expect(await profiles.shownCount()).toBe(1)

    await profiles.toggleFilter('Not Completed')
    await expect(profiles.rows).toHaveCount(INTRO_COURSE_PARTICIPANT_COUNT)
  })

  test('filtering by MacBook excludes the students without one', async ({ page }) => {
    const profiles = new DeveloperProfilesPage(page)
    await profiles.goto()
    await profiles.expectTableLoaded()

    await profiles.toggleFilter('MacBook')

    const shown = await profiles.shownCount()
    expect(shown).toBeGreaterThan(0)
    expect(shown).toBeLessThan(INTRO_COURSE_PARTICIPANT_COUNT)
    await expect(
      profiles.row(BACKGROUND_STUDENTS.withoutMacbook.courseParticipationId),
    ).toBeHidden()
    await expect(profiles.row(BACKGROUND_STUDENTS.first.courseParticipationId)).toBeVisible()
  })
})

test.describe('developer profiles: editing a profile', () => {
  const NEW_APPLE_ID = 'edited.by.e2e@icloud.com'

  // Restore the edited profile to the constants, which mirror the seed, so the
  // other specs keep seeing the fixture they expect however this test ends.
  test.afterEach(async () => {
    await withIntroCourseApi('course-lecturer', (ctx) =>
      putDeveloperProfile(ctx, BACKGROUND_STUDENTS.first.courseParticipationId, {
        appleID: BACKGROUND_STUDENTS.first.appleId,
        gitLabUsername: BACKGROUND_STUDENTS.first.gitlabUsername,
        hasMacBook: BACKGROUND_STUDENTS.first.hasMacbook,
      }),
    )
  })

  test('opens the edit dialog and saves a change', async ({ page }) => {
    const profiles = new DeveloperProfilesPage(page)
    await profiles.goto()
    await profiles.expectTableLoaded()

    await profiles.openEditDialog(BACKGROUND_STUDENTS.first.courseParticipationId)
    await expect(profiles.dialog).toContainText('Edit Developer Profile')
    await expect(profiles.dialog).toContainText(BACKGROUND_STUDENTS.first.lastName)

    await profiles.dialog.getByLabel('Apple ID').fill(NEW_APPLE_ID)
    await profiles.dialog.getByRole('button', { name: 'Save Profile' }).click()

    await expect(profiles.dialog).toBeHidden({ timeout: 30_000 })
    await expect(profiles.row(BACKGROUND_STUDENTS.first.courseParticipationId)).toContainText(
      NEW_APPLE_ID,
    )
  })

  test('shows Create Developer Profile for a participant without one', async ({ page }) => {
    const profiles = new DeveloperProfilesPage(page)
    await profiles.goto()
    await profiles.expectTableLoaded()

    await profiles.openEditDialog(STUDENT_WITHOUT_PROFILE.courseParticipationId)
    await expect(profiles.dialog).toContainText('Create Developer Profile')
    await profiles.dialog.getByRole('button', { name: 'Cancel' }).click()
    await expect(profiles.dialog).toBeHidden()
  })
})
