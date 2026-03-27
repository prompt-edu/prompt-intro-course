package infrastructureSetup

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	db "github.com/prompt-edu/prompt-intro-course/server/db/sqlc"
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

// newCourseProjectOptions returns the shared project configuration for all
// course projects (student repos and the demo repo). Features unrelated to
// the intro course workflow are disabled to keep the UI clean for students.
func newCourseProjectOptions(name string, namespaceID int64) *gitlab.CreateProjectOptions {
	return &gitlab.CreateProjectOptions{
		Name:                             gitlab.Ptr(name),
		NamespaceID:                      gitlab.Ptr(namespaceID),
		SharedRunnersEnabled:             gitlab.Ptr(true),
		OnlyAllowMergeIfPipelineSucceeds: gitlab.Ptr(true),
		BuildsAccessLevel:                gitlab.Ptr(gitlab.PrivateAccessControl),
		ContainerRegistryAccessLevel:     gitlab.Ptr(gitlab.DisabledAccessControl),
		EnvironmentsAccessLevel:          gitlab.Ptr(gitlab.DisabledAccessControl),
		FeatureFlagsAccessLevel:          gitlab.Ptr(gitlab.DisabledAccessControl),
		ForkingAccessLevel:               gitlab.Ptr(gitlab.DisabledAccessControl),
		InfrastructureAccessLevel:        gitlab.Ptr(gitlab.DisabledAccessControl),
		PackagesEnabled:                  gitlab.Ptr(false),
		ReleasesAccessLevel:              gitlab.Ptr(gitlab.DisabledAccessControl),
		SecurityAndComplianceAccessLevel: gitlab.Ptr(gitlab.DisabledAccessControl),
		SnippetsAccessLevel:              gitlab.Ptr(gitlab.DisabledAccessControl),
		WikiAccessLevel:                  gitlab.Ptr(gitlab.DisabledAccessControl),
		RequirementsAccessLevel:          gitlab.Ptr(gitlab.DisabledAccessControl),
		ModelExperimentsAccessLevel:      gitlab.Ptr(gitlab.DisabledAccessControl),
		ModelRegistryAccessLevel:         gitlab.Ptr(gitlab.DisabledAccessControl),
		PagesAccessLevel:                 gitlab.Ptr(gitlab.DisabledAccessControl),
		MonitorAccessLevel:               gitlab.Ptr(gitlab.DisabledAccessControl),
	}
}

// createOrGetProject creates a GitLab project or returns the existing one if
// it already exists. This is the shared create-or-fetch pattern used by both
// student and demo project creation.
func createOrGetProject(git *gitlab.Client, opts *gitlab.CreateProjectOptions, groupPath string) (*gitlab.Project, error) {
	project, _, err := git.Projects.CreateProject(opts)
	if err != nil {
		if !isAlreadyExistsError(err) {
			return nil, fmt.Errorf("create project %q: %w", *opts.Name, err)
		}
		projectPath := groupPath + "/" + *opts.Name
		project, _, err = git.Projects.GetProject(projectPath, nil)
		if err != nil {
			return nil, fmt.Errorf("fetch existing project %q: %w", projectPath, err)
		}
		log.WithField("project", *opts.Name).Info("project already exists, continuing with setup")
	}
	return project, nil
}

func CreateStudentProject(repoName string, devID, tutorID, introCourseID int64, introCourseGroupPath string, devGroupID int64, studentName, submissionDeadline string) error {
	git, err := getClient()
	if err != nil {
		return fmt.Errorf("get client for project %q: %w", repoName, err)
	}

	// 1. Create project (idempotent: handle conflict by fetching existing)
	project, err := createOrGetProject(git, newCourseProjectOptions(repoName, introCourseID), introCourseGroupPath)
	if err != nil {
		return err
	}

	// 2. Create project files (idempotent: skip files that already exist)
	err = createProjectFiles(git, project.ID, repoName, templateVars{
		StudentName:        studentName,
		SubmissionDeadline: submissionDeadline,
	})
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
	err = addProjectMembers(git, project.ID, repoName, devID, devGroupID)
	if err != nil {
		return err
	}

	// 6. Approval rule (idempotent: skip if "Tutor Approval" rule exists)
	err = ensureApprovalRule(git, project.ID, repoName, tutorID)
	if err != nil {
		return err
	}

	// 7. Daily issues (non-fatal: log and continue)
	if err = createDailyIssues(git, project.ID, repoName); err != nil {
		log.WithError(err).WithField("project", repoName).Warn("Failed to create daily issues (non-fatal)")
	}

	return nil
}

