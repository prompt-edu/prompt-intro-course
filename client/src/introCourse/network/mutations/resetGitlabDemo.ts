import { introCourseAxiosInstance } from '../introCourseServerConfig'

export interface ResetGitlabDemoRequest {
  semesterTag: string
  expectedProjectID: number
  expectedSourceSHA: string
}

export interface ResetGitlabDemoResult {
  demoUrl: string
  demoId: number
  archiveUrl: string
  archiveMoveRequired: boolean
  sourceSha: string
}

export const resetGitlabDemo = async (
  phaseId: string,
  request: ResetGitlabDemoRequest,
): Promise<ResetGitlabDemoResult> => {
  const response = await introCourseAxiosInstance.post<ResetGitlabDemoResult>(
    `intro-course/api/course_phase/${phaseId}/infrastructure/gitlab/demo/reset`,
    request,
  )
  return response.data
}
