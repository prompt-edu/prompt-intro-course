import type { CoursePhaseParticipationWithStudent } from '@tumaet/prompt-shared-state'
import { create } from 'zustand'
import type { DeveloperProfile } from '../interfaces/DeveloperProfile'
import type { OwnPeerAssignment } from '../interfaces/PeerAssignment'
import type { SeatAssignment } from '../pages/SeatAssignment/interfaces/SeatAssignment'

interface IntroCourseStoreState {
  developerProfile?: DeveloperProfile
  coursePhaseParticipation?: CoursePhaseParticipationWithStudent
  seatAssignment?: SeatAssignment
  peerAssignment?: OwnPeerAssignment
}

interface IntroCourseStoreAction {
  setDeveloperProfile: (developerProfile?: DeveloperProfile) => void
  setCoursePhaseParticipation: (
    coursePhaseParticipation: CoursePhaseParticipationWithStudent,
  ) => void
  setSeatAssignment: (seatAssignment?: SeatAssignment) => void
  setPeerAssignment: (peerAssignment?: OwnPeerAssignment) => void
}

export const useIntroCourseStore = create<IntroCourseStoreState & IntroCourseStoreAction>(
  (set) => ({
    developerProfile: undefined,
    coursePhaseParticipation: undefined,
    seatAssignment: undefined,
    peerAssignment: undefined,
    setDeveloperProfile: (developerProfile) => set({ developerProfile }),
    setCoursePhaseParticipation: (coursePhaseParticipation) => set({ coursePhaseParticipation }),
    setSeatAssignment: (seatAssignment) => set({ seatAssignment }),
    setPeerAssignment: (peerAssignment) => set({ peerAssignment }),
  }),
)
