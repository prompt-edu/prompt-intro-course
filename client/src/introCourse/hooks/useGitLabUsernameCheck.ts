import { useEffect, useRef, useState } from 'react'
import {
  checkGitLabUsername,
  type GitLabUsernameCheck,
} from '../network/queries/checkGitLabUsername'

export const useGitLabUsernameCheck = (phaseId: string) => {
  const currentUsername = useRef('')
  const timer = useRef<ReturnType<typeof setTimeout> | null>(null)
  const [result, setResult] = useState<GitLabUsernameCheck | null>(null)
  const [isChecking, setIsChecking] = useState(false)

  const reset = (username: string) => {
    if (timer.current) clearTimeout(timer.current)
    currentUsername.current = username
    setResult(null)
    setIsChecking(false)
  }

  const check = async (username: string): Promise<GitLabUsernameCheck | null> => {
    if (timer.current) clearTimeout(timer.current)
    if (!username || !/^[A-Za-z0-9_.-]+$/.test(username)) return null
    currentUsername.current = username
    setIsChecking(true)
    try {
      const response = await checkGitLabUsername(phaseId, username)
      if (currentUsername.current === username) setResult(response)
      return response
    } catch {
      const failed: GitLabUsernameCheck = {
        username,
        status: 'check_failed',
        gitLabName: '',
        gitLabURL: '',
      }
      if (currentUsername.current === username) setResult(failed)
      return failed
    } finally {
      if (currentUsername.current === username) setIsChecking(false)
    }
  }

  const schedule = (username: string) => {
    reset(username)
    if (/^[A-Za-z0-9_.-]+$/.test(username)) {
      timer.current = setTimeout(() => void check(username), 500)
    }
  }

  useEffect(
    () => () => {
      if (timer.current) clearTimeout(timer.current)
    },
    [],
  )

  return { result, isChecking, schedule, check }
}
