import type { ReactNode } from 'react'
import type { GitLabUsernameCheck } from '../network/queries/checkGitLabUsername'

export const GitLabUsernameCheckMessage = ({
  result,
  isChecking,
}: {
  result: GitLabUsernameCheck | null
  isChecking: boolean
}) => {
  if (isChecking) {
    return (
      <div className='min-h-10 text-muted-foreground text-sm' aria-live='polite'>
        Checking LRZ GitLab…
      </div>
    )
  }
  if (!result) return <div className='min-h-10' aria-live='polite' />
  let content: ReactNode
  if (result.status === 'found') {
    content = (
      <p className='text-sm'>
        Found{' '}
        <a className='underline' href={result.gitLabURL} target='_blank' rel='noreferrer'>
          {result.gitLabName || result.username}
        </a>{' '}
        on LRZ GitLab. Confirm this is the correct account.
      </p>
    )
  } else if (result.status === 'check_failed') {
    content = (
      <p className='text-sm text-amber-700'>
        GitLab could not be checked right now. You can save, but verify this username before
        repository setup.
      </p>
    )
  } else {
    content = (
      <p className='text-sm text-red-700'>
        This username was not found on LRZ GitLab. Sign in to GitLab and check your profile URL.
      </p>
    )
  }
  return (
    <div className='min-h-10' aria-live='polite'>
      {content}
    </div>
  )
}
