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
import { parseSeatName, TUTOR_COLORS } from '../../utils/seatGrid'
import { SeatCell } from './SeatCell'
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
  const seatByName = useMemo(() => {
    const map = new Map<string, Seat>()
    for (const s of seats) map.set(s.seatName, s)
    return map
  }, [seats])

  // Group seats by tutor, sorted within each group by row then position
  const seatsByTutor = useMemo(() => {
    const map = new Map<string, Seat[]>()
    for (const seat of seats) {
      const tutorId = seat.assignedTutor ?? 'unassigned'
      if (!map.has(tutorId)) map.set(tutorId, [])
      map.get(tutorId)!.push(seat)
    }
    for (const [, tutorSeats] of map) {
      tutorSeats.sort((a, b) => {
        const pa = parseSeatName(a.seatName)
        const pb = parseSeatName(b.seatName)
        if (!pa || !pb) return a.seatName.localeCompare(b.seatName)
        if (pa.row !== pb.row) return pa.row - pb.row
        return pa.position - pb.position
      })
    }
    return map
  }, [seats])

  // Max seats in any tutor group (for uniform grid columns)
  const maxSeatsPerTutor = useMemo(() => {
    let max = 0
    for (const [, tutorSeats] of seatsByTutor) {
      max = Math.max(max, tutorSeats.length)
    }
    return max
  }, [seatsByTutor])

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
      if (!seat) return

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

  // Ordered tutor IDs (sorted by color index for stable rendering)
  const orderedTutorIds = useMemo(() => {
    const ids = Array.from(seatsByTutor.keys()).filter((id) => id !== 'unassigned')
    ids.sort((a, b) => (tutorColorMap.get(a) ?? 0) - (tutorColorMap.get(b) ?? 0))
    if (seatsByTutor.has('unassigned')) ids.push('unassigned')
    return ids
  }, [seatsByTutor, tutorColorMap])

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

      <div
        className='grid gap-1'
        style={{
          gridTemplateColumns: `10rem repeat(${maxSeatsPerTutor}, minmax(0, 1fr))`,
        }}
      >
        {orderedTutorIds.map((tutorId) => {
          const tutorSeats = seatsByTutor.get(tutorId) ?? []
          const tutor = tutors.find((t) => t.id === tutorId)
          const colorIdx = tutorColorMap.get(tutorId) ?? -1
          const color = colorIdx >= 0 ? TUTOR_COLORS[colorIdx % TUTOR_COLORS.length] : null
          const label = tutor
            ? `${tutor.firstName} ${tutor.lastName}`
            : tutorId === 'unassigned'
              ? 'Unassigned'
              : tutorId.slice(0, 8)
          const studentCount = tutorSeats.filter((s) => s.assignedStudent).length

          return (
            <Fragment key={tutorId}>
              {/* Tutor row label */}
              <div className='flex items-center text-sm font-medium truncate pr-2'>
                {color && (
                  <div className={`w-3 h-3 rounded-full mr-2 flex-shrink-0 ${color.dot}`} />
                )}
                <span className='truncate'>{label}</span>
                <span className='text-muted-foreground ml-1 flex-shrink-0 text-xs'>
                  ({studentCount})
                </span>
              </div>

              {/* Tutor's seats */}
              {tutorSeats.map((seat) => {
                const isPeerOfSelected =
                  seat.assignedStudent != null && selectedPeers.has(seat.assignedStudent)

                return (
                  <SeatCell
                    key={seat.seatName}
                    seat={seat}
                    tutorColorIndex={
                      seat.assignedTutor ? (tutorColorMap.get(seat.assignedTutor) ?? -1) : -1
                    }
                    studentLabel={getStudentInitials(seat.assignedStudent)}
                    isSelected={selectedSeat === seat.seatName}
                    isPeerOfSelected={isPeerOfSelected}
                    onClick={() => handleCellClick(seat.seatName)}
                  />
                )
              })}

              {/* Pad remaining columns for uniform grid */}
              {Array.from({ length: maxSeatsPerTutor - tutorSeats.length }, (_, i) => (
                <div key={`pad-${i}`} className='aspect-square' />
              ))}
            </Fragment>
          )
        })}
      </div>

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
