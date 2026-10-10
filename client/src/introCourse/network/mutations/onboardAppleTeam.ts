import { introCourseAxiosInstance } from '../introCourseServerConfig'

const profilePath = (phaseId: string, participationId: string) =>
  `intro-course/api/course_phase/${phaseId}/infrastructure/apple/profile/${participationId}`

export const inviteToAppleTeam = async (
  phaseId: string,
  participationId: string,
  expectedEmail: string,
) => {
  const response = await introCourseAxiosInstance.post<{ created: boolean }>(
    `${profilePath(phaseId, participationId)}/invitation`,
    { expectedEmail },
  )
  return response.data
}

export const registerAppleDevice = async (
  phaseId: string,
  participationId: string,
  kind: 'iphone' | 'ipad' | 'watch',
  expectedUDID: string,
) => {
  const response = await introCourseAxiosInstance.post<{ created: boolean }>(
    `${profilePath(phaseId, participationId)}/devices/${kind}`,
    { expectedUDID },
  )
  return response.data
}
