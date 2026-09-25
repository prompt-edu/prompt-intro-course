import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useCourseStore, useModifyCoursePhase } from '@tumaet/prompt-shared-state'
import {
  Button,
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  Input,
  ManagementPageHeader,
} from '@tumaet/prompt-ui-components'
import { isAxiosError } from 'axios'
import { ExternalLink, Loader2, RefreshCw } from 'lucide-react'
import { useState } from 'react'
import { useParams } from 'react-router-dom'
import type { GitlabResource } from '../../interfaces/GitlabCourseSetupStatus'
import { createIntroCourseGitlabInfrastructure } from '../../network/mutations/createIntroCourseGitlabInfrastructure'
import { resetGitlabDemo } from '../../network/mutations/resetGitlabDemo'
import { getGitlabCourseSetup } from '../../network/queries/getGitlabCourseSetup'
import { gitlabCourseGroup } from '../../utils/gitlabCourseGroup'

const resetConfirmation = 'RESET DEMO'

const errorMessage = (error: unknown): string => {
  if (typeof error === 'string') return error
  if (isAxiosError(error) && typeof error.response?.data?.error === 'string') {
    return error.response.data.error
  }
  return error instanceof Error ? error.message : 'The request failed.'
}

const Resource = ({ resource, empty }: { resource: GitlabResource | null; empty: string }) =>
  resource ? (
    <a
      className='inline-flex items-center gap-1 text-primary underline underline-offset-2'
      href={resource.url}
      target='_blank'
      rel='noopener noreferrer'
    >
      Open in GitLab <ExternalLink className='h-3 w-3' />
    </a>
  ) : (
    <span className='text-muted-foreground'>{empty}</span>
  )

const Check = ({ ok, label }: { ok: boolean; label: string }) => (
  <li className='flex items-center gap-2'>
    <span aria-hidden='true' className={ok ? 'text-green-700' : 'text-amber-700'}>
      {ok ? '●' : '○'}
    </span>
    {label}
  </li>
)

