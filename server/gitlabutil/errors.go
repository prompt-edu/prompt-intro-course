package gitlabutil

import (
	"errors"
	"net/http"
	"strings"

	gitlab "gitlab.com/gitlab-org/api/client-go"
)

// IsAlreadyExistsError checks whether a GitLab API error indicates the resource
// already exists. It checks for 409 Conflict by status code, and for APIs that
// return 400 with "already exists" in the structured response message.
//
// String matching is intentionally restricted to the gitlab.ErrorResponse.Message
// field (not the full wrapped error chain) to avoid false positives from network
// errors or wrapping context that happens to contain matching substrings.
func IsAlreadyExistsError(err error) bool {
	if err == nil {
		return false
	}
	var errResp *gitlab.ErrorResponse
	if errors.As(err, &errResp) && errResp.Response != nil {
		code := errResp.Response.StatusCode
		if code == http.StatusConflict {
			return true
		}
		msg := errResp.Message
		return strings.Contains(msg, "already exists") ||
			strings.Contains(msg, "already a member") ||
			strings.Contains(msg, "already been taken")
	}
	return false
}

// IsNotFoundError checks whether a GitLab API error is a 404 Not Found.
func IsNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	var errResp *gitlab.ErrorResponse
	if errors.As(err, &errResp) && errResp.Response != nil {
		return errResp.Response.StatusCode == http.StatusNotFound
	}
	return false
}
