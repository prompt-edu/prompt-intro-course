import { GraduationCap } from 'lucide-react'
import { TUTOR_COLORS } from '../../utils/seatGrid'
import { Tutor } from '../../../../interfaces/Tutor'
import { Seat } from '../../../../interfaces/Seat'

interface SeatGridLegendProps {
  tutors: Tutor[]
  seats: Seat[]
  tutorColorMap: Map<string, number>
}

export const SeatGridLegend = ({ tutors, seats, tutorColorMap }: SeatGridLegendProps) => {
  // Count students per tutor
  const studentCounts = new Map<string, number>()
  for (const seat of seats) {
    if (seat.assignedTutor && seat.assignedStudent) {
      studentCounts.set(seat.assignedTutor, (studentCounts.get(seat.assignedTutor) ?? 0) + 1)
    }
  }

  const hasTutorSeats = seats.some((s) => s.isTutorSeat)

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
      {tutors.map((tutor) => {
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
      })}
    </div>
  )
}
