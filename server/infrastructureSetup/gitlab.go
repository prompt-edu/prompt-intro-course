package infrastructureSetup

import (
	"errors"
	"fmt"
	"path"
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
	errExistingMainMissingFiles   = errors.New("missing template files on an existing main branch")
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
		if opts.CIConfigPath != nil {
			if err = reconcileCourseProjectSettings(git, project, opts); err != nil {
				return nil, fmt.Errorf("repair existing project %q settings: %w", projectPath, err)
			}
		}
		log.WithField("project", *opts.Name).Info("project already exists, continuing with setup")
	}
	return project, nil
}

func courseProjectSettingsMatch(project *gitlab.Project, opts *gitlab.CreateProjectOptions) bool {
	return project != nil && project.CIConfigPath == *opts.CIConfigPath &&
		project.Visibility == *opts.Visibility && project.MergeMethod == *opts.MergeMethod &&
		project.SquashOption == *opts.SquashOption &&
		project.RemoveSourceBranchAfterMerge == *opts.RemoveSourceBranchAfterMerge &&
		project.OnlyAllowMergeIfPipelineSucceeds == *opts.OnlyAllowMergeIfPipelineSucceeds &&
		project.OnlyAllowMergeIfAllDiscussionsAreResolved == *opts.OnlyAllowMergeIfAllDiscussionsAreResolved
}

func reconcileCourseProjectSettings(git *gitlab.Client, project *gitlab.Project, opts *gitlab.CreateProjectOptions) error {
	if courseProjectSettingsMatch(project, opts) {
		return nil
	}
	_, _, err := git.Projects.EditProject(project.ID, &gitlab.EditProjectOptions{
		CIConfigPath: opts.CIConfigPath, Visibility: opts.Visibility,
		MergeMethod: opts.MergeMethod, SquashOption: opts.SquashOption,
		RemoveSourceBranchAfterMerge:              opts.RemoveSourceBranchAfterMerge,
		OnlyAllowMergeIfPipelineSucceeds:          opts.OnlyAllowMergeIfPipelineSucceeds,
		OnlyAllowMergeIfAllDiscussionsAreResolved: opts.OnlyAllowMergeIfAllDiscussionsAreResolved,
	})
	if err != nil {
		return err
	}
	updated, _, err := git.Projects.GetProject(project.ID, nil)
	if err != nil {
		return err
	}
	if !courseProjectSettingsMatch(updated, opts) {
		return fmt.Errorf("GitLab did not confirm the shared CI and merge settings")
	}
	return nil
}

