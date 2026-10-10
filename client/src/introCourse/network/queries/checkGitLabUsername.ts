import { introCourseAxiosInstance } from '../introCourseServerConfig'

export type GitLabUsernameCheck = {
  username: string
  status: 'found' | 'missing' | 'invalid_format' | 'not_found' | 'check_failed'
  gitLabName: string
  gitLabURL: string
}

export const checkGitLabUsername = async (
  coursePhaseID: string,
  username: string,
): Promise<GitLabUsernameCheck> => {
  const response = await introCourseAxiosInstance.get<GitLabUsernameCheck>(
    `intro-course/api/course_phase/${coursePhaseID}/developer_profile/gitlab-user/${encodeURIComponent(username)}`,
  )
  return response.data
}
