import { GraduationCap, Laptop } from 'lucide-react'
import type { Seat } from '../../../../interfaces/Seat'
import type { Tutor } from '../../../../interfaces/Tutor'
import { type SeatGridViewMode, TUTOR_COLORS } from '../../utils/seatGrid'

interface SeatGridLegendProps {
  tutors: Tutor[]
  seats: Seat[]
  tutorColorMap: Map<string, number>
  viewMode: SeatGridViewMode
  peerGroupCount: number
  peerGroupMap: Map<string, number>
}

export const SeatGridLegend = ({
  tutors,
  seats,
  tutorColorMap,
  viewMode,
  peerGroupCount,
  peerGroupMap,
}: SeatGridLegendProps) => {
  const hasTutorSeats = seats.some((s) => s.isTutorSeat)
  const hasMacSeats = seats.some((s) => s.hasMac)

  // Count students per tutor
  const studentCounts = new Map<string, number>()
  for (const seat of seats) {
    if (seat.assignedTutor && seat.assignedStudent) {
      studentCounts.set(seat.assignedTutor, (studentCounts.get(seat.assignedTutor) ?? 0) + 1)
    }
  }

  // Count students per peer group
  const peerGroupSizes = new Map<number, number>()
  for (const [, group] of peerGroupMap) {
    peerGroupSizes.set(group, (peerGroupSizes.get(group) ?? 0) + 1)
  }

  return (
    <div className='flex flex-wrap gap-3 mt-4'>
      {hasTutorSeats && (
        <div className='flex items-center gap-1.5 text-sm'>
          <div className='w-4 h-4 rounded-md border-2 border-muted-foreground/50 flex items-center justify-center'>
            <GraduationCap className='h-2.5 w-2.5 text-muted-foreground' />
          </div>
          <span>Tutor Seat</span>
        </div>
      )}

      {hasMacSeats && (
        <div className='flex items-center gap-1.5 text-sm'>
          <div className='w-4 h-4 rounded-md border border-muted-foreground/30 flex items-center justify-center'>
            <Laptop className='h-2.5 w-2.5 text-muted-foreground' />
          </div>
          <span>Mac Seat</span>
        </div>
      )}

      {viewMode === 'peerGroup' && peerGroupCount > 0
        ? // Peer group legend
          Array.from({ length: peerGroupCount }, (_, i) => i + 1).map((group) => {
            const color = TUTOR_COLORS[(group - 1) % TUTOR_COLORS.length]
            const size = peerGroupSizes.get(group) ?? 0
            return (
              <div key={group} className='flex items-center gap-1.5 text-sm'>
                <div className={`w-3 h-3 rounded-full ${color.dot}`} />
                <span>P{group}</span>
                <span className='text-muted-foreground'>({size})</span>
              </div>
            )
          })
        : viewMode !== 'seat'
          ? // Tutor legend
            tutors.map((tutor) => {
              const colorIdx = tutorColorMap.get(tutor.id) ?? 0
              const color = TUTOR_COLORS[colorIdx % TUTOR_COLORS.length]
              const count = studentCounts.get(tutor.id) ?? 0

              return (
                <div key={tutor.id} className='flex items-center gap-1.5 text-sm'>
                  <div className={`w-3 h-3 rounded-full ${color.dot}`} />
                  <span>
                    {tutor.firstName} {tutor.lastName}
                  </span>
                  <span className='text-muted-foreground'>({count})</span>
                </div>
              )
            })
          : null}
    </div>
  )
}
