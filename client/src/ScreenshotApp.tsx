/**
 * E2E screenshot harness — fetches real data from Go API, renders real components.
 * Usage: start Go dev server on :8082, then Rspack dev server on :3006
 */

import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import type { CoursePhaseParticipationWithStudent } from '@tumaet/prompt-shared-state'
import {
  Alert,
  AlertDescription,
  Avatar,
  AvatarFallback,
  Badge,
  Button,
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
  Skeleton,
} from '@tumaet/prompt-ui-components'
import { ExternalLink, Monitor, User, Users } from 'lucide-react'
import { useEffect, useState } from 'react'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import type { DeveloperProfile } from './introCourse/interfaces/DeveloperProfile'
import type { PeerAssignment } from './introCourse/interfaces/PeerAssignment'
import type { Seat } from './introCourse/interfaces/Seat'
import type { Tutor } from './introCourse/interfaces/Tutor'
import { PeerGroupList } from './introCourse/pages/PeerAssignment/components/PeerGroupList'
import { SeatGrid } from './introCourse/pages/SeatAssignment/components/SeatGrid/SeatGrid'
import { SeatTutorTable } from './introCourse/pages/SeatAssignment/components/SeatTutorAssigner/SeatTutorTable'

const queryClient = new QueryClient({
  defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
})

// Use relative URL — Rspack dev server proxies /intro-course to Go server
const API = '/intro-course/api/course_phase/4179d58a-d00d-4fa7-94a5-397bc69fab02'

// ── Student name lookup (PROMPT Core isn't running, so we provide names locally) ──
const studentNames: [string, string][] = [
  ['Max', 'Mueller'],
  ['Anna', 'Schneider'],
  ['Lukas', 'Wagner'],
  ['Sophie', 'Fischer'],
  ['Leon', 'Weber'],
  ['Emma', 'Braun'],
  ['Paul', 'Hoffmann'],
  ['Marie', 'Schulz'],
  ['Jonas', 'Koch'],
  ['Tim', 'Klein'],
  ['Felix', 'Groß'],
  ['Hannah', 'Bauer'],
  ['Lena', 'Berger'],
  ['Tom', 'Richter'],
  ['Laura', 'Krause'],
  ['Nico', 'Wolf'],
  ['Mia', 'Schmitt'],
  ['Finn', 'Neumann'],
  ['Sara', 'Schwarz'],
  ['Eric', 'Zimmermann'],
  ['Robin', 'Braun'],
  ['Jan', 'Beck'],
  ['Fiona', 'Keller'],
  ['Henry', 'Hartmann'],
  ['Ben', 'Lang'],
  ['Lisa', 'Schäfer'],
  ['Lea', 'Werner'],
  ['Lars', 'Seidel'],
  ['Timo', 'Meyer'],
  ['Julia', 'Lange'],
  ['Nina', 'Schmid'],
  ['Alex', 'Meier'],
  ['Diana', 'Krug'],
  ['Nora', 'Hahn'],
  ['Jakob', 'Kaiser'],
  ['Clara', 'Weiß'],
  ['Max', 'König'],
  ['Anne', 'Frank'],
  ['Hugo', 'Peters'],
  ['Pia', 'Brandt'],
  ['Cleo', 'Ludwig'],
  ['Oscar', 'Sommer'],
  ['Ella', 'Maier'],
  ['Karl', 'Wirth'],
  ['Kurt', 'Jung'],
  ['Eva', 'Horn'],
  ['Zoe', 'Stein'],
  ['Sam', 'Vogel'],
  ['Noah', 'Fiedler'],
  ['Ralf', 'Krüger'],
  ['Lara', 'Koenig'],
  ['Theo', 'Günther'],
  ['Peter', 'Fuchs'],
  ['Ida', 'Becker'],
  ['Tina', 'Wendt'],
  ['Vera', 'Roth'],
]

function buildParticipations(seats: Seat[]): CoursePhaseParticipationWithStudent[] {
  const studentIds = new Set<string>()
  for (const s of seats) {
    if (s.assignedStudent) studentIds.add(s.assignedStudent)
  }
  return Array.from(studentIds).map((id) => {
    const idx = parseInt(id.slice(-2), 10) - 1
    const [first, last] = studentNames[idx] ?? ['Student', id.slice(-4)]
    return {
      coursePhaseID: '4179d58a-d00d-4fa7-94a5-397bc69fab02',
      courseParticipationID: id,
      passStatus: 'not_assessed' as any,
      restrictedData: {},
      studentReadableData: {},
      prevData: {},
      student: {
        id,
        firstName: first,
        lastName: last,
        email: `${first.toLowerCase()}.${last.toLowerCase()}@example.com`,
        hasUniversityAccount: true,
      },
    }
  })
}

