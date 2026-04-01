/**
 * Standalone screenshot harness for SeatGrid.
 * Renders the real React components with mock data for Playwright screenshots.
 * Usage: yarn dev → http://localhost:3005
 */
import { useState } from 'react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { SeatGrid } from './introCourse/pages/SeatAssignment/components/SeatGrid/SeatGrid'
import { Seat } from './introCourse/interfaces/Seat'
import { Tutor } from './introCourse/interfaces/Tutor'
import { PeerAssignment } from './introCourse/interfaces/PeerAssignment'
import { RECHNERHALLE_LAYOUT, getPhysicalPositions } from './introCourse/pages/SeatAssignment/utils/rechnerHalle'
import type { CoursePhaseParticipationWithStudent } from '@tumaet/prompt-shared-state'

const queryClient = new QueryClient({
  defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
})

// ── Tutors ─────────────────────────────────────────────────────────────
const tutorData = [
  { firstName: 'Alice', lastName: 'Mueller' },
  { firstName: 'Bob', lastName: 'Schmidt' },
  { firstName: 'Clara', lastName: 'Weber' },
  { firstName: 'David', lastName: 'Fischer' },
  { firstName: 'Eva', lastName: 'Braun' },
  { firstName: 'Felix', lastName: 'Wagner' },
]
const tutors: Tutor[] = tutorData.map((t, i) => ({
  id: `tutor-${i}`,
  firstName: t.firstName,
  lastName: t.lastName,
  email: `${t.firstName.toLowerCase()}@example.com`,
}))

// ── Students ───────────────────────────────────────────────────────────
const studentNames = [
  ['Max','Mueller'],['Anna','Schneider'],['Lukas','Wagner'],['Sophie','Fischer'],
  ['Leon','Weber'],['Emma','Braun'],['Paul','Hoffmann'],['Marie','Schulz'],
  ['Jonas','Koch'],['Tim','Klein'],['Felix','Groß'],['Hannah','Bauer'],
  ['Lena','Berger'],['Tom','Richter'],['Laura','Krause'],['Nico','Wolf'],
  ['Mia','Schmitt'],['Finn','Neumann'],['Sara','Schwarz'],['Eric','Zimmermann'],
  ['Robin','Braun'],['Jan','Beck'],['Fiona','Keller'],['Henry','Hartmann'],
  ['Ben','Lang'],['Lisa','Schäfer'],['Lea','Werner'],['Lars','Seidel'],
  ['Timo','Meyer'],['Julia','Lange'],['Nina','Schmid'],['Alex','Meier'],
  ['Diana','Krug'],['Nora','Hahn'],['Jakob','Kaiser'],['Clara','Weiß'],
  ['Max','König'],['Anne','Frank'],['Hugo','Peters'],['Pia','Brandt'],
  ['Cleo','Ludwig'],['Oscar','Sommer'],['Ella','Maier'],['Karl','Wirth'],
  ['Kurt','Jung'],['Eva','Horn'],['Zoe','Stein'],['Sam','Vogel'],
  ['Noah','Fiedler'],['Ralf','Krüger'],['Lara','Koenig'],['Theo','Günther'],
  ['Peter','Fuchs'],['Ida','Becker'],['Tina','Wendt'],['Vera','Roth'],
]
const macNeedy = new Set([5, 8, 13, 18, 23, 29, 33, 37, 41, 45, 50])

const participations: CoursePhaseParticipationWithStudent[] = studentNames.map((n, i) => ({
  coursePhaseID: 'phase-1',
  courseParticipationID: `student-${i}`,
  passStatus: 'not_assessed' as any,
  restrictedData: {},
  studentReadableData: {},
  prevData: {},
  student: {
    id: `student-${i}`,
    firstName: n[0],
    lastName: n[1],
    email: `${n[0].toLowerCase()}.${n[1].toLowerCase()}@example.com`,
    hasUniversityAccount: true,
  },
}))

// ── Build seats ────────────────────────────────────────────────────────
// Row → tutor assignment: [Alice, Bob, Clara, David, David, Eva, Eva, Felix, Felix]
const rowTutor = [0, 1, 2, 3, 3, 4, 4, 5, 5]
// Mac seats: R5 local 1-6 → phys 3-8, R6 local 1-6 → phys 3,4,5,7,8,9
const macSeatKeys = new Set<string>()
;[3,4,5,6,7,8].forEach(p => macSeatKeys.add(`5-${p}`))
;[3,4,5,7,8,9].forEach(p => macSeatKeys.add(`6-${p}`))

// Tutor seats
const tutorSeatPositions: Record<number, { row: number; physPos: number }> = {
  0: { row: 1, physPos: 12 },
  1: { row: 2, physPos: 12 },
  2: { row: 3, physPos: 12 },
  3: { row: 4, physPos: 12 },
  4: { row: 7, physPos: 12 },
  5: { row: 8, physPos: 12 },
}

// Build all seat objects
const allSeatMeta: { name: string; row: number; physPos: number; hasMac: boolean }[] = []
RECHNERHALLE_LAYOUT.forEach(r => {
  const positions = getPhysicalPositions(r)
  positions.forEach((physPos, idx) => {
    allSeatMeta.push({
      name: `1-${r.row}-${idx + 1}`,
      row: r.row,
      physPos,
      hasMac: macSeatKeys.has(`${r.row}-${physPos}`),
    })
  })
})

