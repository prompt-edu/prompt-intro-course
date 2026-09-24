import { isAxiosError } from 'axios'
import type { GitlabCourseInfrastructureRequest } from '../../interfaces/GitlabCourseInfrastructureRequest'
import { introCourseAxiosInstance } from '../introCourseServerConfig'

export const createIntroCourseGitlabInfrastructure = async (
  coursePhaseID: string,
  request: GitlabCourseInfrastructureRequest,
): Promise<void> => {
  try {
    await introCourseAxiosInstance.post(
      `intro-course/api/course_phase/${coursePhaseID}/infrastructure/gitlab/course-setup`,
      request,
    )
  } catch (err: unknown) {
    if (isAxiosError(err) && typeof err.response?.data?.error === 'string') {
      throw err.response.data.error
    }
    throw err
  }
}
