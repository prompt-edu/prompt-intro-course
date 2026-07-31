import type { CoursePhaseParticipationWithStudent } from '@tumaet/prompt-shared-state'
import type { DeveloperProfile } from '../../../interfaces/DeveloperProfile'
import type { GitlabStatus } from '../../../interfaces/GitlabStatus'

export type ParticipationWithDevProfiles = {
  participation: CoursePhaseParticipationWithStudent
  devProfile: DeveloperProfile | undefined
  gitlabStatus: GitlabStatus | undefined
}
