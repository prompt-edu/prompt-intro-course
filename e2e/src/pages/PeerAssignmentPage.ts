import { Locator, expect } from '@playwright/test'
import { INTRO_COURSE_ROUTES } from '../data/constants'
import { IntroCoursePhasePage } from './IntroCoursePhasePage'

// /peer-assignments — generate, edit, and sync peer review groups.
export class PeerAssignmentPage extends IntroCoursePhasePage {
  protected readonly route = INTRO_COURSE_ROUTES.peerAssignments
  protected readonly headingName = 'Peer Assignments'

  get status(): Locator {
    return this.page.getByTestId('peer-assignment-status')
  }

  get error(): Locator {
    return this.page.getByTestId('peer-assignment-error')
  }

  get groups(): Locator {
    return this.page.getByTestId('peer-group')
  }

  // The per-tutor card the groups are bucketed into, keyed by the tutor id the
  // group's first member is seated with.
  tutorCard(tutorId: string): Locator {
    return this.page.getByTestId(`peer-tutor-card-${tutorId}`)
  }

  member(courseParticipationId: string): Locator {
    return this.page.getByTestId(`peer-group-member-${courseParticipationId}`)
  }

  // Every rendered group member, across all tutor cards.
  get members(): Locator {
    return this.page.getByTestId(/^peer-group-member-/)
  }

  get emptyState(): Locator {
    return this.page.getByRole('heading', { level: 3, name: 'No Peer Assignments' })
  }

  get generateButton(): Locator {
    return this.page.getByRole('button', { name: 'Generate Groups' })
  }

  get clearAllButton(): Locator {
    return this.page.getByRole('button', { name: 'Clear All', exact: true })
  }

  get syncButton(): Locator {
    return this.page.getByRole('button', { name: 'Sync to GitLab' })
  }

  get unsyncButton(): Locator {
    return this.page.getByRole('button', { name: 'Unsync from GitLab' })
  }

  get editGroupsButton(): Locator {
    return this.page.getByRole('button', { name: 'Edit Groups' })
  }

  get saveChangesButton(): Locator {
    return this.page.getByRole('button', { name: 'Save Changes' })
  }

  get cancelEditButton(): Locator {
    return this.page.getByRole('button', { name: 'Cancel' })
  }

  get unassignedStudentsCard(): Locator {
    return this.page.getByTestId('peer-unassigned-students')
  }

  async expectGroupsLoaded() {
    await this.expectLoaded()
    await expect(this.status).toBeVisible({ timeout: 30_000 })
  }

  // Groups are recomputed from the flat bidirectional assignments, so the count
  // is the number of connected components.
  async groupCount(): Promise<number> {
    return this.groups.count()
  }

  // Confirms through the AlertDialog. Its action button carries the same label
  // as the trigger, so scope to the dialog.
  async clearAll() {
    await this.clearAllButton.click()
    const dialog = this.page.getByRole('alertdialog')
    await expect(dialog.getByText('Clear all peer assignments?')).toBeVisible()
    await dialog.getByRole('button', { name: 'Clear All' }).click()
  }
}
