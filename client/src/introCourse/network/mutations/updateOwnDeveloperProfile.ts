import type { PostDeveloperProfile } from '../../interfaces/PostDeveloperProfile'
import { introCourseAxiosInstance } from '../introCourseServerConfig'

export const updateOwnDeveloperProfile = async (
  coursePhaseID: string,
  developerProfile: PostDeveloperProfile,
): Promise<void> => {
  await introCourseAxiosInstance.put(
    `intro-course/api/course_phase/${coursePhaseID}/developer_profile/self`,
    developerProfile,
  )
}
