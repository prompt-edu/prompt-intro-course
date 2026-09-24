// PROMPT stores academic semester tags (for example, ws2627), while the
// iPraktikum GitLab groups use IOS tags (for example, ios2627).
export const gitlabCourseGroup = (semesterTag: string): string => {
  const tag = semesterTag.trim().toLowerCase()
  return tag.startsWith('ws') || tag.startsWith('ss') ? `ios${tag.slice(2)}` : tag
}
