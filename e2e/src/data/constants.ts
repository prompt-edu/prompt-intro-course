// Known values from the two seeds:
//   - core:         e2e/seed/core/10_core_seed.sql + 20_intro_course.sql
//   - intro course: server/database_dumps/e2e_seed.sql
// When either seed changes, update these.

// The course that carries the seeded phase graph, with course-scoped Keycloak
// roles for lecturer / course-lecturer / course-editor. Its name is hyphen-free
// because core parses course-scoped roles with split_part(role, '-', N).
export const SEEDED_COURSE = {
  id: 'c0000001-0000-0000-0000-000000000001',
  name: 'iPraktikumFull',
  semesterTag: 'ios2425',
}

// The intro course phase, appended to the tail of that course's phase chain.
// Same id as in server/database_dumps/e2e_seed.sql, which is what ties the two
// seeds together.
export const INTRO_COURSE_PHASE_ID = '4179d58a-d00d-4fa7-94a5-397bc69fab02'
export const INTRO_COURSE_PHASE_NAME = 'Intro Course'
export const INTRO_COURSE_PHASE_TYPE = 'Intro Course Developer'

// A phase of a DIFFERENT type on the same course. Every intro-course API call is
// scoped by :coursePhaseID, so this is the negative fixture for cross-phase
// reads: the caller is authorized on the course but the phase holds no intro data.
export const FOREIGN_PHASE_ID = 'd0000001-0000-0000-0000-000000000001'

// The two Keycloak student users' core course participations on SEEDED_COURSE.
// Course access is DB-derived (matriculation number + university login from the
// Keycloak user attributes), not a Keycloak role.
export const STUDENT_WITHOUT_PROFILE = {
  role: 'student' as const,
  courseParticipationId: 'a0000001-0000-0000-0000-000000000001',
  firstName: 'Stan',
  lastName: 'Stan',
  universityLogin: 'no42tum',
  matriculationNumber: '00000005',
}

export const STUDENT_WITH_PROFILE = {
  role: 'student2' as const,
  courseParticipationId: 'ca000008-0000-4000-8000-000000000008',
  firstName: 'Selma',
  lastName: 'Second',
  universityLogin: 'st70two',
  matriculationNumber: '00000007',
  gitlabUsername: 'ssecond',
  appleId: 'selma.second@icloud.com',
  // The one free non-tutor seat in Alice's row; Selma is assigned to it.
  seatName: '1-1-11',
  // Peer group A1, which Selma joins as its fourth member.
  peerGitlabUsernames: ['mmueller', 'aschneider', 'lwagner'],
}

// The 6 seeded tutors, in row order. Each owns one row of the Rechnerhalle
// layout (David and Eva own two).
export const SEEDED_TUTORS = [
  { id: 'a0000000-0000-0000-0000-000000000001', firstName: 'Alice', lastName: 'Mueller', gitlabUsername: 'amueller', row: 1 },
  { id: 'a0000000-0000-0000-0000-000000000002', firstName: 'Bob', lastName: 'Schmidt', gitlabUsername: 'bschmidt', row: 2 },
  { id: 'a0000000-0000-0000-0000-000000000003', firstName: 'Clara', lastName: 'Weber', gitlabUsername: 'cweber', row: 3 },
  { id: 'a0000000-0000-0000-0000-000000000004', firstName: 'David', lastName: 'Fischer', gitlabUsername: 'dfischer', row: 4 },
  { id: 'a0000000-0000-0000-0000-000000000005', firstName: 'Eva', lastName: 'Braun', gitlabUsername: 'ebraun', row: 6 },
  { id: 'a0000000-0000-0000-0000-000000000006', firstName: 'Felix', lastName: 'Wagner', gitlabUsername: 'fwagner', row: 8 },
]

export const SEEDED_TUTOR_COUNT = SEEDED_TUTORS.length

// The Rechnerhalle seat plan: 89 seats named 1-<row>-<local> across rows 1..9.
export const SEAT_PLAN = {
  totalSeats: 89,
  // 12 Mac seats: row 5 locals 1-6 (David's) and row 6 locals 1-6 (Eva's).
  macSeats: 12,
  // One tutor seat per tutor; rows 5, 6, and 9 have none.
  tutorSeats: ['1-1-12', '1-2-10', '1-3-10', '1-4-10', '1-7-10', '1-8-10'],
  // A seat assigned to a background student (Max Mueller), used for identity
  // assertions that must not depend on row counts.
  firstStudentSeat: '1-1-1',
}

// Background students: 56 developer profiles / seat assignments / peer group
// members, seeded so the lecturer views have realistic data. Nobody can log in
// as them — the browser flows use the two Keycloak users above.
export const BACKGROUND_STUDENT_COUNT = 56

// Two stable identities among them, asserted by identity rather than row count.
export const BACKGROUND_STUDENTS = {
  first: {
    courseParticipationId: 'b0000000-0000-0000-0000-000000000001',
    firstName: 'Max',
    lastName: 'Mueller',
    gitlabUsername: 'mmueller',
    appleId: 'max.mueller@icloud.com',
    hasMacbook: true,
  },
  // One of the 11 students with has_macbook = false, so it must be seated on a
  // chair Mac.
  withoutMacbook: {
    courseParticipationId: 'b0000000-0000-0000-0000-000000000006',
    firstName: 'Emma',
    lastName: 'Braun',
    hasMacbook: false,
  },
}

// Total participants on the intro phase: 56 background students + Stan + Selma.
export const INTRO_COURSE_PARTICIPANT_COUNT = BACKGROUND_STUDENT_COUNT + 2

// 17 peer groups over the 56 background students; Selma joins A1 as its fourth
// member, so 57 of the 58 participants are grouped (Stan is not).
export const GROUPED_STUDENT_COUNT = BACKGROUND_STUDENT_COUNT + 1

// The remote's routes, relative to /management/course/:courseId/:phaseId.
// Kept in sync with client/routes/index.tsx.
export const INTRO_COURSE_ROUTES = {
  overview: '',
  participants: '/participants',
  developerProfiles: '/developer-profiles',
  tutors: '/tutors',
  seatAssignments: '/seat-assignments',
  peerAssignments: '/peer-assignments',
  mailing: '/mailing',
}