// ── Shared loading/error states ───────────────────────────────────────
const LoadingState = () => (
  <div className='p-6 space-y-4'>
    <Skeleton className='h-8 w-48' />
    <Skeleton className='h-64 w-full' />
  </div>
)

const ErrorState = ({ message }: { message: string }) => (
  <div className='p-6'>
    <Alert variant='destructive'>
      <AlertDescription>{message}</AlertDescription>
    </Alert>
  </div>
)

// ── Data fetching hook ────────────────────────────────────────────────
function useApiData() {
  const [seats, setSeats] = useState<Seat[]>([])
  const [tutors, setTutors] = useState<Tutor[]>([])
  const [peers, setPeers] = useState<PeerAssignment[]>([])
  const [profiles, setProfiles] = useState<DeveloperProfile[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    Promise.all([
      fetch(`${API}/seat_plan`).then((r) => r.json()),
      fetch(`${API}/tutor`).then((r) => r.json()),
      fetch(`${API}/peer_assignments`).then((r) => r.json()),
      fetch(`${API}/developer-profile/all`).then((r) => r.json()),
    ])
      .then(([s, t, p, d]) => {
        setSeats(s)
        setTutors(t)
        setPeers(p)
        setProfiles(d)
        setLoading(false)
      })
      .catch((e) => {
        setError(e.message)
        setLoading(false)
      })
  }, [])

  const participations = buildParticipations(seats)
  return { seats, tutors, peers, profiles, participations, loading, error }
}

// ── Page: Seat Grid ───────────────────────────────────────────────────
const SeatGridPage = () => {
  const { seats, tutors, peers, participations, loading, error } = useApiData()
  if (loading) return <LoadingState />
  if (error) return <ErrorState message={error} />

  const studentCount = seats.filter((s) => s.assignedStudent).length
  const totalStudentSeats = seats.filter((s) => !s.isTutorSeat).length

  return (
    <div className='p-6 max-w-5xl mx-auto'>
      <div className='space-y-4'>
        <div>
          <h2 className='text-xl font-bold tracking-tight'>Seat Assignment</h2>
          <p className='text-sm text-muted-foreground mt-1'>
            Rechnerhalle Room 1 — {studentCount} of {totalStudentSeats} student seats assigned
          </p>
        </div>
        <Card>
          <CardContent className='pt-6'>
            <SeatGrid
              seats={seats}
              tutors={tutors}
              participations={participations}
              peerAssignments={peers}
            />
          </CardContent>
        </Card>
      </div>
    </div>
  )
}

// ── Page: Tutor Assignment Table ──────────────────────────────────────
const TutorTablePage = () => {
  const { seats, tutors, loading, error } = useApiData()
  const [selected, setSelected] = useState<string[]>([])
  if (loading) return <LoadingState />
  if (error) return <ErrorState message={error} />

  return (
    <div className='p-6 max-w-4xl mx-auto'>
      <div className='space-y-4'>
        <h2 className='text-xl font-bold tracking-tight'>Tutor Assignment</h2>
        <Card>
          <CardContent className='pt-6'>
            <SeatTutorTable
              allSeats={seats}
              tutors={tutors}
              selectedSeatNames={selected}
              setSelectedSeatNames={setSelected}
              handleTutorAssignment={() => {}}
              handleTutorSeatToggle={() => {}}
            />
          </CardContent>
        </Card>
      </div>
    </div>
  )
}

// ── Page: Peer Group List ─────────────────────────────────────────────
const PeerGroupPage = () => {
  const { seats, tutors, peers, profiles, participations, loading, error } = useApiData()
  if (loading) return <LoadingState />
  if (error) return <ErrorState message={error} />

  return (
    <div className='p-6 max-w-4xl mx-auto'>
      <div className='space-y-4'>
        <h2 className='text-xl font-bold tracking-tight'>Peer Review Groups</h2>
        <PeerGroupList
          peerAssignments={peers}
          seats={seats}
          tutors={tutors}
          developerProfiles={profiles}
          participations={participations}
        />
      </div>
    </div>
  )
}

// ── Page: Empty state (no peer assignments) ──────────────────────────
const EmptyStatePage = () => {
  const { seats, tutors, profiles, participations, loading, error } = useApiData()
  if (loading) return <LoadingState />
  if (error) return <ErrorState message={error} />

  return (
    <div className='p-6 max-w-4xl mx-auto'>
      <div className='space-y-4'>
        <h2 className='text-xl font-bold tracking-tight'>Peer Review Groups</h2>
        <PeerGroupList
          peerAssignments={[]}
          seats={seats}
          tutors={tutors}
          developerProfiles={profiles}
          participations={participations}
        />
      </div>
    </div>
  )
}

