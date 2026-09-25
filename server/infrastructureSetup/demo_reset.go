package infrastructureSetup

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/google/uuid"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

type materialSnapshot struct {
	sha          string
	templates    []templateFile
	issues       []issueTemplate
	ciFiles      []templateFile
	git2Exercise map[string][]templateFile
}

type resetDemoRequest struct {
	SemesterTag       string `json:"semesterTag" binding:"required"`
	ExpectedProjectID int64  `json:"expectedProjectID" binding:"required"`
	ExpectedSourceSHA string `json:"expectedSourceSHA" binding:"required"`
}

type resetDemoResult struct {
	DemoURL   string `json:"demoUrl"`
	DemoID    int64  `json:"demoId"`
	SourceSHA string `json:"sourceSha"`
}

func loadMaterialSnapshot(git *gitlab.Client, projectID, expectedSHA string) (*materialSnapshot, error) {
	if projectID == "" {
		return nil, fmt.Errorf("teaching material project is not configured")
	}
	branch, _, err := git.Branches.GetBranch(projectID, "main")
	if err != nil || branch.Commit == nil {
		return nil, fmt.Errorf("get teaching material main branch: %w", err)
	}
	sha := branch.Commit.ID
	if expectedSHA != "" && sha != expectedSHA {
		return nil, fmt.Errorf("teaching material changed since the status check; reload before resetting the demo")
	}
	templates, err := fetchTemplateFilesAtRef(git, projectID, sha)
	if err != nil {
		return nil, err
	}
	issues, err := fetchIssueTemplatesAtRef(git, projectID, sha)
	if err != nil {
		return nil, err
	}
	ciFiles, err := fetchCICDFilesAtRef(git, projectID, sha)
	if err != nil {
		return nil, err
	}
	git2Exercise, err := fetchGit2ExerciseAtRef(git, projectID, sha)
	if err != nil {
		return nil, err
	}
	if len(templates) == 0 || len(issues) == 0 || len(ciFiles) == 0 {
		return nil, fmt.Errorf("teaching material must contain repository files, daily issues, and CI configuration")
	}
	return &materialSnapshot{sha: sha, templates: templates, issues: issues, ciFiles: ciFiles, git2Exercise: git2Exercise}, nil
}

func verifySharedCI(git *gitlab.Client, ciProjectID int64, material *materialSnapshot) error {
	for _, file := range material.ciFiles {
		actual, _, err := git.RepositoryFiles.GetRawFile(ciProjectID, file.Path, &gitlab.GetRawFileOptions{Ref: gitlab.Ptr("main")})
		if err != nil || string(actual) != file.Content {
			return fmt.Errorf("shared CI does not match teaching material; repair course infrastructure first")
		}
	}
	return nil
}