// configureProjectWithMaterial applies the shared template, branch protection,
// issue board, approval, and daily issue setup to a course project.
func configureProjectWithMaterial(git *gitlab.Client, projectID int64, projectName string, vars templateVars, material *materialSnapshot, mergeOwnerID int64) error {
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
	// On retries, repair an existing branch before any later operation can fail.
	// In particular, missing template files must not leave an old Developer
	// merge rule in place. A brand-new branch is protected after its first commit.
	_, _, branchErr := git.Branches.GetBranch(projectID, "main")
	if branchErr == nil {
		if err := ensureMainBranchProtection(git, projectID, 0); err != nil {
			return fmt.Errorf("secure existing main for %q: %w", projectName, err)
		}
	} else if !isNotFoundError(branchErr) {
		return fmt.Errorf("check existing main for %q: %w", projectName, branchErr)
	}
	// Template files (idempotent: skip files that already exist)
	err := createProjectFilesFromTemplates(git, projectID, projectName, vars, templates)
	if err != nil {
		return err
	}

	// Never unprotect an existing project during a retry: peers already have
	// Developer access there. Restrict merge access before later setup steps can
	// fail, so a retry cannot leave classmates able to merge this student's MR.
	if err = ensureMainBranchProtection(git, projectID, mergeOwnerID); err != nil {
		return fmt.Errorf("protect main for %q: %w", projectName, err)
	}

	if err = ensureCourseStatusBoard(git, projectID); err != nil {
		return fmt.Errorf("configure status board for %q: %w", projectName, err)
	}

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

func ensureMainBranchProtection(git *gitlab.Client, projectID, mergeOwnerID int64) error {
	return ensureMainBranchProtectionAttempt(git, projectID, mergeOwnerID, 0)
}

func ensureMainBranchProtectionAttempt(git *gitlab.Client, projectID, mergeOwnerID int64, attempt int) error {
	// GitLab applies the most permissive of *all* matching rules. A single GET
	// can return a strict rule while another exact or wildcard rule still lets
	// Developers push. Remove duplicate exact rules before configuring one.
	rules, err := listMatchingMainProtections(git, projectID)
	if err != nil {
		return err
	}
	exact := 0
	for _, rule := range rules {
		if rule.Name != "main" {
			return fmt.Errorf("protected branch rule %q also matches main; resolve it before course setup", rule.Name)
		}
		exact++
	}
	for exact > 1 {
		if _, err := git.ProtectedBranches.UnprotectRepositoryBranches(projectID, "main"); err != nil {
			return fmt.Errorf("remove duplicate main protection: %w", err)
		}
		exact--
	}
	branch, _, err := git.ProtectedBranches.GetProtectedBranch(projectID, "main")
	if isNotFoundError(err) {
		branch, _, err = git.ProtectedBranches.ProtectRepositoryBranches(projectID, &gitlab.ProtectRepositoryBranchesOptions{
			Name: gitlab.Ptr("main"), PushAccessLevel: gitlab.Ptr(gitlab.NoPermissions),
			MergeAccessLevel: gitlab.Ptr(gitlab.MaintainerPermissions), AllowForcePush: gitlab.Ptr(false),
		})
		if isAlreadyExistsError(err) {
			branch, _, err = git.ProtectedBranches.GetProtectedBranch(projectID, "main")
		}
	}
	if err != nil {
		return err
	}
	var pushUpdates, mergeUpdates []*gitlab.BranchPermissionOptions
	pushDenied, maintainerPresent, ownerPresent := false, false, false
	for _, access := range branch.PushAccessLevels {
		if access.AccessLevel == gitlab.NoPermissions && access.UserID == 0 && access.GroupID == 0 && !pushDenied {
			pushDenied = true
		} else {
			pushUpdates = append(pushUpdates, &gitlab.BranchPermissionOptions{ID: gitlab.Ptr(access.ID), Destroy: gitlab.Ptr(true)})
		}
	}
	if !pushDenied {
		pushUpdates = append(pushUpdates, &gitlab.BranchPermissionOptions{AccessLevel: gitlab.Ptr(gitlab.NoPermissions)})
	}
	for _, access := range branch.MergeAccessLevels {
		switch {
		case access.UserID == mergeOwnerID && mergeOwnerID != 0 && !ownerPresent:
			ownerPresent = true
		case access.UserID == 0 && access.GroupID == 0 && access.AccessLevel == gitlab.MaintainerPermissions && !maintainerPresent:
			maintainerPresent = true
		default:
			mergeUpdates = append(mergeUpdates, &gitlab.BranchPermissionOptions{ID: gitlab.Ptr(access.ID), Destroy: gitlab.Ptr(true)})
		}
	}
	if !maintainerPresent {
		mergeUpdates = append(mergeUpdates, &gitlab.BranchPermissionOptions{AccessLevel: gitlab.Ptr(gitlab.MaintainerPermissions)})
	}
	if mergeOwnerID != 0 && !ownerPresent {
		mergeUpdates = append(mergeUpdates, &gitlab.BranchPermissionOptions{UserID: gitlab.Ptr(mergeOwnerID)})
	}
	if len(pushUpdates) > 0 || len(mergeUpdates) > 0 || branch.AllowForcePush {
		options := &gitlab.UpdateProtectedBranchOptions{AllowForcePush: gitlab.Ptr(false)}
		if len(pushUpdates) > 0 {
			options.AllowedToPush = gitlab.Ptr(pushUpdates)
		}
		if len(mergeUpdates) > 0 {
			options.AllowedToMerge = gitlab.Ptr(mergeUpdates)
		}
		branch, _, err = git.ProtectedBranches.UpdateProtectedBranch(projectID, "main", options)
		if err != nil {
			return err
		}
	}
	if !mainBranchProtectionMatches(branch, mergeOwnerID) {
		return fmt.Errorf("GitLab did not confirm main branch push and merge restrictions")
	}
	rules, err = listMatchingMainProtections(git, projectID)
	if err != nil {
		return err
	}
	if len(rules) != 1 || rules[0].Name != "main" || !mainBranchProtectionMatches(rules[0], mergeOwnerID) {
		// GitLab may create its default Developer-push rule while the first
		// template commit is being written. The update above can race with
		// that rule and leave two exact `main` entries. Read and collapse them
		// again before any student or peer gets project access.
		if attempt < 3 {
			return ensureMainBranchProtectionAttempt(git, projectID, mergeOwnerID, attempt+1)
		}
		return fmt.Errorf("GitLab did not confirm exactly one strict main branch rule")
	}
	return nil
}

func listMatchingMainProtections(git *gitlab.Client, projectID int64) ([]*gitlab.ProtectedBranch, error) {
	all, err := gitlab.ScanAndCollect(func(p gitlab.PaginationOptionFunc) ([]*gitlab.ProtectedBranch, *gitlab.Response, error) {
		return git.ProtectedBranches.ListProtectedBranches(projectID, &gitlab.ListProtectedBranchesOptions{ListOptions: gitlab.ListOptions{PerPage: 100}}, p)
	})
	if err != nil {
		return nil, fmt.Errorf("list protected branch rules: %w", err)
	}
	var matching []*gitlab.ProtectedBranch
	for _, rule := range all {
		if rule.Name == "main" || protectedPatternMatchesMain(rule.Name) {
			matching = append(matching, rule)
		}
	}
	return matching, nil
}

func protectedPatternMatchesMain(pattern string) bool {
	matched, err := path.Match(pattern, "main")
	return err == nil && matched
}

func mainBranchProtectionMatches(branch *gitlab.ProtectedBranch, mergeOwnerID int64) bool {
	if branch == nil || branch.AllowForcePush || len(branch.PushAccessLevels) != 1 ||
		branch.PushAccessLevels[0].AccessLevel != gitlab.NoPermissions ||
		branch.PushAccessLevels[0].UserID != 0 || branch.PushAccessLevels[0].GroupID != 0 ||
		len(branch.MergeAccessLevels) != 1+boolToInt(mergeOwnerID != 0) {
		return false
	}
	maintainerPresent, ownerPresent := false, mergeOwnerID == 0
	for _, access := range branch.MergeAccessLevels {
		if access.UserID == mergeOwnerID && mergeOwnerID != 0 {
			ownerPresent = true
		} else if access.UserID == 0 && access.GroupID == 0 && access.AccessLevel == gitlab.MaintainerPermissions {
			maintainerPresent = true
		} else {
			return false
		}
	}
	return maintainerPresent && ownerPresent
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
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
	}, material, 0)
	if err != nil {
		return err
	}

	// Seed the two Git 2 practice branches before students receive access.
	// Their app on main remains the untouched course template.
	var exercise map[string][]templateFile
	if material != nil {
		exercise = material.git2Exercise
	} else {
		exercise, err = fetchGit2ExerciseAtRef(git, InfrastructureServiceSingleton.teachingMaterialProjectID, "main")
		if err != nil {
			return err
		}
	}
	if err = ensureGit2ExerciseBranches(git, project.ID, p.RepoName, exercise); err != nil {
		return err
	}

	// 3. Members (idempotent: skip if already a member)
	err = addProjectMembers(git, project.ID, p.RepoName, p.DevID, p.DevGroupID)
	if err != nil {
		return err
	}
	// GitLab requires a protected-branch user grant to already be a project
	// member. Until then, only Maintainers may merge; never open access to all
	// Developers during a partial setup or retry.
	if err = ensureMainBranchProtection(git, project.ID, p.DevID); err != nil {
		return fmt.Errorf("grant student-only main merge access for %q: %w", p.RepoName, err)
	}
	if err = ensureGroupMember(git, peerGroup.ID, p.DevID, gitlab.DeveloperPermissions); err != nil {
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

func restrictStudentMainMergeAccess(git *gitlab.Client, projectID, ownerID int64) error {
	branch, _, err := git.ProtectedBranches.GetProtectedBranch(projectID, "main")
	if err != nil {
		return err
	}
	var updates []*gitlab.BranchPermissionOptions
	ownerPresent, maintainerPresent := false, false
	for _, access := range branch.MergeAccessLevels {
		switch {
		case access.UserID == ownerID:
			ownerPresent = true
		case access.UserID == 0 && access.GroupID == 0 && access.AccessLevel == gitlab.MaintainerPermissions:
			maintainerPresent = true
		default:
			updates = append(updates, &gitlab.BranchPermissionOptions{ID: gitlab.Ptr(access.ID), Destroy: gitlab.Ptr(true)})
		}
	}
	if !ownerPresent {
		updates = append(updates, &gitlab.BranchPermissionOptions{UserID: gitlab.Ptr(ownerID)})
	}
	if !maintainerPresent {
		updates = append(updates, &gitlab.BranchPermissionOptions{AccessLevel: gitlab.Ptr(gitlab.MaintainerPermissions)})
	}
	if len(updates) > 0 {
		branch, _, err = git.ProtectedBranches.UpdateProtectedBranch(projectID, "main", &gitlab.UpdateProtectedBranchOptions{AllowedToMerge: gitlab.Ptr(updates)})
		if err != nil {
			return err
		}
	}
	if len(branch.PushAccessLevels) != 1 || branch.PushAccessLevels[0].AccessLevel != gitlab.NoPermissions {
		return fmt.Errorf("main must reject direct pushes")
	}
	if len(branch.MergeAccessLevels) != 2 {
		return fmt.Errorf("main must allow only the student owner and Maintainers to merge")
	}
	ownerPresent, maintainerPresent = false, false
	for _, access := range branch.MergeAccessLevels {
		if access.UserID == ownerID {
			ownerPresent = true
		} else if access.UserID == 0 && access.GroupID == 0 && access.AccessLevel == gitlab.MaintainerPermissions {
			maintainerPresent = true
		}
	}
	if !ownerPresent || !maintainerPresent {
		return fmt.Errorf("GitLab did not confirm student-only merge access on main")
	}
	return nil
}

// Each tutor group has one direct-membership peer group. It is shared into
// student projects as Developer, while each repository owner gets Developer
// access directly. Request changes requires Developer on this GitLab instance.
// The protected main branch still prevents direct pushes without review.
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
			if shared.GroupAccessLevel == int64(gitlab.DeveloperPermissions) {
				return nil
			}
			if shared.GroupAccessLevel != int64(gitlab.ReporterPermissions) {
				return fmt.Errorf("peer group has access level %d; expected Reporter or Developer", shared.GroupAccessLevel)
			}
			if _, err := git.Projects.DeleteSharedProjectFromGroup(projectID, peerGroupID); err != nil {
				return fmt.Errorf("remove Reporter peer-group share before upgrade: %w", err)
			}
			if _, err := git.Projects.ShareProjectWithGroup(projectID, &gitlab.ShareWithGroupOptions{
				GroupID: gitlab.Ptr(peerGroupID), GroupAccess: gitlab.Ptr(gitlab.DeveloperPermissions),
			}); err != nil {
				_, rollbackErr := git.Projects.ShareProjectWithGroup(projectID, &gitlab.ShareWithGroupOptions{
					GroupID: gitlab.Ptr(peerGroupID), GroupAccess: gitlab.Ptr(gitlab.ReporterPermissions),
				})
				return fmt.Errorf("upgrade peer-group share to Developer: %w; restore Reporter share: %v", err, rollbackErr)
			}
			return verifyPeerGroupShare(git, projectID, peerGroupID)
		}
	}
	_, err = git.Projects.ShareProjectWithGroup(projectID, &gitlab.ShareWithGroupOptions{
		GroupID: gitlab.Ptr(peerGroupID), GroupAccess: gitlab.Ptr(gitlab.DeveloperPermissions),
	})
	if err != nil && !isAlreadyExistsError(err) {
		return err
	}
	return verifyPeerGroupShare(git, projectID, peerGroupID)
}

