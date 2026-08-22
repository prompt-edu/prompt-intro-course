import { Locator, expect } from '@playwright/test'
import { INTRO_COURSE_ROUTES } from '../data/constants'
import { IntroCoursePhasePage } from './IntroCoursePhasePage'

// '' — the phase root. For a student this is the two-step journey (developer
// profile survey, then seat assignment); for staff it renders the same steps
// behind a "not a student of this course" alert.
export class IntroCourseOverviewPage extends IntroCoursePhasePage {
  protected readonly route = INTRO_COURSE_ROUTES.overview
  protected readonly headingName = 'Intro Course'

  get notAStudentAlert(): Locator {
    return this.page.getByText('Your are not a student of this course.')
  }

  // Step headers. Step 2's title carries a "(Available Soon)" suffix while the
  // student has no seat assignment.
  get surveyStep(): Locator {
    return this.page.getByRole('heading', { level: 2, name: 'Developer Profile Survey' })
  }

  seatStep(available: boolean): Locator {
    return this.page.getByRole('heading', {
      level: 2,
      name: available ? 'Seat Assignment' : 'Seat Assignment (Available Soon)',
      exact: true,
    })
  }

  // ── Developer profile survey ──────────────────────────────────────────────
  get appleIdInput(): Locator {
    return this.page.getByPlaceholder('example@icloud.com')
  }

  get gitlabUsernameInput(): Locator {
    return this.page.getByPlaceholder('i.e. ab12cde')
  }

  // Four Yes/No pairs share the same labels, so they are addressed by the testid
  // added in YesNoButtons.tsx.
  deviceAnswer(
    name: 'hasMacBook' | 'hasIPhone' | 'hasIPad' | 'hasAppleWatch',
    answer: 'yes' | 'no',
  ): Locator {
    return this.page.getByTestId(`${name}-${answer}`)
  }

  get submitButton(): Locator {
    return this.page.getByRole('button', { name: 'Submit' })
  }

  get successHeading(): Locator {
    return this.page.getByRole('heading', { level: 2, name: 'Success' })
  }

  get continueButton(): Locator {
    return this.page.getByRole('button', { name: 'Continue to the next step' })
  }

  // ── Seat assignment display ───────────────────────────────────────────────
  get seatInformationCard(): Locator {
    return this.page.getByText('Seat Information')
  }

  get tutorCard(): Locator {
    return this.page.getByText('Your Tutor')
  }

  get reviewPeersCard(): Locator {
    return this.page.getByText('Your Review Peers')
  }

  get noSeatAssigned(): Locator {
    return this.page.getByRole('heading', { level: 3, name: 'No Seat Assigned' })
  }

  get gitlabRepoButtons(): Locator {
    return this.page.getByRole('button', { name: 'GitLab Repo' })
  }

  // Collapsible steps: the trigger is the whole header row, so click the title.
  async openStep(title: 'Developer Profile Survey' | 'Seat Assignment') {
    await this.page.getByRole('heading', { level: 2, name: new RegExp(`^${title}`) }).click()
  }

  async submitSurvey(profile: { appleId: string; gitlabUsername: string; hasMacBook: boolean }) {
    await this.appleIdInput.fill(profile.appleId)
    await this.gitlabUsernameInput.fill(profile.gitlabUsername)
    await this.deviceAnswer('hasMacBook', profile.hasMacBook ? 'yes' : 'no').click()
    await this.deviceAnswer('hasIPhone', 'no').click()
    await this.deviceAnswer('hasIPad', 'no').click()
    await this.deviceAnswer('hasAppleWatch', 'no').click()
    await this.submitButton.click()
    await expect(this.successHeading).toBeVisible({ timeout: 30_000 })
  }
}
