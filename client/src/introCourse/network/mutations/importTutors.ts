import { Student } from '@tumaet/prompt-shared-state'
import { introCourseAxiosInstance } from '../introCourseServerConfig'

export interface ImportTutorsResult {
  imported?: number
  warnings?: string[]
}

export const importTutors = async (
  coursePhaseID: string,
  courseID: string,
  tutors: Student[],
): Promise<ImportTutorsResult> => {
  try {
    const response = await introCourseAxiosInstance.post(
      `intro-course/api/course_phase/${coursePhaseID}/tutor/course/${courseID}`,
      tutors,
    )
    // Server returns JSON body with warnings, or empty 201
    return response.data ?? {}
  } catch (err) {
    console.error(err)
    throw err
  }
}
