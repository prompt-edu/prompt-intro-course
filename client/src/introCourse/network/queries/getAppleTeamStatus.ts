import { introCourseAxiosInstance } from '../introCourseServerConfig'

export interface AppleTeamCapacity {
  registeredIPhones: number
  availableIPhones: number
  limit: number
}

export interface AppleProfileStatus {
  membership: 'none' | 'invited' | 'active'
  provisioningAllowed: boolean
  devices: Record<string, boolean>
}

export const getAppleTeamCapacity = async (phaseId: string): Promise<AppleTeamCapacity> => {
  const response = await introCourseAxiosInstance.get<AppleTeamCapacity>(
    `intro-course/api/course_phase/${phaseId}/infrastructure/apple/status`,
  )
  return response.data
}

export const getOwnAppleTeamStatus = async (phaseId: string): Promise<AppleProfileStatus> => {
  const response = await introCourseAxiosInstance.get<AppleProfileStatus>(
    `intro-course/api/course_phase/${phaseId}/infrastructure/apple/self`,
  )
  return response.data
}

export const getAppleProfileStatus = async (
  phaseId: string,
  participationId: string,
): Promise<AppleProfileStatus> => {
  const response = await introCourseAxiosInstance.get<AppleProfileStatus>(
    `intro-course/api/course_phase/${phaseId}/infrastructure/apple/profile/${participationId}`,
  )
  return response.data
}
