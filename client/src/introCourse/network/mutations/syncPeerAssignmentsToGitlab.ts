import type { SyncResult } from '../../interfaces/PeerAssignment'
import { introCourseAxiosInstance } from '../introCourseServerConfig'

export const syncPeerAssignmentsToGitlab = async (
  coursePhaseID: string,
  semesterTag: string,
): Promise<SyncResult[]> => {
  try {
    return (
      await introCourseAxiosInstance.post(
        `intro-course/api/course_phase/${coursePhaseID}/peer_assignments/sync-gitlab`,
        { semesterTag },
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
