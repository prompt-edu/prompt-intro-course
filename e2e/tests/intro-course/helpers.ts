import { APIRequestContext } from '@playwright/test'
import { introCourseContextFor } from '../../src/fixtures/api'
import { INTRO_COURSE_PHASE_ID } from '../../src/data/constants'
import { INTRO_COURSE_API } from '../../src/env'
import { Role } from '../../src/data/roles'

// Thin typed API primitives for spec setup and teardown. They talk to the
// intro-course server directly (not through the UI), so a spec can restore the
// seeded state in afterAll even when it failed halfway through.

function phaseUrl(path: string): string {
  return `${INTRO_COURSE_API}/course_phase/${INTRO_COURSE_PHASE_ID}${path}`
}

async function expectOk(res: { ok(): boolean; status(): number; text(): Promise<string> }) {
  if (!res.ok()) {
    throw new Error(`${res.status()} ${await res.text()}`)
  }
}

// The server's pgtype.Text / pgtype.UUID fields marshal to a JSON string or
// null, never an object, so these are plain nullable strings on the wire.
export interface Seat {
  seatName: string
  hasMac: boolean
  deviceID: string | null
  assignedStudent: string | null
  assignedTutor: string | null
  isTutorSeat: boolean
}

export interface PeerAssignmentDTO {
  studentID: string
  peerID: string
}

export async function getSeatPlan(ctx: APIRequestContext): Promise<Seat[]> {
  const res = await ctx.get(phaseUrl('/seat_plan'))
  await expectOk(res)
  return (await res.json()) as Seat[]
}

export async function updateSeats(ctx: APIRequestContext, seats: Seat[]): Promise<void> {
  const res = await ctx.put(phaseUrl('/seat_plan'), { data: seats })
  await expectOk(res)
}

export async function getPeerAssignments(ctx: APIRequestContext): Promise<PeerAssignmentDTO[]> {
  const res = await ctx.get(phaseUrl('/peer_assignments'))
  await expectOk(res)
  return ((await res.json()) as PeerAssignmentDTO[] | null) ?? []
}

export async function setPeerAssignments(
  ctx: APIRequestContext,
  assignments: PeerAssignmentDTO[],
): Promise<void> {
  const res = await ctx.put(phaseUrl('/peer_assignments'), { data: assignments })
  await expectOk(res)
}

export interface DeveloperProfileDTO {
  courseParticipationID: string
  gitLabUsername: string
  appleID: string
  hasMacBook: boolean
  iPhoneUDID?: string
  iPadUDID?: string
  appleWatchUDID?: string
}

export async function getDeveloperProfiles(
  ctx: APIRequestContext,
): Promise<DeveloperProfileDTO[]> {
  const res = await ctx.get(phaseUrl('/developer_profile'))
  await expectOk(res)
  return ((await res.json()) as DeveloperProfileDTO[] | null) ?? []
}

// The lecturer-side upsert (PUT), used for restoring state in teardown.
export async function putDeveloperProfile(
  ctx: APIRequestContext,
  courseParticipationId: string,
  profile: {
    appleID: string
    gitLabUsername: string
    hasMacBook: boolean
    iPhoneUDID?: string | null
    iPadUDID?: string | null
    appleWatchUDID?: string | null
  },
): Promise<void> {
  const res = await ctx.put(phaseUrl(`/developer_profile/${courseParticipationId}`), {
    data: {
      iPhoneUDID: null,
      iPadUDID: null,
      appleWatchUDID: null,
      ...profile,
    },
  })
  await expectOk(res)
}

// Mints its own context so teardown works even if the test failed before its
// fixtures were used.
export async function withIntroCourseApi<T>(
  role: Role,
  fn: (ctx: APIRequestContext) => Promise<T>,
): Promise<T> {
  const ctx = await introCourseContextFor(role)
  try {
    return await fn(ctx)
  } finally {
    await ctx.dispose()
  }
}
