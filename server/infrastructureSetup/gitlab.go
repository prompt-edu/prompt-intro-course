package infrastructureSetup

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/prompt-edu/prompt-intro-course/server/infrastructureSetup/data"
	log "github.com/sirupsen/logrus"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

const IN_PROGRESS_LABEL_ID int64 = 53319
const IN_REVIEW_LABEL_ID int64 = 53320
const ASE_GROUP_ID int64 = 186940

var errGitLabClientNotInitialized = errors.New("gitlab client not initialized")

func getClient() (*gitlab.Client, error) {
	if InfrastructureServiceSingleton.gitlabClient == nil {
		return nil, errGitLabClientNotInitialized
	}
	return InfrastructureServiceSingleton.gitlabClient, nil
}

// isAlreadyExistsError checks whether a GitLab API error indicates the resource
// already exists. It checks for 409 Conflict by status code, and for APIs that
// return 400 with "already exists" in the structured response message.
//
// String matching is intentionally restricted to the gitlab.ErrorResponse.Message
// field (not the full wrapped error chain) to avoid false positives from network
// errors or wrapping context that happens to contain matching substrings.
func isAlreadyExistsError(err error) bool {
	if err == nil {
		return false
	}
	var errResp *gitlab.ErrorResponse
	if errors.As(err, &errResp) && errResp.Response != nil {
		code := errResp.Response.StatusCode
		if code == http.StatusConflict {
			return true
		}
		// Only string-match on the structured API response message,
		// not the full wrapped error chain, to avoid false positives.
		msg := errResp.Message
		return strings.Contains(msg, "already exists") ||
			strings.Contains(msg, "already a member") ||
			strings.Contains(msg, "already been taken")
	}
	return false
}

// findSubGroup searches for a subgroup by name under the given parent.
// Returns the group if found, nil if not found, or an error on API failure.
func findSubGroup(groupName string, parentGroupID int64) (*gitlab.Group, error) {
	git, err := getClient()
	if err != nil {
		return nil, err
	}

	groups, _, err := git.Groups.ListSubGroups(parentGroupID, &gitlab.ListSubGroupsOptions{
		Search:       gitlab.Ptr(groupName),
		AllAvailable: gitlab.Ptr(true),
		ListOptions:  gitlab.ListOptions{PerPage: 100},
	})
	if err != nil {
		return nil, fmt.Errorf("list subgroups of %d: %w", parentGroupID, err)
	}

	for _, group := range groups {
		if group.Name == groupName && group.ParentID == parentGroupID {
			return group, nil
		}
	}
	return nil, nil
}

func getSubGroup(groupName string, parentGroupID int64) (*gitlab.Group, error) {
	group, err := findSubGroup(groupName, parentGroupID)
	if err != nil {
		return nil, err
	}
	if group == nil {
		return nil, fmt.Errorf("subgroup %q not found under group %d", groupName, parentGroupID)
	}
	return group, nil
}

func createCourseIterationGroup(courseIteration string, parentID int64) (*gitlab.Group, error) {
	existing, err := findSubGroup(courseIteration, parentID)
	if err != nil {
		return nil, fmt.Errorf("check if course iteration group %q exists: %w", courseIteration, err)
	}
	if existing != nil {
		return existing, nil
	}

	git, err := getClient()
	if err != nil {
		return nil, err
	}

	group, _, err := git.Groups.CreateGroup(&gitlab.CreateGroupOptions{
		Name:                  gitlab.Ptr(courseIteration),
		ParentID:              gitlab.Ptr(parentID),
		ProjectCreationLevel:  gitlab.Ptr(gitlab.MaintainerProjectCreation),
		SubGroupCreationLevel: gitlab.Ptr(gitlab.MaintainerSubGroupCreationLevelValue),
		AutoDevopsEnabled:     gitlab.Ptr(false),
		Path:                  gitlab.Ptr(courseIteration),
	})
	if err != nil {
		if !isAlreadyExistsError(err) {
			return nil, fmt.Errorf("create course iteration group %q: %w", courseIteration, err)
		}
		// Race: another request created it between our check and create
		existing, findErr := findSubGroup(courseIteration, parentID)
		if findErr != nil || existing == nil {
			return nil, fmt.Errorf("course iteration group %q conflict but not found: %w", courseIteration, err)
		}
		return existing, nil
	}

	return group, nil
}

