import { Laptop } from 'lucide-react'
import { Seat } from '../../../../interfaces/Seat'
import { parseSeatName, TUTOR_COLORS } from '../../utils/seatGrid'

interface SeatCellProps {
  seat: Seat
  tutorColorIndex: number
  studentLabel: string | null
  isSelected: boolean
  isPeerOfSelected: boolean
  onClick: () => void
}

export const SeatCell = ({
  seat,
  tutorColorIndex,
  studentLabel,
  isSelected,
  isPeerOfSelected,
  onClick,
}: SeatCellProps) => {
  const parsed = parseSeatName(seat.seatName)
  const color = tutorColorIndex >= 0 ? TUTOR_COLORS[tutorColorIndex % TUTOR_COLORS.length] : null
  const hasStudent = !!seat.assignedStudent

  const classes = [
    'relative w-full aspect-square rounded-md border text-xs flex flex-col items-center justify-center gap-0.5 transition-all cursor-pointer',
    hasStudent && color ? `${color.bg} ${color.border}` : 'bg-muted/20 border-dashed border-muted-foreground/30',
    isSelected ? 'ring-2 ring-primary ring-offset-1' : '',
    isPeerOfSelected ? 'ring-2 ring-amber-400 ring-offset-1' : '',
    hasStudent ? 'hover:opacity-80' : '',
  ]
    .filter(Boolean)
    .join(' ')

  return (
    <button
      onClick={onClick}
      className={classes}
      title={`${seat.seatName}${studentLabel ? ` - ${studentLabel}` : ''}${seat.hasMac ? ' (Mac)' : ''}`}
    >
      {/* Position number */}
      <span className='text-[10px] text-muted-foreground'>
        {parsed?.position}
      </span>

      {/* Student initials or empty */}
      {studentLabel ? (
        <span className={`font-semibold text-xs ${color?.text ?? ''}`}>
          {studentLabel}
        </span>
      ) : (
        <span className='text-muted-foreground/50'>-</span>
      )}

      {/* Mac indicator */}
      {seat.hasMac && (
        <Laptop className='absolute top-0.5 right-0.5 h-2.5 w-2.5 text-muted-foreground' />
      )}
    </button>
  )
}
