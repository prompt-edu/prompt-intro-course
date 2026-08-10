// Seeded Keycloak users from e2e/keycloak/realm.json (a verbatim copy of the
// prompt2 e2e realm). Passwords are the literal dev credentials baked into the
// realm import (username === password for all).
// `permission` is the role that appears in resourceAccess['prompt-server'].roles.

export type Role =
  | 'admin'
  | 'lecturer'
  | 'course-lecturer'
  | 'course-editor'
  | 'student'
  | 'student2'

export interface RoleAccount {
  username: string
  password: string
  permission: string
  email: string
}

export const ROLES: Record<Role, RoleAccount> = {
  admin: {
    username: 'admin',
    password: 'admin',
    permission: 'PROMPT_Admin',
    email: 'admin@example.com',
  },
  lecturer: {
    username: 'lecturer',
    password: 'lecturer',
    permission: 'PROMPT_Lecturer',
    email: 'lecturer@example.com',
  },
  'course-lecturer': {
    username: 'course-lecturer',
    password: 'course-lecturer',
    permission: 'PROMPT_Course_Lecturer',
    email: 'of_course-lecturer@example.com',
  },
  'course-editor': {
    username: 'course-editor',
    password: 'course-editor',
    permission: 'PROMPT_Course_Editor',
    email: 'best_edits@example.com',
  },
  // Stan: participates in the seeded intro course phase and is deliberately left
  // WITHOUT a developer profile, so the student journey can submit the survey.
  student: {
    username: 'student',
    password: 'student',
    permission: 'PROMPT_Student',
    email: 'pgdp_enjoyer@example.com',
  },
  // Selma: participates in the same phase and HAS a profile, a seat, and peers,
  // so the seat-assignment display has something to render.
  student2: {
    username: 'student2',
    password: 'student2',
    permission: 'PROMPT_Student',
    email: 'second_student@example.com',
  },
}

// Roles we pre-authenticate in global-setup (storageState reused by tests).
export const SEEDED_ROLES: Role[] = [
  'admin',
  'lecturer',
  'course-lecturer',
  'course-editor',
  'student',
  'student2',
]
