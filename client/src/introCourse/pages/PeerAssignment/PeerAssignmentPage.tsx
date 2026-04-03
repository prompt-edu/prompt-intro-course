import { ManagementPageHeader, ErrorPage } from '@tumaet/prompt-ui-components'
import { Loader2 } from 'lucide-react'
import { useParams } from 'react-router-dom'
import { PeerAssignmentActions } from './components/PeerAssignmentActions'
import { PeerGroupList } from './components/PeerGroupList'
import { usePeerAssignmentData } from './hooks/usePeerAssignmentData'

export const PeerAssignmentPage = () => {
  const { phaseId } = useParams<{ phaseId: string }>()

  const {
    tutors,
    seats,
    developerProfiles,
    peerAssignments,
    participations,
    isPending,
    isError,
    refetchAll,
  } = usePeerAssignmentData(phaseId)

  if (isPending) {
    return (
      <div className='flex justify-center items-center flex-grow'>
        <Loader2 className='h-12 w-12 animate-spin text-primary' />
      </div>
    )
  }

  if (isError) {
    return <ErrorPage onRetry={refetchAll} />
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
