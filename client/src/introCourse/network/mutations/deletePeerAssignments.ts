import { introCourseAxiosInstance } from '../introCourseServerConfig'

export const deletePeerAssignments = async (coursePhaseID: string): Promise<void> => {
  try {
    await introCourseAxiosInstance.delete(
      `intro-course/api/course_phase/${coursePhaseID}/peer_assignments`,
    )
  } catch (err) {
    console.error(err)
    throw err
  }
}
