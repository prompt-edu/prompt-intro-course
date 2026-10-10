import { useQuery } from '@tanstack/react-query'
import {
  type CoursePhaseParticipationsWithResolution,
  getCoursePhaseParticipations,
  PassStatus,
} from '@tumaet/prompt-shared-state'
import {
  Button,
  Card,
  CardContent,
  Checkbox,
  ErrorPage,
  ManagementPageHeader,
} from '@tumaet/prompt-ui-components'
import { Grid3X3, Loader2, Table2 } from 'lucide-react'
import { useState } from 'react'
import { useParams } from 'react-router-dom'
import type { DeveloperProfile } from '../../interfaces/DeveloperProfile'
import type { PeerAssignment } from '../../interfaces/PeerAssignment'
import type { Seat } from '../../interfaces/Seat'
import type { Tutor } from '../../interfaces/Tutor'
import { getAllDeveloperProfiles } from '../../network/queries/getAllDeveloperProfiles'
import { getAllTutors } from '../../network/queries/getAllTutors'
import { getPeerAssignments } from '../../network/queries/getPeerAssignments'
import { getSeatPlan } from '../../network/queries/getSeatPlan'
import { SeatGrid } from './components/SeatGrid/SeatGrid'
import { SeatMacAssigner } from './components/SeatMacAssigner'
import { SeatStudentAssigner } from './components/SeatStudentAssigner/SeatStudentAssigner'
import { SeatTutorAssigner } from './components/SeatTutorAssigner/SeatTutorAssigner'
import { SeatUploader } from './components/SeatUploader/SeatUploader'
import { useGetParticipationsWithDevProfile } from './hooks/useGetParticipationWithDevProfile'

export const SeatAssignmentPage = () => {
  const { phaseId } = useParams<{ phaseId: string }>()
  const [viewMode, setViewMode] = useState<'table' | 'grid'>('grid')
  const [excludedStudentIDs, setExcludedStudentIDs] = useState<Set<string>>(new Set())

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
    coursePhaseParticipations?.participations.filter(
      (participation) => participation.passStatus !== PassStatus.FAILED,
    ) || [],
    developerProfiles || [],
  )
  const seatCandidates = developerWithProfiles.filter(
    (dev) => !excludedStudentIDs.has(dev.participation.courseParticipationID),
  )

  if (isPending) {
    return (
      <div className='flex justify-center items-center grow'>
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
      <Card>
        <CardContent className='pt-6'>
          <details>
            <summary className='cursor-pointer font-medium'>
              Seat roster: {seatCandidates.length} of {developerWithProfiles.length} participants
              included
            </summary>
            <p className='my-3 text-sm text-muted-foreground'>
              Uncheck anyone who should not receive a seat in this assignment run. This selection
              resets when the page reloads; saved seat assignments remain shared.
            </p>
            <div className='max-h-64 overflow-y-auto grid gap-2 sm:grid-cols-2'>
              {developerWithProfiles.map((dev) => {
                const id = dev.participation.courseParticipationID
                const student = dev.participation.student
                return (
                  <div key={id} className='flex items-center gap-2 text-sm'>
                    <Checkbox
                      id={`seat-roster-${id}`}
                      checked={!excludedStudentIDs.has(id)}
                      onCheckedChange={(checked) =>
                        setExcludedStudentIDs((current) => {
                          const next = new Set(current)
                          if (checked === true) next.delete(id)
                          else next.add(id)
                          return next
                        })
                      }
                    />
                    <label htmlFor={`seat-roster-${id}`} className='cursor-pointer'>
                      {student.firstName} {student.lastName}
                      {!dev.profile && <span className='text-destructive'> (profile missing)</span>}
                    </label>
                  </div>
                )
              })}
            </div>
          </details>
        </CardContent>
      </Card>
      <SeatUploader existingSeats={seats || []} />
      {seats.length > 0 && <SeatMacAssigner existingSeats={seats} />}
      {seats.length > 0 && (
        <SeatTutorAssigner
          seats={seats}
          tutors={tutors || []}
          numberOfStudents={seatCandidates.length}
        />
      )}
      {seats.length > 0 && (
        <SeatStudentAssigner
          seats={seats}
          developerWithProfiles={seatCandidates}
          tutors={tutors}
          peerAssignments={peerAssignments}
        />
      )}
      {seats.length > 0 && seats.some((s) => s.assignedStudent) && (
        <Card>
          <CardContent className='pt-6'>
            <div className='flex items-center gap-2 mb-4'>
              <Button
                variant={viewMode === 'table' ? 'default' : 'outline-solid'}
                size='sm'
                onClick={() => setViewMode('table')}
              >
                <Table2 className='mr-1.5 h-4 w-4' />
                Table
              </Button>
              <Button
                variant={viewMode === 'grid' ? 'default' : 'outline-solid'}
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
