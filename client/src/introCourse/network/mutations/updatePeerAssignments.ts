import { PeerAssignment } from '../../interfaces/PeerAssignment'
import { introCourseAxiosInstance } from '../introCourseServerConfig'

export const updatePeerAssignments = async (
  coursePhaseID: string,
  assignments: PeerAssignment[],
): Promise<void> => {
  try {
    await introCourseAxiosInstance.put(
      `intro-course/api/course_phase/${coursePhaseID}/peer_assignments`,
      assignments,
      {
        headers: {
          'Content-Type': 'application/json',
        },
      },
    )
  } catch (err) {
    console.error(err)
    throw err
  }
}
