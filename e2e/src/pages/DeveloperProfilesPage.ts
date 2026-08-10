import { Locator, expect } from '@playwright/test'
import { INTRO_COURSE_ROUTES } from '../data/constants'
import { IntroCoursePhasePage } from './IntroCoursePhasePage'

// /developer-profiles — the lecturer's table of every participant's survey
// answers, with sorting, filtering, and a per-row edit dialog.
export class DeveloperProfilesPage extends IntroCoursePhasePage {
  protected readonly route = INTRO_COURSE_ROUTES.developerProfiles
  protected readonly headingName = 'Developer Profile Management'

  get summary(): Locator {
    return this.page.getByTestId('developer-profile-summary')
  }

  row(courseParticipationId: string): Locator {
    return this.page.getByTestId(`developer-profile-row-${courseParticipationId}`)
  }

  get rows(): Locator {
    return this.page.locator('[data-testid^="developer-profile-row-"]')
  }

  // `exact` matters: a substring match on 'Name' also hits 'GitLab Username'.
  columnHeader(name: 'Name' | 'Survey'): Locator {
    return this.page.getByRole('columnheader', { name, exact: true })
  }

  get filtersButton(): Locator {
    return this.page.getByRole('button', { name: 'Filters' })
  }

  get downloadButton(): Locator {
    return this.page.getByRole('button', { name: 'Download Profiles' })
  }

  get dialog(): Locator {
    return this.page.getByRole('dialog')
  }

  async expectTableLoaded() {
    await this.expectLoaded()
    await expect(this.summary).toBeVisible({ timeout: 30_000 })
  }

  // "Showing X of Y participants"
  async shownCount(): Promise<number> {
    const text = (await this.summary.textContent()) ?? ''
    return Number(/Showing (\d+) of/.exec(text)?.[1] ?? -1)
  }

  async totalCount(): Promise<number> {
    const text = (await this.summary.textContent()) ?? ''
    return Number(/of (\d+) participants/.exec(text)?.[1] ?? -1)
  }

  // The name cell of every rendered row, in table order — used to assert that
  // sorting actually reorders. Note the per-row scope: `rows.locator('td').first()`
  // would collapse to the first cell of the first row only.
  async visibleNames(): Promise<string[]> {
    return this.page
      .locator('[data-testid^="developer-profile-row-"] td:first-child')
      .allInnerTexts()
  }

  async toggleFilter(label: string) {
    await this.filtersButton.click()
    await this.page.getByRole('menuitemcheckbox', { name: label }).click()
    // The dropdown keeps focus after a click, so close it explicitly.
    await this.page.keyboard.press('Escape')
  }

  async openEditDialog(courseParticipationId: string) {
    await this.row(courseParticipationId).getByRole('button', { name: 'Edit' }).click()
    await expect(this.dialog).toBeVisible()
  }
}