// ResetDemo keeps the project ID and URL. The main branch is rebuilt from its
// original template commit; practice branches and issues are cleaned in place.
// GitLab retains old merge requests and their sequence numbers. Student projects
// are never touched.
func ResetDemo(ctx context.Context, coursePhaseID uuid.UUID, request resetDemoRequest) (*resetDemoResult, error) {
	if request.ExpectedProjectID <= 0 || len(request.ExpectedSourceSHA) != 40 {
		return nil, fmt.Errorf("a current demo ID and source commit are required")
	}
	status, err := CourseInfrastructureStatus(ctx, coursePhaseID, request.SemesterTag)
	if err != nil {
		return nil, err
	}
	if status.Source == nil || status.Source.SHA != request.ExpectedSourceSHA || status.DemoProject == nil || status.DemoProject.ID != request.ExpectedProjectID {
		return nil, fmt.Errorf("GitLab state changed since the status check; reload before resetting the demo")
	}
	if !status.Checks.TutorsReady || status.CIProject == nil || status.Groups["introCourse"] == nil || status.Groups["tutors"] == nil {
		return nil, fmt.Errorf("tutor access, shared CI, and course groups must be ready before resetting the demo")
	}
	git, err := getClient()
	if err != nil {
		return nil, err
	}
	material, err := loadMaterialSnapshot(git, InfrastructureServiceSingleton.teachingMaterialProjectID, request.ExpectedSourceSHA)
	if err != nil {
		return nil, err
	}
	if err := verifySharedCI(git, status.CIProject.ID, material); err != nil {
		return nil, err
	}
	project, _, err := git.Projects.GetProject(request.ExpectedProjectID, nil)
	if err != nil {
		return nil, err
	}
	if project.Path != "demo" || !strings.HasSuffix(project.PathWithNamespace, "/Introcourse/demo") {
		return nil, fmt.Errorf("the expected project is no longer the active demo")
	}
	root, err := originalDemoCommit(git, project.ID)
	if err != nil {
		return nil, err
	}
	if err := rewriteDemoMain(git, project.ID, root, material.templates); err != nil {
		return nil, err
	}
	if err := closeDemoMergeRequests(git, project.ID); err != nil {
		return nil, err
	}
	if err := cleanDemoBranches(git, project.ID); err != nil {
		return nil, err
	}
	if err := ensureGit2ExerciseBranches(git, project.ID, "demo", material.git2Exercise); err != nil {
		return nil, err
	}
	if err := resetDemoIssues(git, project.ID, project.PathWithNamespace, material.issues); err != nil {
		return nil, err
	}
	if err := configureProjectWithMaterial(git, project.ID, "demo", templateVars{
		StudentName: "Demo", SubmissionDeadline: "See the course schedule in Outline",
	}, material, 0); err != nil {
		return nil, err
	}
	if err := ensureApprovalRule(git, project.ID, "demo", status.Groups["tutors"].ID); err != nil {
		return nil, err
	}
	return &resetDemoResult{DemoURL: project.WebURL, DemoID: project.ID, SourceSHA: material.sha}, nil
}

func originalDemoCommit(git *gitlab.Client, projectID int64) (string, error) {
	branch, _, err := git.Branches.GetBranch(projectID, "main")
	if err != nil || branch.Commit == nil {
		return "", fmt.Errorf("read demo main branch: %w", err)
	}
	sha := branch.Commit.ID
	for range 10000 {
		commit, _, err := git.Commits.GetCommit(projectID, sha, nil)
		if err != nil {
			return "", fmt.Errorf("read demo history: %w", err)
		}
		if len(commit.ParentIDs) == 0 {
			if commit.Title != "Initialize repository from course template" {
				return "", fmt.Errorf("demo root commit is not the PROMPT course template; refusing to rewrite its history")
			}
			return sha, nil
		}
		sha = commit.ParentIDs[0]
	}
	return "", fmt.Errorf("demo history is too long to reset safely")
}

type demoRootFile struct {
	content    string
	executable bool
}

func demoTemplateActions(rootFiles map[string]demoRootFile, templates []templateFile) ([]*gitlab.CommitActionOptions, error) {
	vars := templateVars{StudentName: "Demo", SubmissionDeadline: "See the course schedule in Outline"}
	actions := make([]*gitlab.CommitActionOptions, 0, len(templates)+len(rootFiles))
	wanted := make(map[string]bool, len(templates))
	for _, file := range templates {
		if file.Path == "" || wanted[file.Path] {
			return nil, fmt.Errorf("invalid or duplicate teaching material path %q", file.Path)
		}
		wanted[file.Path] = true
		content := applyTemplateVars(file.Content, vars)
		action := gitlab.FileCreate
		if original, exists := rootFiles[file.Path]; exists {
			if original.content == content && original.executable == file.ExecuteFilemode {
				continue
			}
			action = gitlab.FileUpdate
		}
		actions = append(actions, &gitlab.CommitActionOptions{
			Action: gitlab.Ptr(action), FilePath: gitlab.Ptr(file.Path),
			Content: gitlab.Ptr(content), ExecuteFilemode: gitlab.Ptr(file.ExecuteFilemode),
		})
	}
	for path := range rootFiles {
		if !wanted[path] {
			actions = append(actions, &gitlab.CommitActionOptions{Action: gitlab.Ptr(gitlab.FileDelete), FilePath: gitlab.Ptr(path)})
		}
	}
	return actions, nil
}

