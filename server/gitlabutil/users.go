package gitlabutil

import (
	"fmt"

	gitlab "gitlab.com/gitlab-org/api/client-go"
)

// GetUser resolves a GitLab user by their username. Returns an error if the
// user is not found or if multiple users match (which should not happen for
// exact username lookup).
func GetUser(git *gitlab.Client, username string) (*gitlab.User, error) {
	users, _, err := git.Users.ListUsers(&gitlab.ListUsersOptions{
		Username: gitlab.Ptr(username),
	})
	if err != nil {
		return nil, fmt.Errorf("list users for %q: %w", username, err)
	}
	if len(users) != 1 || users[0] == nil {
		return nil, fmt.Errorf("user %q not found on GitLab", username)
	}
	return users[0], nil
}
