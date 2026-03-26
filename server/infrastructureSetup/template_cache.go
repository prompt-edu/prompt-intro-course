package infrastructureSetup

import (
	"fmt"
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

const templateDir = "student_repo_template"
const templateRef = "main"

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
	replacements := map[string]string{
		"{{.StudentName}}":        vars.StudentName,
		"{{.SubmissionDeadline}}": vars.SubmissionDeadline,
	}
	for placeholder, value := range replacements {
		content = strings.ReplaceAll(content, placeholder, value)
	}
	return content
}