// ── Page: Student Seat View (standalone, no zustand) ─────────────────
const StudentViewPage = () => {
  const [data, setData] = useState<any>(null)
  const [peerData, setPeerData] = useState<any>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    Promise.all([
      fetch(`${API}/seat_plan/own-assignment`).then((r) => r.json()),
      fetch(`${API}/peer_assignments/own`).then((r) => r.json()),
    ])
      .then(([seat, peers]) => {
        setData(seat)
        setPeerData(peers)
        setLoading(false)
      })
      .catch(() => setLoading(false))
  }, [])

  if (loading) return <LoadingState />
  if (!data) return <ErrorState message='No seat assignment data available' />

  const { seatName, hasMac, deviceID, tutorFirstName, tutorLastName, tutorEmail } = data
  const tutorFullName = `${tutorFirstName} ${tutorLastName}`
  const tutorInitial = tutorFirstName.charAt(0)
  const peersIReview = peerData?.peersIReview ?? []

  return (
    <div className='p-6 max-w-3xl mx-auto'>
      <div className='space-y-4'>
        <h2 className='text-xl font-bold tracking-tight'>My Seat Assignment</h2>
        <div className='grid grid-cols-1 md:grid-cols-2 gap-4'>
          {/* Seat Information Card */}
          <Card>
            <CardHeader className='pb-3'>
              <CardTitle className='flex items-center gap-2 text-base'>
                <Monitor className='h-5 w-5 text-primary' />
                Seat Information
              </CardTitle>
            </CardHeader>
            <CardContent>
              <div>
                <p className='text-sm text-muted-foreground'>Assigned Seat</p>
                <p className='text-xl font-semibold'>{seatName}</p>
              </div>
              <div className='mt-4'>
                <p className='text-sm text-muted-foreground'>Device Type</p>
                <div className='flex items-center gap-2 mt-1'>
                  <Badge variant='outline'>{hasMac ? 'Chair Mac' : 'Own MacBook'}</Badge>
                  {deviceID && <span className='text-sm'>ID: {deviceID}</span>}
                </div>
              </div>
            </CardContent>
          </Card>

          {/* Tutor Information Card */}
          <Card>
            <CardHeader className='pb-3'>
              <CardTitle className='flex items-center gap-2 text-base'>
                <User className='h-5 w-5 text-primary' />
                Your Tutor
              </CardTitle>
            </CardHeader>
            <CardContent>
              <div className='flex items-center gap-4'>
                <Avatar className='h-16 w-16 border-2 border-background shadow-xs'>
                  <AvatarFallback className='text-lg font-bold'>{tutorInitial}</AvatarFallback>
                </Avatar>
                <div>
                  <p className='font-semibold text-lg'>{tutorFullName}</p>
                  <p className='text-sm text-muted-foreground'>{tutorEmail}</p>
                </div>
              </div>
            </CardContent>
          </Card>

          {/* Peer Review Information Card */}
          {peersIReview.length > 0 && (
            <Card className='md:col-span-2'>
              <CardHeader className='pb-3'>
                <CardTitle className='flex items-center gap-2 text-base'>
                  <Users className='h-5 w-5 text-primary' />
                  Your Review Peers
                </CardTitle>
                <CardDescription>
                  Your main task is to <strong>manually test</strong> your peer&apos;s application.
                </CardDescription>
              </CardHeader>
              <CardContent>
                <div className='grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3'>
                  {peersIReview.map((peer: any) => (
                    <Card key={peer.courseParticipationID}>
                      <CardContent className='p-3 flex flex-col gap-2'>
                        <p className='font-medium'>{peer.gitlabUsername}</p>
                        {peer.seatName && (
                          <p className='text-sm text-muted-foreground'>Seat: {peer.seatName}</p>
                        )}
                        {peer.gitlabUsername && (
                          <Button variant='outline' size='sm' className='w-fit' disabled>
                            <ExternalLink className='h-3 w-3 mr-1' />
                            GitLab Repo
                          </Button>
                        )}
                      </CardContent>
                    </Card>
                  ))}
                </div>
              </CardContent>
            </Card>
          )}
        </div>
      </div>
    </div>
  )
}

// ── App ──────────────────────────────────────────────────────────────
export const ScreenshotApp = ({ initialRoute = '/grid' }: { initialRoute?: string }) => (
  <QueryClientProvider client={queryClient}>
    <MemoryRouter initialEntries={[initialRoute]}>
      <Routes>
        <Route path='/grid' element={<SeatGridPage />} />
        <Route path='/tutor-table' element={<TutorTablePage />} />
        <Route path='/peer-groups' element={<PeerGroupPage />} />
        <Route path='/empty-state' element={<EmptyStatePage />} />
        <Route path='/student-view' element={<StudentViewPage />} />
        <Route path='*' element={<SeatGridPage />} />
      </Routes>
    </MemoryRouter>
  </QueryClientProvider>
)
