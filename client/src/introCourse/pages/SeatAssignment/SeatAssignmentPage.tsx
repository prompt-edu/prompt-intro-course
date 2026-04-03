import { useState } from 'react'
import { ManagementPageHeader, ErrorPage, Button, Card, CardContent } from '@tumaet/prompt-ui-components'
import { getCoursePhaseParticipations } from '@/network/queries/getCoursePhaseParticipations'
import { useQuery } from '@tanstack/react-query'
import { CoursePhaseParticipationsWithResolution } from '@tumaet/prompt-shared-state'
import { Grid3X3, Loader2, Table2 } from 'lucide-react'
import { useParams } from 'react-router-dom'
import { DeveloperProfile } from '../../interfaces/DeveloperProfile'
import { Tutor } from '../../interfaces/Tutor'
import { PeerAssignment } from '../../interfaces/PeerAssignment'
import { getAllDeveloperProfiles } from '../../network/queries/getAllDeveloperProfiles'
import { getAllTutors } from '../../network/queries/getAllTutors'
import { getSeatPlan } from '../../network/queries/getSeatPlan'
import { getPeerAssignments } from '../../network/queries/getPeerAssignments'
import { Seat } from '../../interfaces/Seat'
import { SeatUploader } from './components/SeatUploader/SeatUploader'
import { SeatMacAssigner } from './components/SeatMacAssigner'
import { SeatTutorAssigner } from './components/SeatTutorAssigner/SeatTutorAssigner'
import { SeatStudentAssigner } from './components/SeatStudentAssigner/SeatStudentAssigner'
import { SeatGrid } from './components/SeatGrid/SeatGrid'
import { useGetParticipationsWithDevProfile } from './hooks/useGetParticipationWithDevProfile'

export const SeatAssignmentPage = () => {
  const { phaseId } = useParams<{ phaseId: string }>()
  const [viewMode, setViewMode] = useState<'table' | 'grid'>('grid')

  // Data fetching
  const {
    data: tutors,
    isPending: isPendingTutors,
    isError: isTutorsLoadingError,
    refetch: refetchTutors,
  } = useQuery<Tutor[]>({
    queryKey: ['tutors', phaseId],
    queryFn: () => getAllTutors(phaseId ?? ''),
    enabled: !!phaseId,
  })

  const {
    data: coursePhaseParticipations,
    isPending: isCoursePhaseParticipationsPending,
    isError: isParticipationsError,
    refetch: refetchCoursePhaseParticipations,
  } = useQuery<CoursePhaseParticipationsWithResolution>({
    queryKey: ['participants', phaseId],
    queryFn: () => getCoursePhaseParticipations(phaseId ?? ''),
    enabled: !!phaseId,
  })

  const {
    data: developerProfiles,
    isPending: isDeveloperProfilesPending,
    isError: isDeveloperProfileError,
    refetch: refetchDeveloperProfiles,
  } = useQuery<DeveloperProfile[]>({
    queryKey: ['developerProfiles', phaseId],
    queryFn: () => getAllDeveloperProfiles(phaseId ?? ''),
    enabled: !!phaseId,
  })

  const {
    data: seats,
    isPending: isSeatPlanLoading,
    isError: isSeatPlanError,
    refetch: refetchSeatPlan,
  } = useQuery<Seat[]>({
    queryKey: ['seatPlan', phaseId],
    queryFn: () => getSeatPlan(phaseId ?? ''),
    enabled: !!phaseId,
  })

  const { data: peerAssignments } = useQuery<PeerAssignment[]>({
    queryKey: ['peerAssignments', phaseId],
    queryFn: () => getPeerAssignments(phaseId ?? ''),
    enabled: !!phaseId,
  })

  const isPending =
    isCoursePhaseParticipationsPending ||
    isDeveloperProfilesPending ||
    isPendingTutors ||
    isSeatPlanLoading
  const isError =
    isParticipationsError || isDeveloperProfileError || isTutorsLoadingError || isSeatPlanError

  const developerWithProfiles = useGetParticipationsWithDevProfile(
    coursePhaseParticipations?.participations || [],
    developerProfiles || [],
  )

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
          refetchCoursePhaseParticipations()
          refetchDeveloperProfiles()
          refetchTutors()
          refetchSeatPlan()
        }}
      />
    )
  }

  return (
    <div className='space-y-6'>
      <ManagementPageHeader>Seat Assignment</ManagementPageHeader>
      <SeatUploader existingSeats={seats || []} />
      {seats.length > 0 && <SeatMacAssigner existingSeats={seats} />}
      {seats.length > 0 && (
        <SeatTutorAssigner
          seats={seats}
          tutors={tutors || []}
          numberOfStudents={developerWithProfiles.length}
        />
      )}
      {seats.length > 0 && (
        <SeatStudentAssigner
          seats={seats}
          developerWithProfiles={developerWithProfiles}
          tutors={tutors}
          peerAssignments={peerAssignments}
        />
      )}
      {seats.length > 0 && seats.some((s) => s.assignedStudent) && (
        <Card>
          <CardContent className='pt-6'>
            <div className='flex items-center gap-2 mb-4'>
              <Button
                variant={viewMode === 'table' ? 'default' : 'outline'}
                size='sm'
                onClick={() => setViewMode('table')}
              >
                <Table2 className='mr-1.5 h-4 w-4' />
                Table
              </Button>
              <Button
                variant={viewMode === 'grid' ? 'default' : 'outline'}
                size='sm'
                onClick={() => setViewMode('grid')}
              >
                <Grid3X3 className='mr-1.5 h-4 w-4' />
                Grid
              </Button>
            </div>
            {viewMode === 'grid' && (
              <SeatGrid
                seats={seats}
                tutors={tutors || []}
                participations={coursePhaseParticipations?.participations || []}
                peerAssignments={peerAssignments}
              />
            )}
          </CardContent>
        </Card>
      )}
    </div>
  )
}