export const RepositorySetupPage = () => {
  const { courseId, phaseId } = useParams<{ courseId: string; phaseId: string }>()
  const { courses } = useCourseStore()
  const semesterTag = gitlabCourseGroup(
    courses.find((course) => course.id === courseId)?.semesterTag ?? '',
  )
  const queryClient = useQueryClient()
  const [notice, setNotice] = useState<string | null>(null)
  const [actionError, setActionError] = useState<string | null>(null)
  const [resetOpen, setResetOpen] = useState(false)
  const [typedConfirmation, setTypedConfirmation] = useState('')

  const queryKey = ['gitlab-course-setup', phaseId, semesterTag]
  const { data, isPending, isError, refetch, isFetching } = useQuery({
    queryKey,
    queryFn: () => getGitlabCourseSetup(phaseId ?? '', semesterTag),
    enabled: Boolean(phaseId && semesterTag),
  })

  const { mutate: markInfrastructureSetup } = useModifyCoursePhase(
    () => queryClient.invalidateQueries({ queryKey: ['course_phase', phaseId] }),
    () => setActionError('GitLab setup succeeded, but PROMPT could not save its setup status.'),
  )

  const setup = useMutation({
    mutationFn: () => createIntroCourseGitlabInfrastructure(phaseId ?? '', { semesterTag }),
    onSuccess: async () => {
      markInfrastructureSetup({
        id: phaseId ?? '',
        restrictedData: { gitLabInfrastructureSetup: true },
      })
      await queryClient.invalidateQueries({ queryKey })
      setNotice('Course infrastructure checked. Review the current status below.')
      setActionError(null)
    },
    onError: (error) => {
      setNotice(null)
      setActionError(errorMessage(error))
    },
  })

  const reset = useMutation({
    mutationFn: () => {
      if (!data?.demoProject || !data.source) {
        throw new Error('Refresh the demo status before resetting it.')
      }
      return resetGitlabDemo(phaseId ?? '', {
        semesterTag,
        expectedProjectID: data.demoProject.id,
        expectedSourceSHA: data.source.sha,
      })
    },
    onSuccess: async (result) => {
      setResetOpen(false)
      setTypedConfirmation('')
      await queryClient.invalidateQueries({ queryKey })
      setNotice(
        `Demo reset in place at ${result.demoUrl}. Check its files, daily issues, practice branches, access, and pipeline before use. Old merge requests remain in GitLab history.`,
      )
      setActionError(null)
    },
    onError: (error) => {
      setNotice(null)
      setActionError(errorMessage(error))
    },
  })

  if (!phaseId || !semesterTag) {
    return (
      <div className='space-y-4'>
        <ManagementPageHeader>Repository Setup</ManagementPageHeader>
        <p>The course phase or semester tag is missing.</p>
      </div>
    )
  }

  if (isPending) {
    return (
      <div className='flex justify-center items-center grow'>
        <Loader2 className='h-12 w-12 animate-spin text-primary' />
      </div>
    )
  }

  if (isError) {
    return (
      <div className='space-y-4'>
        <ManagementPageHeader>Repository Setup</ManagementPageHeader>
        <p>Could not load the GitLab infrastructure status.</p>
        <Button variant='outline' onClick={() => refetch()}>
          Retry
        </Button>
      </div>
    )
  }

  return (
    <div className='space-y-6'>
      <div className='flex flex-wrap items-start justify-between gap-4'>
        <div>
          <ManagementPageHeader>Repository Setup</ManagementPageHeader>
          <p className='mt-2 text-muted-foreground'>
            Import tutors and their GitLab usernames, then set up the course groups and demo. After
            a material update, repair shared CI and reset the demo to load the new files. Test it,
            then reset again to leave a clean instructor copy. Assign seats and peers before
            initializing student repositories.
          </p>
        </div>
        <Button variant='outline' onClick={() => refetch()} disabled={isFetching}>
          {isFetching ? (
            <Loader2 className='mr-2 h-4 w-4 animate-spin' />
          ) : (
            <RefreshCw className='mr-2 h-4 w-4' />
          )}
          Refresh status
        </Button>
      </div>

      {notice && <p className='rounded-md border border-green-300 p-3 text-sm'>{notice}</p>}
      {actionError && (
        <p role='alert' className='rounded-md border border-red-300 p-3 text-sm text-red-700'>
          {actionError}
        </p>
      )}

      <section className='rounded-md border p-5 space-y-4' aria-labelledby='material-heading'>
        <div>
          <h2 id='material-heading' className='text-lg font-semibold'>
            1. Teaching material
          </h2>
          <p className='text-sm text-muted-foreground'>
            Source revision used to check this course:{' '}
            {data?.source?.sha?.slice(0, 12) || 'Unavailable'}
          </p>
        </div>
        {data?.source?.url ? (
          <a
            className='inline-flex items-center gap-1 text-primary underline underline-offset-2'
            href={data.source.url}
            target='_blank'
            rel='noopener noreferrer'
          >
            Open teaching material <ExternalLink className='h-3 w-3' />
          </a>
        ) : (
          <p className='text-sm text-muted-foreground'>Source not available yet.</p>
        )}
      </section>

      <section className='rounded-md border p-5 space-y-4' aria-labelledby='infrastructure-heading'>
        <div>
          <h2 id='infrastructure-heading' className='text-lg font-semibold'>
            2. Course infrastructure
          </h2>
          <p className='text-sm text-muted-foreground'>
            Create or repair the GitLab groups, tutor access, shared CI, and demo. This action does
            not create student repositories.
          </p>
        </div>
        <dl className='grid gap-x-4 gap-y-2 text-sm sm:grid-cols-[11rem_1fr]'>
          <dt>Course group</dt>
          <dd>
            <Resource resource={data?.groups?.course ?? null} empty='Not created' />
          </dd>
          <dt>Tutors group</dt>
          <dd>
            <Resource resource={data?.groups?.tutors ?? null} empty='Not created' />
          </dd>
          <dt>Introcourse group</dt>
          <dd>
            <Resource resource={data?.groups?.introCourse ?? null} empty='Not created' />
          </dd>
          <dt>Shared CI project</dt>
          <dd>
            <Resource resource={data?.ciProject ?? null} empty='Not created' />
          </dd>
        </dl>
        <ul className='space-y-1 text-sm'>
          <Check ok={data?.checks?.tutorsReady ?? false} label='GitLab tutor access is ready' />
        </ul>
        <Button
          onClick={() => setup.mutate()}
          disabled={setup.isPending || reset.isPending || !data?.source?.sha}
        >
          {setup.isPending ? 'Checking infrastructure...' : 'Create or repair infrastructure'}
        </Button>
      </section>

      <section className='rounded-md border p-5 space-y-4' aria-labelledby='demo-heading'>
        <div>
          <h2 id='demo-heading' className='text-lg font-semibold'>
            3. Demo repository
          </h2>
          <p className='text-sm text-muted-foreground'>
            Test the starter app, daily issues, tutor approval, and CI here before creating student
            repositories. The status board shows Open, In Progress, In Review, Blocked, and Done.
            After practicing with branches or merge requests, reset the demo so the instructor copy
            is clean. Repair fills missing setup; it does not overwrite changed files or issues.
            Student and peer permissions still need a controlled student-style test.
          </p>
        </div>
        <dl className='grid gap-x-4 gap-y-2 text-sm sm:grid-cols-[11rem_1fr]'>
          <dt>Repository</dt>
          <dd>
            <Resource resource={data?.demoProject ?? null} empty='Not created' />
          </dd>
          <dt>Main commit</dt>
          <dd className='font-mono'>{data?.demoProject?.sha?.slice(0, 12) || '—'}</dd>
          <dt>Daily issues</dt>
          <dd>{data?.demoProject?.issueCount ?? '—'}</dd>
          <dt>Status board</dt>
          <dd>
            {data?.demoProject?.url ? (
              <a
                className='text-primary underline underline-offset-2'
                href={`${data.demoProject.url}/-/boards`}
                target='_blank'
                rel='noopener noreferrer'
              >
                Open board
              </a>
            ) : (
              '—'
            )}
          </dd>
          <dt>Main pipeline</dt>
          <dd>
            {data?.demoProject?.pipelineUrl ? (
              <a
                className='text-primary underline underline-offset-2'
                href={data.demoProject.pipelineUrl}
                target='_blank'
                rel='noopener noreferrer'
              >
                {data.demoProject.pipelineStatus || 'Open pipeline'}
              </a>
            ) : (
              'Not run yet'
            )}
          </dd>
        </dl>
        <ul className='space-y-1 text-sm'>
          <Check ok={data?.checks?.demoReady ?? false} label='Demo setup is ready' />
          <Check
            ok={data?.checks?.signingReady ?? false}
            label='Apple Developer team ID is configured for student repositories'
          />
          <Check
            ok={data?.checks?.materialCurrent ?? false}
            label='Demo matches teaching material'
          />
        </ul>
        <Dialog
          open={resetOpen}
          onOpenChange={(open) => {
            setResetOpen(open)
            if (!open) setTypedConfirmation('')
          }}
        >
          <Button
            variant='outline'
            onClick={() => setResetOpen(true)}
            disabled={!data?.demoProject}
          >
            Reset demo...
          </Button>
          <DialogContent>
            <DialogHeader>
              <DialogTitle>Reset the demo repository</DialogTitle>
              <DialogDescription>
                PROMPT will replace the demo’s main branch with the current teaching material,
                delete practice branches and extra issues, and restore the Git 2 exercise branches.
                The project URL and existing daily issue numbers stay the same. Old comments on
                those issues remain. Open merge requests will close, but their history and merge
                request numbers will remain. Student repositories are unaffected. This reset cannot
                be undone.
              </DialogDescription>
            </DialogHeader>
            <div className='space-y-2'>
              <label htmlFor='reset-demo-confirmation' className='text-sm font-medium'>
                Type {resetConfirmation} to confirm
              </label>
              <Input
                id='reset-demo-confirmation'
                value={typedConfirmation}
                onChange={(event) => setTypedConfirmation(event.target.value)}
                autoComplete='off'
              />
            </div>
            <DialogFooter>
              <Button variant='outline' onClick={() => setResetOpen(false)}>
                Cancel
              </Button>
              <Button
                variant='destructive'
                onClick={() => reset.mutate()}
                disabled={
                  typedConfirmation !== resetConfirmation ||
                  !data?.demoProject ||
                  !data?.source?.sha ||
                  reset.isPending ||
                  setup.isPending
                }
              >
                {reset.isPending ? 'Resetting...' : 'Reset demo in place'}
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>
      </section>

      {(data?.issues?.length ?? 0) > 0 && (
        <section
          className='rounded-md border border-amber-300 p-5 space-y-2'
          aria-labelledby='issues-heading'
        >
          <h2 id='issues-heading' className='text-lg font-semibold'>
            Items to resolve
          </h2>
          <ul className='list-disc pl-5 text-sm space-y-1'>
            {data?.issues?.map((issue) => (
              <li key={issue}>{issue}</li>
            ))}
          </ul>
        </section>
      )}

      <p className='text-sm text-muted-foreground'>
        Student repositories are initialized separately after the demo has been checked and each
        student has a verified GitLab username and an assigned tutor.
      </p>
    </div>
  )
}
