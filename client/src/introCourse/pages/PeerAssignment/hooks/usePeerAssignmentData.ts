import { useQuery } from '@tanstack/react-query'
import { Tutor } from '../../../interfaces/Tutor'
import { Seat } from '../../../interfaces/Seat'
import { DeveloperProfile } from '../../../interfaces/DeveloperProfile'
import { PeerAssignment } from '../../../interfaces/PeerAssignment'
import { getAllTutors } from '../../../network/queries/getAllTutors'
import { getSeatPlan } from '../../../network/queries/getSeatPlan'
import { getAllDeveloperProfiles } from '../../../network/queries/getAllDeveloperProfiles'
import { getPeerAssignments } from '../../../network/queries/getPeerAssignments'
import {
  CoursePhaseParticipationsWithResolution,
  getCoursePhaseParticipations,
} from '@tumaet/prompt-shared-state'

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
    isPendingTutors || isSeatsPending || isProfilesPending || isPeersPending || isParticipationsPending
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
