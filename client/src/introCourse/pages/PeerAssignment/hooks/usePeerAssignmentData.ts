import { useQuery } from '@tanstack/react-query'
import {
  type CoursePhaseParticipationsWithResolution,
  getCoursePhaseParticipations,
} from '@tumaet/prompt-shared-state'
import type { DeveloperProfile } from '../../../interfaces/DeveloperProfile'
import type { PeerAssignment } from '../../../interfaces/PeerAssignment'
import type { Seat } from '../../../interfaces/Seat'
import type { Tutor } from '../../../interfaces/Tutor'
import { getAllDeveloperProfiles } from '../../../network/queries/getAllDeveloperProfiles'
import { getAllTutors } from '../../../network/queries/getAllTutors'
import { getPeerAssignments } from '../../../network/queries/getPeerAssignments'
import { getSeatPlan } from '../../../network/queries/getSeatPlan'

export const usePeerAssignmentData = (phaseId: string | undefined) => {
  const {
    data: tutors,
    isPending: isPendingTutors,
    isError: isTutorsError,
    refetch: refetchTutors,
  } = useQuery<Tutor[]>({
    queryKey: ['tutors', phaseId],
    queryFn: () => getAllTutors(phaseId ?? ''),
    enabled: !!phaseId,
  })

  const {
    data: seats,
    isPending: isSeatsPending,
    isError: isSeatsError,
    refetch: refetchSeats,
  } = useQuery<Seat[]>({
    queryKey: ['seatPlan', phaseId],
    queryFn: () => getSeatPlan(phaseId ?? ''),
    enabled: !!phaseId,
  })

  const {
    data: developerProfiles,
    isPending: isProfilesPending,
    isError: isProfilesError,
    refetch: refetchProfiles,
  } = useQuery<DeveloperProfile[]>({
    queryKey: ['developerProfiles', phaseId],
    queryFn: () => getAllDeveloperProfiles(phaseId ?? ''),
    enabled: !!phaseId,
  })

  const {
    data: peerAssignments,
    isPending: isPeersPending,
    isError: isPeersError,
    refetch: refetchPeers,
  } = useQuery<PeerAssignment[]>({
    queryKey: ['peerAssignments', phaseId],
    queryFn: () => getPeerAssignments(phaseId ?? ''),
    enabled: !!phaseId,
  })

  const {
    data: participations,
    isPending: isParticipationsPending,
    isError: isParticipationsError,
    refetch: refetchParticipations,
  } = useQuery<CoursePhaseParticipationsWithResolution>({
    queryKey: ['participants', phaseId],
    queryFn: () => getCoursePhaseParticipations(phaseId ?? ''),
    enabled: !!phaseId,
  })

  const isPending =
    isPendingTutors ||
    isSeatsPending ||
    isProfilesPending ||
    isPeersPending ||
    isParticipationsPending
  const isError =
    isTutorsError || isSeatsError || isProfilesError || isPeersError || isParticipationsError

  const refetchAll = () => {
    refetchTutors()
    refetchSeats()
    refetchProfiles()
    refetchPeers()
    refetchParticipations()
  }

  return {
    tutors,
    seats,
    developerProfiles,
    peerAssignments,
    participations,
    isPending,
    isError,
    refetchAll,
  }
}
