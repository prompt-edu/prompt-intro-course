import { cn } from '@tumaet/prompt-ui-components'
import { GraduationCap, Laptop } from 'lucide-react'
import type { Seat } from '../../../../interfaces/Seat'
import { parseSeatName, type SeatGridViewMode, TUTOR_COLORS } from '../../utils/seatGrid'

interface SeatCellProps {
  seat: Seat
  tutorColorIndex: number
  peerGroupColorIndex: number
  studentLabel: string | null
  peerGroupLabel: string | null
  isSelected: boolean
  isPeerOfSelected: boolean
  viewMode: SeatGridViewMode
  onClick: () => void
}

export const SeatCell = ({
  seat,
  tutorColorIndex,
  peerGroupColorIndex,
  studentLabel,
  peerGroupLabel,
  isSelected,
  isPeerOfSelected,
  viewMode,
  onClick,
}: SeatCellProps) => {
  const parsed = parseSeatName(seat.seatName)
  const hasStudent = !!seat.assignedStudent
  const isTutor = seat.isTutorSeat

  // Pick color based on view mode
  const colorIndex = viewMode === 'peerGroup' && hasStudent ? peerGroupColorIndex : tutorColorIndex
  const color = colorIndex >= 0 ? TUTOR_COLORS[colorIndex % TUTOR_COLORS.length] : null

  // In seat view, only use color for tutor seats and assigned seats with tutor
  const useColor = viewMode === 'seat' ? isTutor : true

  const classes = cn(
    'relative w-full aspect-square rounded-md text-xs flex flex-col items-center justify-center gap-0.5 transition-all',
    isTutor
      ? `border-2 ${useColor && color ? `${color.bg} ${color.border}` : 'bg-muted/20 border-dashed border-muted-foreground/30'}`
      : `border ${hasStudent && useColor && color ? `${color.bg} ${color.border}` : 'bg-muted/20 border-dashed border-muted-foreground/30'}`,
    'cursor-pointer',
    isSelected && 'ring-2 ring-primary ring-offset-1',
    isPeerOfSelected && 'ring-2 ring-amber-400 ring-offset-1',
    hasStudent && !isTutor && 'hover:opacity-80',
  )

  return (
    <button
      onClick={onClick}
      className={classes}
      title={`${seat.seatName}${isTutor ? ' (Tutor seat)' : ''}${studentLabel ? ` - ${studentLabel}` : ''}${peerGroupLabel ? ` ${peerGroupLabel}` : ''}${seat.hasMac ? ' (Mac)' : ''}`}
    >
      {/* Peer group badge — top right */}
      {peerGroupLabel && !isTutor && (
        <span
          style={{
            position: 'absolute',
            top: 1,
            right: 2,
            fontSize: 9,
            fontWeight: 700,
            lineHeight: 1,
            zIndex: 10,
          }}
          className='text-muted-foreground'
        >
          {peerGroupLabel}
        </span>
      )}

      {/* Mac indicator — top left */}
      {seat.hasMac && (
        <Laptop className='absolute top-0.5 left-0.5 h-2.5 w-2.5 text-muted-foreground' />
      )}

      {/* Position number or tutor icon */}
      {isTutor ? (
        <GraduationCap className='h-3 w-3 text-muted-foreground' />
      ) : viewMode === 'seat' ? (
        <span className='text-[9px] font-medium text-muted-foreground'>
          {seat.seatName.replace(/^1-/, '')}
        </span>
      ) : (
        <span className='text-[10px] text-muted-foreground'>{parsed?.position}</span>
      )}

      {/* Student/tutor initials or empty */}
      {viewMode === 'seat' && !isTutor ? (
        hasStudent ? (
          <span className='text-[8px] text-muted-foreground'>{studentLabel}</span>
        ) : (
          <span className='text-muted-foreground/50'>-</span>
        )
      ) : studentLabel ? (
        <span className={cn('font-semibold text-xs', useColor && color?.text)}>{studentLabel}</span>
      ) : (
        <span className='text-muted-foreground/50'>-</span>
      )}
    </button>
  )
}
