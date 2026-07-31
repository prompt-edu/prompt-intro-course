import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useCourseStore } from '@tumaet/prompt-shared-state'
import {
  Alert,
  AlertDescription,
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
  Badge,
  Button,
} from '@tumaet/prompt-ui-components'
import { GitBranch, Loader2, Shuffle, Trash2, Unlink } from 'lucide-react'
import { useMemo, useState } from 'react'
import { useParams } from 'react-router-dom'
import type { PeerAssignment, SyncResult } from '../../../interfaces/PeerAssignment'
import { deletePeerAssignments } from '../../../network/mutations/deletePeerAssignments'
import { generatePeerAssignments } from '../../../network/mutations/generatePeerAssignments'
import { syncPeerAssignmentsToGitlab } from '../../../network/mutations/syncPeerAssignmentsToGitlab'
import { unsyncPeerAssignmentsFromGitlab } from '../../../network/mutations/unsyncPeerAssignmentsFromGitlab'

interface PeerAssignmentActionsProps {
  peerAssignments: PeerAssignment[]
  totalStudents: number
}

export const PeerAssignmentActions = ({
  peerAssignments,
  totalStudents,
}: PeerAssignmentActionsProps) => {
  const { phaseId, courseId } = useParams<{ phaseId: string; courseId: string }>()
  const queryClient = useQueryClient()
  const [error, setError] = useState<string | null>(null)
  const [syncResults, setSyncResults] = useState<SyncResult[] | null>(null)
  const { courses } = useCourseStore()
  const semesterTag = courses.find((course) => course.id === courseId)?.semesterTag ?? ''

  // Count unique students in peer assignments
  const uniqueStudents = useMemo(
    () => new Set(peerAssignments.flatMap((a) => [a.studentID, a.peerID])).size,
    [peerAssignments],
  )

  const generateMutation = useMutation({
    mutationFn: () => generatePeerAssignments(phaseId ?? ''),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['peerAssignments', phaseId] })
      setError(null)
      setSyncResults(null)
    },
    onError: () => setError('Failed to generate peer assignments.'),
  })

  const deleteMutation = useMutation({
    mutationFn: () => deletePeerAssignments(phaseId ?? ''),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['peerAssignments', phaseId] })
      setError(null)
      setSyncResults(null)
    },
    onError: () => setError('Failed to delete peer assignments.'),
  })

  const syncMutation = useMutation({
    mutationFn: () => syncPeerAssignmentsToGitlab(phaseId ?? '', semesterTag),
    onSuccess: (results) => {
      setSyncResults(results)
      setError(null)
    },
    onError: () => setError('Failed to sync to GitLab. Is the GitLab token configured?'),
  })

  const unsyncMutation = useMutation({
    mutationFn: () => unsyncPeerAssignmentsFromGitlab(phaseId ?? '', semesterTag),
    onSuccess: (results) => {
      setSyncResults(results)
      setError(null)
    },
    onError: () => setError('Failed to unsync from GitLab. Is the GitLab token configured?'),
  })

  const statusVariant =
    uniqueStudents === 0
      ? 'secondary'
      : uniqueStudents >= totalStudents
        ? 'default'
        : 'outline-solid'

  const isLoading =
    generateMutation.isPending ||
    deleteMutation.isPending ||
    syncMutation.isPending ||
    unsyncMutation.isPending

  return (
    <div className='space-y-4'>
      <div className='flex flex-wrap items-center gap-3'>
        <Badge variant={statusVariant}>
          {uniqueStudents} of {totalStudents} students grouped
        </Badge>

        <Button
          onClick={() => generateMutation.mutate()}
          disabled={isLoading || totalStudents === 0}
          size='sm'
        >
          {generateMutation.isPending ? (
            <Loader2 className='h-4 w-4 animate-spin mr-1' />
          ) : (
            <Shuffle className='h-4 w-4 mr-1' />
          )}
          Generate Groups
        </Button>

        <AlertDialog>
          <AlertDialogTrigger asChild>
            <Button
              variant='destructive'
              size='sm'
              disabled={isLoading || peerAssignments.length === 0}
            >
              <Trash2 className='h-4 w-4 mr-1' />
              Clear All
            </Button>
          </AlertDialogTrigger>
          <AlertDialogContent>
            <AlertDialogHeader>
              <AlertDialogTitle>Clear all peer assignments?</AlertDialogTitle>
              <AlertDialogDescription>
                This will remove all peer groups. Use &quot;Unsync from GitLab&quot; first to revoke
                Reporter access and approval rules.
              </AlertDialogDescription>
            </AlertDialogHeader>
            <AlertDialogFooter>
              <AlertDialogCancel>Cancel</AlertDialogCancel>
              <AlertDialogAction onClick={() => deleteMutation.mutate()}>
                Clear All
              </AlertDialogAction>
            </AlertDialogFooter>
          </AlertDialogContent>
        </AlertDialog>

        <Button
          onClick={() => syncMutation.mutate()}
          disabled={isLoading || peerAssignments.length === 0 || !semesterTag}
          size='sm'
          variant='outline'
        >
          {syncMutation.isPending ? (
            <Loader2 className='h-4 w-4 animate-spin mr-1' />
          ) : (
            <GitBranch className='h-4 w-4 mr-1' />
          )}
          Sync to GitLab
        </Button>
        <Button
          onClick={() => unsyncMutation.mutate()}
          disabled={isLoading || peerAssignments.length === 0 || !semesterTag}
          size='sm'
          variant='outline'
        >
          {unsyncMutation.isPending ? (
            <Loader2 className='h-4 w-4 animate-spin mr-1' />
          ) : (
            <Unlink className='h-4 w-4 mr-1' />
          )}
          Unsync from GitLab
        </Button>
      </div>

      {error && (
        <Alert variant='destructive'>
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      )}

      {syncResults && (
        <Alert>
          <AlertDescription>
            <p className='font-medium mb-1'>
              GitLab sync complete: {syncResults.filter((r) => r.success).length}/
              {syncResults.length} successful
            </p>
            {syncResults
              .filter((r) => !r.success)
              .map((r) => (
                <p key={`${r.studentID}-${r.peerID}`} className='text-sm text-destructive'>
                  Failed: {r.studentID.slice(0, 8)}... → {r.peerID.slice(0, 8)}...: {r.error}
                </p>
              ))}
          </AlertDescription>
        </Alert>
      )}
    </div>
  )
}
