import * as z from 'zod'

export const instructorDevProfile = z.object({
  appleID: z.string().email('Invalid email address').or(z.literal('')),
  gitLabUsername: z
    .string()
    .trim()
    .regex(/^$|^[A-Za-z0-9_.-]+$/, 'Enter the username only, without a URL or email address'),
  hasMacBook: z.boolean(),
  // Use union type to handle empty strings properly with correct TypeScript inference
  iPhoneUDID: z
    .string()
    .regex(/^[0-9A-Fa-f]{8}-[0-9A-Fa-f]{16}$/, 'Invalid iPhone UDID')
    .or(z.literal(''))
    .optional()
    .transform((val) => (val === '' ? undefined : val)),
  iPadUDID: z
    .string()
    .regex(/^[0-9A-Fa-f]{8}-[0-9A-Fa-f]{16}$/, 'Invalid iPad UDID')
    .or(z.literal(''))
    .optional()
    .transform((val) => (val === '' ? undefined : val)),
  appleWatchUDID: z
    .string()
    .regex(/^[0-9A-Fa-f]{8}-[0-9A-Fa-f]{16}$/, 'Invalid Apple Watch UDID')
    .or(z.literal(''))
    .optional()
    .transform((val) => (val === '' ? undefined : val)),
})

export type InstructorDeveloperFormValues = z.input<typeof instructorDevProfile>
