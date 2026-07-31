import { useCallback } from 'react'
import type { PeerAssignment } from '../../../interfaces/PeerAssignment'
import type { Seat } from '../../../interfaces/Seat'
import type { Tutor } from '../../../interfaces/Tutor'
import type { DeveloperWithProfile } from '../interfaces/DeveloperWithProfile'
import { getTutorName } from '../utils/getTutorName'
import { buildPeerGroups } from '../utils/seatGrid'

export const useDownloadAssignment = (
  seats: Seat[],
  developerWithProfiles: DeveloperWithProfile[],
  tutors: Tutor[],
  peerAssignments?: PeerAssignment[],
) => {
  function escapeCsvField(field: string): string {
    // If the field contains commas, quotes, or newlines, wrap it in quotes
    // and escape any quotes by doubling them.
    if (typeof field !== 'string') return field

    if (field.includes(',') || field.includes('"') || field.includes('\n')) {
      return `"${field.replace(/"/g, '""')}"`
    }
    return field
  }

  return useCallback(() => {
    const getStudentName = (courseParticipationID: string | null) => {
      if (!courseParticipationID) return 'Unassigned'
      const student = developerWithProfiles.find(
        (dev) => dev.participation.courseParticipationID === courseParticipationID,
      )
      return student
        ? `${student.participation.student.firstName} ${student.participation.student.lastName}`
        : 'Unknown'
    }

    const peerGroupMap =
      peerAssignments && peerAssignments.length > 0
        ? buildPeerGroups(peerAssignments)
        : new Map<string, number>()

    const csvContent = [
      [
        'Seat',
        'Seat Mac',
        'Device ID',
        'Assigned Student',
        'Assigned Tutor',
        'Tutor Seat',
        'Matriculation',
        'Peer Group',
        'Student ID',
        'Tutor ID',
      ].join(','),
      ...seats
        .filter((seat) => seat.assignedStudent || seat.isTutorSeat)
        .map((seat) => {
          const peerGroup = seat.assignedStudent
            ? peerGroupMap.get(seat.assignedStudent)
            : undefined
          return [
            escapeCsvField(seat.seatName),
            seat.hasMac ? 'Yes' : 'No',
            escapeCsvField(seat.deviceID || ''),
            escapeCsvField(getStudentName(seat.assignedStudent)),
            escapeCsvField(getTutorName(seat.assignedTutor, tutors)),
            seat.isTutorSeat ? 'Yes' : 'No',
            escapeCsvField(
              seat.assignedStudent
                ? (developerWithProfiles.find(
                    (d) => d.participation.courseParticipationID === seat.assignedStudent,
                  )?.participation.student.matriculationNumber ?? '')
                : '',
            ),
            peerGroup != null ? `P${peerGroup}` : '',
            escapeCsvField(seat.assignedStudent ?? ''),
            escapeCsvField(seat.assignedTutor ?? ''),
          ].join(',')
        }),
    ].join('\n')

    const blob = new Blob([csvContent], { type: 'text/csv;charset=utf-8;' })
    const url = URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.setAttribute('href', url)
    link.setAttribute('download', 'seat_assignments.csv')
    link.style.visibility = 'hidden'
    document.body.appendChild(link)
    link.click()
    document.body.removeChild(link)
    URL.revokeObjectURL(url)
  }, [seats, developerWithProfiles, tutors, peerAssignments])
}
