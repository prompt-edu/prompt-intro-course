import { Locator, expect } from '@playwright/test'
import { INTRO_COURSE_ROUTES } from '../data/constants'
import { IntroCoursePhasePage } from './IntroCoursePhasePage'

// /seat-assignments — the lecturer's four-step seat plan workflow plus the
// Rechnerhalle grid. Every step card is only rendered once a seat plan exists.
export class SeatAssignmentPage extends IntroCoursePhasePage {
  protected readonly route = INTRO_COURSE_ROUTES.seatAssignments
  protected readonly headingName = 'Seat Assignment'

  stepCard(title: string): Locator {
    return this.page.getByText(title, { exact: true })
  }

  // A seat in the grid. The visible text is only initials or a position number,
  // so seats are addressed by the testid added in SeatCell.tsx.
  seat(seatName: string): Locator {
    return this.page.getByTestId(`seat-${seatName}`)
  }

  get assignmentStatus(): Locator {
    return this.page.getByTestId('seat-assignment-status')
  }

  get tableViewButton(): Locator {
    return this.page.getByRole('button', { name: 'Table', exact: true })
  }

  get gridViewButton(): Locator {
    return this.page.getByRole('button', { name: 'Grid', exact: true })
  }

  // The grid's view-mode toggle group. 'Peer Group' only appears once peer
  // assignments exist.
  viewModeButton(mode: 'Tutor' | 'Peer Group' | 'Seat'): Locator {
    return this.page.getByRole('radio', { name: mode })
  }

  async expectSeatPlanLoaded() {
    await this.expectLoaded()
    await expect(this.stepCard('Step 1: Seat Plan Configuration')).toBeVisible()
    await expect(this.seat('1-1-1')).toBeVisible({ timeout: 30_000 })
  }

  // Selecting a seat opens the swap hint; selecting it again deselects.
  async selectSeat(seatName: string) {
    await this.seat(seatName).click()
    await expect(
      this.page.getByText('Click another seat to swap, or click the same seat to deselect.'),
    ).toBeVisible()
  }

  async swapSeats(from: string, to: string) {
    await this.selectSeat(from)
    await this.seat(to).click()
    await expect(
      this.page.getByText('Click another seat to swap, or click the same seat to deselect.'),
    ).toBeHidden()
  }

  // The seat's title attribute carries the full identity:
  //   `<name>[ (Tutor seat)][ - <initials>][ P<group>][ (Mac)]`
  async seatTitle(seatName: string): Promise<string> {
    return (await this.seat(seatName).getAttribute('title')) ?? ''
  }

  // The seated student's initials, or null for an empty seat.
  async seatStudentInitials(seatName: string): Promise<string | null> {
    const title = await this.seatTitle(seatName)
    return / - (\S+)/.exec(title)?.[1] ?? null
  }

  get resetAssignmentsButton(): Locator {
    return this.page.getByRole('button', { name: 'Reset Assignments' })
  }

  // A plain Dialog (not an AlertDialog), so the role is 'dialog'.
  async resetStudentAssignments() {
    await this.resetAssignmentsButton.click()
    const dialog = this.page.getByRole('dialog')
    await expect(dialog.getByText('Reset Student Assignments')).toBeVisible()
    await dialog.getByRole('button', { name: 'Yes, Reset Assignments' }).click()
  }
}
