import { OwnPeerAssignment } from '../../interfaces/PeerAssignment'
import { introCourseAxiosInstance } from '../introCourseServerConfig'

export const getOwnPeerAssignment = async (
  coursePhaseID: string,
): Promise<OwnPeerAssignment> => {
  try {
    return (
      await introCourseAxiosInstance.get(
        `intro-course/api/course_phase/${coursePhaseID}/peer-assignments/own`,
      )
    ).data
  } catch (err) {
    console.error(err)
    throw err
  }
}
