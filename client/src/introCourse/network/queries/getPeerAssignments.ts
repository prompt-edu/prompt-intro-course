import { PeerAssignment } from '../../interfaces/PeerAssignment'
import { introCourseAxiosInstance } from '../introCourseServerConfig'

export const getPeerAssignments = async (
  coursePhaseID: string,
): Promise<PeerAssignment[]> => {
  try {
    return (
      await introCourseAxiosInstance.get(
        `intro-course/api/course_phase/${coursePhaseID}/peer_assignments`,
      )
    ).data
  } catch (err) {
    console.error(err)
    throw err
  }
}
