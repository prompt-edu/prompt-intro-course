package infrastructureSetup

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/prompt-edu/prompt-intro-course/server/gitlabutil"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

type repositoryLink struct {
	ID  int64  `json:"id"`
	URL string `json:"url"`
}

type sourceStatus struct {
	URL string `json:"url"`
	SHA string `json:"sha"`
}

type demoStatus struct {
	repositoryLink
	SHA            string `json:"sha"`
	IssueCount     int    `json:"issueCount"`
	PipelineStatus string `json:"pipelineStatus"`
	PipelineURL    string `json:"pipelineUrl"`
}

type repositoryChecks struct {
	TutorsReady     bool `json:"tutorsReady"`
	DemoReady       bool `json:"demoReady"`
	MaterialCurrent bool `json:"materialCurrent"`
}

type courseInfrastructureStatus struct {
	SemesterTag string                     `json:"semesterTag"`
	Source      *sourceStatus              `json:"source"`
	Groups      map[string]*repositoryLink `json:"groups"`
	CIProject   *repositoryLink            `json:"ciProject"`
	DemoProject *demoStatus                `json:"demoProject"`
	Checks      repositoryChecks           `json:"checks"`
	Issues      []string                   `json:"issues"`
}

func groupLink(group *gitlab.Group) *repositoryLink {
	if group == nil {
		return nil
	}
	return &repositoryLink{ID: group.ID, URL: group.WebURL}
}

