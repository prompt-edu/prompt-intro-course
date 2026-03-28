package gitlabutil

import "errors"

const (
	// ASEGroupID is the GitLab group ID for the top-level ASE group on gitlab.lrz.de.
	ASEGroupID int64 = 186940

	// IPraktikumGroupName is the path segment for the iPraktikum subgroup.
	IPraktikumGroupName = "iPraktikum"

	// GitLabBaseURL is the base URL for the LRZ GitLab instance API.
	GitLabBaseURL = "https://gitlab.lrz.de/api/v4"
)

// ErrClientNotInitialized is returned when a GitLab operation is attempted
// but the client has not been set up (e.g., missing access token).
var ErrClientNotInitialized = errors.New("gitlab client not initialized")