func ensureProjectSharedWithGroupAtLeast(git *gitlab.Client, projectID, groupID int64, minimum gitlab.AccessLevelValue) error {
	project, _, err := git.Projects.GetProject(projectID, nil)
	if err != nil {
		return err
	}
	for _, shared := range project.SharedWithGroups {
		if shared.GroupID == groupID {
			if shared.GroupAccessLevel >= int64(minimum) {
				return nil
			}
			if _, err = git.Projects.DeleteSharedProjectFromGroup(projectID, groupID); err != nil {
				return fmt.Errorf("remove insufficient shared CI access: %w", err)
			}
			if _, err = git.Projects.ShareProjectWithGroup(projectID, &gitlab.ShareWithGroupOptions{
				GroupID: gitlab.Ptr(groupID), GroupAccess: gitlab.Ptr(minimum),
			}); err != nil {
				_, rollbackErr := git.Projects.ShareProjectWithGroup(projectID, &gitlab.ShareWithGroupOptions{
					GroupID: gitlab.Ptr(groupID), GroupAccess: gitlab.Ptr(gitlab.AccessLevelValue(shared.GroupAccessLevel)),
				})
				return fmt.Errorf("upgrade shared CI access: %w; restore previous access: %v", err, rollbackErr)
			}
			return verifyProjectGroupAccess(git, projectID, groupID, minimum)
		}
	}
	_, err = git.Projects.ShareProjectWithGroup(projectID, &gitlab.ShareWithGroupOptions{
		GroupID: gitlab.Ptr(groupID), GroupAccess: gitlab.Ptr(minimum),
	})
	if err != nil && !isAlreadyExistsError(err) {
		return err
	}
	return verifyProjectGroupAccess(git, projectID, groupID, minimum)
}

