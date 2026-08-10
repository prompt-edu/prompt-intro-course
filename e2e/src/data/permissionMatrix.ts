import { INTRO_COURSE_ROUTES } from './constants'
import { Role } from './roles'

// Declarative access-control matrix for the intro-course remote. The browser and
// API specs generate their cases by looping over it, so a new route or endpoint is
// covered for every role by adding one entry here.
//
// Roles are the six pre-authenticated users in e2e/keycloak/realm.json. On the
// seeded course (iPraktikumFull) they map to:
//   admin           -> PROMPT_Admin (unrestricted)
//   lecturer        -> ios2425-iPraktikumFull-Lecturer
//   course-lecturer -> ios2425-iPraktikumFull-Lecturer
//   course-editor   -> ios2425-iPraktikumFull-Editor  (NOT a lecturer: the
//                      interesting negative -- has course access, but the intro
//                      routes require PROMPT_ADMIN or COURSE_LECTURER)
//   student         -> enrolled, participates in the intro phase
//   student2        -> enrolled, participates in the intro phase

export const MATRIX_ROLES: Role[] = [
  'admin',
  'lecturer',
  'course-lecturer',
  'course-editor',
  'student',
  'student2',
]

export interface BrowserSurface {
  name: string
  route: string
  // The <h1> the remote renders when access is granted.
  heading: string
  allowed: Role[]
}

const STAFF: Role[] = ['admin', 'lecturer', 'course-lecturer']

// Mirrors requiredPermissions in client/routes/index.tsx.
export const BROWSER_SURFACES: BrowserSurface[] = [
  {
    name: 'intro course overview',
    route: INTRO_COURSE_ROUTES.overview,
    heading: 'Intro Course',
    // The only route that also admits COURSE_STUDENT.
    allowed: [...STAFF, 'student', 'student2'],
  },
  {
    name: 'participants',
    route: INTRO_COURSE_ROUTES.participants,
    heading: 'Intro Course Participants',
    allowed: STAFF,
  },
  {
    name: 'developer profiles',
    route: INTRO_COURSE_ROUTES.developerProfiles,
    heading: 'Developer Profile Management',
    allowed: STAFF,
  },
  {
    name: 'tutor import',
    route: INTRO_COURSE_ROUTES.tutors,
    heading: 'Tutor Import',
    allowed: STAFF,
  },
  {
    name: 'seat assignments',
    route: INTRO_COURSE_ROUTES.seatAssignments,
    heading: 'Seat Assignment',
    allowed: STAFF,
  },
  {
    name: 'peer assignments',
    route: INTRO_COURSE_ROUTES.peerAssignments,
    heading: 'Peer Assignments',
    allowed: STAFF,
  },
  {
    name: 'mailing',
    route: INTRO_COURSE_ROUTES.mailing,
    heading: 'Mailing',
    allowed: STAFF,
  },
]

export interface ApiSurface {
  name: string
  // Path under intro-course/api/course_phase/:coursePhaseID.
  path: string
  allowed: Role[]
}

// Mirrors the authMiddleware role lists in the server's routers. Only side-effect
// free GETs, so the matrix can be replayed without mutating seeded data.
export const LECTURER_API_SURFACES: ApiSurface[] = [
  { name: 'tutors', path: '/tutor', allowed: STAFF },
  { name: 'seat plan', path: '/seat_plan', allowed: STAFF },
  { name: 'peer assignments', path: '/peer_assignments', allowed: STAFF },
  { name: 'developer profiles', path: '/developer_profile', allowed: STAFF },
]

// Endpoints scoped to the calling student's own participation.
export const STUDENT_API_SURFACES: ApiSurface[] = [
  { name: 'own seat assignment', path: '/seat_plan/own-assignment', allowed: ['student', 'student2'] },
  { name: 'own developer profile', path: '/developer_profile/self', allowed: ['student', 'student2'] },
  { name: 'own peer assignment', path: '/peer_assignments/own', allowed: ['student', 'student2'] },
]
