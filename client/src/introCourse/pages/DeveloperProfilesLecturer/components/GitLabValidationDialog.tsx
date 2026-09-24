import { useMutation } from '@tanstack/react-query'
import {
  Button,
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@tumaet/prompt-ui-components'
import { Loader2 } from 'lucide-react'
import { useState } from 'react'
import { useParams } from 'react-router-dom'
import {
  type GitLabProfileValidation,
  getGitLabProfileValidation,
} from '../../../network/queries/getGitLabProfileValidation'
import type { ParticipationWithDevProfiles } from '../interfaces/pariticipationWithDevProfiles'

type Props = { participants: ParticipationWithDevProfiles[] }

export const GitLabValidationDialog = ({ participants }: Props) => {
  const { phaseId } = useParams<{ phaseId: string }>()
  const [open, setOpen] = useState(false)
  const check = useMutation({ mutationFn: () => getGitLabProfileValidation(phaseId ?? '') })

  const openAndCheck = () => {
    setOpen(true)
    check.mutate()
  }

  const byParticipation = new Map(
    check.data?.map((result) => [result.courseParticipationID, result]) ?? [],
  )
  const rows = participants
    .map((participant) => ({
      participant,
      result: byParticipation.get(participant.participation.courseParticipationID),
    }))
    .sort((a, b) => {
      const aFound = a.result?.status === 'found' ? 1 : 0
      const bFound = b.result?.status === 'found' ? 1 : 0
      return aFound - bFound
    })
  const found = rows.filter((row) => row.result?.status === 'found').length

  return (
    <>
      <Button type='button' variant='outline' onClick={openAndCheck}>
        Check GitLab usernames
      </Button>
      <Dialog open={open} onOpenChange={setOpen}>
        <DialogContent className='max-w-4xl max-h-[85vh] overflow-y-auto'>
          <DialogHeader>
            <DialogTitle>LRZ GitLab username check</DialogTitle>
            <DialogDescription>
              Checks whether each username resolves to an LRZ GitLab account. Compare the GitLab
              name with the student before creating repositories; an existing username alone does
              not prove ownership.
            </DialogDescription>
          </DialogHeader>
          {check.isPending && (
            <p className='flex items-center gap-2'>
              <Loader2 className='h-4 w-4 animate-spin' /> Checking accounts…
            </p>
          )}
          {check.isError && (
            <p className='text-destructive'>GitLab check failed. Try again before provisioning.</p>
          )}
          {check.data && (
            <>
              <p className='text-sm'>
                {found} of {participants.length} participants have a username found on LRZ GitLab.
              </p>
              <div className='overflow-x-auto'>
                <table className='w-full text-sm'>
                  <thead>
                    <tr className='border-b text-left'>
                      <th className='p-2'>Student</th>
                      <th className='p-2'>PROMPT username</th>
                      <th className='p-2'>Check</th>
                      <th className='p-2'>GitLab account name</th>
                    </tr>
                  </thead>
                  <tbody>
                    {rows.map(({ participant, result }) => {
                      const student = participant.participation.student
                      const status = result?.status ?? 'no_profile'
                      const label: Record<
                        GitLabProfileValidation['status'] | 'no_profile',
                        string
                      > = {
                        found: 'Found',
                        missing: 'Username missing',
                        invalid_format: 'Use username, not URL',
                        not_found: 'Account not found',
                        check_failed: 'Check failed',
                        no_profile: 'No profile',
                      }
                      return (
                        <tr
                          key={participant.participation.courseParticipationID}
                          className='border-b'
                        >
                          <td className='p-2'>
                            {student.firstName} {student.lastName}
                          </td>
                          <td className='p-2 break-all'>{result?.username || '—'}</td>
                          <td
                            className={`p-2 ${status === 'found' ? 'text-green-600' : 'text-destructive'}`}
                          >
                            {label[status]}
                          </td>
                          <td className='p-2'>
                            {result?.gitLabURL ? (
                              <a
                                href={result.gitLabURL}
                                target='_blank'
                                rel='noreferrer'
                                className='underline'
                              >
                                {result.gitLabName}
                              </a>
                            ) : (
                              '—'
                            )}
                          </td>
                        </tr>
                      )
                    })}
                  </tbody>
                </table>
              </div>
              <Button
                type='button'
                variant='outline'
                onClick={() => check.mutate()}
                disabled={check.isPending}
              >
                Check again
              </Button>
            </>
          )}
        </DialogContent>
      </Dialog>
    </>
  )
}