func createDeveloperTopLevelGroup(parentGroupID int64) (*gitlab.Group, error) {
	return createGitlabGroup(parentGroupID, "developer", gitlab.NoOneProjectCreation, gitlab.OwnerSubGroupCreationLevelValue)
}

// create Groups for tutors and coaches
func createTeachingGroup(parentGroupID int64, groupName string) (*gitlab.Group, error) {
	return createGitlabGroup(parentGroupID, groupName, gitlab.DeveloperProjectCreation, gitlab.OwnerSubGroupCreationLevelValue)
}

func createGitlabGroup(parentGroupID int64, groupName string, projectCreationLevel gitlab.ProjectCreationLevelValue, subGroupCreationLevel gitlab.SubGroupCreationLevelValue) (*gitlab.Group, error) {
	existing, err := findSubGroup(groupName, parentGroupID)
	if err != nil {
		return nil, fmt.Errorf("check if group %q exists: %w", groupName, err)
	}
	if existing != nil {
		return existing, nil
	}

	git, err := getClient()
	if err != nil {
		return nil, err
	}

	// Create a group
	group, _, err := git.Groups.CreateGroup(&gitlab.CreateGroupOptions{
		Name:                  gitlab.Ptr(groupName),
		ParentID:              gitlab.Ptr(parentGroupID),
		ProjectCreationLevel:  gitlab.Ptr(projectCreationLevel),
		SubGroupCreationLevel: gitlab.Ptr(subGroupCreationLevel),
		AutoDevopsEnabled:     gitlab.Ptr(false),
		Path:                  gitlab.Ptr(groupName),
	})

	if err != nil {
		if !isAlreadyExistsError(err) {
			return nil, fmt.Errorf("create group %q: %w", groupName, err)
		}
		// Race: another request created it between our check and create
		existing, findErr := findSubGroup(groupName, parentGroupID)
		if findErr != nil || existing == nil {
			return nil, fmt.Errorf("group %q conflict but not found: %w", groupName, err)
		}
		return existing, nil
	}

	return group, nil
}

func getUserID(username string) (*gitlab.User, error) {
	git, err := getClient()
	if err != nil {
		return nil, fmt.Errorf("get client for user lookup %q: %w", username, err)
	}

	userOpts := &gitlab.ListUsersOptions{
		Username: gitlab.Ptr(username),
	}

	users, _, err := git.Users.ListUsers(userOpts)
	if err != nil {
		return nil, fmt.Errorf("list users for %q: %w", username, err)
	}

	if len(users) != 1 || users[0] == nil {
		return nil, fmt.Errorf("user %q not found on GitLab", username)
	}
	return users[0], nil
}

