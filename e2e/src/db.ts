import { Client } from 'pg'

// Direct access to the intro-course database, for the few teardowns the HTTP API
// cannot express. Use it sparingly: assertions belong against the API or the UI,
// this is only for restoring the fixture.
//
// The one case that needs it today is the developer-profile survey. POST
// /developer_profile is a create, not an upsert, and there is no DELETE endpoint —
// so the lecturer PUT can only blank a profile, never remove the row. Blanking is
// enough for the UI (the data shell reads an empty appleID + gitLabUsername as "no
// profile") but not for the server, which would then reject the next POST with a
// duplicate key. Without a real delete the spec could not survive a Playwright
// retry, and CI runs with retries: 2.

const CONFIG = {
  host: process.env.DB_INTRO_COURSE_HOST ?? 'localhost',
  port: Number(process.env.DB_INTRO_COURSE_PORT ?? 5432),
  user: process.env.DB_INTRO_COURSE_USER ?? 'prompt-postgres',
  password: process.env.DB_INTRO_COURSE_PASSWORD ?? 'prompt-postgres',
  database: process.env.DB_INTRO_COURSE_NAME ?? 'prompt',
}

async function withClient<T>(fn: (client: Client) => Promise<T>): Promise<T> {
  const client = new Client(CONFIG)
  await client.connect()
  try {
    return await fn(client)
  } finally {
    await client.end()
  }
}

// Removes a participant's developer profile entirely, returning them to the
// pre-survey state the seed ships them in.
export async function deleteDeveloperProfile(
  coursePhaseId: string,
  courseParticipationId: string,
): Promise<void> {
  await withClient((client) =>
    client.query(
      'DELETE FROM developer_profile WHERE course_phase_id = $1 AND course_participation_id = $2',
      [coursePhaseId, courseParticipationId],
    ),
  )
}
