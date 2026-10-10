package gitlabutil

import "strings"

// CourseGroupName maps PROMPT's academic semester tag to the GitLab group
// convention used by iPraktikum. Existing IOS tags are left unchanged.
func CourseGroupName(semesterTag string) string {
	tag := strings.ToUpper(strings.TrimSpace(semesterTag))
	if strings.HasPrefix(tag, "WS") || strings.HasPrefix(tag, "SS") {
		return "IOS" + tag[2:]
	}
	return tag
}
