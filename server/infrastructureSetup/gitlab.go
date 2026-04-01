package infrastructureSetup

import (
	"fmt"

	"github.com/prompt-edu/prompt-intro-course/server/gitlabutil"
	log "github.com/sirupsen/logrus"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

const inProgressLabelID int64 = 53319
const inReviewLabelID int64 = 53320

// Shared constants and helpers — delegate to gitlabutil to avoid duplication.
var (
	aseGroupID                    = gitlabutil.ASEGroupID
	errGitLabClientNotInitialized = gitlabutil.ErrClientNotInitialized
)

func getClient() (*gitlab.Client, error) {
	if InfrastructureServiceSingleton.gitlabClient == nil {
		return nil, errGitLabClientNotInitialized
	}
	return InfrastructureServiceSingleton.gitlabClient, nil
}

func isAlreadyExistsError(err error) bool { return gitlabutil.IsAlreadyExistsError(err) }
func isNotFoundError(err error) bool      { return gitlabutil.IsNotFoundError(err) }

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

// createTeachingGroup creates a subgroup for tutors or coaches.
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

func getUser(username string) (*gitlab.User, error) {
	git, err := getClient()
	if err != nil {
		return nil, fmt.Errorf("get client for user lookup %q: %w", username, err)
	}
	return gitlabutil.GetUser(git, username)
}

// newCourseProjectOptions returns the shared project configuration for all
// course projects (student repos and the demo repo). Features unrelated to
// the intro course workflow are disabled to keep the UI clean for students.
// ciCDRepoPath is the full GitLab path to the shared CI/CD repo (e.g.
// "ase/ipraktikum/IOS25/introcourse/ci-cd"), used to set CIConfigPath.
func newCourseProjectOptions(name string, namespaceID int64, ciCDRepoPath string) *gitlab.CreateProjectOptions {
	return &gitlab.CreateProjectOptions{
		Name:        gitlab.Ptr(name),
		NamespaceID: gitlab.Ptr(namespaceID),

		// Git & merge settings
		Visibility:                                    gitlab.Ptr(gitlab.PrivateVisibility),
		MergeMethod:                                   gitlab.Ptr(gitlab.NoFastForwardMerge),
		SquashOption:                                  gitlab.Ptr(gitlab.SquashOptionDefaultOn),
		RemoveSourceBranchAfterMerge:                  gitlab.Ptr(true),
		OnlyAllowMergeIfPipelineSucceeds:              gitlab.Ptr(true),
		OnlyAllowMergeIfAllDiscussionsAreResolved:     gitlab.Ptr(true),

		// CI/CD — pipeline config lives in a shared repo
		CIConfigPath:         gitlab.Ptr(".gitlab-ci.yml@" + ciCDRepoPath),
		SharedRunnersEnabled: gitlab.Ptr(true),
		BuildsAccessLevel:    gitlab.Ptr(gitlab.PrivateAccessControl),

		// Disable unneeded features to keep the UI clean for students
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

// configureProject applies the shared setup steps to any course project:
// template files, branch protection, issue board, approval config, daily issues.
// All steps are idempotent — safe to call on both new and existing projects.
func configureProject(git *gitlab.Client, projectID int64, projectName string, vars templateVars) error {
	// Template files (idempotent: skip files that already exist)
	err := createProjectFiles(git, projectID, projectName, vars)
	if err != nil {
		return err
	}

	// Branch protection — GitLab auto-protects 'main' with default settings
	// when the first commit is pushed, so we must unprotect first to apply our
	// desired access levels. Push is set to NoPermissions to force all changes
	// through merge requests; merge access is Developer (tutors are Maintainer).
	_, unprotectErr := git.ProtectedBranches.UnprotectRepositoryBranches(projectID, "main")
	if unprotectErr != nil && !isNotFoundError(unprotectErr) {
		return fmt.Errorf("unprotect branch for %q: %w", projectName, unprotectErr)
	}
	_, _, err = git.ProtectedBranches.ProtectRepositoryBranches(projectID, &gitlab.ProtectRepositoryBranchesOptions{
		Name:             gitlab.Ptr("main"),
		PushAccessLevel:  gitlab.Ptr(gitlab.NoPermissions),
		MergeAccessLevel: gitlab.Ptr(gitlab.DeveloperPermissions),
		AllowForcePush:   gitlab.Ptr(false),
	})
	if err != nil {
		return fmt.Errorf("protect branch for %q: %w", projectName, err)
	}

	// Issue board (idempotent: reuse existing board, add missing lists)
	err = ensureIssueBoard(git, projectID, projectName)
	if err != nil {
		return err
	}

	// Approval configuration (reset approvals on push, prevent self-approval)
	err = ensureApprovalConfiguration(git, projectID, projectName)
	if err != nil {
		return err
	}

	// Daily issues (non-fatal: log and continue)
	if err = createDailyIssues(git, projectID, projectName); err != nil {
		log.WithError(err).WithField("project", projectName).Warn("Failed to create daily issues (non-fatal)")
	}

	return nil
}

// StudentProjectParams bundles the parameters for CreateStudentProject to
// avoid a long positional parameter list with multiple same-typed values.
type StudentProjectParams struct {
	RepoName             string
	DevID                int64
	TutorSubgroupID      int64
	TutorSubgroupPath    string
	TutorsGroupID        int64
	DevGroupID           int64
	IntroCourseGroupPath string
	StudentName          string
	SubmissionDeadline   string
}

func CreateStudentProject(p StudentProjectParams) error {
	git, err := getClient()
	if err != nil {
		return fmt.Errorf("get client for project %q: %w", p.RepoName, err)
	}

	ciCDRepoPath := p.IntroCourseGroupPath + "/ci-cd"

	// 1. Create project (idempotent: handle conflict by fetching existing)
	project, err := createOrGetProject(git, newCourseProjectOptions(p.RepoName, p.TutorSubgroupID, ciCDRepoPath), p.TutorSubgroupPath)
	if err != nil {
		return err
	}

	// 2. Shared project setup (files, branch protection, board, approvals, issues)
	err = configureProject(git, project.ID, p.RepoName, templateVars{
		StudentName:        p.StudentName,
		SubmissionDeadline: p.SubmissionDeadline,
	})
	if err != nil {
		return err
	}

	// 3. Members (idempotent: skip if already a member)
	err = addProjectMembers(git, project.ID, p.RepoName, p.DevID, p.DevGroupID)
	if err != nil {
		return err
	}

	// 4. Approval rule (idempotent: skip if "Tutor Approval" rule exists)
	err = ensureApprovalRule(git, project.ID, p.RepoName, p.TutorsGroupID)
	if err != nil {
		return err
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

	if !hasLabel(inProgressLabelID) {
		_, _, err = git.Boards.CreateIssueBoardList(projectID, board.ID, &gitlab.CreateIssueBoardListOptions{
			LabelID: gitlab.Ptr(inProgressLabelID),
		})
		if err != nil && !isAlreadyExistsError(err) {
			return fmt.Errorf("create 'In Progress' board list for %q: %w", repoName, err)
		}
	}

	if !hasLabel(inReviewLabelID) {
		_, _, err = git.Boards.CreateIssueBoardList(projectID, board.ID, &gitlab.CreateIssueBoardListOptions{
			LabelID: gitlab.Ptr(inReviewLabelID),
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
// The rule uses the tutors group so that any tutor can approve any student's MR,
// not just the assigned tutor. GitLab does not enforce uniqueness on approval
// rule names, so we must check first — create-then-handle-conflict is not
// possible for this resource.
func ensureApprovalRule(git *gitlab.Client, projectID int64, repoName string, tutorsGroupID int64) error {
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
		GroupIDs:          gitlab.Ptr([]int64{tutorsGroupID}),
	})
	if err != nil {
		return fmt.Errorf("create approval rule for %q: %w", repoName, err)
	}

	return nil
}

// ensureApprovalConfiguration sets project-level approval policies:
// - Reset approvals when new commits are pushed (prevents stale approvals)
// - Prevent MR authors from approving their own MRs
// - Prevent committers from approving MRs they contributed to
// - Prevent students from overriding approval rules on their MRs
// - Only reset code owner approvals when relevant files change
// Idempotent: safe to call multiple times.
func ensureApprovalConfiguration(git *gitlab.Client, projectID int64, repoName string) error {
	_, _, err := git.Projects.ChangeApprovalConfiguration(projectID, &gitlab.ChangeApprovalConfigurationOptions{
		ResetApprovalsOnPush:                         gitlab.Ptr(true),
		MergeRequestsAuthorApproval:                  gitlab.Ptr(false),
		MergeRequestsDisableCommittersApproval:       gitlab.Ptr(true),
		DisableOverridingApproversPerMergeRequest:     gitlab.Ptr(true),
		SelectiveCodeOwnerRemovals:                    gitlab.Ptr(true),
	})
	if err != nil {
		return fmt.Errorf("configure approval settings for %q: %w", repoName, err)
	}
	return nil
}

// getOrCreateTutorSubgroup returns the GitLab group ID and path for a tutor's
// subgroup inside the Introcourse group. Creates the subgroup and adds the
// tutor as Maintainer if it doesn't exist yet. GitLab is the sole source of
// truth — no DB caching needed since this runs once per student setup.
func getOrCreateTutorSubgroup(tutorGitlabUsername, tutorFirstName, tutorLastName string, tutorGitlabUserID, introCourseGroupID int64) (int64, string, error) {
	git, err := getClient()
	if err != nil {
		return 0, "", err
	}

	// 1. Check if subgroup already exists in GitLab
	existing, err := findSubGroup(tutorGitlabUsername, introCourseGroupID)
	if err != nil {
		return 0, "", fmt.Errorf("check tutor subgroup %q: %w", tutorGitlabUsername, err)
	}
	if existing != nil {
		return existing.ID, existing.FullPath, nil
	}

	// 2. Create subgroup (display name = "FirstName LastName", path = gitlab_username)
	displayName := tutorFirstName + " " + tutorLastName
	group, _, err := git.Groups.CreateGroup(&gitlab.CreateGroupOptions{
		Name:                  gitlab.Ptr(displayName),
		Path:                  gitlab.Ptr(tutorGitlabUsername),
		ParentID:              gitlab.Ptr(introCourseGroupID),
		ProjectCreationLevel:  gitlab.Ptr(gitlab.MaintainerProjectCreation),
		SubGroupCreationLevel: gitlab.Ptr(gitlab.OwnerSubGroupCreationLevelValue),
		AutoDevopsEnabled:     gitlab.Ptr(false),
	})
	if err != nil {
		if !isAlreadyExistsError(err) {
			return 0, "", fmt.Errorf("create tutor subgroup %q: %w", tutorGitlabUsername, err)
		}
		// Race: another request created it between our check and create
		raceGroup, findErr := findSubGroup(tutorGitlabUsername, introCourseGroupID)
		if findErr != nil || raceGroup == nil {
			return 0, "", fmt.Errorf("tutor subgroup %q conflict but not found: %w", tutorGitlabUsername, err)
		}
		return raceGroup.ID, raceGroup.FullPath, nil
	}

	// 3. Add tutor as Maintainer on subgroup (inherits to all projects)
	_, _, err = git.GroupMembers.AddGroupMember(group.ID, &gitlab.AddGroupMemberOptions{
		UserID:      gitlab.Ptr(tutorGitlabUserID),
		AccessLevel: gitlab.Ptr(gitlab.MaintainerPermissions),
	})
	if err != nil && !isAlreadyExistsError(err) {
		return 0, "", fmt.Errorf("add tutor as maintainer on subgroup %q: %w", tutorGitlabUsername, err)
	}

	return group.ID, group.FullPath, nil
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

// createCICDProject creates the shared CI/CD project in the Introcourse group
// and populates it with pipeline config files from the teaching material repo's
// ci_cd/ directory. All course projects reference this project's .gitlab-ci.yml
// via CIConfigPath. Fully idempotent: safe to re-run on an existing course.
func createCICDProject(git *gitlab.Client, introCourseGroupID int64, introCourseGroupPath string) error {
	const cicdProjectName = "ci-cd"

	project, err := createOrGetProject(git, &gitlab.CreateProjectOptions{
		Name:                 gitlab.Ptr(cicdProjectName),
		NamespaceID:          gitlab.Ptr(introCourseGroupID),
		Visibility:           gitlab.Ptr(gitlab.PrivateVisibility),
		SharedRunnersEnabled: gitlab.Ptr(true),
		BuildsAccessLevel:    gitlab.Ptr(gitlab.PrivateAccessControl),
		// Initialize with an empty repo so the default branch exists
		InitializeWithReadme: gitlab.Ptr(true),
	}, introCourseGroupPath)
	if err != nil {
		return fmt.Errorf("create CI/CD project: %w", err)
	}

	// Push pipeline config files from teaching material repo
	svc := InfrastructureServiceSingleton
	if svc.teachingMaterialProjectID == "" {
		return nil // no teaching material configured, skip
	}

	cicdFiles, err := svc.cicd.get(git, svc.teachingMaterialProjectID)
	if err != nil {
		return fmt.Errorf("fetch CI/CD files: %w", err)
	}
	if len(cicdFiles) == 0 {
		log.Warn("No CI/CD files found in teaching material repo ci_cd/ directory; CI/CD project will be empty")
		return nil
	}

	var actions []*gitlab.CommitActionOptions
	for _, f := range cicdFiles {
		action := &gitlab.CommitActionOptions{
			Action:   gitlab.Ptr(gitlab.FileCreate),
			FilePath: gitlab.Ptr(f.Path),
			Content:  gitlab.Ptr(f.Content),
		}
		if f.ExecuteFilemode {
			action.ExecuteFilemode = gitlab.Ptr(true)
		}
		actions = append(actions, action)
	}

	_, _, err = git.Commits.CreateCommit(project.ID, &gitlab.CreateCommitOptions{
		Branch:        gitlab.Ptr("main"),
		CommitMessage: gitlab.Ptr("Initialize CI/CD pipeline from course template"),
		Actions:       actions,
	})
	if err != nil && !isAlreadyExistsError(err) {
		return fmt.Errorf("push CI/CD files to project: %w", err)
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
	ciCDRepoPath := introCourseGroupPath + "/ci-cd"

	project, err := createOrGetProject(git, newCourseProjectOptions(demoProjectName, introCourseGroupID, ciCDRepoPath), introCourseGroupPath)
	if err != nil {
		return err
	}

	// Shared project setup (files, branch protection, board, approvals, issues)
	err = configureProject(git, project.ID, demoProjectName, templateVars{
		StudentName: "Demo",
	})
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

	return nil
}
