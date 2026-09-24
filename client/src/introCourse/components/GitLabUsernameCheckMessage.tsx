import type { GitLabUsernameCheck } from '../network/queries/checkGitLabUsername'

export const GitLabUsernameCheckMessage = ({
  result,
  isChecking,
}: {
  result: GitLabUsernameCheck | null
  isChecking: boolean
}) => {
  if (isChecking) return <p className='text-muted-foreground text-sm'>Checking LRZ GitLab…</p>
  if (!result) return null
  if (result.status === 'found') {
    return (
      <p className='text-sm'>
        Found{' '}
        <a className='underline' href={result.gitLabURL} target='_blank' rel='noreferrer'>
          {result.gitLabName || result.username}
        </a>{' '}
        on LRZ GitLab. Confirm this is the correct account.
      </p>
    )
  }
  if (result.status === 'check_failed') {
    return (
      <p className='text-sm text-amber-700'>
        GitLab could not be checked right now. You can save, but verify this username before
        repository setup.
      </p>
    )
  }
  return (
    <p className='text-sm text-red-700'>
      This username was not found on LRZ GitLab. Sign in to GitLab and check your profile URL.
    </p>
  )
}
