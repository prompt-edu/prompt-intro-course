import { PeerAssignment } from '../../interfaces/PeerAssignment'
import { introCourseAxiosInstance } from '../introCourseServerConfig'

export const generatePeerAssignments = async (
  coursePhaseID: string,
): Promise<PeerAssignment[]> => {
  try {
    return (
      await introCourseAxiosInstance.post(
        `intro-course/api/course_phase/${coursePhaseID}/peer_assignments/generate`,
        {},
        {
          headers: {
            'Content-Type': 'application/json',
          },
        },
      )
    ).data
  } catch (err) {
    console.error(err)
    throw err
  }
}
