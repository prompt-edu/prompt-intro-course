package infrastructureSetup

import (
	"fmt"
	"sort"
	"strings"
	"sync"

	log "github.com/sirupsen/logrus"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

// templateFile represents a single file fetched from the teaching material
// repository's student_repo_template/ directory.
type templateFile struct {
	// Path is the file path relative to the student repo root
	// (e.g. "README.md", ".githooks/post-checkout").
	Path string
	// Content is the raw file content (may contain template placeholders).
	Content string
	// ExecuteFilemode indicates whether the file should be committed with
	// the executable bit set (mode 100755 in the source repo).
	ExecuteFilemode bool
}

// templateVars holds the values substituted into template placeholders.
type templateVars struct {
	StudentName        string
	SubmissionDeadline string
}

// templateCache provides thread-safe, fetch-once caching of template files
// from the teaching material GitLab repository. On the first successful fetch,
// the result is stored and reused for the lifetime of the process. If the
// initial fetch fails, subsequent calls will retry (not permanently broken).
type templateCache struct {
	mu    sync.Mutex
	files []templateFile
}

const (
	templateDir      = "student_repo_template"
	templateRef      = "main"
	maxTemplateBytes = 1 << 20 // 1 MiB — guard against accidental binary commits
)

// get returns cached template files, fetching them on first call.
// Thread-safe: concurrent callers block until the first fetch completes.
func (tc *templateCache) get(client *gitlab.Client, projectID string) ([]templateFile, error) {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	if tc.files != nil {
		return tc.files, nil
	}

	files, err := fetchTemplateFiles(client, projectID)
	if err != nil {
		return nil, err
	}

	tc.files = files
	log.WithField("count", len(files)).Info("template files cached from teaching material repo")
	return tc.files, nil
}

// fetchTemplateFiles lists all files under student_repo_template/ in the
// teaching material repo and fetches their content. Uses auto-pagination
// to handle directories with more than 100 entries.
func fetchTemplateFiles(client *gitlab.Client, projectID string) ([]templateFile, error) {
	opts := &gitlab.ListTreeOptions{
		Path:      gitlab.Ptr(templateDir),
		Ref:       gitlab.Ptr(templateRef),
		Recursive: gitlab.Ptr(true),
		ListOptions: gitlab.ListOptions{
			PerPage: 100,
		},
	}

	nodes, err := gitlab.ScanAndCollect(func(p gitlab.PaginationOptionFunc) ([]*gitlab.TreeNode, *gitlab.Response, error) {
		return client.Repositories.ListTree(projectID, opts, p)
	})
	if err != nil {
		return nil, fmt.Errorf("list template tree in project %q: %w", projectID, err)
	}

	var files []templateFile
	for _, node := range nodes {
		if node.Type != "blob" {
			continue // skip directories
		}

		raw, _, err := client.RepositoryFiles.GetRawFile(projectID, node.Path, &gitlab.GetRawFileOptions{
			Ref: gitlab.Ptr(templateRef),
		})
		if err != nil {
			return nil, fmt.Errorf("fetch file %q from project %q: %w", node.Path, projectID, err)
		}
		if len(raw) > maxTemplateBytes {
			return nil, fmt.Errorf("file %q in project %q exceeds max size (%d bytes > %d)", node.Path, projectID, len(raw), maxTemplateBytes)
		}

		relPath := strings.TrimPrefix(node.Path, templateDir+"/")

		files = append(files, templateFile{
			Path:            relPath,
			Content:         string(raw),
			ExecuteFilemode: node.Mode == "100755",
		})
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("no template files found in %s/ of project %q", templateDir, projectID)
	}

	return files, nil
}

// applyTemplateVars replaces {{.VarName}} placeholders in content with the
// corresponding values from vars. Files without placeholders are unchanged.
func applyTemplateVars(content string, vars templateVars) string {
	return strings.NewReplacer(
		"{{.StudentName}}", vars.StudentName,
		"{{.SubmissionDeadline}}", vars.SubmissionDeadline,
	).Replace(content)
}

// --- CI/CD Pipeline Templates ---

const cicdDir = "ci_cd"

// cicdCache provides thread-safe, fetch-once caching of CI/CD pipeline files
// from the teaching material repo. Same retry-on-failure semantics as templateCache.
type cicdCache struct {
	mu    sync.Mutex
	files []templateFile
}

// get returns cached CI/CD files, fetching them on first call.
func (cc *cicdCache) get(client *gitlab.Client, projectID string) ([]templateFile, error) {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	if cc.files != nil {
		return cc.files, nil
	}

	files, err := fetchCICDFiles(client, projectID)
	if err != nil {
		return nil, err
	}

	cc.files = files
	log.WithField("count", len(files)).Info("CI/CD pipeline files cached from teaching material repo")
	return cc.files, nil
}

// fetchCICDFiles lists all files under ci_cd/ in the teaching material repo
// and fetches their content. Returns an empty slice (no error) if the directory
// does not exist — CI/CD pipeline config is optional.
func fetchCICDFiles(client *gitlab.Client, projectID string) ([]templateFile, error) {
	opts := &gitlab.ListTreeOptions{
		Path:      gitlab.Ptr(cicdDir),
		Ref:       gitlab.Ptr(templateRef),
		Recursive: gitlab.Ptr(true),
		ListOptions: gitlab.ListOptions{
			PerPage: 100,
		},
	}

	nodes, err := gitlab.ScanAndCollect(func(p gitlab.PaginationOptionFunc) ([]*gitlab.TreeNode, *gitlab.Response, error) {
		return client.Repositories.ListTree(projectID, opts, p)
	})
	if err != nil {
		// If the directory doesn't exist, GitLab returns 404 — treat as empty
		return nil, nil //nolint:nilerr // missing ci_cd/ dir is valid (optional)
	}

	var files []templateFile
	for _, node := range nodes {
		if node.Type != "blob" {
			continue
		}

		raw, _, err := client.RepositoryFiles.GetRawFile(projectID, node.Path, &gitlab.GetRawFileOptions{
			Ref: gitlab.Ptr(templateRef),
		})
		if err != nil {
			return nil, fmt.Errorf("fetch CI/CD file %q from project %q: %w", node.Path, projectID, err)
		}
		if len(raw) > maxTemplateBytes {
			return nil, fmt.Errorf("CI/CD file %q in project %q exceeds max size (%d bytes > %d)", node.Path, projectID, len(raw), maxTemplateBytes)
		}

		relPath := strings.TrimPrefix(node.Path, cicdDir+"/")
		files = append(files, templateFile{
			Path:            relPath,
			Content:         string(raw),
			ExecuteFilemode: node.Mode == "100755",
		})
	}

	return files, nil
}

// --- Daily Issue Templates ---

const dailyIssuesDir = "daily_issues"

// issueTemplate represents a parsed issue from the teaching material repo.
type issueTemplate struct {
	Title       string
	Description string
}

// issueCache provides thread-safe, fetch-once caching of issue templates.
// Same retry-on-failure semantics as templateCache.
type issueCache struct {
	mu     sync.Mutex
	issues []issueTemplate
}

// get returns cached issue templates, fetching them on first call.
func (ic *issueCache) get(client *gitlab.Client, projectID string) ([]issueTemplate, error) {
	ic.mu.Lock()
	defer ic.mu.Unlock()

	if ic.issues != nil {
		return ic.issues, nil
	}

	issues, err := fetchIssueTemplates(client, projectID)
	if err != nil {
		return nil, err
	}

	ic.issues = issues
	log.WithField("count", len(issues)).Info("issue templates cached from teaching material repo")
	return ic.issues, nil
}

// fetchIssueTemplates lists all .md files under daily_issues/ and parses each
// into a title (from the first # heading) and description (the rest).
func fetchIssueTemplates(client *gitlab.Client, projectID string) ([]issueTemplate, error) {
	opts := &gitlab.ListTreeOptions{
		Path:      gitlab.Ptr(dailyIssuesDir),
		Ref:       gitlab.Ptr(templateRef),
		Recursive: gitlab.Ptr(false),
		ListOptions: gitlab.ListOptions{
			PerPage: 100,
		},
	}

	nodes, err := gitlab.ScanAndCollect(func(p gitlab.PaginationOptionFunc) ([]*gitlab.TreeNode, *gitlab.Response, error) {
		return client.Repositories.ListTree(projectID, opts, p)
	})
	if err != nil {
		return nil, fmt.Errorf("list issue templates in project %q: %w", projectID, err)
	}

	// Sort by path for deterministic ordering (day1_* before day2_*)
	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].Path < nodes[j].Path
	})

	var issues []issueTemplate
	for _, node := range nodes {
		if node.Type != "blob" || !strings.HasSuffix(node.Name, ".md") {
			continue
		}

		raw, _, err := client.RepositoryFiles.GetRawFile(projectID, node.Path, &gitlab.GetRawFileOptions{
			Ref: gitlab.Ptr(templateRef),
		})
		if err != nil {
			return nil, fmt.Errorf("fetch issue file %q: %w", node.Path, err)
		}

		title, description := parseIssueContent(string(raw))
		if title == "" {
			continue
		}
		issues = append(issues, issueTemplate{Title: title, Description: description})
	}

	return issues, nil
}

// parseIssueContent extracts the title from the first # heading and the
// description from the remaining content.
func parseIssueContent(content string) (title, description string) {
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		if strings.HasPrefix(line, "# ") {
			title = strings.TrimSpace(strings.TrimPrefix(line, "# "))
			rest := strings.Join(lines[i+1:], "\n")
			description = strings.TrimSpace(rest)
			return
		}
	}
	return "", strings.TrimSpace(content)
}

