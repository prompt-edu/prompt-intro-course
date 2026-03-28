import { Seat } from '../../../interfaces/Seat'
import { PeerAssignment } from '../../../interfaces/PeerAssignment'
import { DeveloperWithProfile } from '../interfaces/DeveloperWithProfile'
import { parseSeatName } from './seatGrid'

function shuffle<T>(arr: T[]): T[] {
  const a = [...arr]
  for (let i = a.length - 1; i > 0; i--) {
    const j = Math.floor(Math.random() * (i + 1))
    ;[a[i], a[j]] = [a[j], a[i]]
  }
  return a
}

/**
 * Smart seating assignment algorithm.
 *
 * Strategy:
 * 1. Use the EXISTING tutor-seat mapping. Distribute students proportionally
 *    to seat count per tutor (max 10 students per tutor).
 * 2. Mac-needy students are redistributed to tutors with Mac seats, balanced
 *    by remaining Mac capacity across multiple Mac tutors.
 * 3. For each tutor group, sort seats: Mac first, then by row/position.
 * 4. Within each group, order students: Mac-needy first, then unknown, then Mac owners.
 * 5. If peer assignments exist, reorder students within each Mac-priority tier
 *    so that peer groups (triples/quads) end up adjacent.
 * 6. Assign students to seats sequentially — Mac-needy students land on Mac seats.
 */
export function smartAssign(
  seats: Seat[],
  developerWithProfiles: DeveloperWithProfile[],
  peerAssignments?: PeerAssignment[],
): Seat[] {
  // Group seats by tutor (use existing tutor-seat mapping)
  const tutorSeatMap = new Map<string, Seat[]>()
  for (const seat of seats) {
    if (!seat.assignedTutor) continue
    if (!tutorSeatMap.has(seat.assignedTutor)) tutorSeatMap.set(seat.assignedTutor, [])
    tutorSeatMap.get(seat.assignedTutor)!.push(seat)
  }

  const tutorIds = [...tutorSeatMap.keys()].sort()

  // Always clear assignments — even if no tutors assigned
  const updatedSeats = seats.map((s) => ({ ...s, assignedStudent: null as string | null }))

  if (tutorIds.length === 0) return updatedSeats

  // Count Mac seats per tutor
  const tutorMacCount = new Map<string, number>()
  for (const id of tutorIds) {
    tutorMacCount.set(id, (tutorSeatMap.get(id) ?? []).filter((s) => s.hasMac).length)
  }

  // Distribute students proportionally to seat count per tutor (max 10 per tutor)
  const MAX_STUDENTS_PER_TUTOR = 10
  const students = shuffle(developerWithProfiles)
  const tutorGroups = new Map<string, DeveloperWithProfile[]>()
  for (const id of tutorIds) tutorGroups.set(id, [])

  // Sort tutors by seat count descending; shuffle among equal-capacity tutors for fairness
  const tutorsByCapacity = shuffle([...tutorIds]).sort(
    (a, b) => (tutorSeatMap.get(b)?.length ?? 0) - (tutorSeatMap.get(a)?.length ?? 0),
  )

  const capacityLeft = new Map<string, number>()
  for (const id of tutorIds) {
    const seatCount = tutorSeatMap.get(id)?.length ?? 0
    capacityLeft.set(id, Math.min(seatCount, MAX_STUDENTS_PER_TUTOR))
  }

  for (const student of students) {
    // Pick tutor with most remaining capacity
    let bestTutor = tutorsByCapacity[0]
    let bestCapacity = capacityLeft.get(bestTutor) ?? 0
    for (const tid of tutorsByCapacity) {
      const cap = capacityLeft.get(tid) ?? 0
      if (cap > bestCapacity) {
        bestTutor = tid
        bestCapacity = cap
      }
    }
    tutorGroups.get(bestTutor)!.push(student)
    capacityLeft.set(bestTutor, bestCapacity - 1)
  }

  // Mac-aware redistribution: move Mac-needy students to tutors with Mac seats
  redistributeMacNeedy(tutorIds, tutorGroups, tutorMacCount)

  // Build symmetric peer adjacency map
  const peerMap = new Map<string, Set<string>>()
  if (peerAssignments) {
    for (const pa of peerAssignments) {
      if (!peerMap.has(pa.studentId)) peerMap.set(pa.studentId, new Set())
      peerMap.get(pa.studentId)!.add(pa.peerId)
      if (!peerMap.has(pa.peerId)) peerMap.set(pa.peerId, new Set())
      peerMap.get(pa.peerId)!.add(pa.studentId)
    }
  }

  // Build seat index for O(1) lookup
  const seatIndex = new Map<string, number>()
  updatedSeats.forEach((s, i) => seatIndex.set(s.seatName, i))

  // For each tutor group, assign students to that tutor's seats
  for (const tutorId of tutorIds) {
    const groupStudents = tutorGroups.get(tutorId) ?? []
    const groupSeats = tutorSeatMap.get(tutorId) ?? []
    if (groupStudents.length === 0 || groupSeats.length === 0) continue

    // Sort seats: Mac seats first (for Mac-needy students), then by row, then position
    const sortedSeats = [...groupSeats].sort((a, b) => {
      if (a.hasMac !== b.hasMac) return a.hasMac ? -1 : 1
      const pa = parseSeatName(a.seatName)
      const pb = parseSeatName(b.seatName)
      if (!pa || !pb) return 0
      if (pa.row !== pb.row) return pa.row - pb.row
      return pa.position - pb.position
    })

    // Separate students by Mac need
    const needMac = shuffle(groupStudents.filter((s) => s.profile?.hasMacBook === false))
    const unknownMac = shuffle(groupStudents.filter((s) => s.profile?.hasMacBook == null))
    const haveMac = shuffle(groupStudents.filter((s) => s.profile?.hasMacBook === true))

    // Within each tier, apply peer adjacency ordering
    const orderedStudents = [
      ...orderByPeerAdjacency(needMac, peerMap),
      ...orderByPeerAdjacency(unknownMac, peerMap),
      ...orderByPeerAdjacency(haveMac, peerMap),
    ]

    // Assign sequentially: Mac-needy students get Mac seats, rest fill remaining
    for (let i = 0; i < orderedStudents.length && i < sortedSeats.length; i++) {
      const idx = seatIndex.get(sortedSeats[i].seatName)
      if (idx !== undefined) {
        updatedSeats[idx].assignedStudent =
          orderedStudents[i].participation.courseParticipationID ?? null
      }
    }
  }

  return updatedSeats
}

