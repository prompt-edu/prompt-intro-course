import { Fragment, useState, useMemo, useCallback } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useParams } from 'react-router-dom'
import {
  Alert,
  AlertDescription,
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@tumaet/prompt-ui-components'
import { Seat } from '../../../../interfaces/Seat'
import { Tutor } from '../../../../interfaces/Tutor'
import { PeerAssignment } from '../../../../interfaces/PeerAssignment'
import { updateSeatPlan } from '../../../../network/mutations/updateSeatPlan'
import { buildPhysicalSeatMap, getMaxPhysicalPosition, TUTOR_COLORS } from '../../utils/seatGrid'
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
  const [crossTutorSwap, setCrossTutorSwap] = useState<{
    seatA: Seat
    seatB: Seat
  } | null>(null)

  const swapMutation = useMutation({
    mutationFn: (updatedSeats: Seat[]) => updateSeatPlan(phaseId ?? '', updatedSeats),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['seatPlan', phaseId] })
      setSelectedSeat(null)
      setError(null)
    },
    onError: () => setError('Failed to swap seats.'),
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

  const executeSwap = useCallback(
    (seatA: Seat, seatB: Seat) => {
      const updatedA = { ...seatA, assignedStudent: seatB.assignedStudent }
      const updatedB = { ...seatB, assignedStudent: seatA.assignedStudent }
      swapSeats([updatedA, updatedB])
    },
    [swapSeats],
  )

  const handleCellClick = useCallback(
    (seatName: string) => {
      const seat = seatByName.get(seatName)
      if (!seat || seat.isTutorSeat) return

      if (!selectedSeat) {
        if (seat.assignedStudent) {
          setSelectedSeat(seatName)
          setError(null)
        }
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

      if (seatA.assignedTutor && seatB.assignedTutor && seatA.assignedTutor !== seatB.assignedTutor) {
        setCrossTutorSwap({ seatA, seatB })
      } else {
        executeSwap(seatA, seatB)
      }
      setSelectedSeat(null)
    },
    [selectedSeat, seatByName, executeSwap],
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

  return (
    <div>
      {selectedSeat && (
        <Alert className='mb-3'>
          <AlertDescription>
            Click another seat to swap students, or click the same seat to deselect.
          </AlertDescription>
        </Alert>
      )}

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

              return (
                <SeatCell
                  key={key}
                  seat={seat}
                  tutorColorIndex={seat.assignedTutor ? (tutorColorMap.get(seat.assignedTutor) ?? -1) : -1}
                  studentLabel={
                    seat.isTutorSeat
                      ? (seat.assignedTutor ? (tutorNameMap.get(seat.assignedTutor) ?? null) : null)
                      : getStudentInitials(seat.assignedStudent)
                  }
                  isSelected={selectedSeat === seat.seatName}
                  isPeerOfSelected={isPeerOfSelected}
                  onClick={() => handleCellClick(seat.seatName)}
                />
              )
            })}
          </Fragment>
        ))}
      </div>

      <SeatGridLegend tutors={tutors} seats={seats} tutorColorMap={tutorColorMap} />

      {/* Cross-tutor swap dialog */}
      <AlertDialog open={!!crossTutorSwap} onOpenChange={() => setCrossTutorSwap(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Cross-tutor seat swap</AlertDialogTitle>
            <AlertDialogDescription>
              These seats belong to different tutors. The students will physically move but their
              GitLab repositories will remain under their original tutor. Continue?
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => {
                if (crossTutorSwap) {
                  executeSwap(crossTutorSwap.seatA, crossTutorSwap.seatB)
                  setCrossTutorSwap(null)
                }
              }}
            >
              Swap Anyway
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}