// CourseInfrastructureStatus always reads GitLab. A course-phase flag is not
// proof that the repo, tutor access, or current teaching material still exists.
func CourseInfrastructureStatus(ctx context.Context, coursePhaseID uuid.UUID, semesterTag string) (*courseInfrastructureStatus, error) {
	git, err := getClient()
	if err != nil {
		return nil, err
	}
	status := &courseInfrastructureStatus{
		SemesterTag: semesterTag,
		Groups:      map[string]*repositoryLink{"course": nil, "tutors": nil, "introCourse": nil},
		Issues:      []string{},
	}
	issue := func(message string) { status.Issues = append(status.Issues, message) }
	svc := InfrastructureServiceSingleton
	if svc.teachingMaterialProjectID == "" {
		issue("Teaching material project is not configured.")
		return status, nil
	}
	source, _, err := git.Projects.GetProject(svc.teachingMaterialProjectID, nil)
	if err != nil {
		return nil, fmt.Errorf("get teaching material project: %w", err)
	}
	sourceBranch, _, err := git.Branches.GetBranch(source.ID, "main")
	if err != nil || sourceBranch.Commit == nil {
		return nil, fmt.Errorf("get teaching material main branch: %w", err)
	}
	status.Source = &sourceStatus{URL: source.WebURL, SHA: sourceBranch.Commit.ID}

	root, err := getiPraktikumGroup()
	if err != nil {
		return nil, err
	}
	course, err := findSubGroup(semesterTag, root.ID)
	if err != nil {
		return nil, err
	}
	if course == nil {
		issue("Semester GitLab group has not been created.")
		return status, nil
	}
	status.Groups["course"] = groupLink(course)
	tutors, err := findSubGroup("tutors", course.ID)
	if err != nil {
		return nil, err
	}
	if tutors == nil {
		issue("Tutor GitLab group has not been created.")
	} else {
		status.Groups["tutors"] = groupLink(tutors)
		rows, rowErr := svc.queries.GetAllTutors(ctx, coursePhaseID)
		if rowErr != nil {
			return nil, fmt.Errorf("get imported tutors: %w", rowErr)
		}
		if len(rows) == 0 {
			issue("Import tutors before preparing the demo.")
		} else {
			ready := 0
			for _, tutor := range rows {
				if !tutor.GitlabUsername.Valid || strings.TrimSpace(tutor.GitlabUsername.String) == "" {
					continue
				}
				user, lookupErr := gitlabutil.GetUser(git, tutor.GitlabUsername.String)
				if lookupErr != nil {
					continue
				}
				member, _, memberErr := git.GroupMembers.GetGroupMember(tutors.ID, user.ID)
				if memberErr == nil && member.AccessLevel >= gitlab.DeveloperPermissions {
					ready++
				}
			}
			status.Checks.TutorsReady = ready == len(rows)
			if !status.Checks.TutorsReady {
				issue(fmt.Sprintf("GitLab tutor access is ready for %d of %d imported tutors.", ready, len(rows)))
			}
		}
	}
	intro, err := findSubGroup("Introcourse", course.ID)
	if err != nil {
		return nil, err
	}
	if intro == nil {
		issue("Introcourse GitLab group has not been created.")
		return status, nil
	}
	status.Groups["introCourse"] = groupLink(intro)
	if tutors != nil {
		introDetails, _, shareErr := git.Groups.GetGroup(intro.ID, nil)
		if shareErr != nil {
			return nil, fmt.Errorf("get Introcourse group access: %w", shareErr)
		}
		shared := false
		for _, granted := range introDetails.SharedWithGroups {
			if granted.GroupID == tutors.ID && granted.GroupAccessLevel >= int64(gitlab.DeveloperPermissions) {
				shared = true
				break
			}
		}
		if !shared {
			status.Checks.TutorsReady = false
			issue("The tutor group has not been granted access to Introcourse projects.")
		}
	}
	ciPath := intro.FullPath + "/ci-cd"
	ci, _, err := git.Projects.GetProject(ciPath, nil)
	if err == nil && strings.EqualFold(ci.PathWithNamespace, ciPath) {
		status.CIProject = &repositoryLink{ID: ci.ID, URL: ci.WebURL}
		developers, groupErr := findSubGroup("developer", course.ID)
		if groupErr != nil {
			return nil, fmt.Errorf("get developer group for CI access: %w", groupErr)
		}
		ciReadable := false
		if developers != nil {
			for _, shared := range ci.SharedWithGroups {
				if shared.GroupID == developers.ID && shared.GroupAccessLevel >= int64(gitlab.ReporterPermissions) {
					ciReadable = true
				}
			}
		}
		if !ciReadable {
			issue("Students do not have read access to the shared CI configuration project.")
		}
	} else if err != nil && !isNotFoundError(err) {
		return nil, fmt.Errorf("get CI project: %w", err)
	} else {
		issue("Shared CI project is missing.")
	}
	demoPath := intro.FullPath + "/demo"
	demo, _, err := git.Projects.GetProject(demoPath, nil)
	if err != nil {
		if isNotFoundError(err) {
			issue("Demo project has not been created.")
			return status, nil
		}
		return nil, fmt.Errorf("get demo project: %w", err)
	}
	if !strings.EqualFold(demo.PathWithNamespace, demoPath) {
		issue("The demo path redirects to an archived project; the path must be released before setup.")
		return status, nil
	}
	status.DemoProject = &demoStatus{repositoryLink: repositoryLink{ID: demo.ID, URL: demo.WebURL}}
	if !courseProjectSettingsMatch(demo, newCourseProjectOptions("demo", "demo", intro.ID, ciPath)) {
		issue("Demo CI path or merge-request settings differ from the course policy.")
	}
	branch, _, err := git.Branches.GetBranch(demo.ID, "main")
	if err != nil {
		issue("Demo main branch is missing.")
		return status, nil
	}
	if branch.Commit != nil {
		status.DemoProject.SHA = branch.Commit.ID
	}
	if !branch.Protected {
		issue("Demo main branch is not protected.")
	}
	matchingRules, protectionErr := listMatchingMainProtections(git, demo.ID)
	if protectionErr != nil {
		return nil, protectionErr
	}
	if len(matchingRules) != 1 || matchingRules[0].Name != "main" || !mainBranchProtectionMatches(matchingRules[0], 0) {
		issue("Demo main must reject direct pushes and allow only Maintainers to merge.")
	}
	if err := checkCourseStatusBoard(git, demo.ID, demo.PathWithNamespace); err != nil {
		issue("Demo needs its GitLab status board: " + err.Error())
	}
	pipelines, _, pipelineErr := git.Pipelines.ListProjectPipelines(demo.ID, &gitlab.ListProjectPipelinesOptions{
		Ref: gitlab.Ptr("main"), ListOptions: gitlab.ListOptions{PerPage: 1},
	})
	if pipelineErr != nil {
		return nil, fmt.Errorf("get demo main pipeline: %w", pipelineErr)
	}
	if len(pipelines) > 0 {
		status.DemoProject.PipelineStatus = pipelines[0].Status
		status.DemoProject.PipelineURL = pipelines[0].WebURL
	}
	if len(pipelines) == 0 || pipelines[0].SHA != status.DemoProject.SHA || pipelines[0].Status != "success" {
		issue("The latest demo main pipeline has not passed for its current commit.")
	}
	branches, err := gitlab.ScanAndCollect(func(p gitlab.PaginationOptionFunc) ([]*gitlab.Branch, *gitlab.Response, error) {
		return git.Branches.ListBranches(demo.ID, &gitlab.ListBranchesOptions{ListOptions: gitlab.ListOptions{PerPage: 100}}, p)
	})
	if err != nil {
		return nil, fmt.Errorf("list demo branches: %w", err)
	}
	for _, candidate := range branches {
		if candidate.Name != "main" {
			issue("Demo has exercise branches; reset it before student initialization.")
			break
		}
	}
	openMRs, err := gitlab.ScanAndCollect(func(p gitlab.PaginationOptionFunc) ([]*gitlab.BasicMergeRequest, *gitlab.Response, error) {
		return git.MergeRequests.ListProjectMergeRequests(demo.ID, &gitlab.ListProjectMergeRequestsOptions{
			State: gitlab.Ptr("opened"), ListOptions: gitlab.ListOptions{PerPage: 100},
		}, p)
	})
	if err != nil {
		return nil, fmt.Errorf("list demo merge requests: %w", err)
	}
	if len(openMRs) > 0 {
		issue("Demo has open merge requests; reset it before student initialization.")
	}
	if tutors != nil {
		rules, _, rulesErr := git.Projects.GetProjectApprovalRules(demo.ID, nil)
		if rulesErr != nil {
			return nil, fmt.Errorf("get demo approval rules: %w", rulesErr)
		}
		valid := false
		for _, rule := range rules {
			if rule.Name == "Tutor Approval" && tutorApprovalRuleIsStrict(rule, tutors.ID) {
				valid = true
			}
		}
		if !valid {
			issue("Demo approval rule does not restrict approval to the tutor group.")
		}
	}
	issues, err := gitlab.ScanAndCollect(func(p gitlab.PaginationOptionFunc) ([]*gitlab.Issue, *gitlab.Response, error) {
		return git.Issues.ListProjectIssues(demo.ID, &gitlab.ListProjectIssuesOptions{ListOptions: gitlab.ListOptions{PerPage: 100}}, p)
	})
	if err != nil {
		return nil, fmt.Errorf("list demo issues: %w", err)
	}
	status.DemoProject.IssueCount = len(issues)
	current, compareErr := demoMatchesSource(git, svc.teachingMaterialProjectID, status.Source.SHA, demo.ID, issues, status.CIProject)
	if compareErr != nil {
		return nil, compareErr
	}
	status.Checks.MaterialCurrent = current
	if !current {
		issue("Demo files, daily issues, or shared CI differ from current teaching material. Reset the demo after testing.")
	}
	status.Checks.DemoReady = branch.Protected && current && status.Checks.TutorsReady && len(status.Issues) == 0
	return status, nil
}

