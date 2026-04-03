export interface RowLayout {
  row: number
  physicalStart: number // 1-indexed physical position of first seat
  physicalEnd: number // inclusive
  gaps: number[] // physical positions within range that have NO seat
}

export const RECHNERHALLE_LAYOUT: RowLayout[] = [
  { row: 1, physicalStart: 1, physicalEnd: 12, gaps: [] }, // 12 seats
  { row: 2, physicalStart: 3, physicalEnd: 12, gaps: [6] }, // 9 seats
  { row: 3, physicalStart: 3, physicalEnd: 12, gaps: [] }, // 10 seats
  { row: 4, physicalStart: 3, physicalEnd: 12, gaps: [] }, // 10 seats
  { row: 5, physicalStart: 3, physicalEnd: 12, gaps: [] }, // 10 seats
  { row: 6, physicalStart: 3, physicalEnd: 12, gaps: [6, 12] }, // 8 seats
  { row: 7, physicalStart: 3, physicalEnd: 12, gaps: [] }, // 10 seats
  { row: 8, physicalStart: 3, physicalEnd: 12, gaps: [] }, // 10 seats
  { row: 9, physicalStart: 3, physicalEnd: 12, gaps: [] }, // 10 seats
]

/** Get the physical positions that actually have seats for a row layout. */
export function getPhysicalPositions(layout: RowLayout): number[] {
  const positions: number[] = []
  for (let p = layout.physicalStart; p <= layout.physicalEnd; p++) {
    if (!layout.gaps.includes(p)) positions.push(p)
  }
  return positions
}

/** Map 0-based local seat index to physical position for a row. */
export function localToPhysical(layout: RowLayout, localIndex: number): number {
  const positions = getPhysicalPositions(layout)
  return positions[localIndex]
}

// Backward-compat flat list for SeatUploader
export const RECHNERHALLE_SEATS = RECHNERHALLE_LAYOUT.flatMap(
  ({ row, physicalStart, physicalEnd, gaps }) => {
    const seatCount = physicalEnd - physicalStart + 1 - gaps.length
    return Array.from({ length: seatCount }, (_, i) => `1-${row}-${i + 1}`)
  },
)
