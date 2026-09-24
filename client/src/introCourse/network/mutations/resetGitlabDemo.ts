import { introCourseAxiosInstance } from '../introCourseServerConfig'

export interface ResetGitlabDemoRequest {
  semesterTag: string
  expectedProjectID: number
  expectedSourceSHA: string
}

export const resetGitlabDemo = async (
  phaseId: string,
  request: ResetGitlabDemoRequest,
): Promise<void> => {
  await introCourseAxiosInstance.post(
    `intro-course/api/course_phase/${phaseId}/infrastructure/gitlab/demo/reset`,
    request,
  )
}
