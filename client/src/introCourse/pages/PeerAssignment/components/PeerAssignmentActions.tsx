import { useState } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useParams } from 'react-router-dom'
import { Shuffle, Trash2, GitBranch, Loader2 } from 'lucide-react'
import {
  Badge,
  Button,
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
  Input,
  Label,
} from '@tumaet/prompt-ui-components'
import { PeerAssignment, SyncResult } from '../../../interfaces/PeerAssignment'
import { generatePeerAssignments } from '../../../network/mutations/generatePeerAssignments'
import { deletePeerAssignments } from '../../../network/mutations/deletePeerAssignments'
import { syncPeerAssignmentsToGitlab } from '../../../network/mutations/syncPeerAssignmentsToGitlab'

interface PeerAssignmentActionsProps {
  peerAssignments: PeerAssignment[]
  totalStudents: number
}

export const PeerAssignmentActions = ({
  peerAssignments,
  totalStudents,
}: PeerAssignmentActionsProps) => {
  const { phaseId } = useParams<{ phaseId: string }>()
  const queryClient = useQueryClient()
  const [error, setError] = useState<string | null>(null)
  const [syncResults, setSyncResults] = useState<SyncResult[] | null>(null)
  const [semesterTag, setSemesterTag] = useState('')

  // Count unique students in peer assignments
  const uniqueStudents = new Set(
    peerAssignments.flatMap((a) => [a.studentID, a.peerID]),
  ).size

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

  const statusVariant =
    uniqueStudents === 0 ? 'secondary' : uniqueStudents >= totalStudents ? 'default' : 'outline'

  const isLoading = generateMutation.isPending || deleteMutation.isPending || syncMutation.isPending

  return (
    <div className='space-y-4'>
      <div className='flex flex-wrap items-center gap-3'>
        <Badge variant={statusVariant}>
          {uniqueStudents} of {totalStudents} students paired
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
          Generate Pairs
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
                This will remove all peer pairings. GitLab access will not be revoked automatically.
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

        <div className='flex items-center gap-2'>
          <Input
            placeholder='Semester tag (e.g. IOS25)'
            value={semesterTag}
            onChange={(e) => setSemesterTag(e.target.value)}
            className='w-48 h-9'
          />
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
        </div>
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
              .map((r, i) => (
                <p key={i} className='text-sm text-destructive'>
                  Failed: {r.studentID.slice(0, 8)}... → {r.peerID.slice(0, 8)}...: {r.error}
                </p>
              ))}
          </AlertDescription>
        </Alert>
      )}
    </div>
  )
}
