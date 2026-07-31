import { type ExtendedRouteObject, Role } from '@tumaet/prompt-shared-state'
import { IntroCourseDataShell } from '../src/introCourse/IntroCourseDataShell'
import { IntroCoursePage } from '../src/introCourse/IntroCoursePage'
import { DeveloperProfilesLecturerPage } from '../src/introCourse/pages/DeveloperProfilesLecturer/DeveloperProfilesLecturerPage'
import { IntroCourseParticipantsPage } from '../src/introCourse/pages/IntroCourseParticipantsPage/IntroCourseParticipantsPage'
import { MailingPage } from '../src/introCourse/pages/Mailing/MailingPage'
import { PeerAssignmentPage } from '../src/introCourse/pages/PeerAssignment/PeerAssignmentPage'
import { SeatAssignmentPage } from '../src/introCourse/pages/SeatAssignment/SeatAssignmentPage'
import { TutorImportPage } from '../src/introCourse/pages/TutorImport/TutorImportPage'

const routes: ExtendedRouteObject[] = [
  {
    path: '',
    element: (
      <IntroCourseDataShell>
        <IntroCoursePage />
      </IntroCourseDataShell>
    ),
    requiredPermissions: [Role.PROMPT_ADMIN, Role.COURSE_LECTURER, Role.COURSE_STUDENT], // empty means no permissions required
  },
  {
    path: '/participants',
    element: <IntroCourseParticipantsPage />,
    requiredPermissions: [Role.PROMPT_ADMIN, Role.COURSE_LECTURER],
  },
  {
    path: '/developer-profiles',
    element: <DeveloperProfilesLecturerPage />,
    requiredPermissions: [Role.PROMPT_ADMIN, Role.COURSE_LECTURER],
  },
  {
    path: '/tutors',
    element: <TutorImportPage />,
    requiredPermissions: [Role.PROMPT_ADMIN, Role.COURSE_LECTURER],
  },
  {
    path: '/seat-assignments',
    element: <SeatAssignmentPage />,
    requiredPermissions: [Role.PROMPT_ADMIN, Role.COURSE_LECTURER],
  },
  {
    path: '/peer-assignments',
    element: <PeerAssignmentPage />,
    requiredPermissions: [Role.PROMPT_ADMIN, Role.COURSE_LECTURER],
  },
  {
    path: '/mailing',
    element: <MailingPage />,
    requiredPermissions: [Role.PROMPT_ADMIN, Role.COURSE_LECTURER],
  },
  // Add more routes here as needed
]

export default routes
