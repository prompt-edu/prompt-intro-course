import { zodResolver } from '@hookform/resolvers/zod'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  Button,
  Checkbox,
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
  Input,
  Separator,
} from '@tumaet/prompt-ui-components'
import { isAxiosError } from 'axios'
import { AlertTriangle, CheckCircle, Laptop, Smartphone, Tablet, Watch } from 'lucide-react'
import type React from 'react'
import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { GitLabUsernameCheckMessage } from '../../../components/GitLabUsernameCheckMessage'
import { useGitLabUsernameCheck } from '../../../hooks/useGitLabUsernameCheck'
import type { PostDeveloperProfile } from '../../../interfaces/PostDeveloperProfile'
import { inviteToAppleTeam, registerAppleDevice } from '../../../network/mutations/onboardAppleTeam'
import { updateDeveloperProfile } from '../../../network/mutations/updateDeveloperProfile'
import { updateGitLabStatusCreated } from '../../../network/mutations/updateGitlabStatus'
import { getAppleProfileStatus } from '../../../network/queries/getAppleTeamStatus'
import {
  type InstructorDeveloperFormValues,
  instructorDevProfile,
} from '../../../validations/instructorDevProfile'
import type { ParticipationWithDevProfiles } from '../interfaces/pariticipationWithDevProfiles'

interface ProfileDetailsDialogProps {
  participantWithProfile: ParticipationWithDevProfiles
  phaseId: string
  onClose: () => void
  onSaved: () => void
}

