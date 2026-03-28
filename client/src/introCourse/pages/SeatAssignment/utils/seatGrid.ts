import { Seat } from '../../../interfaces/Seat'

export interface ParsedSeat {
  room: number
  row: number
  position: number
  original: string
}

export function parseSeatName(seatName: string): ParsedSeat | null {
  const parts = seatName.split('-')
  if (parts.length !== 3) return null
  const room = parseInt(parts[0], 10)
  const row = parseInt(parts[1], 10)
  const position = parseInt(parts[2], 10)
  if (isNaN(room) || isNaN(row) || isNaN(position)) return null
  return { room, row, position, original: seatName }
}

export function getGridDimensions(seats: Seat[]): { maxRow: number; maxPosition: number } {
  let maxRow = 0
  let maxPosition = 0
  for (const seat of seats) {
    const parsed = parseSeatName(seat.seatName)
    if (!parsed) continue
    if (parsed.row > maxRow) maxRow = parsed.row
    if (parsed.position > maxPosition) maxPosition = parsed.position
  }
  return { maxRow, maxPosition }
}

// Build a lookup map: "row-position" -> Seat
export function buildSeatLookup(seats: Seat[]): Map<string, Seat> {
  const map = new Map<string, Seat>()
  for (const seat of seats) {
    const parsed = parseSeatName(seat.seatName)
    if (!parsed) continue
    map.set(`${parsed.row}-${parsed.position}`, seat)
  }
  return map
}

// 9-color palette for tutor groups
export const TUTOR_COLORS = [
  { bg: 'bg-blue-100 dark:bg-blue-900/30', border: 'border-blue-300', text: 'text-blue-800 dark:text-blue-200', dot: 'bg-blue-500' },
  { bg: 'bg-green-100 dark:bg-green-900/30', border: 'border-green-300', text: 'text-green-800 dark:text-green-200', dot: 'bg-green-500' },
  { bg: 'bg-amber-100 dark:bg-amber-900/30', border: 'border-amber-300', text: 'text-amber-800 dark:text-amber-200', dot: 'bg-amber-500' },
  { bg: 'bg-purple-100 dark:bg-purple-900/30', border: 'border-purple-300', text: 'text-purple-800 dark:text-purple-200', dot: 'bg-purple-500' },
  { bg: 'bg-pink-100 dark:bg-pink-900/30', border: 'border-pink-300', text: 'text-pink-800 dark:text-pink-200', dot: 'bg-pink-500' },
  { bg: 'bg-cyan-100 dark:bg-cyan-900/30', border: 'border-cyan-300', text: 'text-cyan-800 dark:text-cyan-200', dot: 'bg-cyan-500' },
  { bg: 'bg-orange-100 dark:bg-orange-900/30', border: 'border-orange-300', text: 'text-orange-800 dark:text-orange-200', dot: 'bg-orange-500' },
  { bg: 'bg-teal-100 dark:bg-teal-900/30', border: 'border-teal-300', text: 'text-teal-800 dark:text-teal-200', dot: 'bg-teal-500' },
  { bg: 'bg-rose-100 dark:bg-rose-900/30', border: 'border-rose-300', text: 'text-rose-800 dark:text-rose-200', dot: 'bg-rose-500' },
]
