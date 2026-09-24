package infrastructureSetup

import (
	"errors"
	"fmt"
	"strings"
	"unicode"

	"github.com/prompt-edu/prompt-intro-course/server/gitlabutil"
	log "github.com/sirupsen/logrus"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

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
		if (group.Name == groupName || group.Path == groupName) && group.ParentID == parentGroupID {
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
func newCourseProjectOptions(name, path string, namespaceID int64, ciCDRepoPath string) *gitlab.CreateProjectOptions {
	return &gitlab.CreateProjectOptions{
		Name:        gitlab.Ptr(name),
		Path:        gitlab.Ptr(path),
		NamespaceID: gitlab.Ptr(namespaceID),

		// Git & merge settings
		Visibility:                       gitlab.Ptr(gitlab.PrivateVisibility),
		MergeMethod:                      gitlab.Ptr(gitlab.NoFastForwardMerge),
		SquashOption:                     gitlab.Ptr(gitlab.SquashOptionDefaultOn),
		RemoveSourceBranchAfterMerge:     gitlab.Ptr(true),
		OnlyAllowMergeIfPipelineSucceeds: gitlab.Ptr(true),
		OnlyAllowMergeIfAllDiscussionsAreResolved: gitlab.Ptr(true),

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
		path := *opts.Name
		if opts.Path != nil {
			path = *opts.Path
		}
		projectPath := groupPath + "/" + path
		project, _, err = git.Projects.GetProject(projectPath, nil)
		if err != nil {
			return nil, fmt.Errorf("fetch existing project %q: %w", projectPath, err)
		}
		// GitLab can redirect an old path after a project is renamed. A reset
		// must never treat that archived project as the new demo.
		if !strings.EqualFold(project.PathWithNamespace, projectPath) {
			return nil, fmt.Errorf("project path %q resolves to %q; release the old path before retrying", projectPath, project.PathWithNamespace)
		}
		log.WithField("project", *opts.Name).Info("project already exists, continuing with setup")
	}
	return project, nil
}

// configureProjectWithMaterial applies the shared template, branch protection,
// issue board, approval, and daily issue setup to a course project.
func configureProjectWithMaterial(git *gitlab.Client, projectID int64, projectName string, vars templateVars, material *materialSnapshot) (resultErr error) {
	var templates []templateFile
	if material == nil {
		svc := InfrastructureServiceSingleton
		if svc.teachingMaterialProjectID == "" {
			return fmt.Errorf("GITLAB_TEACHING_MATERIAL_PROJECT_ID not configured")
		}
		var err error
		templates, err = svc.templates.get(git, svc.teachingMaterialProjectID)
		if err != nil {
			return fmt.Errorf("fetch templates for %q: %w", projectName, err)
		}
	} else {
		templates = material.templates
	}
	// Branch protection — GitLab auto-protects 'main' with default settings
	// when the first commit is pushed, so we must unprotect first to apply our
	// desired access levels. We unprotect BEFORE creating files so the initial
	// commit doesn't trigger default protection rules.
	_, unprotectErr := git.ProtectedBranches.UnprotectRepositoryBranches(projectID, "main")
	if unprotectErr != nil && !isNotFoundError(unprotectErr) {
		return fmt.Errorf("unprotect branch for %q: %w", projectName, unprotectErr)
	}
	protectionRestored := false
	// A failed template fetch or commit must not leave an existing project
	// writable. Retry protection before returning any setup error.
	defer func() {
		if protectionRestored {
			return
		}
		_, _, protectErr := git.ProtectedBranches.ProtectRepositoryBranches(projectID, &gitlab.ProtectRepositoryBranchesOptions{
			Name:             gitlab.Ptr("main"),
			PushAccessLevel:  gitlab.Ptr(gitlab.NoPermissions),
			MergeAccessLevel: gitlab.Ptr(gitlab.DeveloperPermissions),
			AllowForcePush:   gitlab.Ptr(false),
		})
		if protectErr != nil && !isAlreadyExistsError(protectErr) {
			log.WithError(protectErr).WithField("project", projectName).Error("failed to restore main branch protection")
			resultErr = errors.Join(resultErr, fmt.Errorf("restore main protection for %q: %w", projectName, protectErr))
		}
	}()

	// Template files (idempotent: skip files that already exist)
	err := createProjectFilesFromTemplates(git, projectID, projectName, vars, templates)
	if err != nil {
		return err
	}

	// Re-protect branch with our desired settings. Push is set to NoPermissions
	// to force all changes through merge requests; merge access is Developer
	// (tutors are Maintainer). Tolerate already-exists in case of retry.
	_, _, err = git.ProtectedBranches.ProtectRepositoryBranches(projectID, &gitlab.ProtectRepositoryBranchesOptions{
		Name:             gitlab.Ptr("main"),
		PushAccessLevel:  gitlab.Ptr(gitlab.NoPermissions),
		MergeAccessLevel: gitlab.Ptr(gitlab.DeveloperPermissions),
		AllowForcePush:   gitlab.Ptr(false),
	})
	if err != nil && !isAlreadyExistsError(err) {
		return fmt.Errorf("protect branch for %q: %w", projectName, err)
	}
	protectionRestored = true

	// Issue board — skipped; GitLab provides a default board and custom lists
	// add complexity with no clear benefit for the intro course workflow.

	// Approval configuration (reset approvals on push, prevent self-approval)
	err = ensureApprovalConfiguration(git, projectID, projectName)
	if err != nil {
		return err
	}

	// A repository without its course issues is incomplete. A retry will skip
	// issues that were created successfully and fill in any missing ones.
	if material == nil {
		err = createDailyIssues(git, projectID, projectName)
	} else {
		err = createDailyIssuesFromTemplates(git, projectID, projectName, material.issues)
	}
	if err != nil {
		return fmt.Errorf("create daily issues for %q: %w", projectName, err)
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

// GitLab project names must begin with a letter or digit and may only contain
// letters, digits, spaces, hyphens, underscores, periods, and plus signs.
func studentProjectDisplayName(studentName, login string) string {
	clean := func(value string) string {
		value = strings.Map(func(r rune) rune {
			if unicode.IsLetter(r) || unicode.IsDigit(r) || r == ' ' || r == '-' || r == '_' || r == '.' || r == '+' {
				return r
			}
			return ' '
		}, value)
		value = strings.Join(strings.Fields(value), " ")
		return strings.TrimLeftFunc(value, func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) })
	}
	return fmt.Sprintf("%s - %s", clean(studentName), clean(login))
}

func CreateStudentProject(p StudentProjectParams) error {
	return createStudentProjectWithMaterial(p, nil)
}

func createStudentProjectWithMaterial(p StudentProjectParams, material *materialSnapshot) error {
	if p.RepoName == "" || p.StudentName == "" {
		return fmt.Errorf("student project needs a university login and student name")
	}
	git, err := getClient()
	if err != nil {
		return fmt.Errorf("get client for project %q: %w", p.RepoName, err)
	}
	peerGroup, err := getOrCreatePeerReviewGroup(git, p.TutorSubgroupID, p.TutorSubgroupPath)
	if err != nil {
		return fmt.Errorf("prepare peer review group for %q: %w", p.RepoName, err)
	}

	ciCDRepoPath := p.IntroCourseGroupPath + "/ci-cd"

	// 1. Create project (idempotent: handle conflict by fetching existing)
	// Keep the university login as the stable URL path while showing both the
	// student's name and login in GitLab's project lists.
	displayName := studentProjectDisplayName(p.StudentName, p.RepoName)
	project, err := createOrGetProject(git, newCourseProjectOptions(displayName, p.RepoName, p.TutorSubgroupID, ciCDRepoPath), p.TutorSubgroupPath)
	if err != nil {
		return err
	}
	if project.Name != displayName {
		_, _, err = git.Projects.EditProject(project.ID, &gitlab.EditProjectOptions{Name: gitlab.Ptr(displayName)})
		if err != nil {
			return fmt.Errorf("update student project display name %q: %w", displayName, err)
		}
	}

	// 2. Shared project setup (files, branch protection, board, approvals, issues)
	err = configureProjectWithMaterial(git, project.ID, p.RepoName, templateVars{
		StudentName:        p.StudentName,
		SubmissionDeadline: p.SubmissionDeadline,
	}, material)
	if err != nil {
		return err
	}

	// 3. Members (idempotent: skip if already a member)
	err = addProjectMembers(git, project.ID, p.RepoName, p.DevID, p.DevGroupID)
	if err != nil {
		return err
	}
	if err = ensureGroupMember(git, peerGroup.ID, p.DevID, gitlab.ReporterPermissions); err != nil {
		return fmt.Errorf("add student to peer review group for %q: %w", p.RepoName, err)
	}
	if err = ensureProjectSharedWithPeerGroup(git, project.ID, peerGroup.ID); err != nil {
		return fmt.Errorf("share %q with peer reviewers: %w", p.RepoName, err)
	}

	// 4. Approval rule (idempotent: skip if "Tutor Approval" rule exists)
	err = ensureApprovalRule(git, project.ID, p.RepoName, p.TutorsGroupID)
	if err != nil {
		return err
	}
	if err = ensurePeerReviewRule(git, project.ID, p.RepoName, peerGroup.ID); err != nil {
		return err
	}

	return nil
}

// Each tutor group has one direct-membership peer group. It is shared into
// student projects as Reporter, while each repository owner gets Developer
// access directly. This allows peer approval without letting classmates push.
func getOrCreatePeerReviewGroup(git *gitlab.Client, tutorGroupID int64, tutorGroupPath string) (*gitlab.Group, error) {
	const groupPath = "peer-reviewers"
	fullPath := tutorGroupPath + "/" + groupPath
	group, _, err := git.Groups.GetGroup(fullPath, nil)
	if err == nil {
		if group.ParentID != tutorGroupID || !strings.EqualFold(group.FullPath, fullPath) {
			return nil, fmt.Errorf("peer review group path %q resolves to a different group", fullPath)
		}
		return group, nil
	}
	if !isNotFoundError(err) {
		return nil, err
	}
	group, _, err = git.Groups.CreateGroup(&gitlab.CreateGroupOptions{
		Name:                  gitlab.Ptr("Peer Reviewers"),
		Path:                  gitlab.Ptr(groupPath),
		ParentID:              gitlab.Ptr(tutorGroupID),
		Visibility:            gitlab.Ptr(gitlab.PrivateVisibility),
		ProjectCreationLevel:  gitlab.Ptr(gitlab.NoOneProjectCreation),
		SubGroupCreationLevel: gitlab.Ptr(gitlab.OwnerSubGroupCreationLevelValue),
	})
	if err == nil {
		return group, nil
	}
	if !isAlreadyExistsError(err) {
		return nil, err
	}
	group, _, err = git.Groups.GetGroup(fullPath, nil)
	if err != nil {
		return nil, fmt.Errorf("find peer review group after create conflict: %w", err)
	}
	if group.ParentID != tutorGroupID || !strings.EqualFold(group.FullPath, fullPath) {
		return nil, fmt.Errorf("peer review group path %q resolves to a different group", fullPath)
	}
	return group, nil
}

func ensureProjectSharedWithPeerGroup(git *gitlab.Client, projectID, peerGroupID int64) error {
	project, _, err := git.Projects.GetProject(projectID, nil)
	if err != nil {
		return err
	}
	for _, shared := range project.SharedWithGroups {
		if shared.GroupID == peerGroupID {
			if shared.GroupAccessLevel != int64(gitlab.ReporterPermissions) {
				return fmt.Errorf("peer group has access level %d; expected Reporter", shared.GroupAccessLevel)
			}
			return nil
		}
	}
	_, err = git.Projects.ShareProjectWithGroup(projectID, &gitlab.ShareWithGroupOptions{
		GroupID: gitlab.Ptr(peerGroupID), GroupAccess: gitlab.Ptr(gitlab.ReporterPermissions),
	})
	if err != nil && !isAlreadyExistsError(err) {
		return err
	}
	project, _, err = git.Projects.GetProject(projectID, nil)
	if err != nil {
		return err
	}
	for _, shared := range project.SharedWithGroups {
		if shared.GroupID == peerGroupID && shared.GroupAccessLevel == int64(gitlab.ReporterPermissions) {
			return nil
		}
	}
	return fmt.Errorf("GitLab did not confirm Reporter sharing for peer group %d", peerGroupID)
}

func ensurePeerReviewRule(git *gitlab.Client, projectID int64, repoName string, peerGroupID int64) error {
	branch, _, err := git.ProtectedBranches.GetProtectedBranch(projectID, "main")
	if err != nil {
		return fmt.Errorf("get protected main for peer review in %q: %w", repoName, err)
	}
	rules, _, err := git.Projects.GetProjectApprovalRules(projectID, nil)
	if err != nil {
		return fmt.Errorf("list peer review rules for %q: %w", repoName, err)
	}
	branchIDs := []int64{branch.ID}
	groupIDs := []int64{peerGroupID}
	for _, rule := range rules {
		if rule.Name != "Peer Review" {
			continue
		}
		if rule.RuleType == "regular" && rule.ApprovalsRequired == 0 &&
			len(rule.Groups) == 1 && approvalRuleIncludesGroup(rule, peerGroupID) &&
			len(rule.Users) == 0 && !rule.AppliesToAllProtectedBranches &&
			len(rule.ProtectedBranches) == 1 && rule.ProtectedBranches[0].ID == branch.ID {
			return nil
		}
		if rule.RuleType == "any_approver" {
			return fmt.Errorf("peer review rule for %q is an any-approver rule; remove it before repair", repoName)
		}
		updated, _, updateErr := git.Projects.UpdateProjectApprovalRule(projectID, rule.ID, &gitlab.UpdateProjectLevelRuleOptions{
			ApprovalsRequired:             gitlab.Ptr(int64(0)),
			GroupIDs:                      gitlab.Ptr(groupIDs),
			UserIDs:                       gitlab.Ptr([]int64{}),
			ProtectedBranchIDs:            gitlab.Ptr(branchIDs),
			AppliesToAllProtectedBranches: gitlab.Ptr(false),
		})
		if updateErr != nil {
			return fmt.Errorf("update peer review rule for %q: %w", repoName, updateErr)
		}
		if updated.RuleType != "regular" || updated.ApprovalsRequired != 0 || !approvalRuleIncludesGroup(updated, peerGroupID) {
			return fmt.Errorf("peer review rule for %q was not configured as an optional group rule", repoName)
		}
		return nil
	}
	created, _, err := git.Projects.CreateProjectApprovalRule(projectID, &gitlab.CreateProjectLevelRuleOptions{
		Name:                          gitlab.Ptr("Peer Review"),
		ApprovalsRequired:             gitlab.Ptr(int64(0)),
		GroupIDs:                      gitlab.Ptr(groupIDs),
		ProtectedBranchIDs:            gitlab.Ptr(branchIDs),
		AppliesToAllProtectedBranches: gitlab.Ptr(false),
	})
	if err != nil {
		return fmt.Errorf("create peer review rule for %q: %w", repoName, err)
	}
	if created.RuleType != "regular" || created.ApprovalsRequired != 0 || !approvalRuleIncludesGroup(created, peerGroupID) {
		return fmt.Errorf("peer review rule for %q was not configured as an optional group rule", repoName)
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

// createProjectFiles adds missing template files without overwriting files a
// student may already have edited. Retrying a partly initialized project fills
// gaps instead of treating an existing file as a successful full setup.
func createProjectFiles(git *gitlab.Client, projectID int64, repoName string, vars templateVars) error {
	svc := InfrastructureServiceSingleton
	if svc.teachingMaterialProjectID == "" {
		return fmt.Errorf("GITLAB_TEACHING_MATERIAL_PROJECT_ID not configured")
	}

	templates, err := svc.templates.get(git, svc.teachingMaterialProjectID)
	if err != nil {
		return fmt.Errorf("fetch templates for %q: %w", repoName, err)
	}
	return createProjectFilesFromTemplates(git, projectID, repoName, vars, templates)
}

func createProjectFilesFromTemplates(git *gitlab.Client, projectID int64, repoName string, vars templateVars, templates []templateFile) error {
	existingFiles := make(map[string]bool)
	nodes, treeErr := gitlab.ScanAndCollect(func(p gitlab.PaginationOptionFunc) ([]*gitlab.TreeNode, *gitlab.Response, error) {
		return git.Repositories.ListTree(projectID, &gitlab.ListTreeOptions{
			Ref: gitlab.Ptr("main"), Recursive: gitlab.Ptr(true),
			ListOptions: gitlab.ListOptions{PerPage: 100},
		}, p)
	})
	if treeErr != nil && !isNotFoundError(treeErr) {
		return fmt.Errorf("list existing files in %q: %w", repoName, treeErr)
	}
	for _, node := range nodes {
		if node.Type == "blob" {
			existingFiles[node.Path] = true
		}
	}

	actions := make([]*gitlab.CommitActionOptions, 0, len(templates))
	for _, tmpl := range templates {
		if existingFiles[tmpl.Path] {
			continue
		}
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
	if len(actions) == 0 {
		return nil
	}

	_, _, err := git.Commits.CreateCommit(projectID, &gitlab.CreateCommitOptions{
		Branch:        gitlab.Ptr("main"),
		CommitMessage: gitlab.Ptr("Initialize repository from course template"),
		Actions:       actions,
	})
	if err != nil {
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
			if r.ApprovalsRequired == 1 && r.RuleType != "any_approver" && approvalRuleIncludesGroup(r, tutorsGroupID) {
				return nil
			}
			// GitLab cannot convert an any-approver rule into a regular group
			// rule. Verify its replacement before removing the old rule.
			if r.RuleType == "any_approver" {
				created, _, createErr := git.Projects.CreateProjectApprovalRule(projectID, &gitlab.CreateProjectLevelRuleOptions{
					Name:              gitlab.Ptr("Tutor Approval"),
					ApprovalsRequired: gitlab.Ptr(int64(1)),
					GroupIDs:          gitlab.Ptr([]int64{tutorsGroupID}),
				})
				if createErr != nil {
					return fmt.Errorf("replace tutor approval rule for %q: %w", repoName, createErr)
				}
				if created.RuleType == "any_approver" || !approvalRuleIncludesGroup(created, tutorsGroupID) {
					return fmt.Errorf("replacement tutor approval rule for %q does not restrict approvers to the tutors group", repoName)
				}
				if _, deleteErr := git.Projects.DeleteProjectApprovalRule(projectID, r.ID); deleteErr != nil {
					return fmt.Errorf("remove old any-approver rule for %q: %w", repoName, deleteErr)
				}
				return nil
			}
			updated, _, updateErr := git.Projects.UpdateProjectApprovalRule(projectID, r.ID, &gitlab.UpdateProjectLevelRuleOptions{
				Name:              gitlab.Ptr("Tutor Approval"),
				ApprovalsRequired: gitlab.Ptr(int64(1)),
				GroupIDs:          gitlab.Ptr([]int64{tutorsGroupID}),
			})
			if updateErr != nil {
				return fmt.Errorf("repair tutor approval rule for %q: %w", repoName, updateErr)
			}
			if updated.RuleType == "any_approver" || !approvalRuleIncludesGroup(updated, tutorsGroupID) {
				return fmt.Errorf("tutor approval rule for %q does not restrict approvers to the tutors group", repoName)
			}
			return nil
		}
	}

	created, _, err := git.Projects.CreateProjectApprovalRule(projectID, &gitlab.CreateProjectLevelRuleOptions{
		Name:              gitlab.Ptr("Tutor Approval"),
		ApprovalsRequired: gitlab.Ptr(int64(1)),
		GroupIDs:          gitlab.Ptr([]int64{tutorsGroupID}),
	})
	if err != nil {
		return fmt.Errorf("create approval rule for %q: %w", repoName, err)
	}
	if created.RuleType == "any_approver" || !approvalRuleIncludesGroup(created, tutorsGroupID) {
		return fmt.Errorf("tutor approval rule for %q does not restrict approvers to the tutors group", repoName)
	}

	return nil
}

func approvalRuleIncludesGroup(rule *gitlab.ProjectApprovalRule, groupID int64) bool {
	for _, group := range rule.Groups {
		if group.ID == groupID {
			return true
		}
	}
	return false
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
		ResetApprovalsOnPush:                      gitlab.Ptr(true),
		MergeRequestsAuthorApproval:               gitlab.Ptr(false),
		MergeRequestsDisableCommittersApproval:    gitlab.Ptr(true),
		DisableOverridingApproversPerMergeRequest: gitlab.Ptr(true),
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
		if err := ensureGroupMember(git, existing.ID, tutorGitlabUserID, gitlab.MaintainerPermissions); err != nil {
			return 0, "", fmt.Errorf("repair tutor subgroup membership: %w", err)
		}
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
		if err := ensureGroupMember(git, raceGroup.ID, tutorGitlabUserID, gitlab.MaintainerPermissions); err != nil {
			return 0, "", fmt.Errorf("repair tutor subgroup membership: %w", err)
		}
		return raceGroup.ID, raceGroup.FullPath, nil
	}

	// 3. Add tutor as Maintainer on subgroup (inherits to all projects)
	if err := ensureGroupMember(git, group.ID, tutorGitlabUserID, gitlab.MaintainerPermissions); err != nil {
		return 0, "", fmt.Errorf("add tutor as maintainer on subgroup %q: %w", tutorGitlabUsername, err)
	}

	return group.ID, group.FullPath, nil
}

func ensureGroupMember(git *gitlab.Client, groupID, userID int64, access gitlab.AccessLevelValue) error {
	member, _, err := git.GroupMembers.GetGroupMember(groupID, userID)
	if err == nil {
		if member.AccessLevel >= access {
			return nil
		}
		_, _, err = git.GroupMembers.EditGroupMember(groupID, userID, &gitlab.EditGroupMemberOptions{AccessLevel: gitlab.Ptr(access)})
		if err != nil {
			return err
		}
		return verifyDirectGroupMember(git, groupID, userID, access)
	}
	if !isNotFoundError(err) {
		return err
	}
	_, _, err = git.GroupMembers.AddGroupMember(groupID, &gitlab.AddGroupMemberOptions{UserID: gitlab.Ptr(userID), AccessLevel: gitlab.Ptr(access)})
	if err != nil && !isAlreadyExistsError(err) {
		return err
	}
	return verifyDirectGroupMember(git, groupID, userID, access)
}

func verifyDirectGroupMember(git *gitlab.Client, groupID, userID int64, access gitlab.AccessLevelValue) error {
	member, _, err := git.GroupMembers.GetGroupMember(groupID, userID)
	if err != nil {
		return fmt.Errorf("GitLab did not confirm direct group membership: %w", err)
	}
	if member.AccessLevel < access {
		return fmt.Errorf("GitLab group membership has access level %d; need at least %d", member.AccessLevel, access)
	}
	return nil
}

// createDailyIssues creates all daily issues from the teaching material repo's
// daily_issues/ directory. Each .md file becomes a GitLab issue (title from
// first # heading, description from remaining content). Existing issues with
// matching titles are skipped for idempotency.
func createDailyIssues(git *gitlab.Client, projectID int64, repoName string) error {
	svc := InfrastructureServiceSingleton
	if svc.teachingMaterialProjectID == "" {
		return fmt.Errorf("GITLAB_TEACHING_MATERIAL_PROJECT_ID not configured")
	}

	templates, err := svc.issues.get(git, svc.teachingMaterialProjectID)
	if err != nil {
		return fmt.Errorf("fetch issue templates: %w", err)
	}
	return createDailyIssuesFromTemplates(git, projectID, repoName, templates)
}

func createDailyIssuesFromTemplates(git *gitlab.Client, projectID int64, repoName string, templates []issueTemplate) error {

	if len(templates) == 0 {
		return fmt.Errorf("no daily issue templates found in teaching material repo daily_issues/ directory")
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

	var issueErrors []error
	for _, tmpl := range templates {
		if existingTitles[tmpl.Title] {
			continue
		}
		_, _, err := git.Issues.CreateIssue(projectID, &gitlab.CreateIssueOptions{
			Title:       gitlab.Ptr(tmpl.Title),
			Description: gitlab.Ptr(tmpl.Description),
		})
		if err != nil {
			issueErrors = append(issueErrors, fmt.Errorf("%q: %w", tmpl.Title, err))
			continue
		}
	}

	return errors.Join(issueErrors...)
}

// createCICDProject creates the shared CI/CD project in the Introcourse group
// and populates it with pipeline config files from the teaching material repo's
// ci_cd/ directory. All course projects reference this project's .gitlab-ci.yml
// via CIConfigPath. Fully idempotent: safe to re-run on an existing course.
func createCICDProject(git *gitlab.Client, introCourseGroupID int64, introCourseGroupPath string) error {
	return createCICDProjectWithMaterial(git, introCourseGroupID, introCourseGroupPath, nil)
}

func createCICDProjectWithMaterial(git *gitlab.Client, introCourseGroupID int64, introCourseGroupPath string, material *materialSnapshot) error {
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
		return fmt.Errorf("GITLAB_TEACHING_MATERIAL_PROJECT_ID not configured")
	}

	var cicdFiles []templateFile
	if material == nil {
		cicdFiles, err = svc.cicd.get(git, svc.teachingMaterialProjectID)
		if err != nil {
			return fmt.Errorf("fetch CI/CD files: %w", err)
		}
	} else {
		cicdFiles = material.ciFiles
	}
	if len(cicdFiles) == 0 {
		return fmt.Errorf("no CI/CD files found in teaching material repo ci_cd/ directory")
	}

	var actions []*gitlab.CommitActionOptions
	for _, f := range cicdFiles {
		existing, _, readErr := git.RepositoryFiles.GetRawFile(project.ID, f.Path, &gitlab.GetRawFileOptions{
			Ref: gitlab.Ptr("main"),
		})
		actionType := gitlab.FileCreate
		if readErr == nil {
			if string(existing) == f.Content {
				continue
			}
			actionType = gitlab.FileUpdate
		} else if !isNotFoundError(readErr) {
			return fmt.Errorf("read existing CI/CD file %q: %w", f.Path, readErr)
		}
		action := &gitlab.CommitActionOptions{
			Action:   gitlab.Ptr(actionType),
			FilePath: gitlab.Ptr(f.Path),
			Content:  gitlab.Ptr(f.Content),
		}
		if f.ExecuteFilemode {
			action.ExecuteFilemode = gitlab.Ptr(true)
		}
		actions = append(actions, action)
	}
	if len(actions) == 0 {
		return nil
	}

	_, _, err = git.Commits.CreateCommit(project.ID, &gitlab.CreateCommitOptions{
		Branch:        gitlab.Ptr("main"),
		CommitMessage: gitlab.Ptr("Synchronize CI/CD pipeline with course template"),
		Actions:       actions,
	})
	if err != nil {
		return fmt.Errorf("push CI/CD files to project: %w", err)
	}

	return nil
}

// createDemoProject creates a "demo" project in the Introcourse group,
// initialized from the same template as student repos. Tutors inherit access
// from the shared parent group.
// Fully idempotent: safe to re-run on an existing course.
func createDemoProject(git *gitlab.Client, introCourseGroupID int64, introCourseGroupPath string, tutorsGroupID int64) error {
	return createDemoProjectWithMaterial(git, introCourseGroupID, introCourseGroupPath, tutorsGroupID, nil)
}

func createDemoProjectWithMaterial(git *gitlab.Client, introCourseGroupID int64, introCourseGroupPath string, tutorsGroupID int64, material *materialSnapshot) error {
	const demoProjectName = "demo"
	ciCDRepoPath := introCourseGroupPath + "/ci-cd"

	project, err := createOrGetProject(git, newCourseProjectOptions(demoProjectName, demoProjectName, introCourseGroupID, ciCDRepoPath), introCourseGroupPath)
	if err != nil {
		return err
	}

	// Shared project setup (files, branch protection, board, approvals, issues)
	err = configureProjectWithMaterial(git, project.ID, demoProjectName, templateVars{
		StudentName:        "Demo",
		SubmissionDeadline: "See the course schedule in Outline",
	}, material)
	if err != nil {
		return err
	}

	if err = ensureApprovalRule(git, project.ID, demoProjectName, tutorsGroupID); err != nil {
		return err
	}

	return nil
}
