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
import { buildSeatLookup, getGridDimensions, parseSeatName } from '../../utils/seatGrid'
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
  const seatLookup = useMemo(() => buildSeatLookup(seats), [seats])
  const seatByName = useMemo(() => {
    const map = new Map<string, Seat>()
    for (const s of seats) map.set(s.seatName, s)
    return map
  }, [seats])
  const { maxRow, maxPosition } = useMemo(() => getGridDimensions(seats), [seats])

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
          gridTemplateColumns: `2.5rem repeat(${maxPosition}, minmax(0, 1fr))`,
        }}
      >
        {/* Header row: position numbers */}
        <div /> {/* empty corner */}
        {Array.from({ length: maxPosition }, (_, i) => i + 1).map((pos) => (
          <div key={pos} className='text-center text-xs text-muted-foreground font-mono'>
            {pos}
          </div>
        ))}

        {/* Seat rows */}
        {Array.from({ length: maxRow }, (_, i) => i + 1).map((row) => (
          <Fragment key={row}>
            {/* Row label */}
            <div
              className='flex items-center justify-center text-xs font-medium text-muted-foreground'
            >
              R{row}
            </div>
            {/* Seat cells */}
            {Array.from({ length: maxPosition }, (_, i) => i + 1).map((pos) => {
              const key = `${row}-${pos}`
              const seat = seatLookup.get(key)

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
                  studentLabel={getStudentInitials(seat.assignedStudent)}
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