func rewriteDemoMain(git *gitlab.Client, projectID int64, root string, templates []templateFile) (err error) {
	nodes, err := gitlab.ScanAndCollect(func(p gitlab.PaginationOptionFunc) ([]*gitlab.TreeNode, *gitlab.Response, error) {
		return git.Repositories.ListTree(projectID, &gitlab.ListTreeOptions{
			Ref: gitlab.Ptr(root), Recursive: gitlab.Ptr(true), ListOptions: gitlab.ListOptions{PerPage: 100},
		}, p)
	})
	if err != nil {
		return fmt.Errorf("read original demo files: %w", err)
	}
	rootFiles := make(map[string]demoRootFile)
	for _, node := range nodes {
		if node.Type == "blob" {
			raw, _, readErr := git.RepositoryFiles.GetRawFile(projectID, node.Path, &gitlab.GetRawFileOptions{Ref: gitlab.Ptr(root)})
			if readErr != nil {
				return fmt.Errorf("read original demo file %q: %w", node.Path, readErr)
			}
			rootFiles[node.Path] = demoRootFile{content: string(raw), executable: node.Mode == "100755"}
		}
	}
	if len(rootFiles) == 0 {
		return fmt.Errorf("original demo has no files; refusing to rewrite main")
	}
	actions, err := demoTemplateActions(rootFiles, templates)
	if err != nil {
		return err
	}
	bot, _, err := git.Users.CurrentUser()
	if err != nil {
		return fmt.Errorf("identify GitLab service user: %w", err)
	}
	protected, _, err := git.ProtectedBranches.GetProtectedBranch(projectID, "main")
	if err != nil {
		return fmt.Errorf("read demo main branch protection: %w", err)
	}
	if !mainBranchProtectionMatches(protected, 0) {
		return fmt.Errorf("demo main branch must have the expected strict protection before reset")
	}
	rules, err := listMatchingMainProtections(git, projectID)
	if err != nil {
		return err
	}
	if len(rules) != 1 || rules[0].Name != "main" {
		return fmt.Errorf("demo main has overlapping protection rules; resolve them before reset")
	}
	push := make([]*gitlab.BranchPermissionOptions, 0, len(protected.PushAccessLevels)+1)
	for _, access := range protected.PushAccessLevels {
		push = append(push, &gitlab.BranchPermissionOptions{ID: gitlab.Ptr(access.ID), Destroy: gitlab.Ptr(true)})
	}
	push = append(push, &gitlab.BranchPermissionOptions{UserID: gitlab.Ptr(bot.ID)})
	defer func() {
		if restoreErr := ensureMainBranchProtection(git, projectID, 0); restoreErr != nil {
			err = errors.Join(err, fmt.Errorf("restore demo main protection immediately: %w", restoreErr))
		}
	}()
	temporary, _, err := git.ProtectedBranches.UpdateProtectedBranch(projectID, "main", &gitlab.UpdateProtectedBranchOptions{
		AllowedToPush: gitlab.Ptr(push), AllowForcePush: gitlab.Ptr(true),
	})
	if err != nil {
		return fmt.Errorf("temporarily permit service user to rewrite demo main: %w", err)
	}
	if !temporary.AllowForcePush || len(temporary.PushAccessLevels) != 1 || temporary.PushAccessLevels[0].UserID != bot.ID {
		return fmt.Errorf("GitLab did not restrict temporary demo push access to the service user")
	}
	// client-go v1.46 lacks allow_empty, which GitLab needs when the template
	// already matches the root commit. Use the same authenticated client.
	body := struct {
		*gitlab.CreateCommitOptions
		AllowEmpty bool `json:"allow_empty"`
	}{&gitlab.CreateCommitOptions{
		Branch: gitlab.Ptr("main"), StartSHA: gitlab.Ptr(root), Force: gitlab.Ptr(true),
		CommitMessage: gitlab.Ptr("Reset demo from current teaching material"), Actions: actions,
	}, true}
	req, err := git.NewRequest("POST", fmt.Sprintf("projects/%d/repository/commits", projectID), body, nil)
	if err != nil {
		return err
	}
	var commit gitlab.Commit
	if _, err = git.Do(req, &commit); err != nil {
		return fmt.Errorf("rewrite demo main from its original commit: %w", err)
	}
	if len(commit.ParentIDs) != 1 || commit.ParentIDs[0] != root {
		return fmt.Errorf("GitLab did not confirm the expected reset commit ancestry")
	}
	return nil
}

