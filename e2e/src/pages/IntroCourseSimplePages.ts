import { Locator } from '@playwright/test'
import { INTRO_COURSE_ROUTES } from '../data/constants'
import { IntroCoursePhasePage } from './IntroCoursePhasePage'

// /participants — a thin wrapper around the shared CoursePhaseParticipationsTable,
// fed entirely by core. The table renders first and last name in separate cells and
// paginates, so locate participants through its search box rather than by text.
export class IntroCourseParticipantsPage extends IntroCoursePhasePage {
  protected readonly route = INTRO_COURSE_ROUTES.participants
  protected readonly headingName = 'Intro Course Participants'

  get searchInput(): Locator {
    return this.page.getByRole('textbox', { name: /Search/ })
  }

  row(firstName: string, lastName: string): Locator {
    return this.page.getByRole('row', { name: new RegExp(`${firstName}\\s+${lastName}`) })
  }

  // "58 rows · …" above the table.
  get rowCount(): Locator {
    return this.page.getByText(/\d+ rows/)
  }

  async search(term: string) {
    await this.searchInput.fill(term)
    await this.searchInput.press('Enter')
  }
}

// /mailing — a reminder alert plus the shared CoursePhaseMailing component.
export class MailingPage extends IntroCoursePhasePage {
  protected readonly route = INTRO_COURSE_ROUTES.mailing
  protected readonly headingName = 'Mailing'

  get reminder(): Locator {
    return this.page.getByText('Important Reminder')
  }

  get participantsLink(): Locator {
    return this.page.getByRole('link', { name: 'Go To Course Phase Participants' })
  }
}
