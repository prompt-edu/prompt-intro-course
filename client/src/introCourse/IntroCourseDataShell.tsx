import { useCourseStore } from '@tumaet/prompt-shared-state'
import { useParams } from 'react-router-dom'
import { Loader2, TriangleAlert } from 'lucide-react'
import { useQuery } from '@tanstack/react-query'
import {
  CoursePhaseParticipationWithStudent,
  getOwnCoursePhaseParticipation,
} from '@tumaet/prompt-shared-state'
import { useEffect, useState } from 'react'
import { useIntroCourseStore } from './zustand/useIntroCourseStore'
import { getOwnDeveloperProfile } from './network/queries/getOwnDeveloperProfile'
import { DeveloperProfile } from './interfaces/DeveloperProfile'
import { OwnPeerAssignment } from './interfaces/PeerAssignment'
import {
  Alert,
  AlertDescription,
  AlertTitle,
  ErrorPage,
  UnauthorizedPage,
} from '@tumaet/prompt-ui-components'
import { SeatAssignment } from './pages/SeatAssignment/interfaces/SeatAssignment'
import { getOwnSeatPlanAssignment } from './network/queries/getOwnSeatPlanAssignment'
import { getOwnPeerAssignment } from './network/queries/getOwnPeerAssignment'

interface IntroCourseDataShellProps {
  children: React.ReactNode
}

export const IntroCourseDataShell = ({ children }: IntroCourseDataShellProps) => {
  const { isStudentOfCourse } = useCourseStore()
  const { courseId, phaseId } = useParams<{ courseId: string; phaseId: string }>()
  const isStudent = isStudentOfCourse(courseId ?? '')
  const { setCoursePhaseParticipation, setDeveloperProfile, setSeatAssignment, setPeerAssignment } =
    useIntroCourseStore()

  const [devProfileSet, setDevProfileSet] = useState(false)
  const [participationSet, setParticipationSet] = useState(false)
  const [seatAssignmentSet, setSeatAssignmentSet] = useState(false)
  const [peerAssignmentSet, setPeerAssignmentSet] = useState(false)

  // getting the course phase participation
  const {
    data: fetchedParticipation,
    error,
    isPending: isParticipationPending,
    isError: isParticipationError,
    refetch: refetchParticipation,
  } = useQuery<CoursePhaseParticipationWithStudent>({
    queryKey: ['course_phase_participation', phaseId],
    queryFn: () => getOwnCoursePhaseParticipation(phaseId ?? ''),
    enabled: isStudent,
  })

  // trying to get the developerProfile
  const {
    data: fetchedProfile,
    isPending: isProfilePending,
    isError: isProfileError,
    refetch: refetchProfile,
  } = useQuery<DeveloperProfile>({
    queryKey: ['developer_profile'],
    queryFn: () => getOwnDeveloperProfile(phaseId ?? ''),
    enabled: isStudent,
  })

  const {
    data: fetchedSeatAssignment,
    isPending: isSeatAssignmentPending,
    isError: isSeatAssignmentError,
    refetch: refetchSeatAssignment,
  } = useQuery<SeatAssignment>({
    queryKey: ['seat_assignment', phaseId],
    queryFn: () => getOwnSeatPlanAssignment(phaseId ?? ''),
    enabled: isStudent,
  })

  const {
    data: fetchedPeerAssignment,
    isPending: isPeerAssignmentPending,
    isError: isPeerAssignmentError,
    refetch: refetchPeerAssignment,
  } = useQuery<OwnPeerAssignment>({
    queryKey: ['peer_assignment', phaseId],
    queryFn: () => getOwnPeerAssignment(phaseId ?? ''),
    enabled: isStudent,
  })

  const isPending = isStudent
    ? isParticipationPending ||
      isProfilePending ||
      !devProfileSet ||
      !participationSet ||
      isSeatAssignmentPending ||
      !seatAssignmentSet ||
      isPeerAssignmentPending ||
      !peerAssignmentSet
    : false
  const isError =
    isParticipationError || isProfileError || isSeatAssignmentError || isPeerAssignmentError

  useEffect(() => {
    if (fetchedParticipation) {
      setCoursePhaseParticipation(fetchedParticipation)
      setParticipationSet(true)
    }
  }, [fetchedParticipation, setCoursePhaseParticipation])

  useEffect(() => {
    if (fetchedProfile) {
      if (fetchedProfile.appleID === '' && fetchedProfile.gitLabUsername === '') {
        setDeveloperProfile(undefined)
      } else {
        setDeveloperProfile(fetchedProfile)
      }
      setDevProfileSet(true)
    }
  }, [fetchedProfile, setDeveloperProfile])

  useEffect(() => {
    if (fetchedSeatAssignment && fetchedSeatAssignment.seatName !== '') {
      setSeatAssignment(fetchedSeatAssignment)
      setSeatAssignmentSet(true)
    } else if (fetchedSeatAssignment && fetchedSeatAssignment.seatName === '') {
      setSeatAssignment(undefined)
      setSeatAssignmentSet(true)
    }
  }, [fetchedSeatAssignment, setSeatAssignment])

  useEffect(() => {
    if (fetchedPeerAssignment) {
      if (
        fetchedPeerAssignment.peersIReview?.length > 0 ||
        fetchedPeerAssignment.peersWhoReviewMe?.length > 0
      ) {
        setPeerAssignment(fetchedPeerAssignment)
      } else {
        setPeerAssignment(undefined)
      }
      setPeerAssignmentSet(true)
    } else if (fetchedPeerAssignment === undefined && !isPeerAssignmentPending) {
      setPeerAssignment(undefined)
      setPeerAssignmentSet(true)
    }
  }, [fetchedPeerAssignment, isPeerAssignmentPending, setPeerAssignment])

  // if he is not a student -> we do not wait for the participation
  if (isStudent && isPending) {
    return (
      <div className='flex justify-center items-center h-64'>
        <Loader2 className='h-12 w-12 animate-spin text-primary' />
      </div>
    )
  }

  // Data only relevant for students - not for lecturers
  if (isStudent && isError) {
    // if the participation is not found, we show the unauthorized page bc then the student has not yet processed to this phase
    if (isParticipationError && error.message.includes('404')) {
      return <UnauthorizedPage backUrl={`/management/course/${courseId}`} />
    } else {
      return (
        <ErrorPage
          onRetry={() => {
            refetchProfile()
            refetchParticipation()
            refetchSeatAssignment()
            refetchPeerAssignment()
          }}
        />
      )
    }
  }

  return (
    <>
      {!isStudent && (
        <Alert>
          <TriangleAlert className='h-4 w-4' />
          <AlertTitle>Your are not a student of this course.</AlertTitle>
          <AlertDescription>
            The following components are disabled because you are not a student of this course. For
            configuring this view, please refer to the Intro Course in the Tutor Course.
          </AlertDescription>
        </Alert>
      )}
      {children}
    </>
  )
}
