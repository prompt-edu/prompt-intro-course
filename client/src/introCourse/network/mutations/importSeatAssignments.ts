import { introCourseAxiosInstance } from '../introCourseServerConfig'

export interface ImportSeatAssignment {
  seatName: string
  seatMac: boolean
  assignedStudent: string
  assignedTutor: string
  isTutorSeat: boolean
  peerGroup: string
}

export interface ImportResult {
  seatsUpdated: number
  peerGroupsImported: number
  warnings?: string[]
}

export const importSeatAssignments = async (
  coursePhaseID: string,
  assignments: ImportSeatAssignment[],
): Promise<ImportResult> => {
  const response = await introCourseAxiosInstance.post(
    `intro-course/api/course_phase/${coursePhaseID}/seat_plan/import`,
    { assignments },
  )
  return response.data
}
