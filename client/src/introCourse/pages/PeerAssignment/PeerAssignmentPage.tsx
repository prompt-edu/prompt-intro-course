import { ManagementPageHeader, ErrorPage } from '@tumaet/prompt-ui-components'
import { useQuery } from '@tanstack/react-query'
import { Loader2 } from 'lucide-react'
import { useParams } from 'react-router-dom'
import { Tutor } from '../../interfaces/Tutor'
import { Seat } from '../../interfaces/Seat'
import { DeveloperProfile } from '../../interfaces/DeveloperProfile'
import { PeerAssignment } from '../../interfaces/PeerAssignment'
import { getAllTutors } from '../../network/queries/getAllTutors'
import { getSeatPlan } from '../../network/queries/getSeatPlan'
import { getAllDeveloperProfiles } from '../../network/queries/getAllDeveloperProfiles'
import { getPeerAssignments } from '../../network/queries/getPeerAssignments'
import { getCoursePhaseParticipations } from '@/network/queries/getCoursePhaseParticipations'
import { CoursePhaseParticipationsWithResolution } from '@tumaet/prompt-shared-state'
import { PeerAssignmentActions } from './components/PeerAssignmentActions'
import { PeerGroupList } from './components/PeerGroupList'

export const PeerAssignmentPage = () => {
  const { phaseId } = useParams<{ phaseId: string }>()

  const {
    data: tutors,
    isPending: isPendingTutors,
    isError: isTutorsError,
    refetch: refetchTutors,
  } = useQuery<Tutor[]>({
    queryKey: ['tutors', phaseId],
    queryFn: () => getAllTutors(phaseId ?? ''),
  })

  const {
    data: seats,
    isPending: isSeatsPending,
    isError: isSeatsError,
    refetch: refetchSeats,
  } = useQuery<Seat[]>({
    queryKey: ['seatPlan', phaseId],
    queryFn: () => getSeatPlan(phaseId ?? ''),
  })

  const {
    data: developerProfiles,
    isPending: isProfilesPending,
    isError: isProfilesError,
    refetch: refetchProfiles,
  } = useQuery<DeveloperProfile[]>({
    queryKey: ['developerProfiles', phaseId],
    queryFn: () => getAllDeveloperProfiles(phaseId ?? ''),
  })

  const {
    data: peerAssignments,
    isPending: isPeersPending,
    isError: isPeersError,
    refetch: refetchPeers,
  } = useQuery<PeerAssignment[]>({
    queryKey: ['peerAssignments', phaseId],
    queryFn: () => getPeerAssignments(phaseId ?? ''),
  })

  const {
    data: participations,
    isPending: isParticipationsPending,
    isError: isParticipationsError,
    refetch: refetchParticipations,
  } = useQuery<CoursePhaseParticipationsWithResolution>({
    queryKey: ['participants', phaseId],
    queryFn: () => getCoursePhaseParticipations(phaseId ?? ''),
  })

  const isPending =
    isPendingTutors || isSeatsPending || isProfilesPending || isPeersPending || isParticipationsPending
  const isError =
    isTutorsError || isSeatsError || isProfilesError || isPeersError || isParticipationsError

  if (isPending) {
    return (
      <div className='flex justify-center items-center flex-grow'>
        <Loader2 className='h-12 w-12 animate-spin text-primary' />
      </div>
    )
  }

  if (isError) {
    return (
      <ErrorPage
        onRetry={() => {
          refetchTutors()
          refetchSeats()
          refetchProfiles()
          refetchPeers()
          refetchParticipations()
        }}
      />
    )
  }

  // Count assigned students
  const assignedStudents = (seats ?? []).filter((s) => s.assignedStudent).length

  return (
    <div className='space-y-6'>
      <ManagementPageHeader>Peer Assignments</ManagementPageHeader>
      <PeerAssignmentActions
        peerAssignments={peerAssignments ?? []}
        totalStudents={assignedStudents}
      />
      <PeerGroupList
        peerAssignments={peerAssignments ?? []}
        seats={seats ?? []}
        tutors={tutors ?? []}
        developerProfiles={developerProfiles ?? []}
        participations={participations?.participations ?? []}
      />
    </div>
  )
}
