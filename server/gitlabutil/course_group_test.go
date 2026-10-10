package gitlabutil

import "testing"

func TestCourseGroupName(t *testing.T) {
	for input, want := range map[string]string{
		"ws2627": "IOS2627",
		"SS26":   "IOS26",
		"ios25":  "IOS25",
	} {
		if got := CourseGroupName(input); got != want {
			t.Errorf("CourseGroupName(%q) = %q, want %q", input, got, want)
		}
	}
}
