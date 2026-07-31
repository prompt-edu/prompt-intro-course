import { useQuery } from '@tanstack/react-query'
import { useGetCoursePhase } from '@tumaet/prompt-shared-state'
import {
  Alert,
  AlertDescription,
  Button,
  ErrorPage,
  ManagementPageHeader,
  Separator,
} from '@tumaet/prompt-ui-components'
import { AlertTriangle, Loader2, RefreshCw } from 'lucide-react'
import { useState } from 'react'
import { useParams } from 'react-router-dom'
import type { Tutor } from '../../interfaces/Tutor'
import { importTutors } from '../../network/mutations/importTutors'
import { getAllTutors } from '../../network/queries/getAllTutors'
import { KeycloakGroupCreation } from './components/KeycloakGroupCreation'
import { TutorImportDialog } from './components/TutorImportDialog'
import { TutorTable } from './components/TutorTable'

export const TutorImportPage = () => {
  const { courseId, phaseId } = useParams<{ courseId: string; phaseId: string }>()
  const [syncStatus, setSyncStatus] = useState<{ loading: boolean; warnings: string[] | null }>({
    loading: false,
    warnings: null,
  })

  const {
    data: coursePhase,
    isPending: isCoursePhasePending,
    isError: isCoursePhaseError,
    refetch: refetchCoursePhase,
  } = useGetCoursePhase()

  const { data: tutors } = useQuery<Tutor[]>({
    queryKey: ['tutors', phaseId],
    queryFn: () => getAllTutors(phaseId ?? ''),
    enabled: !!phaseId,
  })

  const handleSyncKeycloak = async () => {
    if (!tutors || tutors.length === 0 || !phaseId || !courseId) return
    setSyncStatus({ loading: true, warnings: null })
    try {
      // Re-import the same tutors — the server upserts and retries Keycloak
      const result = await importTutors(
        phaseId,
        courseId,
        tutors.map(
          (t) =>
            ({
              id: t.id,
              firstName: t.firstName,
              lastName: t.lastName,
              email: t.email,
              matriculationNumber: t.matriculationNumber,
              universityLogin: t.universityLogin,
            }) as any,
        ),
      )
      if (result.warnings && result.warnings.length > 0) {
        setSyncStatus({ loading: false, warnings: result.warnings })
      } else {
        setSyncStatus({ loading: false, warnings: null })
      }
    } catch {
      setSyncStatus({ loading: false, warnings: ['Failed to sync tutors to Keycloak.'] })
    }
  }

  if (isCoursePhasePending) {
    return (
      <div className='flex justify-center items-center grow'>
        <Loader2 className='h-12 w-12 animate-spin text-primary' />
      </div>
    )
  }

  if (isCoursePhaseError) {
    return <ErrorPage onRetry={refetchCoursePhase} />
  }

  const groupExists = !!coursePhase?.restrictedData?.keycloakGroup

  return (
    <div className='space-y-8'>
      <ManagementPageHeader>Tutor Import</ManagementPageHeader>

      {/* Keycloak Group Section */}
      <div className='space-y-4'>
        <h2 className='text-xl font-semibold'>Keycloak Group Status</h2>
        <KeycloakGroupCreation coursePhase={coursePhase} />
      </div>

      <Separator className='my-6' />

      {groupExists ? (
        <div className='space-y-4'>
          <div className='flex items-center justify-between'>
            <h2 className='text-xl font-semibold'>Imported Tutors</h2>
            <div className='flex gap-2'>
              {tutors && tutors.length > 0 && (
                <Button
                  variant='outline'
                  size='sm'
                  onClick={handleSyncKeycloak}
                  disabled={syncStatus.loading}
                >
                  {syncStatus.loading ? (
                    <Loader2 className='mr-1.5 h-4 w-4 animate-spin' />
                  ) : (
                    <RefreshCw className='mr-1.5 h-4 w-4' />
                  )}
                  Sync to Keycloak
                </Button>
              )}
              <TutorImportDialog />
            </div>
          </div>
          {syncStatus.warnings && (
            <Alert>
              <AlertTriangle className='h-4 w-4' />
              <AlertDescription>
                {syncStatus.warnings.map((w) => (
                  <p key={w} className='text-sm'>
                    {w}
                  </p>
                ))}
              </AlertDescription>
            </Alert>
          )}
          <TutorTable />
        </div>
      ) : (
        <div> Please create a Keycloak group first before adding Tutors</div>
      )}
    </div>
  )
}
