export interface GitlabResource {
  id: number
  url: string
}

export interface GitlabCourseSetupStatus {
  semesterTag: string
  source: {
    url: string
    sha: string
  } | null
  groups: {
    course: GitlabResource | null
    tutors: GitlabResource | null
    introCourse: GitlabResource | null
  }
  ciProject: GitlabResource | null
  demoProject:
    | (GitlabResource & {
        sha: string
        issueCount: number
        pipelineStatus: string
        pipelineUrl: string
      })
    | null
  checks: {
    tutorsReady: boolean
    demoReady: boolean
    materialCurrent: boolean
  }
  issues: string[]
}