func CreateStudentProject(repoName string, devID, tutorID, introCourseID int64, introCourseGroupPath string, devGroupID int64, studentName, submissionDeadline string) error {
	git, err := getClient()
	if err != nil {
		return fmt.Errorf("get client for project %q: %w", repoName, err)
	}

	// 1. Create project (idempotent: handle conflict by fetching existing)
	p := &gitlab.CreateProjectOptions{
		Name:                             gitlab.Ptr(repoName),
		NamespaceID:                      gitlab.Ptr(introCourseID),
		SharedRunnersEnabled:             gitlab.Ptr(true),
		OnlyAllowMergeIfPipelineSucceeds: gitlab.Ptr(true),
		BuildsAccessLevel:                gitlab.Ptr(gitlab.PrivateAccessControl),
		ContainerRegistryAccessLevel:     gitlab.Ptr(gitlab.DisabledAccessControl),
		EnvironmentsAccessLevel:          gitlab.Ptr(gitlab.DisabledAccessControl), // disable environments
		FeatureFlagsAccessLevel:          gitlab.Ptr(gitlab.DisabledAccessControl), // disable feature flags
		ForkingAccessLevel:               gitlab.Ptr(gitlab.DisabledAccessControl), // disable forking
		InfrastructureAccessLevel:        gitlab.Ptr(gitlab.DisabledAccessControl), // disable infrastructure
		PackagesEnabled:                  gitlab.Ptr(false),                        // disable packages
		ReleasesAccessLevel:              gitlab.Ptr(gitlab.DisabledAccessControl), // disable releases
		SecurityAndComplianceAccessLevel: gitlab.Ptr(gitlab.DisabledAccessControl), // disable security & compliance
		SnippetsAccessLevel:              gitlab.Ptr(gitlab.DisabledAccessControl), // disable snippets
		WikiAccessLevel:                  gitlab.Ptr(gitlab.DisabledAccessControl), // disable wiki
		RequirementsAccessLevel:          gitlab.Ptr(gitlab.DisabledAccessControl), // disable requirements
		ModelExperimentsAccessLevel:      gitlab.Ptr(gitlab.DisabledAccessControl), // disable model experiments
		ModelRegistryAccessLevel:         gitlab.Ptr(gitlab.DisabledAccessControl), // disable model registry
		PagesAccessLevel:                 gitlab.Ptr(gitlab.DisabledAccessControl), // disable pages
		MonitorAccessLevel:               gitlab.Ptr(gitlab.DisabledAccessControl), // disable monitor
	}

	project, _, err := git.Projects.CreateProject(p)
	if err != nil {
		if !isAlreadyExistsError(err) {
			return fmt.Errorf("create project %q: %w", repoName, err)
		}
		// Project already exists — fetch by deterministic path
		projectPath := introCourseGroupPath + "/" + repoName
		project, _, err = git.Projects.GetProject(projectPath, nil)
		if err != nil {
			return fmt.Errorf("fetch existing project %q: %w", projectPath, err)
		}
		log.WithField("project", repoName).Info("project already exists, continuing with setup")
	}

	// 2. Create project files (idempotent: skip files that already exist)
	err = createProjectFiles(git, project.ID, repoName, studentName, submissionDeadline)
	if err != nil {
		return err
	}

	// 3. Branch protection (idempotent: skip if already protected)
	_, _, err = git.ProtectedBranches.ProtectRepositoryBranches(project.ID, &gitlab.ProtectRepositoryBranchesOptions{
		Name:             gitlab.Ptr("main"),
		PushAccessLevel:  gitlab.Ptr(gitlab.MaintainerPermissions),
		MergeAccessLevel: gitlab.Ptr(gitlab.DeveloperPermissions),
	})
	if err != nil && !isAlreadyExistsError(err) {
		return fmt.Errorf("protect branch for %q: %w", repoName, err)
	}

	// 4. Issue board (idempotent: reuse existing board, add missing lists)
	err = ensureIssueBoard(git, project.ID, repoName)
	if err != nil {
		return err
	}

	// 5. Members (idempotent: skip if already a member)
	// Last step before approval rule as this might fail if the tutor is already a member of a "higher" group
	err = addProjectMembers(git, project.ID, repoName, tutorID, devID, devGroupID)
	if err != nil {
		return err
	}

	// 6. Approval rule (idempotent: skip if "Tutor Approval" rule exists)
	err = ensureApprovalRule(git, project.ID, repoName, tutorID)
	if err != nil {
		return err
	}

	return nil
}

func addProjectMembers(git *gitlab.Client, projectID int64, repoName string, tutorID, devID, devGroupID int64) error {
	// Add student to the project
	_, _, err := git.ProjectMembers.AddProjectMember(projectID, &gitlab.AddProjectMemberOptions{
		UserID:      gitlab.Ptr(devID),
		AccessLevel: gitlab.Ptr(gitlab.DeveloperPermissions),
	})
	if err != nil && !isAlreadyExistsError(err) {
		return fmt.Errorf("add student %d to project %q: %w", devID, repoName, err)
	}

	// Add student to the developer group
	_, _, err = git.GroupMembers.AddGroupMember(devGroupID, &gitlab.AddGroupMemberOptions{
		UserID:      gitlab.Ptr(devID),
		AccessLevel: gitlab.Ptr(gitlab.DeveloperPermissions),
	})
	if err != nil && !isAlreadyExistsError(err) {
		return fmt.Errorf("add student %d to developer group for %q: %w", devID, repoName, err)
	}

	// Add tutor to the project
	_, _, err = git.ProjectMembers.AddProjectMember(projectID, &gitlab.AddProjectMemberOptions{
		UserID:      gitlab.Ptr(tutorID),
		AccessLevel: gitlab.Ptr(gitlab.DeveloperPermissions),
	})
	if err != nil && !isAlreadyExistsError(err) {
		return fmt.Errorf("add tutor %d to project %q: %w", tutorID, repoName, err)
	}

	return nil
}

