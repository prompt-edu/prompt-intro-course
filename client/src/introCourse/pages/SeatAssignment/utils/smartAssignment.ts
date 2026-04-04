import { Seat } from '../../../interfaces/Seat'
import { PeerAssignment } from '../../../interfaces/PeerAssignment'
import { Tutor } from '../../../interfaces/Tutor'
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
 * Auto-assign tutors to contiguous blocks of seats.
 * Clears existing tutor state, sorts seats numerically by room/row/position,
 * distributes tutors evenly in contiguous blocks, and places each tutor seat
 * on the last NON-Mac seat in each block.
 */
export function autoAssignTutors(seats: Seat[], tutors: Tutor[]): Seat[] {
  if (tutors.length === 0) return seats

  const updated = seats.map((s) => ({
    ...s,
    assignedTutor: null as string | null,
    isTutorSeat: false,
  }))

  // Sort seats numerically by parsed name
  const sorted = [...updated].sort((a, b) => {
    const pa = parseSeatName(a.seatName)
    const pb = parseSeatName(b.seatName)
    if (!pa || !pb) return a.seatName.localeCompare(b.seatName)
    if (pa.room !== pb.room) return pa.room - pb.room
    if (pa.row !== pb.row) return pa.row - pb.row
    return pa.position - pb.position
  })

  const seatIndex = new Map<string, number>()
  updated.forEach((s, i) => seatIndex.set(s.seatName, i))

  const totalSeats = sorted.length
  const blockSize = Math.floor(totalSeats / tutors.length)
  const remainder = totalSeats % tutors.length

  let offset = 0
  for (let t = 0; t < tutors.length; t++) {
    const size = blockSize + (t < remainder ? 1 : 0)
    const blockSeats = sorted.slice(offset, offset + size)

    // Assign tutor to all seats in block
    for (const seat of blockSeats) {
      const idx = seatIndex.get(seat.seatName)
      if (idx !== undefined) {
        updated[idx].assignedTutor = tutors[t].id
      }
    }

    // Tutor seat = last non-Mac seat in block (fallback: last seat)
    let tutorSeatName: string | null = null
    for (let i = blockSeats.length - 1; i >= 0; i--) {
      if (!blockSeats[i].hasMac) {
        tutorSeatName = blockSeats[i].seatName
        break
      }
    }
    if (!tutorSeatName && blockSeats.length > 0) {
      tutorSeatName = blockSeats[blockSeats.length - 1].seatName
    }
    if (tutorSeatName) {
      const idx = seatIndex.get(tutorSeatName)
      if (idx !== undefined) {
        updated[idx].isTutorSeat = true
      }
    }

    offset += size
  }

  return updated
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
  tutors?: Tutor[],
): Seat[] {
  // If no tutor assignments exist, auto-assign tutors first
  const hasTutorAssignments = seats.some((s) => s.assignedTutor)
  let workingSeats = seats
  if (!hasTutorAssignments && tutors && tutors.length > 0) {
    workingSeats = autoAssignTutors(seats, tutors)
  }

  // Filter out tutor seats — they are not available for student assignment
  const studentSeats = workingSeats.filter((s) => !s.isTutorSeat)

  // Preserve existing assignments: track which students are already seated
  const alreadyAssignedStudents = new Set<string>()
  const alreadyOccupiedSeats = new Set<string>()
  for (const seat of studentSeats) {
    if (seat.assignedStudent) {
      alreadyAssignedStudents.add(seat.assignedStudent)
      alreadyOccupiedSeats.add(seat.seatName)
    }
  }

  // Group EMPTY seats by tutor (use existing tutor-seat mapping)
  const tutorSeatMap = new Map<string, Seat[]>()
  for (const seat of studentSeats) {
    if (!seat.assignedTutor) continue
    if (alreadyOccupiedSeats.has(seat.seatName)) continue
    if (!tutorSeatMap.has(seat.assignedTutor)) tutorSeatMap.set(seat.assignedTutor, [])
    tutorSeatMap.get(seat.assignedTutor)!.push(seat)
  }

  const tutorIds = [...tutorSeatMap.keys()].sort()

  // Carry over existing assignments, clear only unoccupied seats
  const updatedSeats = workingSeats.map((s) => ({ ...s }))

  if (tutorIds.length === 0) return updatedSeats

  // Count Mac seats per tutor (empty seats only)
  const tutorMacCount = new Map<string, number>()
  for (const id of tutorIds) {
    tutorMacCount.set(id, (tutorSeatMap.get(id) ?? []).filter((s) => s.hasMac).length)
  }

  // Only assign unassigned students
  const MAX_STUDENTS_PER_TUTOR = 10
  const students = shuffle(
    developerWithProfiles.filter(
      (d) => !alreadyAssignedStudents.has(d.participation.courseParticipationID),
    ),
  )
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
      if (!peerMap.has(pa.studentID)) peerMap.set(pa.studentID, new Set())
      peerMap.get(pa.studentID)!.add(pa.peerID)
      if (!peerMap.has(pa.peerID)) peerMap.set(pa.peerID, new Set())
      peerMap.get(pa.peerID)!.add(pa.studentID)
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

  // Clear assignedTutor from empty non-tutor seats (cleanup)
  for (const seat of updatedSeats) {
    if (!seat.isTutorSeat && !seat.assignedStudent && seat.assignedTutor) {
      // Keep assignedTutor — it's needed for the tutor color mapping
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
 * Reorder students so peer groups are adjacent in the list.
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
