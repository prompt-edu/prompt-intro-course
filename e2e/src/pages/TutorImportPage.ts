import { Locator, expect } from '@playwright/test'
import { INTRO_COURSE_ROUTES } from '../data/constants'
import { IntroCoursePhasePage } from './IntroCoursePhasePage'

// /tutors — Keycloak group management plus the imported tutor table. The table is
// GATED on the phase having a keycloakGroup in its restrictedData; until then the
// page shows a plain-text hint instead.
export class TutorImportPage extends IntroCoursePhasePage {
  protected readonly route = INTRO_COURSE_ROUTES.tutors
  protected readonly headingName = 'Tutor Import'

  get keycloakGroupStatus(): Locator {
    return this.page.getByRole('heading', { level: 2, name: 'Keycloak Group Status' })
  }

  get createGroupButton(): Locator {
    return this.page.getByRole('button', { name: 'Create Keycloak Group' })
  }

  get gatedHint(): Locator {
    return this.page.getByText('Please create a Keycloak group first before adding Tutors')
  }

  get importedTutorsHeading(): Locator {
    return this.page.getByRole('heading', { level: 2, name: 'Imported Tutors' })
  }

  get searchInput(): Locator {
    return this.page.getByPlaceholder('Search tutors...')
  }

  get syncToKeycloakButton(): Locator {
    return this.page.getByRole('button', { name: 'Sync to Keycloak' })
  }

  get importDialogTrigger(): Locator {
    return this.page.getByRole('button', { name: 'Import Tutors' })
  }

  tutorRow(tutorId: string): Locator {
    return this.page.getByTestId(`tutor-row-${tutorId}`)
  }

  gitlabInput(tutorId: string): Locator {
    return this.page.getByTestId(`tutor-gitlab-input-${tutorId}`)
  }

  get groupCreatedStatus(): Locator {
    return this.page.getByText('Keycloak group has been created')
  }

  // Idempotent: creates the Keycloak group only if it does not exist yet, so the
  // spec behaves the same on a fresh stack and on a retry.
  //
  // The status starts as "Checking Keycloak group status..." until the phase query
  // resolves, so wait for one of the two settled states before branching — a bare
  // isVisible() check races the fetch and silently skips the click.
  async ensureKeycloakGroup() {
    await this.expectLoaded()
    await expect(this.keycloakGroupStatus).toBeVisible()
    await expect(this.groupCreatedStatus.or(this.createGroupButton).first()).toBeVisible({
      timeout: 30_000,
    })
    if (!(await this.groupCreatedStatus.isVisible())) {
      await this.createGroupButton.click()
    }
    await expect(this.groupCreatedStatus).toBeVisible({ timeout: 30_000 })
    await expect(this.importedTutorsHeading).toBeVisible({ timeout: 30_000 })
  }
}