/**
 * Redistribute Mac-needy students: swap Mac-needy students stuck under
 * tutors without Mac seats with non-Mac-needy students under tutors that
 * have Mac seats. Distributes across Mac tutors proportionally to their
 * remaining Mac capacity so no single tutor is overloaded.
 */
function redistributeMacNeedy(
  tutorIds: string[],
  tutorGroups: Map<string, DeveloperWithProfile[]>,
  tutorMacCount: Map<string, number>,
): void {
  // Collect Mac-needy students in groups without Mac seats
  const misplacedMacNeedy: { tutorId: string; index: number; student: DeveloperWithProfile }[] = []
  for (const tid of tutorIds) {
    if ((tutorMacCount.get(tid) ?? 0) > 0) continue
    const group = tutorGroups.get(tid) ?? []
    for (let i = 0; i < group.length; i++) {
      if (group[i].profile?.hasMacBook === false) {
        misplacedMacNeedy.push({ tutorId: tid, index: i, student: group[i] })
      }
    }
  }

  if (misplacedMacNeedy.length === 0) return

  // For each Mac tutor, track remaining Mac capacity and swappable students
  const macTutors: {
    tutorId: string
    remainingCap: number
    swappable: { index: number; student: DeveloperWithProfile }[]
  }[] = []
  for (const tid of tutorIds) {
    const macSeats = tutorMacCount.get(tid) ?? 0
    if (macSeats === 0) continue
    const group = tutorGroups.get(tid) ?? []
    const currentMacNeedy = group.filter((s) => s.profile?.hasMacBook === false).length
    const swappable: { index: number; student: DeveloperWithProfile }[] = []
    for (let i = 0; i < group.length; i++) {
      if (group[i].profile?.hasMacBook !== false) {
        swappable.push({ index: i, student: group[i] })
      }
    }
    macTutors.push({ tutorId: tid, remainingCap: macSeats - currentMacNeedy, swappable })
  }

  // Distribute misplaced students to the Mac tutor with the most remaining capacity
  for (const misplaced of misplacedMacNeedy) {
    let best: (typeof macTutors)[number] | null = null
    for (const mt of macTutors) {
      if (mt.swappable.length === 0) continue
      if (!best || mt.remainingCap > best.remainingCap) {
        best = mt
      }
    }
    if (!best) break

    const swap = best.swappable.shift()!
    tutorGroups.get(misplaced.tutorId)![misplaced.index] = swap.student
    tutorGroups.get(best.tutorId)![swap.index] = misplaced.student
    best.remainingCap--
  }
}

/**
 * Reorder students so peer pairs are adjacent in the list.
 * Only reorders within the given subset — does not pull students from other tiers.
 */
function orderByPeerAdjacency(
  students: DeveloperWithProfile[],
  peerMap: Map<string, Set<string>>,
): DeveloperWithProfile[] {
  if (students.length <= 1) return students

  const result: DeveloperWithProfile[] = []
  const placed = new Set<string>()
  const studentLookup = new Map<string, DeveloperWithProfile>()
  for (const s of students) {
    if (s.participation.courseParticipationID) {
      studentLookup.set(s.participation.courseParticipationID, s)
    }
  }

  for (const student of students) {
    const sid = student.participation.courseParticipationID
    if (!sid || placed.has(sid)) continue

    result.push(student)
    placed.add(sid)

    // Place this student's peers (from same tier only) immediately after
    const peers = peerMap.get(sid)
    if (peers) {
      for (const peerId of peers) {
        if (placed.has(peerId)) continue
        const peerStudent = studentLookup.get(peerId)
        if (peerStudent) {
          result.push(peerStudent)
          placed.add(peerId)
        }
      }
    }
  }

  return result
}
