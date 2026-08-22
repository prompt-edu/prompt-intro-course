import { test, expect } from '../../src/fixtures/auth'
import { PeerAssignmentPage } from '../../src/pages/PeerAssignmentPage'
import {
  BACKGROUND_STUDENTS,
  GROUPED_STUDENT_COUNT,
  SEEDED_TUTORS,
  STUDENT_WITH_PROFILE,
} from '../../src/data/constants'
import {
  getPeerAssignments,
  setPeerAssignments,
  withIntroCourseApi,
  type PeerAssignmentDTO,
} from './helpers'

// The seed has 17 peer groups over the 56 background students, with Selma joining
// group A1 as its fourth member — so 57 of the 57 seated students are grouped.
test.use({ role: 'course-lecturer' })

let peerBackup: PeerAssignmentDTO[] = []

test.beforeAll(async () => {
  peerBackup = await withIntroCourseApi('course-lecturer', getPeerAssignments)
  expect(peerBackup.length).toBeGreaterThan(0)
})

// PUT /peer_assignments replaces the whole set, so replaying the snapshot after
// each test keeps the tests order-independent and restores the fixture for the
// seat-assignment and student specs.
test.afterEach(async () => {
  if (peerBackup.length > 0) {
    await withIntroCourseApi('course-lecturer', (ctx) => setPeerAssignments(ctx, peerBackup))
  }
})

test.describe('peer assignments: reading the seeded groups', () => {
  test('the status badge counts every grouped student', async ({ page }) => {
    const peers = new PeerAssignmentPage(page)
    await peers.goto()
    await peers.expectGroupsLoaded()

    await expect(peers.status).toHaveText(
      `${GROUPED_STUDENT_COUNT} of ${GROUPED_STUDENT_COUNT} students grouped`,
    )
  })

  test('groups are bucketed under the tutor their members sit with', async ({ page }) => {
    const peers = new PeerAssignmentPage(page)
    await peers.goto()
    await peers.expectGroupsLoaded()

    // Alice owns row 1, which holds group A1.
    const alice = SEEDED_TUTORS[0]
    await expect(peers.tutorCard(alice.id)).toBeVisible()
    await expect(peers.tutorCard(alice.id)).toContainText(`${alice.firstName} ${alice.lastName}`)

    // Selma joined A1, so she and Max Mueller share a card and a group.
    await expect(peers.member(STUDENT_WITH_PROFILE.courseParticipationId)).toBeVisible()
    await expect(peers.member(BACKGROUND_STUDENTS.first.courseParticipationId)).toBeVisible()

    const group = peers.groups.filter({
      has: peers.member(STUDENT_WITH_PROFILE.courseParticipationId),
    })
    await expect(group).toHaveCount(1)
    await expect(group).toContainText(BACKGROUND_STUDENTS.first.gitlabUsername)
    // A1 has four members once Selma joins.
    await expect(group).toContainText('Quad')
  })

  test('every seeded group is rendered', async ({ page }) => {
    const peers = new PeerAssignmentPage(page)
    await peers.goto()
    await peers.expectGroupsLoaded()

    // The UI recomputes groups as connected components of the flat assignment
    // list, so this cross-checks the client's grouping against the server's data.
    const expected = new Set<string>()
    for (const a of peerBackup) {
      expected.add(a.studentID)
    }
    expect(await peers.groupCount()).toBeGreaterThan(0)
    await expect(peers.members).toHaveCount(expected.size)
  })
})

test.describe('peer assignments: mutating the groups', () => {
  test('Clear All removes every group after confirmation', async ({ page }) => {
    const peers = new PeerAssignmentPage(page)
    await peers.goto()
    await peers.expectGroupsLoaded()

    await peers.clearAll()

    await expect(peers.emptyState).toBeVisible({ timeout: 30_000 })
    await expect(peers.status).toHaveText(`0 of ${GROUPED_STUDENT_COUNT} students grouped`)
  })

  test('Generate Groups regroups every seated student', async ({ page }) => {
    const peers = new PeerAssignmentPage(page)
    await peers.goto()
    await peers.expectGroupsLoaded()

    // Start from empty so the assertion is about generation, not about the seed.
    await peers.clearAll()
    await expect(peers.status).toHaveText(`0 of ${GROUPED_STUDENT_COUNT} students grouped`)

    await peers.generateButton.click()

    await expect(peers.status).toHaveText(
      `${GROUPED_STUDENT_COUNT} of ${GROUPED_STUDENT_COUNT} students grouped`,
      { timeout: 30_000 },
    )
    // Generation partitions into 3s and 4s within each tutor group.
    expect(await peers.groupCount()).toBeGreaterThan(0)
    await expect(peers.groups.first()).toContainText(/Triple|Quad/)
  })

  test('Edit Groups then Save Changes persists a membership change', async ({ page }) => {
    const peers = new PeerAssignmentPage(page)
    await peers.goto()
    await peers.expectGroupsLoaded()

    await peers.editGroupsButton.click()
    await expect(peers.saveChangesButton).toBeVisible()

    // Remove Selma from her group; the remaining members stay together.
    const selma = peers.member(STUDENT_WITH_PROFILE.courseParticipationId)
    await selma.getByRole('button').click()
    await expect(selma).toBeHidden()

    await peers.saveChangesButton.click()
    await expect(peers.editGroupsButton).toBeVisible({ timeout: 30_000 })

    // Persisted: she is gone after a reload, and the group is a Triple again.
    await page.reload()
    await peers.expectGroupsLoaded()
    await expect(peers.member(STUDENT_WITH_PROFILE.courseParticipationId)).toBeHidden()
    await expect(peers.status).toHaveText(
      `${GROUPED_STUDENT_COUNT - 1} of ${GROUPED_STUDENT_COUNT} students grouped`,
    )
  })

  test('Cancel discards an unsaved membership change', async ({ page }) => {
    const peers = new PeerAssignmentPage(page)
    await peers.goto()
    await peers.expectGroupsLoaded()

    await peers.editGroupsButton.click()
    const selma = peers.member(STUDENT_WITH_PROFILE.courseParticipationId)
    await selma.getByRole('button').click()
    await expect(selma).toBeHidden()

    await peers.cancelEditButton.click()
    await expect(peers.editGroupsButton).toBeVisible()
    await expect(peers.member(STUDENT_WITH_PROFILE.courseParticipationId)).toBeVisible()
  })
})

test.describe('peer assignments: GitLab sync without a token', () => {
  // GITLAB_ACCESS_TOKEN is deliberately unset in the e2e stack, so the server's
  // GitLab client is nil and these endpoints fail by construction. Assert the
  // surfaced error rather than talking to a real GitLab.
  test('Sync to GitLab surfaces an error', async ({ page }) => {
    const peers = new PeerAssignmentPage(page)
    await peers.goto()
    await peers.expectGroupsLoaded()

    await peers.syncButton.click()
    await expect(peers.error).toBeVisible({ timeout: 30_000 })
    await expect(peers.error).toContainText('GitLab')
  })

  test('Unsync from GitLab surfaces an error', async ({ page }) => {
    const peers = new PeerAssignmentPage(page)
    await peers.goto()
    await peers.expectGroupsLoaded()

    await peers.unsyncButton.click()
    await expect(peers.error).toBeVisible({ timeout: 30_000 })
    await expect(peers.error).toContainText('GitLab')
  })
})
