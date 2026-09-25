import {
  Button,
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
  ScrollArea,
  Separator,
} from '@tumaet/prompt-ui-components'
import { ExternalLink, HelpCircle } from 'lucide-react'
import * as React from 'react'

export const AppleIDHelperDialog = () => {
  const [open, setOpen] = React.useState(false)

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button variant='outline' data-testid='apple-id-help'>
          <HelpCircle className='h-4 w-4 mr-1' />
          Help
        </Button>
      </DialogTrigger>
      <DialogContent className='sm:max-w-[625px]'>
        <DialogHeader>
          <DialogTitle>Apple Account for Xcode</DialogTitle>
          <DialogDescription>
            Use an existing Apple Account or create one before submitting your profile.
          </DialogDescription>
        </DialogHeader>
        <ScrollArea className='max-h-[80vh] pr-4'>
          <style>
            {`
              .appleID ol {
                  list-style: decimal; /* Restore list-style for ordered lists */
                  margin: 0 0 1rem 1.5rem; /* Adjust spacing around lists */
                  padding: 0;
              }

            `}
          </style>
          <div className='appleID'>
            <ol className='space-y-4 pl-5 text-sm'>
              <li>
                If you need an account, visit{' '}
                <a
                  href='https://account.apple.com/'
                  target='_blank'
                  rel='noopener noreferrer'
                  className='font-medium text-primary hover:underline'
                >
                  account.apple.com
                  <ExternalLink className='ml-1 inline-block h-4 w-4' />
                </a>
              </li>
              <li>Create your Apple Account and verify your email address and phone number.</li>
              <li>Enter that account&apos;s email address in your developer profile.</li>
              <li>
                If the course sends you a team invitation, accept it within three days. Sign in with
                the same account in Xcode.
              </li>
            </ol>
          </div>
        </ScrollArea>
        <Separator className='my-4' />
        <div className='flex justify-end'>
          <Button onClick={() => setOpen(false)}>Close</Button>
        </div>
      </DialogContent>
    </Dialog>
  )
}
