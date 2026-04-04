import { Fragment, useState, useMemo, useCallback } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useParams } from 'react-router-dom'
import { Users, GraduationCap, Armchair } from 'lucide-react'
import {
  Alert,
  AlertDescription,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
  ToggleGroup,
  ToggleGroupItem,
} from '@tumaet/prompt-ui-components'
import { Seat } from '../../../../interfaces/Seat'
import { Tutor } from '../../../../interfaces/Tutor'
import { PeerAssignment } from '../../../../interfaces/PeerAssignment'
import { updateSeatPlan } from '../../../../network/mutations/updateSeatPlan'
import { updatePeerAssignments } from '../../../../network/mutations/updatePeerAssignments'
import {
  buildPhysicalSeatMap,
  getMaxPhysicalPosition,
  buildPeerGroups,
  TUTOR_COLORS,
  type SeatGridViewMode,
} from '../../utils/seatGrid'
import { RECHNERHALLE_LAYOUT } from '../../utils/rechnerHalle'
import { SeatCell } from './SeatCell'
import { SeatGridLegend } from './SeatGridLegend'
import { CoursePhaseParticipationWithStudent } from '@tumaet/prompt-shared-state'

interface SeatGridProps {
  seats: Seat[]
  tutors: Tutor[]
  participations: CoursePhaseParticipationWithStudent[]
  peerAssignments?: PeerAssignment[]
}

