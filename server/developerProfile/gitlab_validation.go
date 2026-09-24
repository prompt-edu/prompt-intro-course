package developerProfile

import (
	"net/http"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

var gitLabUsernamePattern = regexp.MustCompile(`^[A-Za-z0-9_.-]+$`)

type gitLabValidationResult struct {
	CourseParticipationID uuid.UUID `json:"courseParticipationID"`
	Username              string    `json:"username"`
	Status                string    `json:"status"`
	GitLabName            string    `json:"gitLabName,omitempty"`
	GitLabURL             string    `json:"gitLabURL,omitempty"`
}

func validateGitLabProfiles(c *gin.Context) {
	phaseID, err := uuid.Parse(c.Param("coursePhaseID"))
	if err != nil {
		handleError(c, http.StatusBadRequest, err)
		return
	}
	git := DeveloperProfileServiceSingleton.gitlabClient
	if git == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "GitLab validation is not configured"})
		return
	}
	profiles, err := GetAllDeveloperProfiles(c, phaseID)
	if err != nil {
		handleError(c, http.StatusInternalServerError, err)
		return
	}
	results := make([]gitLabValidationResult, 0, len(profiles))
	for _, profile := range profiles {
		result := checkGitLabUsername(git, profile.GitLabUsername)
		result.CourseParticipationID = profile.CourseParticipationID
		results = append(results, result)
	}
	c.JSON(http.StatusOK, results)
}

func checkGitLabUsername(git *gitlab.Client, username string) gitLabValidationResult {
	result := gitLabValidationResult{Username: username}
	if username == "" {
		result.Status = "missing"
		return result
	}
	if strings.TrimSpace(username) != username || !gitLabUsernamePattern.MatchString(username) {
		result.Status = "invalid_format"
		return result
	}
	users, _, err := git.Users.ListUsers(&gitlab.ListUsersOptions{Username: gitlab.Ptr(username)})
	if err != nil {
		result.Status = "check_failed"
		return result
	}
	if len(users) != 1 || users[0] == nil || users[0].Username != username {
		result.Status = "not_found"
		return result
	}
	result.Status = "found"
	result.GitLabName = users[0].Name
	result.GitLabURL = users[0].WebURL
	return result
}
