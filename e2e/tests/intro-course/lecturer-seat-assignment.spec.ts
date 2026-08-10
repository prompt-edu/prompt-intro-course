import { test, expect } from '../../src/fixtures/auth'
import { SeatAssignmentPage } from '../../src/pages/SeatAssignmentPage'
import {
  INTRO_COURSE_PARTICIPANT_COUNT,
  SEAT_PLAN,
  SEEDED_TUTORS,
  STUDENT_WITH_PROFILE,
} from '../../src/data/constants'
import { getSeatPlan, updateSeats, withIntroCourseApi } from './helpers'
import type { Seat } from './helpers'

// The seeded Rechnerhalle plan: 89 seats over 9 rows, 6 tutor seats, 12 chair
// Macs, and 57 of the 58 participants seated (Stan has no profile and no seat).
test.use({ role: 'course-lecturer' })

let seatPlanBackup: Seat[] = []

test.beforeAll(async () => {
  seatPlanBackup = await withIntroCourseApi('course-lecturer', getSeatPlan)
  expect(seatPlanBackup).toHaveLength(SEAT_PLAN.totalSeats)
})

// PUT /seat_plan upserts every seat it is given, so replaying the snapshot after
// each test restores the fixture — including when a test failed mid-mutation.
// This keeps the tests below independent of each other's order and leaves the
// seed intact for the peer-assignment and student specs.
test.afterEach(async () => {
  if (seatPlanBackup.length > 0) {
    await withIntroCourseApi('course-lecturer', (ctx) => updateSeats(ctx, seatPlanBackup))
  }
})

test.describe('seat assignment: reading the seeded plan', () => {
  test('renders the four workflow steps and the grid', async ({ page }) => {
    const seats = new SeatAssignmentPage(page)
    await seats.goto()
    await seats.expectSeatPlanLoaded()

    for (const step of [
      'Step 1: Seat Plan Configuration',
      'Step 2: Mac Assignment',
      'Step 3: Tutor Assignment',
      'Step 4: Student Assignment',
    ]) {
      await expect(seats.stepCard(step)).toBeVisible()
    }
  })

  test('seeded seats keep their tutor, Mac, and student identity', async ({ page }) => {
    const seats = new SeatAssignmentPage(page)
    await seats.goto()
    await seats.expectSeatPlanLoaded()

    // Identity assertions rather than row counts, so the spec survives reruns.
    expect(await seats.seatTitle(SEAT_PLAN.tutorSeats[0])).toContain('(Tutor seat)')
    // Row 5 locals 1-6 are David's chair Macs.
    expect(await seats.seatTitle('1-5-1')).toContain('(Mac)')
    // Selma sits on the one free non-tutor seat in Alice's row.
    expect(await seats.seatStudentInitials(STUDENT_WITH_PROFILE.seatName)).toBe(
      `${STUDENT_WITH_PROFILE.firstName.charAt(0)}${STUDENT_WITH_PROFILE.lastName.charAt(0)}`,
    )
  })

  test('the assignment status reflects the seeded participants', async ({ page }) => {
    const seats = new SeatAssignmentPage(page)
    await seats.goto()
    await seats.expectSeatPlanLoaded()

    // 57 of 58: every background student plus Selma, but not Stan.
    await expect(seats.assignmentStatus).toContainText(
      `Partially Assigned (${INTRO_COURSE_PARTICIPANT_COUNT - 1}/${INTRO_COURSE_PARTICIPANT_COUNT})`,
    )
  })

  test('the tutor distribution lists all six tutors', async ({ page }) => {
    const seats = new SeatAssignmentPage(page)
    await seats.goto()
    await seats.expectSeatPlanLoaded()

    for (const tutor of SEEDED_TUTORS) {
      await expect(
        page.getByText(`${tutor.firstName} ${tutor.lastName}`, { exact: true }).first(),
      ).toBeVisible()
    }
  })
})