func closeDemoMergeRequests(git *gitlab.Client, projectID int64) error {
	mrs, err := gitlab.ScanAndCollect(func(p gitlab.PaginationOptionFunc) ([]*gitlab.BasicMergeRequest, *gitlab.Response, error) {
		return git.MergeRequests.ListProjectMergeRequests(projectID, &gitlab.ListProjectMergeRequestsOptions{
			State: gitlab.Ptr("opened"), ListOptions: gitlab.ListOptions{PerPage: 100},
		}, p)
	})
	if err != nil {
		return err
	}
	for _, mr := range mrs {
		if _, _, err := git.MergeRequests.UpdateMergeRequest(projectID, mr.IID, &gitlab.UpdateMergeRequestOptions{StateEvent: gitlab.Ptr("close")}); err != nil {
			return fmt.Errorf("close demo merge request !%d: %w", mr.IID, err)
		}
	}
	return nil
}

func cleanDemoBranches(git *gitlab.Client, projectID int64) error {
	branches, err := gitlab.ScanAndCollect(func(p gitlab.PaginationOptionFunc) ([]*gitlab.Branch, *gitlab.Response, error) {
		return git.Branches.ListBranches(projectID, &gitlab.ListBranchesOptions{ListOptions: gitlab.ListOptions{PerPage: 100}}, p)
	})
	if err != nil {
		return err
	}
	for _, branch := range branches {
		if branch.Name == "main" {
			continue
		}
		if branch.Protected {
			if _, err := git.ProtectedBranches.UnprotectRepositoryBranches(projectID, branch.Name); err != nil {
				return fmt.Errorf("unprotect demo branch %q: %w", branch.Name, err)
			}
		}
		if _, err := git.Branches.DeleteBranch(projectID, branch.Name); err != nil {
			return fmt.Errorf("delete demo branch %q: %w", branch.Name, err)
		}
	}
	return nil
}