func demoMatchesSource(git *gitlab.Client, sourceProjectID, sourceSHA string, demoID int64, demoIssues []*gitlab.Issue, ci *repositoryLink) (bool, error) {
	templates, err := fetchTemplateFilesAtRef(git, sourceProjectID, sourceSHA)
	if err != nil {
		return false, err
	}
	nodes, err := gitlab.ScanAndCollect(func(p gitlab.PaginationOptionFunc) ([]*gitlab.TreeNode, *gitlab.Response, error) {
		return git.Repositories.ListTree(demoID, &gitlab.ListTreeOptions{
			Ref: gitlab.Ptr("main"), Recursive: gitlab.Ptr(true), ListOptions: gitlab.ListOptions{PerPage: 100},
		}, p)
	})
	if err != nil {
		return false, fmt.Errorf("list demo files: %w", err)
	}
	paths := make(map[string]string)
	for _, node := range nodes {
		if node.Type == "blob" {
			paths[node.Path] = node.Mode
		}
	}
	if len(paths) != len(templates) {
		return false, nil
	}
	for _, file := range templates {
		mode := "100644"
		if file.ExecuteFilemode {
			mode = "100755"
		}
		if paths[file.Path] != mode {
			return false, nil
		}
		actual, _, readErr := git.RepositoryFiles.GetRawFile(demoID, file.Path, &gitlab.GetRawFileOptions{Ref: gitlab.Ptr("main")})
		if readErr != nil || string(actual) != applyTemplateVars(file.Content, templateVars{StudentName: "Demo", SubmissionDeadline: "See the course schedule in Outline"}) {
			return false, nil
		}
	}
	expectedIssues, err := fetchIssueTemplatesAtRef(git, sourceProjectID, sourceSHA)
	if err != nil {
		return false, err
	}
	if len(expectedIssues) == 0 || len(expectedIssues) != len(demoIssues) {
		return false, nil
	}
	byTitle := make(map[string]string, len(demoIssues))
	for _, item := range demoIssues {
		byTitle[item.Title] = item.Description
	}
	for _, expected := range expectedIssues {
		if byTitle[expected.Title] != expected.Description {
			return false, nil
		}
	}
	if ci == nil {
		return false, nil
	}
	ciFiles, err := fetchCICDFilesAtRef(git, sourceProjectID, sourceSHA)
	if err != nil {
		return false, err
	}
	if len(ciFiles) == 0 {
		return false, nil
	}
	for _, file := range ciFiles {
		actual, _, readErr := git.RepositoryFiles.GetRawFile(ci.ID, file.Path, &gitlab.GetRawFileOptions{Ref: gitlab.Ptr("main")})
		if readErr != nil || string(actual) != file.Content {
			return false, nil
		}
	}
	return true, nil
}
