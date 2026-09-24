import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useCourseStore } from '@tumaet/prompt-shared-state'
import {
  Button,
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
  Input,
} from '@tumaet/prompt-ui-components'
import { useEffect, useRef, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import type { GitlabRepoRequest } from '../../../interfaces/GitlabRepoRequest'
import { createGitlabRepo } from '../../../network/mutations/createGitlabRepo'
import { getGitlabCourseSetup } from '../../../network/queries/getGitlabCourseSetup'
import { getPeerAssignments } from '../../../network/queries/getPeerAssignments'
import { getSeatPlan } from '../../../network/queries/getSeatPlan'
import { gitlabCourseGroup } from '../../../utils/gitlabCourseGroup'
import type { ParticipationWithDevProfiles } from '../interfaces/pariticipationWithDevProfiles'

interface CreateGitlabReposDialogProps {
  participantsWithDevProfiles: ParticipationWithDevProfiles[]
}

export const CreateGitlabReposDialog = ({
  participantsWithDevProfiles,
}: CreateGitlabReposDialogProps) => {
  const [isDialogOpen, setIsDialogOpen] = useState(false)
  const [successCount, setSuccessCount] = useState(0)
  const [errorCount, setErrorCount] = useState(0)
  const [isCreatingRepos, setIsCreatingRepos] = useState(false)
  const stopRequested = useRef(false)
  const [logs, setLogs] = useState<string[]>([])

  // State for storing the user-entered deadline
  const [deadline, setDeadline] = useState('')
  const [demoTested, setDemoTested] = useState(false)

  const { phaseId, courseId } = useParams<{ phaseId: string; courseId: string }>()
  const queryClient = useQueryClient()

  const { courses } = useCourseStore()
  const semesterTag = gitlabCourseGroup(
    courses.find((course) => course.id === courseId)?.semesterTag ?? '',
  )
  const { data: courseSetup, isError: courseSetupError } = useQuery({
    queryKey: ['gitlab-course-setup', phaseId, semesterTag],
    queryFn: () => getGitlabCourseSetup(phaseId ?? '', semesterTag),
    enabled: Boolean(isDialogOpen && phaseId && semesterTag),
  })
  const { data: seats, isError: seatsError } = useQuery({
    queryKey: ['seatPlan', phaseId],
    queryFn: () => getSeatPlan(phaseId ?? ''),
    enabled: Boolean(isDialogOpen && phaseId),
  })
  const { data: peers, isError: peersError } = useQuery({
    queryKey: ['peerAssignments', phaseId],
    queryFn: () => getPeerAssignments(phaseId ?? ''),
    enabled: Boolean(isDialogOpen && phaseId),
  })

  const createGitlabRepoMutation = useMutation({
    mutationFn: ({
      coursePhaseParticipationID,
      createGitlabRepoDTO,
    }: {
      coursePhaseParticipationID: string
      createGitlabRepoDTO: GitlabRepoRequest
    }) => createGitlabRepo(phaseId ?? '', coursePhaseParticipationID, createGitlabRepoDTO),
  })

  const pendingProfiles = participantsWithDevProfiles.filter(
    (participation) => !participation.devProfile && !participation.gitlabStatus?.gitlabSuccess,
  )
  const missingGitlabUsername = participantsWithDevProfiles.filter(
    (participation) =>
      participation.devProfile &&
      !participation.devProfile.gitLabUsername &&
      !participation.gitlabStatus?.gitlabSuccess,
  )
  const missingUniversityLogin = participantsWithDevProfiles.filter(
    (participation) =>
      participation.devProfile?.gitLabUsername &&
      !participation.participation.student.universityLogin &&
      !participation.gitlabStatus?.gitlabSuccess,
  )
  const missingStudentName = participantsWithDevProfiles.filter(
    (participation) =>
      participation.devProfile?.gitLabUsername &&
      participation.participation.student.universityLogin &&
      !`${participation.participation.student.firstName ?? ''} ${participation.participation.student.lastName ?? ''}`.trim() &&
      !participation.gitlabStatus?.gitlabSuccess,
  )
  const participationsReadyForGitlab = participantsWithDevProfiles.filter(
    (participation) =>
      participation.devProfile?.gitLabUsername &&
      participation.participation.student.universityLogin &&
      `${participation.participation.student.firstName ?? ''} ${participation.participation.student.lastName ?? ''}`.trim() &&
      !participation.gitlabStatus?.gitlabSuccess,
  )
  const readyIDs = new Set(
    participationsReadyForGitlab.map((item) => item.participation.courseParticipationID),
  )
  const seatedWithTutor = new Set(
    (seats ?? [])
      .filter((seat) => !seat.isTutorSeat && seat.assignedStudent && seat.assignedTutor)
      .map((seat) => seat.assignedStudent),
  )
  const studentsWithPeers = new Set((peers ?? []).flatMap((peer) => [peer.studentID, peer.peerID]))
  const missingTutorSeats = [...readyIDs].filter((id) => !seatedWithTutor.has(id)).length
  const missingPeerGroups = [...readyIDs].filter((id) => !studentsWithPeers.has(id)).length
  const assignmentsReady =
    Boolean(seats && peers) &&
    !seatsError &&
    !peersError &&
    missingTutorSeats === 0 &&
    missingPeerGroups === 0

  const triggerCreateRepos = async () => {
    stopRequested.current = false
    setIsCreatingRepos(true)
    setLogs([])
    setSuccessCount(0)
    setErrorCount(0)

    for (const participation of participationsReadyForGitlab) {
      if (stopRequested.current) break
      try {
        await createGitlabRepoMutation.mutateAsync({
          coursePhaseParticipationID: participation.participation.courseParticipationID,
          createGitlabRepoDTO: {
            // The server keeps the TUM ID as the stable URL path and uses
            // "Student Name - TUM ID" as the visible GitLab project name.
            repoName: participation.participation.student.universityLogin ?? '',
            studentName:
              `${participation.participation.student.firstName ?? ''} ${participation.participation.student.lastName ?? ''}`.trim(),

            semesterTag,
            submissionDeadline: deadline, // Use the user-entered deadline here
          },
        })
        setSuccessCount((count) => count + 1)
        setLogs((preLogs) => [
          ...preLogs,
          `✅ Created repo for ${participation.devProfile?.gitLabUsername}`,
        ])
      } catch (error) {
        setErrorCount((count) => count + 1)
        setLogs((preLogs) => [
          ...preLogs,
          `❌ Failed to create repo for ${participation.devProfile?.gitLabUsername}: ${error}`,
        ])
      }
    }

    await queryClient.invalidateQueries({ queryKey: ['gitlab_statuses', phaseId] })
    setIsCreatingRepos(false)
  }

  useEffect(() => {
    if (isDialogOpen) {
      setSuccessCount(0)
      setErrorCount(0)
      setLogs([])
      setDeadline('') // Reset the deadline field whenever the dialog opens
      setDemoTested(false)
    }
  }, [isDialogOpen])

  return (
    <Dialog open={isDialogOpen} onOpenChange={setIsDialogOpen}>
      <DialogTrigger asChild>
        <Button>Create Student Repositories</Button>
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Create Student Repositories</DialogTitle>
          <DialogDescription>
            Each ready student receives a repository from the verified teaching material. Seat,
            tutor, and peer assignments must be complete before the batch starts.
          </DialogDescription>
        </DialogHeader>

        <section className='space-y-2 py-4 border-b'>
          <p className='text-sm'>
            Set up and test the demo in{' '}
            <Link className='underline' to='../repository-setup'>
              Repository Setup
            </Link>{' '}
            first.
          </p>
          {courseSetupError ? (
            <p className='text-sm text-destructive'>Could not verify GitLab setup.</p>
          ) : courseSetup?.checks.demoReady ? (
            <p className='text-sm text-green-700'>
              Demo configuration matches current teaching material.
            </p>
          ) : (
            <p className='text-sm text-amber-700'>
              Demo configuration is not ready for student repositories.
            </p>
          )}
          <label className='flex items-start gap-2 text-sm'>
            <input
              type='checkbox'
              checked={demoTested}
              onChange={(event) => setDemoTested(event.target.checked)}
            />
            <span>I tested the demo app, daily issues, merge request, review, and CI.</span>
          </label>
        </section>

        <section className='mt-4'>
          <div className='mb-2'>
            <label htmlFor='deadline' className='block text-sm font-medium'>
              Project Submission Deadline
            </label>
            <p className='text-xs text-muted-foreground'>
              Will be inserted as Submission Deadline in the Readme of the student repository.
            </p>
          </div>
          <Input
            id='deadline'
            placeholder='e.g. 2025-06-01 23:59'
            value={deadline}
            onChange={(e) => setDeadline(e.target.value)}
            className='w-full mb-4'
          />

          <p className='mb-3 text-sm text-muted-foreground'>
            {participationsReadyForGitlab.length} ready to create; {pendingProfiles.length} waiting
            for a developer profile; {missingGitlabUsername.length} missing a GitLab username;{' '}
            {missingUniversityLogin.length} missing a TUM ID; {missingStudentName.length} missing a
            name. Students still completing a profile are skipped for now and can be created in a
            later run.
          </p>
          <p className='mb-3 text-sm'>
            Assignment check: {missingTutorSeats} ready students without a tutor seat;{' '}
            {missingPeerGroups} without a peer group.
            {(seatsError || peersError) && ' Could not load assignments.'}
          </p>
          {pendingProfiles.length > 0 && (
            <p className='mb-3 text-sm'>
              Waiting for profile:{' '}
              {pendingProfiles
                .map(({ participation }) =>
                  `${participation.student.firstName} ${participation.student.lastName}`.trim(),
                )
                .join(', ')}
            </p>
          )}

          <Button
            disabled={
              isCreatingRepos ||
              !courseSetup?.checks.demoReady ||
              !assignmentsReady ||
              !demoTested ||
              !deadline.trim() ||
              participationsReadyForGitlab.length === 0
            }
            onClick={triggerCreateRepos}
          >
            Create Repositories ({participationsReadyForGitlab.length})
          </Button>

          {isCreatingRepos && (
            <Button
              variant='outline'
              className='ml-2'
              onClick={() => {
                stopRequested.current = true
              }}
            >
              Stop after current student
            </Button>
          )}

          <div className='mt-4 max-h-40 overflow-auto border rounded p-2 text-xs'>
            {logs.map((log) => (
              <div key={log}>{log}</div>
            ))}
          </div>

          {(successCount > 0 || errorCount > 0) && (
            <div className='mt-2'>
              <span className='text-green-600'>Success: {successCount}</span>
              <span className='ml-4 text-red-600'>Failed: {errorCount}</span>
            </div>
          )}
        </section>
      </DialogContent>
    </Dialog>
  )
}
