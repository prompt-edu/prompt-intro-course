import { useMutation, useQueryClient } from '@tanstack/react-query'
import {
  useCourseStore,
  useGetCoursePhase,
  useModifyCoursePhase,
} from '@tumaet/prompt-shared-state'
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
import { AlertCircle, CheckCircle, Loader2 } from 'lucide-react'
import { useEffect, useRef, useState } from 'react'
import { useParams } from 'react-router-dom'
import type { GitlabRepoRequest } from '../../../interfaces/GitlabRepoRequest'
import { createGitlabRepo } from '../../../network/mutations/createGitlabRepo'
import { createIntroCourseGitlabInfrastructure } from '../../../network/mutations/createIntroCourseGitlabInfrastructure'
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

  const { phaseId, courseId } = useParams<{ phaseId: string; courseId: string }>()
  const queryClient = useQueryClient()

  const { courses } = useCourseStore()
  const semesterTag = courses.find((course) => course.id === courseId)?.semesterTag ?? ''

  const { data: coursePhase, isPending, isError } = useGetCoursePhase()
  const { mutate: mutateCoursePhase } = useModifyCoursePhase(
    () => queryClient.invalidateQueries({ queryKey: ['course_phase', phaseId] }),
    () => setLogs((prev) => [...prev, `❌ Failed to update course phase`]),
  )

  const infraStructureExists: boolean =
    coursePhase?.restrictedData?.gitLabInfrastructureSetup ?? false

  const createGitlabRepoMutation = useMutation({
    mutationFn: ({
      coursePhaseParticipationID,
      createGitlabRepoDTO,
    }: {
      coursePhaseParticipationID: string
      createGitlabRepoDTO: GitlabRepoRequest
    }) => createGitlabRepo(phaseId ?? '', coursePhaseParticipationID, createGitlabRepoDTO),
  })

  const createInfrastructureSetup = useMutation({
    mutationFn: () => createIntroCourseGitlabInfrastructure(phaseId ?? '', { semesterTag }),
    onSuccess: () =>
      mutateCoursePhase({
        id: phaseId ?? '',
        restrictedData: { gitLabInfrastructureSetup: true },
      }),
    onError: (error) => setLogs((prev) => [...prev, `❌ Infrastructure setup error: ${error}`]),
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
    }
  }, [isDialogOpen])

  if (isPending) {
    return (
      <div className='flex justify-center items-center h-64'>
        <Loader2 className='h-12 w-12 animate-spin text-primary' />
      </div>
    )
  }

  if (isError) {
    return (
      <div className='flex justify-center items-center h-64 text-red-600'>
        <AlertCircle className='h-12 w-12 mr-2' /> Failed to load course phase data.
      </div>
    )
  }

  return (
    <Dialog open={isDialogOpen} onOpenChange={setIsDialogOpen}>
      <DialogTrigger asChild>
        <Button>Create Gitlab Repositories</Button>
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Create Gitlab Repositories</DialogTitle>
          <DialogDescription>
            Infrastructure setup and repo creation for students.
            <br />
            <strong>Important: </strong>Make sure that every student has a tutor assigned!
          </DialogDescription>
        </DialogHeader>

        <section className='flex items-center justify-between py-4 border-b'>
          <span>Create Gitlab Course Group</span>
          <div className='flex items-center gap-2'>
            {infraStructureExists && <CheckCircle className='text-green-500' />}
            <Button
              disabled={createInfrastructureSetup.isPending || isCreatingRepos}
              onClick={() => createInfrastructureSetup.mutate()}
            >
              {createInfrastructureSetup.isPending
                ? 'Checking Infrastructure...'
                : infraStructureExists
                  ? 'Check and repair infrastructure'
                  : 'Create infrastructure'}
            </Button>
          </div>
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
              !infraStructureExists ||
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