export const SeatGrid = ({ seats, tutors, participations, peerAssignments }: SeatGridProps) => {
  const { phaseId } = useParams<{ phaseId: string }>()
  const queryClient = useQueryClient()
  const [selectedSeat, setSelectedSeat] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [viewMode, setViewMode] = useState<SeatGridViewMode>('tutor')

  const swapMutation = useMutation({
    mutationFn: (updatedSeats: Seat[]) => updateSeatPlan(phaseId ?? '', updatedSeats),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['seatPlan', phaseId] })
      setSelectedSeat(null)
      setError(null)
    },
    onError: () => setError('Failed to swap seats.'),
  })

  const peerMutation = useMutation({
    mutationFn: (assignments: PeerAssignment[]) => updatePeerAssignments(phaseId ?? '', assignments),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['peerAssignments', phaseId] })
    },
  })

  // Build lookup maps
  const layout = RECHNERHALLE_LAYOUT
  const maxPhysicalPos = useMemo(() => getMaxPhysicalPosition(layout), [layout])
  const physicalSeatMap = useMemo(() => buildPhysicalSeatMap(seats, layout), [seats, layout])

  const seatByName = useMemo(() => {
    const map = new Map<string, Seat>()
    for (const s of seats) map.set(s.seatName, s)
    return map
  }, [seats])

  const participationMap = useMemo(() => {
    const map = new Map<string, CoursePhaseParticipationWithStudent>()
    for (const p of participations) {
      if (p.courseParticipationID) map.set(p.courseParticipationID, p)
    }
    return map
  }, [participations])

  // Tutor color assignment: stable mapping of tutorID -> color index
  const tutorColorMap = useMemo(() => {
    const map = new Map<string, number>()
    const uniqueTutors = [...new Set(seats.filter((s) => s.assignedTutor).map((s) => s.assignedTutor!))]
    uniqueTutors.sort()
    uniqueTutors.forEach((id, idx) => map.set(id, idx))
    return map
  }, [seats])

  // Tutor name lookup for tutor seats
  const tutorNameMap = useMemo(() => {
    const map = new Map<string, string>()
    for (const t of tutors) {
      const initials = `${t.firstName?.charAt(0) ?? ''}${t.lastName?.charAt(0) ?? ''}`.toUpperCase()
      map.set(t.id, initials)
    }
    return map
  }, [tutors])

  // Peer lookup: studentID -> Set of peer IDs
  const peerMap = useMemo(() => {
    const map = new Map<string, Set<string>>()
    if (!peerAssignments) return map
    for (const a of peerAssignments) {
      if (!map.has(a.studentID)) map.set(a.studentID, new Set())
      map.get(a.studentID)!.add(a.peerID)
    }
    return map
  }, [peerAssignments])

  // Peer groups: studentID -> group number
  const peerGroupMap = useMemo(() => {
    if (!peerAssignments || peerAssignments.length === 0) return new Map<string, number>()
    return buildPeerGroups(peerAssignments)
  }, [peerAssignments])

  const peerGroupCount = useMemo(() => {
    if (peerGroupMap.size === 0) return 0
    return Math.max(...peerGroupMap.values())
  }, [peerGroupMap])

  const getStudentInitials = useCallback(
    (studentId: string | null): string | null => {
      if (!studentId) return null
      const p = participationMap.get(studentId)
      if (!p?.student) return studentId.slice(0, 2).toUpperCase()
      const first = p.student.firstName?.charAt(0) ?? ''
      const last = p.student.lastName?.charAt(0) ?? ''
      return (first + last).toUpperCase() || studentId.slice(0, 2).toUpperCase()
    },
    [participationMap],
  )

  const { mutate: swapSeats } = swapMutation

  // Change tutor assignment for a single seat
  const changeTutor = useCallback(
    (seatName: string, newTutorId: string) => {
      const seat = seatByName.get(seatName)
      if (!seat) return
      swapSeats([{ ...seat, assignedTutor: newTutorId }])
    },
    [seatByName, swapSeats],
  )

  // Swap all seats between two tutors (swap entire blocks)
  const swapTutors = useCallback(
    (tutorA: string, tutorB: string) => {
      const updates: Seat[] = []
      for (const s of seats) {
        if (s.assignedTutor === tutorA) {
          updates.push({ ...s, assignedTutor: tutorB })
        } else if (s.assignedTutor === tutorB) {
          updates.push({ ...s, assignedTutor: tutorA })
        }
      }
      if (updates.length > 0) swapSeats(updates)
    },
    [seats, swapSeats],
  )

  // Build group members lookup
  const groupMembers = useMemo(() => {
    const map = new Map<number, string[]>()
    for (const [studentId, groupNum] of peerGroupMap) {
      if (!map.has(groupNum)) map.set(groupNum, [])
      map.get(groupNum)!.push(studentId)
    }
    return map
  }, [peerGroupMap])

  // Move a student to a different peer group
  const moveStudentToGroup = useCallback(
    (studentId: string, targetGroup: number) => {
      if (!peerAssignments) return
      // Remove student from current group assignments
      const filtered = peerAssignments.filter(
        (a) => a.studentID !== studentId && a.peerID !== studentId,
      )
      // Add student to target group
      const targetMembers = groupMembers.get(targetGroup) ?? []
      for (const member of targetMembers) {
        if (member === studentId) continue
        filtered.push({ studentID: studentId, peerID: member })
        filtered.push({ studentID: member, peerID: studentId })
      }
      peerMutation.mutate(filtered)
    },
    [peerAssignments, groupMembers, peerMutation],
  )

  const executeSwap = useCallback(
    (seatA: Seat, seatB: Seat) => {
      const updatedA = { ...seatA, assignedStudent: seatB.assignedStudent }
      const updatedB = { ...seatB, assignedStudent: seatA.assignedStudent }
      // When one seat is empty and the other has a student, tutor follows the student
      if (!seatA.assignedStudent && seatB.assignedStudent) {
        updatedB.assignedTutor = seatA.assignedTutor
      } else if (!seatB.assignedStudent && seatA.assignedStudent) {
        updatedA.assignedTutor = seatB.assignedTutor
      }
      swapSeats([updatedA, updatedB])
    },
    [swapSeats],
  )

  const handleCellClick = useCallback(
    (seatName: string) => {
      const seat = seatByName.get(seatName)
      if (!seat) return

      if (!selectedSeat) {
        // Allow selecting any seat (including tutor seats and empty seats)
        setSelectedSeat(seatName)
        setError(null)
        return
      }

      if (selectedSeat === seatName) {
        setSelectedSeat(null)
        return
      }

      // Second selection — perform swap
      const seatA = seatByName.get(selectedSeat)
      if (!seatA) return
      const seatB = seat

      // Tutor-to-tutor: swap entire blocks
      if (seatA.isTutorSeat && seatB.isTutorSeat && seatA.assignedTutor && seatB.assignedTutor && seatA.assignedTutor !== seatB.assignedTutor) {
        swapTutors(seatA.assignedTutor, seatB.assignedTutor)
        setSelectedSeat(null)
        return
      }

      // Tutor seat to student seat: move tutor seat (carry assignedTutor, clear duplicates)
      if (seatA.isTutorSeat && !seatB.isTutorSeat && seatA.assignedTutor) {
        const tutorId = seatA.assignedTutor
        const updates: Seat[] = [
          { ...seatA, isTutorSeat: false },
          { ...seatB, isTutorSeat: true, assignedTutor: tutorId },
        ]
        swapSeats(updates)
        setSelectedSeat(null)
        return
      }
      if (!seatA.isTutorSeat && seatB.isTutorSeat && seatB.assignedTutor) {
        const tutorId = seatB.assignedTutor
        const updates: Seat[] = [
          { ...seatB, isTutorSeat: false },
          { ...seatA, isTutorSeat: true, assignedTutor: tutorId },
        ]
        swapSeats(updates)
        setSelectedSeat(null)
        return
      }

      // Student-to-student (or empty) — always works
      executeSwap(seatA, seatB)
      setSelectedSeat(null)
    },
    [selectedSeat, seatByName, executeSwap, swapTutors, swapSeats],
  )

  // Get peers of selected student
  const selectedPeers = useMemo(() => {
    if (!selectedSeat) return new Set<string>()
    const seat = seatByName.get(selectedSeat)
    if (!seat?.assignedStudent) return new Set<string>()
    return peerMap.get(seat.assignedStudent) ?? new Set<string>()
  }, [selectedSeat, seatByName, peerMap])

  // Build gap set for quick lookup
  const gapSet = useMemo(() => {
    const set = new Set<string>()
    for (const r of layout) {
      for (const g of r.gaps) set.add(`${r.row}-${g}`)
    }
    return set
  }, [layout])

  const hasPeerGroups = peerGroupCount > 0

  return (
    <div>
      {/* View mode toggle */}
      <ToggleGroup
        type='single'
        value={viewMode}
        onValueChange={(value) => { if (value) setViewMode(value as SeatGridViewMode) }}
        className='mb-3'
      >
        <ToggleGroupItem value='tutor' size='sm'>
          <GraduationCap className='mr-1.5 h-3.5 w-3.5' />
          Tutor
        </ToggleGroupItem>
        {hasPeerGroups && (
          <ToggleGroupItem value='peerGroup' size='sm'>
            <Users className='mr-1.5 h-3.5 w-3.5' />
            Peer Group
          </ToggleGroupItem>
        )}
        <ToggleGroupItem value='seat' size='sm'>
          <Armchair className='mr-1.5 h-3.5 w-3.5' />
          Seat
        </ToggleGroupItem>
      </ToggleGroup>

      {selectedSeat && (() => {
        const selSeat = seatByName.get(selectedSeat)
        const selStudent = selSeat?.assignedStudent
        const selPeerGroup = selStudent ? peerGroupMap.get(selStudent) : undefined
        return (
          <Alert className='mb-3'>
            <AlertDescription className='flex flex-wrap items-center gap-3'>
              <span>Click another seat to swap, or click the same seat to deselect.</span>
              {selSeat && !selSeat.isTutorSeat && (
                <Select
                  value={selSeat.assignedTutor ?? ''}
                  onValueChange={(val) => changeTutor(selectedSeat, val)}
                >
                  <SelectTrigger className='w-40 h-8'>
                    <SelectValue placeholder='Tutor' />
                  </SelectTrigger>
                  <SelectContent>
                    {tutors.map((t) => (
                      <SelectItem key={t.id} value={t.id}>
                        {t.firstName} {t.lastName}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              )}
              {selStudent && hasPeerGroups && (
                <Select
                  value={selPeerGroup != null ? String(selPeerGroup) : ''}
                  onValueChange={(val) => moveStudentToGroup(selStudent, Number(val))}
                >
                  <SelectTrigger className='w-28 h-8'>
                    <SelectValue placeholder='Peer Group' />
                  </SelectTrigger>
                  <SelectContent>
                    {Array.from({ length: peerGroupCount }, (_, i) => i + 1).map((g) => (
                      <SelectItem key={g} value={String(g)}>
                        P{g}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              )}
            </AlertDescription>
          </Alert>
        )
      })()}

      {error && (
        <Alert variant='destructive' className='mb-3'>
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      )}

      {/* Transposed grid: columns = physical rows (R1-R9), rows = physical positions (1-12) */}
      <div
        className='grid gap-1'
        style={{
          gridTemplateColumns: `2.5rem repeat(${layout.length}, minmax(0, 1fr))`,
        }}
      >
        {/* Header row: row numbers as column headers */}
        <div /> {/* empty corner */}
        {layout.map((r) => (
          <div key={r.row} className='text-center text-xs text-muted-foreground font-mono'>
            R{r.row}
          </div>
        ))}

        {/* Position rows */}
        {Array.from({ length: maxPhysicalPos }, (_, i) => i + 1).map((physPos) => (
          <Fragment key={physPos}>
            {/* Position label */}
            <div className='flex items-center justify-center text-xs font-medium text-muted-foreground'>
              {physPos}
            </div>
            {/* Cells: one per physical row */}
            {layout.map((rowLayout) => {
              const key = `${rowLayout.row}-${physPos}`

              // Outside this row's range
              if (physPos < rowLayout.physicalStart || physPos > rowLayout.physicalEnd) {
                return <div key={key} className='aspect-square' />
              }

              // Gap position (door, etc.)
              if (gapSet.has(key)) {
                return <div key={key} className='aspect-square' />
              }

              // Look up the seat at this physical position
              const seat = physicalSeatMap.get(key)
              if (!seat) {
                return <div key={key} className='aspect-square' />
              }

              const isPeerOfSelected =
                seat.assignedStudent != null && selectedPeers.has(seat.assignedStudent)

              const peerGroup = seat.assignedStudent ? peerGroupMap.get(seat.assignedStudent) : undefined

              return (
                <SeatCell
                  key={key}
                  seat={seat}
                  tutorColorIndex={seat.assignedTutor ? (tutorColorMap.get(seat.assignedTutor) ?? -1) : -1}
                  peerGroupColorIndex={peerGroup != null ? (peerGroup - 1) : -1}
                  studentLabel={
                    seat.isTutorSeat
                      ? (seat.assignedTutor ? (tutorNameMap.get(seat.assignedTutor) ?? null) : null)
                      : getStudentInitials(seat.assignedStudent)
                  }
                  peerGroupLabel={peerGroup != null ? `P${peerGroup}` : null}
                  isSelected={selectedSeat === seat.seatName}
                  isPeerOfSelected={isPeerOfSelected}
                  viewMode={viewMode}
                  onClick={() => handleCellClick(seat.seatName)}
                />
              )
            })}
          </Fragment>
        ))}
      </div>

      <SeatGridLegend
        tutors={tutors}
        seats={seats}
        tutorColorMap={tutorColorMap}
        viewMode={viewMode}
        peerGroupCount={peerGroupCount}
        peerGroupMap={peerGroupMap}
      />

    </div>
  )
}
