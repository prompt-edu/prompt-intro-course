import { introCourseAxiosInstance } from '../introCourseServerConfig'

const profilePath = (phaseId: string, participationId: string) =>
  `intro-course/api/course_phase/${phaseId}/infrastructure/apple/profile/${participationId}`

export const inviteToAppleTeam = async (phaseId: string, participationId: string) => {
  const response = await introCourseAxiosInstance.post<{ created: boolean }>(
    `${profilePath(phaseId, participationId)}/invitation`,
  )
  return response.data
}

export const registerAppleDevice = async (
  phaseId: string,
  participationId: string,
  kind: 'iphone' | 'ipad' | 'watch',
) => {
  const response = await introCourseAxiosInstance.post<{ created: boolean }>(
    `${profilePath(phaseId, participationId)}/devices/${kind}`,
  )
  return response.data
}
