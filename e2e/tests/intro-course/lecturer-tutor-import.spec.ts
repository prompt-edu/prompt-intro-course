import { test, expect } from '../../src/fixtures/auth'
import { TutorImportPage } from '../../src/pages/TutorImportPage'
import { SEEDED_TUTORS } from '../../src/data/constants'

// The tutor table is gated on the phase carrying a keycloakGroup in its
// restrictedData. The seed leaves restricted_data empty, so the first test walks
// through creating the group — which exercises core's Keycloak realm-management
// path with the prompt-server service account (granted realm-admin in the e2e
// realm). ensureKeycloakGroup() is idempotent, so retries and reruns behave the
// same on a stack where the group already exists.
test.use({ role: 'course-lecturer' })

test.describe('tutor import: the Keycloak group gate', () => {
  test('creating the Keycloak group unlocks the tutor table', async ({ page }) => {
    const tutors = new TutorImportPage(page)
    await tutors.goto()
    await tutors.expectLoaded()

    await expect(tutors.keycloakGroupStatus).toBeVisible()

    // The status starts as "Checking..."; wait for a settled state before branching.
    await expect(tutors.groupCreatedStatus.or(tutors.createGroupButton).first()).toBeVisible({
      timeout: 30_000,
    })

    if (!(await tutors.groupCreatedStatus.isVisible())) {
      // Before creation the table is replaced by a plain-text hint.
      await expect(tutors.gatedHint).toBeVisible()
      await expect(tutors.importedTutorsHeading).toBeHidden()
      await tutors.createGroupButton.click()
    }

    await expect(tutors.groupCreatedStatus).toBeVisible({ timeout: 30_000 })
    await expect(tutors.importedTutorsHeading).toBeVisible({ timeout: 30_000 })
    await expect(tutors.gatedHint).toBeHidden()
  })
})

test.describe('tutor import: the tutor table', () => {
  test.beforeEach(async ({ page }) => {
    const tutors = new TutorImportPage(page)
    await tutors.goto()
    await tutors.ensureKeycloakGroup()
  })

  test('lists all six seeded tutors with their details', async ({ page }) => {
    const tutors = new TutorImportPage(page)

    for (const tutor of SEEDED_TUTORS) {
      const row = tutors.tutorRow(tutor.id)
      await expect(row).toBeVisible()
      await expect(row).toContainText(tutor.firstName)
      await expect(row).toContainText(tutor.lastName)
      await expect(tutors.gitlabInput(tutor.id)).toHaveValue(tutor.gitlabUsername)
    }

    await expect(page.getByText('No tutors found.')).toBeHidden()
  })

  test('searching narrows the table to the matching tutor', async ({ page }) => {
    const tutors = new TutorImportPage(page)

    await tutors.searchInput.fill(SEEDED_TUTORS[0].lastName)
    await expect(tutors.tutorRow(SEEDED_TUTORS[0].id)).toBeVisible()

    // The search input is wired to local state; asserting on a non-match keeps
    // the test honest about the filter actually being applied.
    await tutors.searchInput.fill('definitely-no-such-tutor')
    await expect(page.getByText('No tutors found.')).toBeVisible()
  })

  // Nested so only the mutating test pays for the restore hook. The restore lives
  // in a hook rather than the test body: a failed assertion or a timed-out reload
  // would otherwise leak the edited username into `lists all six seeded tutors`,
  // which asserts the seeded value against this same shared stack.
  test.describe('editing a GitLab username', () => {
    test.afterEach(async ({ page }) => {
      const tutors = new TutorImportPage(page)
      const tutor = SEEDED_TUTORS[1]

      await tutors.goto()
      await tutors.ensureKeycloakGroup()
      const input = tutors.gitlabInput(tutor.id)
      if ((await input.inputValue()) === tutor.gitlabUsername) return

      // Restore through the same UI path, then confirm the save landed.
      await input.fill(tutor.gitlabUsername)
      await input.blur()
      await page.reload()
      await tutors.ensureKeycloakGroup()
      await expect(tutors.gitlabInput(tutor.id)).toHaveValue(tutor.gitlabUsername)
    })

    test('saves on blur and persists', async ({ page }) => {
      const tutors = new TutorImportPage(page)
      const tutor = SEEDED_TUTORS[1]
      const edited = `${tutor.gitlabUsername}-e2e`

      const input = tutors.gitlabInput(tutor.id)
      await input.fill(edited)
      await input.blur()

      await page.reload()
      await tutors.ensureKeycloakGroup()
      await expect(tutors.gitlabInput(tutor.id)).toHaveValue(edited)
    })
  })

  test('the import dialog opens', async ({ page }) => {
    const tutors = new TutorImportPage(page)

    await tutors.importDialogTrigger.click()
    await expect(page.getByRole('dialog')).toBeVisible()
    await page.keyboard.press('Escape')
    await expect(page.getByRole('dialog')).toBeHidden()
  })

  test('Sync to Keycloak reports back without breaking the page', async ({ page }) => {
    const tutors = new TutorImportPage(page)

    // Re-imports the same tutors, which the server upserts and then retries
    // against Keycloak. It either succeeds silently or surfaces warnings; both
    // are acceptable, what matters is that the table survives.
    await tutors.syncToKeycloakButton.click()
    await expect(tutors.syncToKeycloakButton).toBeEnabled({ timeout: 60_000 })
    await expect(tutors.tutorRow(SEEDED_TUTORS[0].id)).toBeVisible()
  })
})
