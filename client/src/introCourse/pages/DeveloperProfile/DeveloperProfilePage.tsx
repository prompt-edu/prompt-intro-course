import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Button } from '@tumaet/prompt-ui-components'
import { AlertCircle, CheckCircle } from 'lucide-react'
import { useState } from 'react'
import { useParams } from 'react-router-dom'
import type { PostDeveloperProfile } from '../../interfaces/PostDeveloperProfile'
import { updateOwnDeveloperProfile } from '../../network/mutations/updateOwnDeveloperProfile'
import { getOwnAppleTeamStatus } from '../../network/queries/getAppleTeamStatus'
import { useIntroCourseStore } from '../../zustand/useIntroCourseStore'
import { DeveloperProfileForm } from './DeveloperProfileForm'

interface DeveloperProfilePageProps {
  onContinue: () => void
}

export const DeveloperProfilePage = ({ onContinue }: DeveloperProfilePageProps) => {
  const { phaseId } = useParams<{ phaseId: string }>()
  const queryClient = useQueryClient()
  const { developerProfile } = useIntroCourseStore()
  const [currState, setCurrState] = useState<'input' | 'success' | 'error'>(
    developerProfile?.appleID && developerProfile.gitLabUsername ? 'success' : 'input',
  )
  const { data: appleStatus } = useQuery({
    queryKey: ['apple-team-self', phaseId],
    queryFn: () => getOwnAppleTeamStatus(phaseId ?? ''),
    enabled: Boolean(phaseId && currState === 'success'),
    retry: false,
  })

  const mutation = useMutation({
    mutationFn: (devProfile: PostDeveloperProfile) =>
      updateOwnDeveloperProfile(phaseId ?? '', devProfile),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['developer_profile'] })
      queryClient.invalidateQueries({ queryKey: ['apple-team-self', phaseId] })
      setCurrState('success')
    },
    onError: () => {
      setCurrState('error')
    },
  })

  return (
    <div>
      {currState === 'input' && (
        <DeveloperProfileForm
          phaseId={phaseId ?? ''}
          developerProfile={developerProfile}
          onSubmit={(profile) => {
            mutation.mutate(profile)
          }}
        />
      )}
      {currState === 'success' && (
        <div className='flex items-center justify-center min-h-[300px]'>
          {currState === 'success' && (
            <div className='text-center space-y-4'>
              <div className='flex flex-col items-center space-y-2 text-green-500'>
                <CheckCircle className='h-12 w-12' />
                <h2 className='text-2xl font-semibold'>Success</h2>
              </div>
              <p className='text-muted-foreground max-w-md mx-auto'>
                You have successfully submitted your developer profile.
              </p>
              {appleStatus && (
                <div className='text-muted-foreground max-w-md mx-auto space-y-2'>
                  <p>
                    {appleStatus.membership === 'active'
                      ? appleStatus.provisioningAllowed
                        ? 'Your course Apple team access is active.'
                        : 'Your Apple team membership is active. Ask a tutor if Xcode says provisioning access is missing.'
                      : appleStatus.membership === 'invited'
                        ? 'Your course Apple team invitation is pending. Check your Apple Account email.'
                        : 'A course Apple team invitation has not been sent yet. You can use a simulated device in Xcode.'}
                  </p>
                  {Object.entries(appleStatus.devices).map(([device, registered]) => (
                    <p key={device}>
                      {device}:{' '}
                      {registered
                        ? 'registered with the course team'
                        : 'not registered with the course team'}
                    </p>
                  ))}
                </div>
              )}
              <div className='pt-4'>
                <div className='flex justify-center gap-3'>
                  <Button variant='outline' onClick={() => setCurrState('input')}>
                    Edit profile
                  </Button>
                  <Button onClick={onContinue}>Continue to the next step</Button>
                </div>
              </div>
            </div>
          )}
        </div>
      )}
      {currState === 'error' && (
        <div className='text-center space-y-4'>
          <div className='flex flex-col items-center space-y-2 text-red-600'>
            <AlertCircle className='h-12 w-12' />
            <h2 className='text-2xl font-semibold'>Error</h2>
          </div>
          <p className='text-muted-foreground max-w-md mx-auto'>
            Something went wrong. Please try again later or contact support.
          </p>
          <div className='flex space-x-4 pt-4 max-w-md mx-auto'>
            <Button onClick={() => setCurrState('input')} variant='outline' className='flex-1'>
              Back
            </Button>
          </div>
        </div>
      )}
    </div>
  )
}
