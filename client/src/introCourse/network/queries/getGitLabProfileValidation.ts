import { introCourseAxiosInstance } from '../introCourseServerConfig'

export type GitLabProfileValidation = {
  courseParticipationID: string
  username: string
  status: 'found' | 'missing' | 'invalid_format' | 'not_found' | 'check_failed'
  gitLabName?: string
  gitLabURL?: string
}

export const getGitLabProfileValidation = async (
  coursePhaseID: string,
): Promise<GitLabProfileValidation[]> => {
  const response = await introCourseAxiosInstance.get<GitLabProfileValidation[]>(
    `intro-course/api/course_phase/${coursePhaseID}/developer_profile/gitlab-validation`,
  )
  return response.data
}