export const ProfileDetailsDialog: React.FC<ProfileDetailsDialogProps> = ({
  participantWithProfile,
  phaseId,
  onClose,
  onSaved,
}) => {
  const queryClient = useQueryClient()
  const participationId = participantWithProfile.participation.courseParticipationID
  type AppleAction = 'invite' | 'iphone' | 'ipad' | 'watch'
  const [appleAction, setAppleAction] = useState<AppleAction | null>(null)
  const [appleActionError, setAppleActionError] = useState<string | null>(null)
  const deviceToRegister =
    appleAction === 'iphone'
      ? { label: 'iPhone', udid: participantWithProfile.devProfile?.iPhoneUDID }
      : appleAction === 'ipad'
        ? { label: 'iPad', udid: participantWithProfile.devProfile?.iPadUDID }
        : appleAction === 'watch'
          ? { label: 'Apple Watch', udid: participantWithProfile.devProfile?.appleWatchUDID }
          : null
  const { data: appleStatus, isError: appleStatusError } = useQuery({
    queryKey: ['apple-team-profile', phaseId, participationId],
    queryFn: () => getAppleProfileStatus(phaseId, participationId),
    enabled: Boolean(participantWithProfile.devProfile),
    retry: false,
  })
  const appleMutation = useMutation({
    mutationFn: (action: AppleAction) => {
      const profile = participantWithProfile.devProfile
      if (!profile) throw new Error('Save the developer profile first.')
      if (action === 'invite') return inviteToAppleTeam(phaseId, participationId, profile.appleID)
      const udid =
        action === 'iphone'
          ? profile.iPhoneUDID
          : action === 'ipad'
            ? profile.iPadUDID
            : profile.appleWatchUDID
      if (!udid) throw new Error('Save the device UDID first.')
      return registerAppleDevice(phaseId, participationId, action, udid)
    },
    onSuccess: () => {
      setAppleAction(null)
      setAppleActionError(null)
      queryClient.invalidateQueries({ queryKey: ['apple-team-profile', phaseId, participationId] })
      queryClient.invalidateQueries({ queryKey: ['apple-team-capacity', phaseId] })
    },
    onError: (error: unknown) => {
      const message =
        isAxiosError(error) && typeof error.response?.data?.error === 'string'
          ? error.response.data.error
          : 'Apple onboarding failed. Check the team connection and try again.'
      setAppleActionError(message)
      setAppleAction(null)
    },
  })
  const gitLabCheck = useGitLabUsernameCheck(phaseId)
  const form = useForm<InstructorDeveloperFormValues>({
    resolver: zodResolver(instructorDevProfile),
    defaultValues: {
      appleID: participantWithProfile.devProfile?.appleID || '',
      gitLabUsername: participantWithProfile.devProfile?.gitLabUsername || '',
      hasMacBook: participantWithProfile.devProfile?.hasMacBook || false,
      iPhoneUDID: participantWithProfile.devProfile?.iPhoneUDID || '',
      iPadUDID: participantWithProfile.devProfile?.iPadUDID || '',
      appleWatchUDID: participantWithProfile.devProfile?.appleWatchUDID || '',
    },
  })

  const [gitlabSuccess, setGitlabSuccess] = useState(
    participantWithProfile.gitlabStatus?.gitlabSuccess || false,
  )
  const gitlabStatusErrorMessage = participantWithProfile.gitlabStatus?.errorMessage || ''

  const { mutate, isPending } = useMutation({
    mutationFn: (devProfile: PostDeveloperProfile) =>
      updateDeveloperProfile(
        phaseId,
        participantWithProfile.participation.courseParticipationID,
        devProfile,
      ),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['apple-team-profile', phaseId, participationId] })
      onSaved()
      onClose()
    },
    onError: (error: unknown) => {
      console.error('Error saving profile:', error)
      let message = 'An error occurred while saving the profile.'
      if (error instanceof Error) {
        message = error.message
      }
      form.setError('root', {
        type: 'manual',
        message,
      })
    },
  })

  const updateGitlabStatusMutation = useMutation({
    mutationFn: () =>
      updateGitLabStatusCreated(
        phaseId,
        participantWithProfile.participation.courseParticipationID,
      ),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['gitlab_statuses'] })
      setGitlabSuccess(true)
    },
    onError: (error: unknown) => {
      console.error('Error updating GitLab status:', error)
      let message = 'An error occurred while updating the GitLab status.'
      if (error instanceof Error) {
        message = error.message
      }
      form.setError('root', {
        type: 'manual',
        message,
      })
    },
  })

  const verifyGitLabUsername = async (username: string) => {
    if (!username) return true
    const result = await gitLabCheck.check(username)
    if (form.getValues('gitLabUsername').trim() !== username) return false
    if (result?.status === 'found' || result?.status === 'check_failed') {
      form.clearErrors('gitLabUsername')
      return true
    }
    form.setError('gitLabUsername', {
      message: 'Username not found on LRZ GitLab. Check the profile URL.',
    })
    return false
  }

  const onSubmit = async (data: InstructorDeveloperFormValues) => {
    if (!(await verifyGitLabUsername(data.gitLabUsername))) return
    if (form.getValues('gitLabUsername').trim() !== data.gitLabUsername) return
    mutate(data)
  }

  return (
    <Dialog open onOpenChange={onClose}>
      <DialogContent className='sm:max-w-[600px] max-h-[90vh] overflow-y-auto'>
        <DialogHeader>
          <DialogTitle>
            {participantWithProfile.devProfile
              ? 'Edit Developer Profile'
              : 'Create Developer Profile'}
          </DialogTitle>
          <DialogDescription>
            {participantWithProfile.participation.student.firstName}{' '}
            {participantWithProfile.participation.student.lastName} (
            {participantWithProfile.participation.student.email})
          </DialogDescription>
        </DialogHeader>

        {participantWithProfile.devProfile && (
          <div className='rounded-md border p-3 text-sm'>
            <strong>Apple team</strong>
            <p>
              {appleStatus
                ? appleStatus.membership === 'active'
                  ? appleStatus.provisioningAllowed
                    ? 'Access and provisioning are active.'
                    : 'Membership is active; provisioning access is missing.'
                  : appleStatus.membership === 'invited'
                    ? 'Invitation pending.'
                    : 'No active membership or pending invitation.'
                : appleStatusError
                  ? 'Could not check Apple team access.'
                  : 'Checking Apple team access...'}
            </p>
            {appleStatus &&
              Object.entries(appleStatus.devices).map(([device, registered]) => (
                <p key={device}>
                  {device}: {registered ? 'registered' : 'not registered'}
                </p>
              ))}
            {appleStatus && (
              <div className='mt-3 flex flex-wrap gap-2'>
                {appleStatus.membership === 'none' && participantWithProfile.devProfile.appleID && (
                  <Button
                    type='button'
                    size='sm'
                    variant='outline'
                    onClick={() => setAppleAction('invite')}
                  >
                    Invite to Apple team
                  </Button>
                )}
                {(
                  [
                    ['iphone', 'iPhone', participantWithProfile.devProfile.iPhoneUDID],
                    ['ipad', 'iPad', participantWithProfile.devProfile.iPadUDID],
                    ['watch', 'Apple Watch', participantWithProfile.devProfile.appleWatchUDID],
                  ] as const
                ).map(([kind, label, udid]) =>
                  udid && !appleStatus.devices[label] ? (
                    <Button
                      key={kind}
                      type='button'
                      size='sm'
                      variant='outline'
                      onClick={() => setAppleAction(kind)}
                    >
                      Register {label}
                    </Button>
                  ) : null,
                )}
              </div>
            )}
            {appleActionError && <p className='mt-2 text-destructive'>{appleActionError}</p>}
          </div>
        )}

        <AlertDialog
          open={appleAction !== null}
          onOpenChange={(open) => {
            if (!open) setAppleAction(null)
          }}
        >
          <AlertDialogContent>
            <AlertDialogHeader>
              <AlertDialogTitle>
                {appleAction === 'invite'
                  ? 'Send Apple team invitation?'
                  : 'Register this device with Apple?'}
              </AlertDialogTitle>
              <AlertDialogDescription>
                {appleAction === 'invite'
                  ? `Apple will email ${participantWithProfile.devProfile?.appleID}. The invitation grants Developer access with provisioning across the TUM Apple team.`
                  : `Apple will register ${deviceToRegister?.label} UDID ${deviceToRegister?.udid}. This uses a team device slot.`}
              </AlertDialogDescription>
            </AlertDialogHeader>
            <AlertDialogFooter>
              <AlertDialogCancel>Cancel</AlertDialogCancel>
              <AlertDialogAction
                disabled={appleMutation.isPending}
                onClick={() => {
                  if (appleAction) appleMutation.mutate(appleAction)
                }}
              >
                {appleAction === 'invite' ? 'Send invitation' : 'Register device'}
              </AlertDialogAction>
            </AlertDialogFooter>
          </AlertDialogContent>
        </AlertDialog>

        {form.formState.errors.root && (
          <div className='mb-4 rounded bg-red-100 p-2 text-red-700'>
            {form.formState.errors.root.message}
          </div>
        )}

        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)} className='space-y-6 py-4'>
            <div className='grid grid-cols-2 gap-4'>
              <FormField
                control={form.control}
                name='appleID'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Apple Account email</FormLabel>
                    <FormDescription>
                      Leave empty if unconfirmed. The student can add or correct it in their own
                      profile.
                    </FormDescription>
                    <FormControl>
                      <Input placeholder='example@icloud.com' disabled={isPending} {...field} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name='gitLabUsername'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>GitLab Username</FormLabel>
                    <FormDescription>
                      Enter the verified LRZ GitLab username, without the profile URL.
                    </FormDescription>
                    <FormControl>
                      <Input
                        placeholder='username'
                        disabled={isPending}
                        {...field}
                        onChange={(event) => {
                          field.onChange(event)
                          gitLabCheck.schedule(event.target.value)
                          form.clearErrors('gitLabUsername')
                        }}
                        onBlur={() => {
                          field.onBlur()
                          if (field.value) void verifyGitLabUsername(field.value.trim())
                        }}
                      />
                    </FormControl>
                    <FormMessage />
                    <GitLabUsernameCheckMessage
                      result={gitLabCheck.result}
                      isChecking={gitLabCheck.isChecking}
                    />
                  </FormItem>
                )}
              />
            </div>

            <Separator />

            <div className='space-y-4'>
              <h3 className='text-lg font-medium'>Devices</h3>

              <FormField
                control={form.control}
                name='hasMacBook'
                render={({ field }) => (
                  <FormItem className='flex flex-row items-start space-x-3 space-y-0 rounded-md'>
                    <FormControl>
                      <Checkbox
                        checked={field.value}
                        onCheckedChange={field.onChange}
                        disabled={isPending}
                      />
                    </FormControl>
                    <div className='space-y-1 leading-none'>
                      <FormLabel className='flex items-center gap-2'>
                        <Laptop className='h-5 w-5' /> MacBook
                      </FormLabel>
                    </div>
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name='iPhoneUDID'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel className='flex items-center gap-2'>
                      <Smartphone className='h-5 w-5' /> iPhone UDID
                    </FormLabel>
                    <FormControl>
                      <Input placeholder='iPhone UDID (optional)' disabled={isPending} {...field} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name='iPadUDID'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel className='flex items-center gap-2'>
                      <Tablet className='h-5 w-5' /> iPad UDID
                    </FormLabel>
                    <FormControl>
                      <Input placeholder='iPad UDID (optional)' disabled={isPending} {...field} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name='appleWatchUDID'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel className='flex items-center gap-2'>
                      <Watch className='h-5 w-5' /> Apple Watch UDID
                    </FormLabel>
                    <FormControl>
                      <Input
                        placeholder='Apple Watch UDID (optional)'
                        disabled={isPending}
                        {...field}
                      />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
            </div>

            <Separator />

            <div className='space-y-4'>
              <h3 className='text-lg font-medium'>GitLab Status</h3>
              {gitlabSuccess ? (
                <div className='text-green-600 flex items-center gap-2'>
                  <CheckCircle />
                  Repository created successfully.
                </div>
              ) : (
                <div className='text-orange-600 flex items-center gap-2'>
                  <AlertTriangle />
                  {gitlabStatusErrorMessage || 'Not created yet.'}
                </div>
              )}
              {!gitlabSuccess && (
                <Button
                  type='button'
                  variant='outline'
                  onClick={() => updateGitlabStatusMutation.mutate()}
                  disabled={updateGitlabStatusMutation.isPending}
                >
                  Mark as Created
                </Button>
              )}
            </div>

            <DialogFooter>
              <Button type='button' variant='outline' onClick={onClose} disabled={isPending}>
                Cancel
              </Button>
              <Button type='submit' disabled={isPending || form.formState.isSubmitting}>
                {isPending ? 'Saving...' : 'Save Profile'}
              </Button>
            </DialogFooter>
          </form>
        </Form>
      </DialogContent>
    </Dialog>
  )
}