func demoIssueStatusIDs(git *gitlab.Client, projectPath string) (map[int64]string, string, error) {
	board, err := readStatusBoard(git, projectPath)
	if err != nil {
		return nil, "", err
	}
	ids, err := requiredStatusIDs(board)
	if err != nil {
		return nil, "", err
	}
	var result struct {
		Data struct {
			Project struct {
				WorkItems struct {
					Nodes []struct {
						IID     string `json:"iid"`
						Widgets []struct {
							Status *struct {
								ID string `json:"id"`
							} `json:"status"`
						} `json:"widgets"`
					} `json:"nodes"`
					PageInfo struct {
						HasNextPage bool `json:"hasNextPage"`
					} `json:"pageInfo"`
				} `json:"workItems"`
			} `json:"project"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	_, err = git.GraphQL.Do(gitlab.GraphQLQuery{
		Query:     `query($path: ID!) { project(fullPath: $path) { workItems(first: 100) { nodes { iid widgets { ... on WorkItemWidgetStatus { status { id } } } } pageInfo { hasNextPage } } } }`,
		Variables: map[string]any{"path": projectPath},
	}, &result)
	if err != nil {
		return nil, "", fmt.Errorf("read demo issue statuses: %w", err)
	}
	if len(result.Errors) != 0 || result.Data.Project.WorkItems.PageInfo.HasNextPage {
		return nil, "", fmt.Errorf("could not read all demo issue statuses before reset: %v", result.Errors)
	}
	statuses := make(map[int64]string, len(result.Data.Project.WorkItems.Nodes))
	for _, item := range result.Data.Project.WorkItems.Nodes {
		iid, parseErr := strconv.ParseInt(item.IID, 10, 64)
		if parseErr != nil {
			return nil, "", fmt.Errorf("read demo issue IID %q: %w", item.IID, parseErr)
		}
		for _, widget := range item.Widgets {
			if widget.Status != nil {
				statuses[iid] = widget.Status.ID
			}
		}
	}
	return statuses, ids["Open"], nil
}

func resetDemoIssues(git *gitlab.Client, projectID int64, projectPath string, templates []issueTemplate) error {
	issues, err := gitlab.ScanAndCollect(func(p gitlab.PaginationOptionFunc) ([]*gitlab.Issue, *gitlab.Response, error) {
		return git.Issues.ListProjectIssues(projectID, &gitlab.ListProjectIssuesOptions{
			State: gitlab.Ptr("all"), Scope: gitlab.Ptr("all"), ListOptions: gitlab.ListOptions{PerPage: 100},
		}, p)
	})
	if err != nil {
		return err
	}
	statuses, openStatusID, err := demoIssueStatusIDs(git, projectPath)
	if err != nil {
		return err
	}
	sort.Slice(issues, func(i, j int) bool { return issues[i].IID < issues[j].IID })
	wanted := make(map[string]issueTemplate, len(templates))
	for _, tmpl := range templates {
		if tmpl.Title == "" {
			return fmt.Errorf("teaching material has an issue without a title")
		}
		if _, exists := wanted[tmpl.Title]; exists {
			return fmt.Errorf("teaching material has duplicate issue title %q", tmpl.Title)
		}
		wanted[tmpl.Title] = tmpl
	}
	seen := make(map[string]bool)
	for _, issue := range issues {
		tmpl, keep := wanted[issue.Title]
		if !keep || seen[issue.Title] {
			if _, err := git.Issues.DeleteIssue(projectID, issue.IID); err != nil {
				return fmt.Errorf("delete demo practice issue #%d: %w", issue.IID, err)
			}
			continue
		}
		seen[issue.Title] = true
		// Avoid churn and notifications on already-clean daily issues. Close and
		// reopen only when needed to restore the lifecycle's Open status.
		needsReopen := issue.State != "opened" || statuses[issue.IID] != openStatusID
		needsMetadata := issue.Description != tmpl.Description || len(issue.Labels) != 0 ||
			len(issue.Assignees) != 0 || issue.Milestone != nil || issue.DueDate != nil ||
			issue.Weight != 0 || issue.Confidential || issue.DiscussionLocked
		if !needsReopen && !needsMetadata {
			continue
		}
		if needsReopen && issue.State == "opened" {
			if _, _, err := git.Issues.UpdateIssue(projectID, issue.IID, &gitlab.UpdateIssueOptions{StateEvent: gitlab.Ptr("close")}); err != nil {
				return fmt.Errorf("close demo daily issue #%d: %w", issue.IID, err)
			}
		}
		update := &gitlab.UpdateIssueOptions{
			Description: gitlab.Ptr(tmpl.Description), Confidential: gitlab.Ptr(false), DiscussionLocked: gitlab.Ptr(false),
			Labels: gitlab.Ptr(gitlab.LabelOptions{}), AssigneeIDs: gitlab.Ptr([]int64{}),
			ResetMilestoneID: true, ResetDueDate: true, ResetWeight: true,
		}
		if needsReopen {
			update.StateEvent = gitlab.Ptr("reopen")
		}
		if _, _, err := git.Issues.UpdateIssue(projectID, issue.IID, update); err != nil {
			return fmt.Errorf("restore demo daily issue #%d: %w", issue.IID, err)
		}
	}
	for _, tmpl := range templates {
		if seen[tmpl.Title] {
			continue
		}
		if _, _, err := git.Issues.CreateIssue(projectID, &gitlab.CreateIssueOptions{
			Title: gitlab.Ptr(tmpl.Title), Description: gitlab.Ptr(tmpl.Description),
		}); err != nil {
			return fmt.Errorf("restore demo daily issue %q: %w", tmpl.Title, err)
		}
	}
	return nil
}