func addProjectMembers(git *gitlab.Client, projectID int64, repoName string, devID, devGroupID int64) error {
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

	// Tutor access is inherited from the tutor subgroup (Maintainer permission)

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

// createProjectFiles fetches template files from the teaching material repo,
// applies variable substitution, and pushes all files in a single atomic commit.
// If the project already has commits on its default branch, file creation is
// skipped entirely to maintain idempotency.
func createProjectFiles(git *gitlab.Client, projectID int64, repoName string, vars templateVars) error {
	svc := InfrastructureServiceSingleton
	if svc.teachingMaterialProjectID == "" {
		return fmt.Errorf("GITLAB_TEACHING_MATERIAL_PROJECT_ID not configured")
	}

	templates, err := svc.templates.get(git, svc.teachingMaterialProjectID)
	if err != nil {
		return fmt.Errorf("fetch templates for %q: %w", repoName, err)
	}

	actions := make([]*gitlab.CommitActionOptions, 0, len(templates))
	for _, tmpl := range templates {
		content := applyTemplateVars(tmpl.Content, vars)
		action := &gitlab.CommitActionOptions{
			Action:   gitlab.Ptr(gitlab.FileCreate),
			FilePath: gitlab.Ptr(tmpl.Path),
			Content:  gitlab.Ptr(content),
		}
		if tmpl.ExecuteFilemode {
			action.ExecuteFilemode = gitlab.Ptr(true)
		}
		actions = append(actions, action)
	}

	_, _, err = git.Commits.CreateCommit(projectID, &gitlab.CreateCommitOptions{
		Branch:        gitlab.Ptr("main"),
		CommitMessage: gitlab.Ptr("Initialize repository from course template"),
		Actions:       actions,
	})
	if err != nil && !isAlreadyExistsError(err) {
		return fmt.Errorf("initialize %q from template: %w", repoName, err)
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

// getOrCreateTutorSubgroup returns the GitLab group ID and path for a tutor's
// subgroup inside the Introcourse group. Creates the subgroup and adds the
// tutor as Maintainer if it doesn't exist yet. The mapping is cached in the DB
// for fast lookups on subsequent calls.
func getOrCreateTutorSubgroup(ctx context.Context, coursePhaseID uuid.UUID, tutorID pgtype.UUID, tutorGitlabUsername, tutorFirstName, tutorLastName string, tutorGitlabUserID, introCourseGroupID int64) (int64, string, error) {
	svc := InfrastructureServiceSingleton

	// 1. Check DB for existing mapping
	subgroup, err := svc.queries.GetTutorGitlabSubgroup(ctx, db.GetTutorGitlabSubgroupParams{
		CoursePhaseID: coursePhaseID,
		TutorID:       uuid.UUID(tutorID.Bytes),
	})
	if err == nil {
		return subgroup.GitlabGroupID, subgroup.GitlabGroupPath, nil
	}

	git, err := getClient()
	if err != nil {
		return 0, "", err
	}

	// 2. Check if subgroup already exists in GitLab
	existing, err := findSubGroup(tutorGitlabUsername, introCourseGroupID)
	if err != nil {
		return 0, "", fmt.Errorf("check tutor subgroup %q: %w", tutorGitlabUsername, err)
	}

	var groupID int64
	var groupPath string

	if existing != nil {
		groupID = existing.ID
		groupPath = existing.FullPath
	} else {
		// 3. Create subgroup (display name = "FirstName LastName", path = gitlab_username)
		displayName := tutorFirstName + " " + tutorLastName
		group, _, createErr := git.Groups.CreateGroup(&gitlab.CreateGroupOptions{
			Name:                  gitlab.Ptr(displayName),
			Path:                  gitlab.Ptr(tutorGitlabUsername),
			ParentID:              gitlab.Ptr(introCourseGroupID),
			ProjectCreationLevel:  gitlab.Ptr(gitlab.MaintainerProjectCreation),
			SubGroupCreationLevel: gitlab.Ptr(gitlab.OwnerSubGroupCreationLevelValue),
			AutoDevopsEnabled:     gitlab.Ptr(false),
		})
		if createErr != nil {
			if !isAlreadyExistsError(createErr) {
				return 0, "", fmt.Errorf("create tutor subgroup %q: %w", tutorGitlabUsername, createErr)
			}
			// Race: another request created it between our check and create
			raceGroup, findErr := findSubGroup(tutorGitlabUsername, introCourseGroupID)
			if findErr != nil || raceGroup == nil {
				return 0, "", fmt.Errorf("tutor subgroup %q conflict but not found: %w", tutorGitlabUsername, createErr)
			}
			groupID = raceGroup.ID
			groupPath = raceGroup.FullPath
		} else {
			groupID = group.ID
			groupPath = group.FullPath
		}

		// 4. Add tutor as Maintainer on subgroup (inherits to all projects)
		_, _, err = git.GroupMembers.AddGroupMember(groupID, &gitlab.AddGroupMemberOptions{
			UserID:      gitlab.Ptr(tutorGitlabUserID),
			AccessLevel: gitlab.Ptr(gitlab.MaintainerPermissions),
		})
		if err != nil && !isAlreadyExistsError(err) {
			return 0, "", fmt.Errorf("add tutor as maintainer on subgroup %q: %w", tutorGitlabUsername, err)
		}
	}

	// 5. Store in DB for future lookups
	_ = svc.queries.CreateTutorGitlabSubgroup(ctx, db.CreateTutorGitlabSubgroupParams{
		CoursePhaseID:   coursePhaseID,
		TutorID:         uuid.UUID(tutorID.Bytes),
		GitlabGroupID:   groupID,
		GitlabGroupPath: groupPath,
	})

	return groupID, groupPath, nil
}

// createDailyIssues creates all daily issues from the teaching material repo's
// daily_issues/ directory. Each .md file becomes a GitLab issue (title from
// first # heading, description from remaining content). Existing issues with
// matching titles are skipped for idempotency.
func createDailyIssues(git *gitlab.Client, projectID int64, repoName string) error {
	svc := InfrastructureServiceSingleton
	if svc.teachingMaterialProjectID == "" {
		return nil // no teaching material configured, skip silently
	}

	templates, err := svc.issues.get(git, svc.teachingMaterialProjectID)
	if err != nil {
		return fmt.Errorf("fetch issue templates: %w", err)
	}

	if len(templates) == 0 {
		return nil
	}

	// Fetch existing issue titles for idempotency check
	existingTitles := make(map[string]bool)
	existingIssues, _, err := git.Issues.ListProjectIssues(projectID, &gitlab.ListProjectIssuesOptions{
		ListOptions: gitlab.ListOptions{PerPage: 100},
	})
	if err != nil {
		return fmt.Errorf("list existing issues for %q: %w", repoName, err)
	}
	for _, issue := range existingIssues {
		existingTitles[issue.Title] = true
	}

	for _, tmpl := range templates {
		if existingTitles[tmpl.Title] {
			continue
		}
		_, _, err := git.Issues.CreateIssue(projectID, &gitlab.CreateIssueOptions{
			Title:       gitlab.Ptr(tmpl.Title),
			Description: gitlab.Ptr(tmpl.Description),
		})
		if err != nil {
			// Log and continue — one failed issue should not block the others
			log.WithFields(log.Fields{
				"issue":   tmpl.Title,
				"project": repoName,
			}).WithError(err).Warn("Failed to create daily issue")
			continue
		}
	}

	return nil
}

// createDemoProject creates a "demo" project in the Introcourse group,
// initialized from the same template as student repos. This gives
// instructors a reference repository for live demonstrations and testing.
// The project is shared with the tutors group so all tutors have access.
// Fully idempotent: safe to re-run on an existing course.
func createDemoProject(git *gitlab.Client, introCourseGroupID int64, introCourseGroupPath string, tutorsGroupID int64) error {
	const demoProjectName = "demo"

	project, err := createOrGetProject(git, newCourseProjectOptions(demoProjectName, introCourseGroupID), introCourseGroupPath)
	if err != nil {
		return err
	}

	// Push template files with demo-specific substitutions
	err = createProjectFiles(git, project.ID, demoProjectName, templateVars{
		StudentName: "Demo",
	})
	if err != nil {
		return err
	}

	// Protect main branch (same as student repos)
	_, _, err = git.ProtectedBranches.ProtectRepositoryBranches(project.ID, &gitlab.ProtectRepositoryBranchesOptions{
		Name:             gitlab.Ptr("main"),
		PushAccessLevel:  gitlab.Ptr(gitlab.MaintainerPermissions),
		MergeAccessLevel: gitlab.Ptr(gitlab.DeveloperPermissions),
	})
	if err != nil && !isAlreadyExistsError(err) {
		return fmt.Errorf("protect branch for %q: %w", demoProjectName, err)
	}

	// Issue board (same as student repos)
	err = ensureIssueBoard(git, project.ID, demoProjectName)
	if err != nil {
		return err
	}

	// Share with tutors group so all tutors can access the demo
	_, err = git.Projects.ShareProjectWithGroup(project.ID, &gitlab.ShareWithGroupOptions{
		GroupID:     gitlab.Ptr(tutorsGroupID),
		GroupAccess: gitlab.Ptr(gitlab.DeveloperPermissions),
	})
	if err != nil && !isAlreadyExistsError(err) {
		return fmt.Errorf("share %q with tutors group: %w", demoProjectName, err)
	}

	// Daily issues (non-fatal)
	if err = createDailyIssues(git, project.ID, demoProjectName); err != nil {
		log.WithError(err).WithField("project", demoProjectName).Warn("Failed to create daily issues for demo (non-fatal)")
	}

	return nil
}
