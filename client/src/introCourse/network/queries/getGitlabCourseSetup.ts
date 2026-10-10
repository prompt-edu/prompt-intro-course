import type { GitlabCourseSetupStatus } from '../../interfaces/GitlabCourseSetupStatus'
import { introCourseAxiosInstance } from '../introCourseServerConfig'

export const getGitlabCourseSetup = async (
  phaseId: string,
  semesterTag: string,
): Promise<GitlabCourseSetupStatus> => {
  const response = await introCourseAxiosInstance.get<GitlabCourseSetupStatus>(
    `intro-course/api/course_phase/${phaseId}/infrastructure/gitlab/course-setup`,
    { params: { semesterTag } },
  )
  return response.data
}