func ensureIssueBoard(git *gitlab.Client, projectID int64, repoName string) error {
	boards, _, err := git.Boards.ListIssueBoards(projectID, nil)
	if err != nil {
		return fmt.Errorf("list issue boards for %q: %w", repoName, err)
	}

	// Find existing board by name, or create one.
	// GitLab allows duplicate board names, so we search first to avoid
	// creating duplicates on partial-failure retries.
	var board *gitlab.IssueBoard
	for _, b := range boards {
		if b.Name == "Issue Board" {
			board = b
			break
		}
	}
	if board == nil {
		board, _, err = git.Boards.CreateIssueBoard(projectID, &gitlab.CreateIssueBoardOptions{
			Name: gitlab.Ptr("Issue Board"),
		})
		if err != nil {
			return fmt.Errorf("create issue board for %q: %w", repoName, err)
		}
	}

	// Add lists individually (idempotent: skip if label list already exists on this board)
	hasLabel := func(labelID int64) bool {
		for _, l := range board.Lists {
			if l.Label != nil && l.Label.ID == labelID {
				return true
			}
		}
		return false
	}

	if !hasLabel(IN_PROGRESS_LABEL_ID) {
		_, _, err = git.Boards.CreateIssueBoardList(projectID, board.ID, &gitlab.CreateIssueBoardListOptions{
			LabelID: gitlab.Ptr(IN_PROGRESS_LABEL_ID),
		})
		if err != nil && !isAlreadyExistsError(err) {
			return fmt.Errorf("create 'In Progress' board list for %q: %w", repoName, err)
		}
	}

	if !hasLabel(IN_REVIEW_LABEL_ID) {
		_, _, err = git.Boards.CreateIssueBoardList(projectID, board.ID, &gitlab.CreateIssueBoardListOptions{
			LabelID: gitlab.Ptr(IN_REVIEW_LABEL_ID),
		})
		if err != nil && !isAlreadyExistsError(err) {
			return fmt.Errorf("create 'In Review' board list for %q: %w", repoName, err)
		}
	}

	return nil
}

// createProjectFiles adds scaffold files to a student project.
// Existing files are skipped (create-if-absent), not overwritten.
func createProjectFiles(git *gitlab.Client, projectID int64, repoName, studentName, submissionDeadline string) error {
	// Add custom README
	_, _, err := git.RepositoryFiles.CreateFile(projectID, "README.md", &gitlab.CreateFileOptions{
		Branch:        gitlab.Ptr("main"),
		Content:       gitlab.Ptr(data.GetReadme(studentName, submissionDeadline)),
		CommitMessage: gitlab.Ptr("Add custom README"),
	})
	if err != nil && !isAlreadyExistsError(err) {
		return fmt.Errorf("create README for %q: %w", repoName, err)
	}

	// Add custom swiftlint
	_, _, err = git.RepositoryFiles.CreateFile(projectID, ".swiftlint.yml", &gitlab.CreateFileOptions{
		Branch:        gitlab.Ptr("main"),
		Content:       gitlab.Ptr(data.GetSwiftlint()),
		CommitMessage: gitlab.Ptr("Add custom .swiftlint.yml"),
	})
	if err != nil && !isAlreadyExistsError(err) {
		return fmt.Errorf("create .swiftlint.yml for %q: %w", repoName, err)
	}

	// Add custom gitignore
	_, _, err = git.RepositoryFiles.CreateFile(projectID, ".gitignore", &gitlab.CreateFileOptions{
		Branch:        gitlab.Ptr("main"),
		Content:       gitlab.Ptr(data.GetGitignore()),
		CommitMessage: gitlab.Ptr("Add custom .gitignore"),
	})
	if err != nil && !isAlreadyExistsError(err) {
		return fmt.Errorf("create .gitignore for %q: %w", repoName, err)
	}

	return nil
}

// ensureApprovalRule creates the "Tutor Approval" rule if it doesn't exist.
// GitLab does not enforce uniqueness on approval rule names, so we must
// check first — create-then-handle-conflict is not possible for this resource.
func ensureApprovalRule(git *gitlab.Client, projectID int64, repoName string, tutorID int64) error {
	rules, _, err := git.Projects.GetProjectApprovalRules(projectID, nil)
	if err != nil {
		return fmt.Errorf("list approval rules for %q: %w", repoName, err)
	}
	for _, r := range rules {
		if r.Name == "Tutor Approval" {
			return nil
		}
	}

	_, _, err = git.Projects.CreateProjectApprovalRule(projectID, &gitlab.CreateProjectLevelRuleOptions{
		Name:              gitlab.Ptr("Tutor Approval"),
		ApprovalsRequired: gitlab.Ptr(int64(1)),
		UserIDs:           gitlab.Ptr([]int64{tutorID}),
	})
	if err != nil {
		return fmt.Errorf("create approval rule for %q: %w", repoName, err)
	}

	return nil
}
