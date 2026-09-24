package infrastructureSetup

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

type materialSnapshot struct {
	sha       string
	templates []templateFile
	issues    []issueTemplate
	ciFiles   []templateFile
}

type resetDemoRequest struct {
	SemesterTag       string `json:"semesterTag" binding:"required"`
	ExpectedProjectID int64  `json:"expectedProjectID" binding:"required"`
	ExpectedSourceSHA string `json:"expectedSourceSHA" binding:"required"`
}

type resetDemoResult struct {
	DemoURL    string `json:"demoUrl"`
	DemoID     int64  `json:"demoId"`
	ArchiveURL string `json:"archiveUrl"`
	SourceSHA  string `json:"sourceSha"`
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
	if len(templates) == 0 || len(issues) == 0 || len(ciFiles) == 0 {
		return nil, fmt.Errorf("teaching material must contain repository files, daily issues, and CI configuration")
	}
	return &materialSnapshot{sha: sha, templates: templates, issues: issues, ciFiles: ciFiles}, nil
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

// ResetDemo preserves the old project and its merge requests under a dated
// path, then creates a new student-like demo from one immutable source commit.
// No student project is created, edited, or deleted here.
func ResetDemo(ctx context.Context, coursePhaseID uuid.UUID, request resetDemoRequest) (*resetDemoResult, error) {
	if request.ExpectedProjectID <= 0 || len(request.ExpectedSourceSHA) != 40 {
		return nil, fmt.Errorf("a current demo ID and source commit are required")
	}
	semesterTag := request.SemesterTag
	status, err := CourseInfrastructureStatus(ctx, coursePhaseID, semesterTag)
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
	introPath := ""
	oldProject, _, err := git.Projects.GetProject(request.ExpectedProjectID, nil)
	if err != nil {
		return nil, err
	}
	introPath = strings.TrimSuffix(oldProject.PathWithNamespace, "/demo")
	if oldProject.Path != "demo" || oldProject.PathWithNamespace != introPath+"/demo" || !strings.HasSuffix(oldProject.PathWithNamespace, "/Introcourse/demo") {
		return nil, fmt.Errorf("the expected project is no longer the active demo")
	}
	archiveGroup, err := createTeachingGroup(status.Groups["introCourse"].ID, "demo-archives")
	if err != nil {
		return nil, fmt.Errorf("prepare demo archive group: %w", err)
	}
	archivePath := fmt.Sprintf("demo-before-reset-%s-%s", time.Now().UTC().Format("20060102-150405"), uuid.NewString()[:6])
	archived, _, err := git.Projects.EditProject(oldProject.ID, &gitlab.EditProjectOptions{
		Name: gitlab.Ptr("Demo before reset"), Path: gitlab.Ptr(archivePath),
	})
	if err != nil {
		return nil, fmt.Errorf("archive current demo: %w", err)
	}
	if archived.ID != oldProject.ID || archived.Path != archivePath {
		return nil, fmt.Errorf("GitLab did not confirm the demo archive path")
	}
	result := &resetDemoResult{ArchiveURL: archived.WebURL, SourceSHA: material.sha}
	createErr := createDemoProjectWithMaterial(git, status.Groups["introCourse"].ID, introPath, status.Groups["tutors"].ID, material)
	if createErr != nil {
		// Roll back only when the original path is still vacant. If another
		// actor has claimed it, keep the archive and report both paths.
		current, _, readErr := git.Projects.GetProject(introPath+"/demo", nil)
		if readErr != nil || current.ID == oldProject.ID || !strings.EqualFold(current.PathWithNamespace, introPath+"/demo") {
			_, _, rollbackErr := git.Projects.EditProject(oldProject.ID, &gitlab.EditProjectOptions{Name: gitlab.Ptr("demo"), Path: gitlab.Ptr("demo")})
			if rollbackErr != nil {
				return result, fmt.Errorf("create replacement demo: %w; restore archived demo: %v (archive: %s)", createErr, rollbackErr, archived.WebURL)
			}
			return nil, fmt.Errorf("create replacement demo: %w; original demo restored", createErr)
		}
		return result, fmt.Errorf("replacement demo needs repair: %w (archive: %s)", createErr, archived.WebURL)
	}
	newProject, _, err := git.Projects.GetProject(introPath+"/demo", nil)
	if err != nil || newProject.ID == oldProject.ID || !strings.EqualFold(newProject.PathWithNamespace, introPath+"/demo") {
		return result, fmt.Errorf("new demo path could not be verified; archived demo remains at %s", archived.WebURL)
	}
	result.DemoURL = newProject.WebURL
	result.DemoID = newProject.ID
	// Keep repeated experiments out of the active Introcourse project list.
	_, _, err = git.Projects.TransferProject(oldProject.ID, &gitlab.TransferProjectOptions{Namespace: archiveGroup.ID})
	if err != nil {
		return result, fmt.Errorf("replacement demo exists, but move old demo to archives failed: %w", err)
	}
	// GitLab accepts transfers before its background worker moves the project.
	// An immediate GET can still return the old namespace; the old URL remains
	// usable and redirects after the transfer completes.
	return result, nil
}