func verifyProjectGroupAccess(git *gitlab.Client, projectID, groupID int64, minimum gitlab.AccessLevelValue) error {
	project, _, err := git.Projects.GetProject(projectID, nil)
	if err != nil {
		return err
	}
	for _, shared := range project.SharedWithGroups {
		if shared.GroupID == groupID && shared.GroupAccessLevel >= int64(minimum) {
			return nil
		}
	}
	return fmt.Errorf("GitLab did not confirm shared CI access for developer group")
}

func verifyPeerGroupShare(git *gitlab.Client, projectID, peerGroupID int64) error {
	project, _, err := git.Projects.GetProject(projectID, nil)
	if err != nil {
		return err
	}
	for _, shared := range project.SharedWithGroups {
		if shared.GroupID == peerGroupID && shared.GroupAccessLevel == int64(gitlab.DeveloperPermissions) {
			return nil
		}
	}
	return fmt.Errorf("GitLab did not confirm Developer sharing for peer group %d", peerGroupID)
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
	if err := ensureStudentProjectMember(git, projectID, devID); err != nil {
		return fmt.Errorf("ensure student %d has Developer access to project %q: %w", devID, repoName, err)
	}
	if err := ensureGroupMember(git, devGroupID, devID, gitlab.DeveloperPermissions); err != nil {
		return fmt.Errorf("ensure student %d has Developer access to developer group for %q: %w", devID, repoName, err)
	}

	// Tutor access is inherited from the tutor subgroup (Maintainer permission)

	return nil
}