test.describe('seat assignment: grid interaction', () => {
  test('switches between the Table and Grid views', async ({ page }) => {
    const seats = new SeatAssignmentPage(page)
    await seats.goto()
    await seats.expectSeatPlanLoaded()

    await seats.tableViewButton.click()
    await expect(seats.seat(SEAT_PLAN.firstStudentSeat)).toBeHidden()

    await seats.gridViewButton.click()
    await expect(seats.seat(SEAT_PLAN.firstStudentSeat)).toBeVisible()
  })

  test('switches the grid between tutor, peer group, and seat view modes', async ({ page }) => {
    const seats = new SeatAssignmentPage(page)
    await seats.goto()
    await seats.expectSeatPlanLoaded()

    // 'Peer Group' is only offered because the seed has peer assignments.
    for (const mode of ['Tutor', 'Peer Group', 'Seat'] as const) {
      await seats.viewModeButton(mode).click()
      await expect(seats.viewModeButton(mode)).toHaveAttribute('data-state', 'on')
      await expect(seats.seat(SEAT_PLAN.firstStudentSeat)).toBeVisible()
    }

    // In seat view the cell label is the row-local seat name, not initials.
    await expect(seats.seat(SEAT_PLAN.firstStudentSeat)).toContainText('1-1')
  })

  test('selecting a seat offers the swap hint and deselects again', async ({ page }) => {
    const seats = new SeatAssignmentPage(page)
    await seats.goto()
    await seats.expectSeatPlanLoaded()

    await seats.selectSeat(SEAT_PLAN.firstStudentSeat)
    await seats.seat(SEAT_PLAN.firstStudentSeat).click()
    await expect(
      page.getByText('Click another seat to swap, or click the same seat to deselect.'),
    ).toBeHidden()
  })

  test('swapping two seats moves the students and persists', async ({ page }) => {
    const seats = new SeatAssignmentPage(page)
    await seats.goto()
    await seats.expectSeatPlanLoaded()

    const before1 = await seats.seatStudentInitials('1-1-1')
    const before2 = await seats.seatStudentInitials('1-1-2')
    expect(before1).not.toBeNull()
    expect(before2).not.toBeNull()
    expect(before1).not.toEqual(before2)

    await seats.swapSeats('1-1-1', '1-1-2')

    await expect.poll(() => seats.seatStudentInitials('1-1-1'), { timeout: 15_000 }).toBe(before2)
    expect(await seats.seatStudentInitials('1-1-2')).toBe(before1)

    // Reload to prove the swap reached the server rather than only local state.
    await page.reload()
    await seats.expectSeatPlanLoaded()
    expect(await seats.seatStudentInitials('1-1-1')).toBe(before2)
  })
})

test.describe('seat assignment: assigning students', () => {
  test('Assign Random is disabled while students are already seated', async ({ page }) => {
    const seats = new SeatAssignmentPage(page)
    await seats.goto()
    await seats.expectSeatPlanLoaded()

    await expect(page.getByRole('button', { name: 'Assign Random' })).toBeDisabled()
  })

  test('Smart Assign seats the last unseated student', async ({ page }) => {
    const seats = new SeatAssignmentPage(page)
    await seats.goto()
    await seats.expectSeatPlanLoaded()

    // Stan is the only participant without a seat, so one Smart Assign completes
    // the plan.
    await page.getByRole('button', { name: 'Smart Assign' }).click()
    await expect(seats.assignmentStatus).toContainText(
      `Fully Assigned (${INTRO_COURSE_PARTICIPANT_COUNT}/${INTRO_COURSE_PARTICIPANT_COUNT})`,
      { timeout: 30_000 },
    )
  })

  test('Reset Assignments clears every student after confirmation', async ({ page }) => {
    const seats = new SeatAssignmentPage(page)
    await seats.goto()
    await seats.expectSeatPlanLoaded()

    await seats.resetStudentAssignments()
    await expect(seats.assignmentStatus).toContainText('Not Assigned', { timeout: 30_000 })

    // The grid is only rendered while at least one seat has a student, so it
    // disappears entirely — but the seat plan itself survives, which the step
    // cards still being present proves.
    await expect(seats.seat(SEAT_PLAN.firstStudentSeat)).toBeHidden()
    await expect(seats.stepCard('Step 1: Seat Plan Configuration')).toBeVisible()

    const remaining = await withIntroCourseApi('course-lecturer', getSeatPlan)
    expect(remaining).toHaveLength(SEAT_PLAN.totalSeats)
    expect(remaining.every((s) => s.assignedStudent === null)).toBe(true)
  })
})
