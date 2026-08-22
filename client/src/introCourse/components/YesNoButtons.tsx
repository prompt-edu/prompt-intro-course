import { Button } from '@tumaet/prompt-ui-components'

export const YesNoButtons = ({
  name,
  value,
  onChange,
}: {
  // Identifies the question this pair belongs to. The developer profile form
  // renders four of them, all labelled Yes/No, so the testid is what makes an
  // individual button addressable.
  name: string
  value?: boolean
  onChange: (value: boolean) => void
}) => (
  <div className='flex space-x-4'>
    <Button
      type='button'
      variant={value ? 'default' : 'outline-solid'}
      onClick={() => onChange(true)}
      data-testid={`${name}-yes`}
    >
      Yes
    </Button>
    <Button
      type='button'
      variant={value === false ? 'default' : 'outline-solid'}
      onClick={() => onChange(false)}
      data-testid={`${name}-no`}
    >
      No
    </Button>
  </div>
)
