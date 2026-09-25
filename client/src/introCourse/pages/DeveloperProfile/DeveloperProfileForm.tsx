import { zodResolver } from '@hookform/resolvers/zod'
import {
  Button,
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
  Input,
  useScreenSize,
} from '@tumaet/prompt-ui-components'
import { useForm } from 'react-hook-form'
import { GitLabUsernameCheckMessage } from '../../components/GitLabUsernameCheckMessage'
import { YesNoButtons } from '../../components/YesNoButtons'
import { useGitLabUsernameCheck } from '../../hooks/useGitLabUsernameCheck'
import type { DeveloperProfile } from '../../interfaces/DeveloperProfile'
import type { PostDeveloperProfile } from '../../interfaces/PostDeveloperProfile'
import { type DeveloperFormValues, developerFormSchema } from '../../validations/developerProfile'
import { AppleIDHelperDialog } from './components/AppleIDHelperDialog'
import { GitLabHelperDialog } from './components/GitLabHelperDialog'
import IOSUDIDDialog from './components/IOSUDIDDialog'

interface DeveloperProfileFormProps {
  phaseId: string
  developerProfile?: DeveloperProfile
  status?: string
  onSubmit: (developerProfile: PostDeveloperProfile) => void
}

export const DeveloperProfileForm = ({
  phaseId,
  developerProfile,
  status,
  onSubmit,
}: DeveloperProfileFormProps) => {
  const { width } = useScreenSize()
  const gitLabCheck = useGitLabUsernameCheck(phaseId)

  const form = useForm<DeveloperFormValues>({
    resolver: zodResolver(developerFormSchema),
    defaultValues: {
      appleID: developerProfile?.appleID || '',
      gitLabUsername: developerProfile?.gitLabUsername || '',
      hasMacBook: developerProfile?.hasMacBook,
      hasIPhone:
        developerProfile?.iPhoneUDID === undefined
          ? undefined
          : developerProfile?.iPhoneUDID !== '',
      iPhoneUDID: developerProfile?.iPhoneUDID || '',
      hasIPad:
        developerProfile?.iPadUDID === undefined ? undefined : developerProfile?.iPadUDID !== '',
      iPadUDID: developerProfile?.iPadUDID || '',
      hasAppleWatch:
        developerProfile?.appleWatchUDID === undefined
          ? undefined
          : developerProfile?.appleWatchUDID !== '',
      appleWatchUDID: developerProfile?.appleWatchUDID || '',
    },
  })

  const verifyGitLabUsername = async (username: string) => {
    const result = await gitLabCheck.check(username)
    if (form.getValues('gitLabUsername').trim() !== username) return false
    if (result?.status === 'found' || result?.status === 'check_failed') {
      form.clearErrors('gitLabUsername')
      return true
    }
    form.setError('gitLabUsername', {
      message: 'Username not found on LRZ GitLab. Check your profile URL.',
    })
    return false
  }

  const handleSubmit = async (values: DeveloperFormValues) => {
    if (!(await verifyGitLabUsername(values.gitLabUsername))) return
    if (form.getValues('gitLabUsername').trim() !== values.gitLabUsername) return
    const submittedProfile: PostDeveloperProfile = {
      appleID: values.appleID,
      gitLabUsername: values.gitLabUsername,
      hasMacBook: values.hasMacBook,
      iPhoneUDID: values.hasIPhone ? values.iPhoneUDID : undefined,
      iPadUDID: values.hasIPad ? values.iPadUDID : undefined,
      appleWatchUDID: values.hasAppleWatch ? values.appleWatchUDID : undefined,
    }
    onSubmit(submittedProfile)
  }

  return (
    <div>
      {status && <p className='text-muted-foreground mb-4'>{status}</p>}
      <Form {...form}>
        <form onSubmit={form.handleSubmit(handleSubmit)} className='space-y-8'>
          {/* Apple Account email for the course team invitation */}
          <FormField
            control={form.control}
            name='appleID'
            render={({ field }) => (
              <FormItem>
                <FormLabel>Apple Account email</FormLabel>
                <FormDescription>
                  Enter the email address of the Apple Account you will use in Xcode. We will send
                  the course developer team invitation to this address.
                </FormDescription>
                <FormControl>
                  <div className='flex items-center space-x-2'>
                    <Input placeholder='example@icloud.com' {...field} className='grow' />
                    <AppleIDHelperDialog />
                  </div>
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />

          {/* GitLab Username Field */}
          <FormField
            control={form.control}
            name='gitLabUsername'
            render={({ field }) => (
              <FormItem>
                <FormLabel>GitLab Username</FormLabel>
                <FormDescription>
                  Enter the username shown on your LRZ GitLab profile, without the full URL. If you
                  have not signed in to LRZ GitLab yet, sign in there first and then return here. If
                  your course repository already exists, tell your tutor when correcting this field
                  so access can be updated.
                </FormDescription>
                <FormControl>
                  <div className='flex items-center space-x-2'>
                    <Input
                      placeholder='i.e. ab12cde'
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
                      className='grow'
                    />
                    <GitLabHelperDialog />
                  </div>
                </FormControl>
                <FormMessage />
                <GitLabUsernameCheckMessage
                  result={gitLabCheck.result}
                  isChecking={gitLabCheck.isChecking}
                />
              </FormItem>
            )}
          />

          {/* MacBook Section */}
          <FormField
            control={form.control}
            name='hasMacBook'
            render={({ field }) => (
              <FormItem>
                <FormLabel>Do you have a MacBook?</FormLabel>
                <FormDescription>
                  Please only respond with Yes if you can bring the MacBook to the Intro Course
                  every day. If you do not have access you will get a device from the chair. There
                  are just limited devices and you are NOT allowed to take them home.
                </FormDescription>
                <FormControl>
                  <YesNoButtons name='hasMacBook' value={field.value} onChange={field.onChange} />
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />

          {/* iPhone Section: Yes/No + UDID Input */}
          <div className={`grid ${width > 800 ? 'grid-cols-4' : 'grid-cols-1'} gap-4 items-center`}>
            <div>
              <FormField
                control={form.control}
                name='hasIPhone'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Do you have an iPhone?</FormLabel>
                    <FormControl>
                      <YesNoButtons
                        name='hasIPhone'
                        value={field.value}
                        onChange={field.onChange}
                      />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
            </div>
            {form.watch('hasIPhone') && (
              <div className='col-span-3 h-full flex flex-col justify-end'>
                <FormField
                  control={form.control}
                  name='iPhoneUDID'
                  render={({ field }) => (
                    <FormItem>
                      <FormControl>
                        <div className='flex items-center space-x-2'>
                          <Input
                            placeholder="Enter your iPhone's UDID."
                            {...field}
                            className='w-full'
                          />
                          <IOSUDIDDialog />
                        </div>
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />
              </div>
            )}
          </div>

          {/* iPad Section: Yes/No + UDID Input */}
          <div className={`grid ${width > 800 ? 'grid-cols-4' : 'grid-cols-1'} gap-4 items-center`}>
            <div>
              <FormField
                control={form.control}
                name='hasIPad'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Do you have an iPad?</FormLabel>
                    <FormControl>
                      <YesNoButtons name='hasIPad' value={field.value} onChange={field.onChange} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
            </div>
            {form.watch('hasIPad') && (
              <div className='col-span-3 h-full flex flex-col justify-end'>
                <FormField
                  control={form.control}
                  name='iPadUDID'
                  render={({ field }) => (
                    <FormItem>
                      <FormControl>
                        <div className='flex items-center space-x-2'>
                          <Input
                            placeholder="Enter your iPad's UDID."
                            {...field}
                            className='w-full'
                          />
                          <IOSUDIDDialog />
                        </div>
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />
              </div>
            )}
          </div>

          {/* Apple Watch Section: Yes/No + UDID Input */}
          <div className={`grid ${width > 800 ? 'grid-cols-4' : 'grid-cols-1'} gap-4 items-center`}>
            <div>
              <FormField
                control={form.control}
                name='hasAppleWatch'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Do you have an Apple Watch?</FormLabel>
                    <FormControl>
                      <YesNoButtons
                        name='hasAppleWatch'
                        value={field.value}
                        onChange={field.onChange}
                      />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
            </div>
            {form.watch('hasAppleWatch') && (
              <div className='col-span-3 h-full flex flex-col justify-end'>
                <FormField
                  control={form.control}
                  name='appleWatchUDID'
                  render={({ field }) => (
                    <FormItem>
                      <FormControl>
                        <div className='flex items-center space-x-2'>
                          <Input
                            placeholder="Enter your Apple Watch's UDID."
                            {...field}
                            className='w-full'
                          />
                          <IOSUDIDDialog />
                        </div>
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />
              </div>
            )}
          </div>

          <div className='flex justify-end mt-3'>
            <Button type='submit' size='lg' disabled={form.formState.isSubmitting}>
              Submit
            </Button>
          </div>
        </form>
      </Form>
    </div>
  )
}