// Create seat objects with tutor and student assignments
const seats: Seat[] = []
const tutorSeatKeys = new Set<string>()
for (const [tIdx, pos] of Object.entries(tutorSeatPositions)) {
  tutorSeatKeys.add(`${pos.row}-${pos.physPos}`)
}

// Group student seats by tutor, Mac-first sort
type SeatMeta = typeof allSeatMeta[number]
const tutorSeatGroups: SeatMeta[][] = [[], [], [], [], [], []]

for (const sm of allSeatMeta) {
  const key = `${sm.row}-${sm.physPos}`
  if (tutorSeatKeys.has(key)) continue
  tutorSeatGroups[rowTutor[sm.row - 1]].push(sm)
}
for (let t = 0; t < 6; t++) {
  tutorSeatGroups[t].sort((a, b) => {
    if (a.hasMac !== b.hasMac) return a.hasMac ? -1 : 1
    if (a.row !== b.row) return a.row - b.row
    return a.physPos - b.physPos
  })
}

// Smart Mac-aware student distribution
const macNeedyStudents = [...macNeedy].map(i => i)
const macOwnerStudents = studentNames.map((_, i) => i).filter(i => !macNeedy.has(i))
const caps = [10, 8, 9, 10, 10, 9]
const tutorStudents: number[][] = [[], [], [], [], [], []]

// David gets first 6 Mac-needy, Eva gets remaining
let mnIdx = 0
for (let i = 0; i < 6 && mnIdx < macNeedyStudents.length; i++) tutorStudents[3].push(macNeedyStudents[mnIdx++])
for (let i = 0; i < 6 && mnIdx < macNeedyStudents.length; i++) tutorStudents[4].push(macNeedyStudents[mnIdx++])

// Fill remaining with Mac-owners
let moIdx = 0
for (let t = 0; t < 6; t++) {
  const remaining = caps[t] - tutorStudents[t].length
  for (let i = 0; i < remaining && moIdx < macOwnerStudents.length; i++) {
    tutorStudents[t].push(macOwnerStudents[moIdx++])
  }
}

// Sort students within group: Mac-needy first
for (let t = 0; t < 6; t++) {
  tutorStudents[t].sort((a, b) => {
    const aMac = macNeedy.has(a) ? 0 : 1
    const bMac = macNeedy.has(b) ? 0 : 1
    return aMac - bMac
  })
}

// Assign students to sorted seats
const studentSeatAssignment = new Map<string, string>() // seatName → studentId
for (let t = 0; t < 6; t++) {
  const seatGroup = tutorSeatGroups[t]
  const studs = tutorStudents[t]
  for (let i = 0; i < studs.length && i < seatGroup.length; i++) {
    studentSeatAssignment.set(seatGroup[i].name, `student-${studs[i]}`)
  }
}

// Build final Seat[]
for (const sm of allSeatMeta) {
  const key = `${sm.row}-${sm.physPos}`
  const isTutorSeat = tutorSeatKeys.has(key)
  const tIdx = rowTutor[sm.row - 1]

  seats.push({
    seatName: sm.name,
    hasMac: sm.hasMac,
    deviceID: null,
    assignedStudent: isTutorSeat ? null : (studentSeatAssignment.get(sm.name) ?? null),
    assignedTutor: `tutor-${tIdx}`,
    isTutorSeat,
  })
}

// ── Peer assignments ───────────────────────────────────────────────────
// Build peer groups per tutor using 3a + 4b = n partitioning
function partition(n: number): number[] {
  const groups: number[] = []
  const rem = n % 3
  const numQuads = rem === 0 ? 0 : rem === 1 ? 1 : 2
  const numTriples = (n - numQuads * 4) / 3
  for (let i = 0; i < numTriples; i++) groups.push(3)
  for (let i = 0; i < numQuads; i++) groups.push(4)
  return groups
}

const peerAssignments: PeerAssignment[] = []
for (let t = 0; t < 6; t++) {
  const studs = tutorStudents[t]
  const groups = partition(studs.length)
  let offset = 0
  for (const size of groups) {
    const members = studs.slice(offset, offset + size)
    // Create peer pairs: each member is paired with every other member
    for (let i = 0; i < members.length; i++) {
      for (let j = i + 1; j < members.length; j++) {
        peerAssignments.push({
          studentID: `student-${members[i]}`,
          peerID: `student-${members[j]}`,
        })
      }
    }
    offset += size
  }
}

// ── The screenshot page ────────────────────────────────────────────────
const SeatGridPage = () => {
  const [, setForceUpdate] = useState(0)

  return (
    <div className='p-6 max-w-5xl mx-auto bg-background text-foreground'>
      <div className='space-y-4'>
        <div>
          <h2 className='text-xl font-bold tracking-tight'>Seat Assignment</h2>
          <p className='text-sm text-muted-foreground mt-1'>
            Rechnerhalle Room 1 — {studentNames.length} of {allSeatMeta.length - 6} student seats assigned. 12 Mac seats (R5 &amp; R6).
          </p>
        </div>

        <div className='bg-card border rounded-lg p-6 shadow-sm'>
          <SeatGrid
            seats={seats}
            tutors={tutors}
            participations={participations}
            peerAssignments={peerAssignments}
          />
        </div>
      </div>
    </div>
  )
}

export const ScreenshotApp = () => (
  <QueryClientProvider client={queryClient}>
    <MemoryRouter initialEntries={['/course/c1/phase/p1']}>
      <Routes>
        <Route path='/course/:courseId/phase/:phaseId' element={<SeatGridPage />} />
      </Routes>
    </MemoryRouter>
  </QueryClientProvider>
)