func ensureStudentProjectMember(git *gitlab.Client, projectID, userID int64) error {
	member, _, err := git.ProjectMembers.GetProjectMember(projectID, userID)
	if err == nil {
		if member.AccessLevel >= gitlab.DeveloperPermissions {
			return nil
		}
		_, _, err = git.ProjectMembers.EditProjectMember(projectID, userID, &gitlab.EditProjectMemberOptions{AccessLevel: gitlab.Ptr(gitlab.DeveloperPermissions)})
	} else if isNotFoundError(err) {
		_, _, err = git.ProjectMembers.AddProjectMember(projectID, &gitlab.AddProjectMemberOptions{
			UserID: gitlab.Ptr(userID), AccessLevel: gitlab.Ptr(gitlab.DeveloperPermissions),
		})
	} else {
		return err
	}
	if err != nil && !isAlreadyExistsError(err) {
		return err
	}
	member, _, err = git.ProjectMembers.GetProjectMember(projectID, userID)
	if err != nil {
		return fmt.Errorf("GitLab did not confirm direct Developer access: %w", err)
	}
	if member.AccessLevel < gitlab.DeveloperPermissions {
		return fmt.Errorf("GitLab project membership has access level %d; need Developer", member.AccessLevel)
	}
	return nil
}

// createProjectFiles adds the template to a new project without overwriting
// files that a student may already have edited.
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
	// An existing repository may already be shared with Developer peers. Do not
	// open its protected main branch to repair missing template files. A fresh
	// project has no main branch yet and can be initialized safely here.
	_, _, branchErr := git.Branches.GetBranch(projectID, "main")
	if branchErr == nil {
		return fmt.Errorf("%q has %w; repair them through a reviewed MR or reset the demo", repoName, errExistingMainMissingFiles)
	}
	if !isNotFoundError(branchErr) {
		return fmt.Errorf("check main branch for %q: %w", repoName, branchErr)
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
			if tutorApprovalRuleIsStrict(r, tutorsGroupID) {
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
				if !tutorApprovalRuleIsStrict(created, tutorsGroupID) {
					return fmt.Errorf("replacement tutor approval rule for %q does not restrict approvers to the tutors group", repoName)
				}
				if _, deleteErr := git.Projects.DeleteProjectApprovalRule(projectID, r.ID); deleteErr != nil {
					return fmt.Errorf("remove old any-approver rule for %q: %w", repoName, deleteErr)
				}
				return nil
			}
			updated, _, updateErr := git.Projects.UpdateProjectApprovalRule(projectID, r.ID, &gitlab.UpdateProjectLevelRuleOptions{
				Name:                          gitlab.Ptr("Tutor Approval"),
				ApprovalsRequired:             gitlab.Ptr(int64(1)),
				UserIDs:                       gitlab.Ptr([]int64{}),
				GroupIDs:                      gitlab.Ptr([]int64{tutorsGroupID}),
				AppliesToAllProtectedBranches: gitlab.Ptr(true),
			})
			if updateErr != nil {
				return fmt.Errorf("repair tutor approval rule for %q: %w", repoName, updateErr)
			}
			if !tutorApprovalRuleIsStrict(updated, tutorsGroupID) {
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
	if !tutorApprovalRuleIsStrict(created, tutorsGroupID) {
		return fmt.Errorf("tutor approval rule for %q does not restrict approvers to the tutors group", repoName)
	}

	return nil
}

func tutorApprovalRuleIsStrict(rule *gitlab.ProjectApprovalRule, groupID int64) bool {
	if rule == nil || rule.RuleType != "regular" || rule.ApprovalsRequired != 1 ||
		len(rule.Users) != 0 || len(rule.Groups) != 1 || rule.Groups[0].ID != groupID {
		return false
	}
	// An unscoped rule applies to all branches. A scoped rule must cover main.
	if rule.AppliesToAllProtectedBranches || len(rule.ProtectedBranches) == 0 {
		return true
	}
	for _, branch := range rule.ProtectedBranches {
		if branch.Name == "main" {
			return true
		}
	}
	return false
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
	existingIssues, err := gitlab.ScanAndCollect(func(p gitlab.PaginationOptionFunc) ([]*gitlab.Issue, *gitlab.Response, error) {
		return git.Issues.ListProjectIssues(projectID, &gitlab.ListProjectIssuesOptions{ListOptions: gitlab.ListOptions{PerPage: 100}}, p)
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

	options := newCourseProjectOptions(demoProjectName, demoProjectName, introCourseGroupID, ciCDRepoPath)
	// The demo is repeatedly reset for teaching-material checks. Do not email
	// every inherited course member when GitLab moves an older demo snapshot.
	options.EmailsEnabled = gitlab.Ptr(false)
	project, err := createOrGetProject(git, options, introCourseGroupPath)
	if err != nil {
		return err
	}
	if err := ensureDemoEmailNotificationsDisabled(git, project); err != nil {
		return err
	}

	// Shared project setup (files, branch protection, board, approvals, issues)
	err = configureProjectWithMaterial(git, project.ID, demoProjectName, templateVars{
		StudentName:        "Demo",
		SubmissionDeadline: "See the course schedule in Outline",
	}, material, 0)
	if err != nil {
		return err
	}

	var exercise map[string][]templateFile
	if material != nil {
		exercise = material.git2Exercise
	} else {
		exercise, err = fetchGit2ExerciseAtRef(git, InfrastructureServiceSingleton.teachingMaterialProjectID, "main")
		if err != nil {
			return err
		}
	}
	if err = ensureGit2ExerciseBranches(git, project.ID, demoProjectName, exercise); err != nil {
		return err
	}

	if err = ensureApprovalRule(git, project.ID, demoProjectName, tutorsGroupID); err != nil {
		return err
	}

	return nil
}

func ensureDemoEmailNotificationsDisabled(git *gitlab.Client, project *gitlab.Project) error {
	if project == nil {
		return fmt.Errorf("demo project is missing")
	}
	if !project.EmailsEnabled {
		return nil
	}
	updated, _, err := git.Projects.EditProject(project.ID, &gitlab.EditProjectOptions{EmailsEnabled: gitlab.Ptr(false)})
	if err != nil {
		return fmt.Errorf("disable demo project emails: %w", err)
	}
	if updated.EmailsEnabled {
		return fmt.Errorf("GitLab did not disable demo project emails")
	}
	return nil
}
