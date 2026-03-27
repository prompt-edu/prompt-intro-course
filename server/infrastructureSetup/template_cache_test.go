package infrastructureSetup

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

// makeGitLabError constructs a gitlab.ErrorResponse for testing isAlreadyExistsError.
func makeGitLabError(statusCode int, message string) *gitlab.ErrorResponse {
	return &gitlab.ErrorResponse{
		Response: &http.Response{
			StatusCode: statusCode,
			Request:    &http.Request{URL: &url.URL{Path: "/test"}},
		},
		Message: message,
	}
}

func TestApplyTemplateVars(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		vars     templateVars
		expected string
	}{
		{
			name:    "replaces student name and deadline",
			content: "# {{.StudentName}}'s App\n\n**Deadline:** **{{.SubmissionDeadline}}**",
			vars: templateVars{
				StudentName:        "Alice",
				SubmissionDeadline: "2026-04-01",
			},
			expected: "# Alice's App\n\n**Deadline:** **2026-04-01**",
		},
		{
			name:    "no placeholders returns content unchanged",
			content: "This file has no variables at all.",
			vars: templateVars{
				StudentName:        "Bob",
				SubmissionDeadline: "2026-05-01",
			},
			expected: "This file has no variables at all.",
		},
		{
			name:    "multiple occurrences of same placeholder",
			content: "Hello {{.StudentName}}, welcome {{.StudentName}}!",
			vars: templateVars{
				StudentName:        "Charlie",
				SubmissionDeadline: "2026-06-01",
			},
			expected: "Hello Charlie, welcome Charlie!",
		},
		{
			name:     "empty vars produce empty replacements",
			content:  "# {{.StudentName}}'s App",
			vars:     templateVars{},
			expected: "# 's App",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := applyTemplateVars(tt.content, tt.vars)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// fakeGitLabServer creates a test HTTP server that mimics the GitLab API
// endpoints used by fetchTemplateFiles. fileContents keys are the decoded
// file paths (e.g. "student_repo_template/README.md").
func fakeGitLabServer(t *testing.T, treeEntries []map[string]interface{}, fileContents map[string]string) *httptest.Server {
	t.Helper()

	return fakeGitLabServerPaginated(t, [][]map[string]interface{}{treeEntries}, fileContents)
}

// fakeGitLabServerPaginated creates a test HTTP server that supports paginated
// tree listing. Each element of treePages is a page of tree entries.
// The server sets X-Page, X-Next-Page, and X-Total-Pages headers to drive
// go-gitlab's ScanAndCollect pagination.
func fakeGitLabServerPaginated(t *testing.T, treePages [][]map[string]interface{}, fileContents map[string]string) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// Tree listing endpoint
		if path == "/api/v4/projects/42/repository/tree" {
			pageStr := r.URL.Query().Get("page")
			page := 1
			if pageStr != "" {
				if _, err := fmt.Sscanf(pageStr, "%d", &page); err != nil {
					t.Errorf("invalid page param %q: %v", pageStr, err)
					http.Error(w, "bad page", http.StatusBadRequest)
					return
				}
			}
			if page < 1 || page > len(treePages) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = fmt.Fprint(w, "[]")
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("X-Page", fmt.Sprintf("%d", page))
			w.Header().Set("X-Total-Pages", fmt.Sprintf("%d", len(treePages)))
			if page < len(treePages) {
				w.Header().Set("X-Next-Page", fmt.Sprintf("%d", page+1))
			}
			if err := json.NewEncoder(w).Encode(treePages[page-1]); err != nil {
				t.Fatalf("encode tree page %d: %v", page, err)
			}
			return
		}

		// File content endpoint: /api/v4/projects/42/repository/files/<path>/raw
		prefix := "/api/v4/projects/42/repository/files/"
		suffix := "/raw"
		if strings.HasPrefix(path, prefix) && strings.HasSuffix(path, suffix) {
			filePath := strings.TrimPrefix(path, prefix)
			filePath = strings.TrimSuffix(filePath, suffix)
			if content, ok := fileContents[filePath]; ok {
				w.Header().Set("Content-Type", "application/octet-stream")
				_, _ = fmt.Fprint(w, content)
				return
			}
		}

		http.NotFound(w, r)
	}))
}

func TestFetchTemplateFiles(t *testing.T) {
	treeEntries := []map[string]interface{}{
		{"id": "abc1", "name": "README.md", "type": "blob", "path": "student_repo_template/README.md", "mode": "100644"},
		{"id": "abc2", "name": ".githooks", "type": "tree", "path": "student_repo_template/.githooks", "mode": "040000"},
		{"id": "abc3", "name": "post-checkout", "type": "blob", "path": "student_repo_template/.githooks/post-checkout", "mode": "100755"},
		{"id": "abc4", "name": ".gitignore", "type": "blob", "path": "student_repo_template/.gitignore", "mode": "100644"},
	}

	// Keys are the decoded file paths (matching r.URL.Path after ServeMux decoding)
	fileContents := map[string]string{
		"student_repo_template/README.md":                "# {{.StudentName}}'s App\nDeadline: {{.SubmissionDeadline}}",
		"student_repo_template/.githooks/post-checkout":  "#!/bin/bash\nxcodegen generate\n",
		"student_repo_template/.gitignore":               "*.xcodeproj\n",
	}

	server := fakeGitLabServer(t, treeEntries, fileContents)
	defer server.Close()

	client, err := gitlab.NewClient("test-token", gitlab.WithBaseURL(server.URL+"/api/v4"))
	require.NoError(t, err)

	files, err := fetchTemplateFiles(client, "42")
	require.NoError(t, err)
	require.Len(t, files, 3) // 3 blobs, 1 tree skipped

	// Verify paths are stripped of the template directory prefix
	pathMap := make(map[string]templateFile)
	for _, f := range files {
		pathMap[f.Path] = f
	}

	readme := pathMap["README.md"]
	assert.Contains(t, readme.Content, "{{.StudentName}}")
	assert.False(t, readme.ExecuteFilemode)

	hook := pathMap[".githooks/post-checkout"]
	assert.Contains(t, hook.Content, "xcodegen generate")
	assert.True(t, hook.ExecuteFilemode, "githooks should have executable flag")

	gitignore := pathMap[".gitignore"]
	assert.Contains(t, gitignore.Content, "*.xcodeproj")
	assert.False(t, gitignore.ExecuteFilemode)
}

func TestTemplateCacheThreadSafety(t *testing.T) {
	var fetchCount atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		if path == "/api/v4/projects/42/repository/tree" {
			fetchCount.Add(1)
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode([]map[string]interface{}{
				{"id": "a1", "name": "file.txt", "type": "blob", "path": "student_repo_template/file.txt", "mode": "100644"},
			})
			return
		}

		if strings.Contains(path, "/repository/files/") && strings.HasSuffix(path, "/raw") {
			_, _ = fmt.Fprint(w, "content")
			return
		}

		http.NotFound(w, r)
	}))
	defer server.Close()

	client, err := gitlab.NewClient("test-token", gitlab.WithBaseURL(server.URL+"/api/v4"))
	require.NoError(t, err)

	cache := &templateCache{}

	// Launch concurrent goroutines that all try to load the cache
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			files, err := cache.get(client, "42")
			assert.NoError(t, err)
			assert.Len(t, files, 1)
		}()
	}
	wg.Wait()

	// The tree endpoint should have been called exactly once (cached after first call)
	assert.Equal(t, int32(1), fetchCount.Load(), "template tree should be fetched exactly once")
}

func TestTemplateCacheRetryOnError(t *testing.T) {
	var callCount atomic.Int32

	// Use 401 Unauthorized because the go-gitlab client automatically retries
	// on 5xx errors, making 500-based failure tests unreliable.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		if path == "/api/v4/projects/42/repository/tree" {
			count := callCount.Add(1)
			if count == 1 {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = fmt.Fprint(w, `{"message":"401 Unauthorized"}`)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode([]map[string]interface{}{
				{"id": "a1", "name": "file.txt", "type": "blob", "path": "student_repo_template/file.txt", "mode": "100644"},
			})
			return
		}

		if strings.Contains(path, "/repository/files/") && strings.HasSuffix(path, "/raw") {
			_, _ = fmt.Fprint(w, "file content")
			return
		}

		http.NotFound(w, r)
	}))
	defer server.Close()

	client, err := gitlab.NewClient("test-token", gitlab.WithBaseURL(server.URL+"/api/v4"))
	require.NoError(t, err)

	cache := &templateCache{}

	// First call should fail
	_, err = cache.get(client, "42")
	assert.Error(t, err)

	// Second call should succeed (retry after failure)
	files, err := cache.get(client, "42")
	assert.NoError(t, err)
	assert.Len(t, files, 1)

	// Third call should use cache (no additional fetch)
	files, err = cache.get(client, "42")
	assert.NoError(t, err)
	assert.Len(t, files, 1)
	assert.Equal(t, int32(2), callCount.Load(), "should fetch twice (1 failure + 1 success)")
}

func TestFetchTemplateFilesEmptyRepo(t *testing.T) {
	treeEntries := []map[string]interface{}{
		// Only tree entries, no blobs
		{"id": "abc1", "name": ".githooks", "type": "tree", "path": "student_repo_template/.githooks", "mode": "040000"},
	}

	server := fakeGitLabServer(t, treeEntries, nil)
	defer server.Close()

	client, err := gitlab.NewClient("test-token", gitlab.WithBaseURL(server.URL+"/api/v4"))
	require.NoError(t, err)

	_, err = fetchTemplateFiles(client, "42")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no template files found")
}

func TestCreateProjectFiles(t *testing.T) {
	// Captures the full commit body sent to the Commits API
	var commitBody map[string]interface{}
	var targetProjectPath string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// Tree listing endpoint (teaching material repo, project 100)
		if path == "/api/v4/projects/100/repository/tree" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode([]map[string]interface{}{
				{"id": "a1", "name": "README.md", "type": "blob", "path": "student_repo_template/README.md", "mode": "100644"},
				{"id": "a2", "name": "post-checkout", "type": "blob", "path": "student_repo_template/.githooks/post-checkout", "mode": "100755"},
			})
			return
		}

		// Raw file content endpoints (teaching material repo)
		filePrefix := "/api/v4/projects/100/repository/files/"
		fileSuffix := "/raw"
		if strings.HasPrefix(path, filePrefix) && strings.HasSuffix(path, fileSuffix) {
			filePath := strings.TrimPrefix(path, filePrefix)
			filePath = strings.TrimSuffix(filePath, fileSuffix)
			contents := map[string]string{
				"student_repo_template/README.md":               "# {{.StudentName}}'s App\nDeadline: {{.SubmissionDeadline}}",
				"student_repo_template/.githooks/post-checkout": "#!/bin/bash\nxcodegen generate\n",
			}
			if content, ok := contents[filePath]; ok {
				w.Header().Set("Content-Type", "application/octet-stream")
				_, _ = fmt.Fprint(w, content)
				return
			}
		}

		// CreateCommit endpoint (student repo, project 200)
		if strings.HasSuffix(path, "/repository/commits") && r.Method == http.MethodPost {
			targetProjectPath = path
			if err := json.NewDecoder(r.Body).Decode(&commitBody); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"id":      "abc123",
				"message": commitBody["commit_message"],
			})
			return
		}

		http.NotFound(w, r)
	}))
	defer server.Close()

	client, err := gitlab.NewClient("test-token", gitlab.WithBaseURL(server.URL+"/api/v4"))
	require.NoError(t, err)

	originalSingleton := InfrastructureServiceSingleton
	defer func() { InfrastructureServiceSingleton = originalSingleton }()

	InfrastructureServiceSingleton = &InfrastructureService{
		teachingMaterialProjectID: "100",
	}

	err = createProjectFiles(client, 200, "test-student-repo", templateVars{
		StudentName:        "Alice",
		SubmissionDeadline: "2026-04-01",
	})
	require.NoError(t, err)

	// Verify commit targets the correct project
	assert.Equal(t, "/api/v4/projects/200/repository/commits", targetProjectPath)

	// Verify commit metadata
	assert.Equal(t, "main", commitBody["branch"])
	assert.Equal(t, "Initialize repository from course template", commitBody["commit_message"])

	// Verify file actions
	actions := commitBody["actions"].([]interface{})
	require.Len(t, actions, 2, "should create exactly one commit with 2 file actions")

	actionMap := make(map[string]map[string]interface{})
	for _, a := range actions {
		action := a.(map[string]interface{})
		actionMap[action["file_path"].(string)] = action
	}

	readme := actionMap["README.md"]
	assert.Equal(t, "create", readme["action"])
	assert.Contains(t, readme["content"], "Alice")
	assert.Contains(t, readme["content"], "2026-04-01")
	assert.NotContains(t, readme["content"], "{{.StudentName}}")

	hook := actionMap[".githooks/post-checkout"]
	assert.Equal(t, "create", hook["action"])
	assert.Contains(t, hook["content"], "xcodegen generate")
	assert.Equal(t, true, hook["execute_filemode"], "githook should have executable flag")
}

func TestCreateProjectFilesIdempotent(t *testing.T) {
	// Verify that createProjectFiles handles "already exists" errors gracefully
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		if path == "/api/v4/projects/100/repository/tree" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode([]map[string]interface{}{
				{"id": "a1", "name": "file.txt", "type": "blob", "path": "student_repo_template/file.txt", "mode": "100644"},
			})
			return
		}

		if strings.Contains(path, "/repository/files/") && strings.HasSuffix(path, "/raw") {
			_, _ = fmt.Fprint(w, "content")
			return
		}

		// CreateCommit returns "already exists" error (repo already initialized)
		if path == "/api/v4/projects/200/repository/commits" && r.Method == http.MethodPost {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"message": "A file with this name already exists",
			})
			return
		}

		http.NotFound(w, r)
	}))
	defer server.Close()

	client, err := gitlab.NewClient("test-token", gitlab.WithBaseURL(server.URL+"/api/v4"))
	require.NoError(t, err)

	originalSingleton := InfrastructureServiceSingleton
	defer func() { InfrastructureServiceSingleton = originalSingleton }()

	InfrastructureServiceSingleton = &InfrastructureService{
		teachingMaterialProjectID: "100",
	}

	// Should succeed (silently skip since files already exist)
	err = createProjectFiles(client, 200, "test-repo", templateVars{StudentName: "Alice"})
	assert.NoError(t, err)
}

func TestCreateProjectFilesNotConfigured(t *testing.T) {
	// Verify that createProjectFiles fails when teaching material project ID is not set
	originalSingleton := InfrastructureServiceSingleton
	defer func() { InfrastructureServiceSingleton = originalSingleton }()

	InfrastructureServiceSingleton = &InfrastructureService{
		teachingMaterialProjectID: "",
	}

	client, _ := gitlab.NewClient("test-token")
	err := createProjectFiles(client, 200, "test-repo", templateVars{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "GITLAB_TEACHING_MATERIAL_PROJECT_ID not configured")
}

func TestFetchTemplateFilesPagination(t *testing.T) {
	// Simulate two pages of tree entries to verify ScanAndCollect pagination
	page1 := []map[string]interface{}{
		{"id": "abc1", "name": "README.md", "type": "blob", "path": "student_repo_template/README.md", "mode": "100644"},
		{"id": "abc2", "name": ".gitignore", "type": "blob", "path": "student_repo_template/.gitignore", "mode": "100644"},
	}
	page2 := []map[string]interface{}{
		{"id": "abc3", "name": "post-checkout", "type": "blob", "path": "student_repo_template/.githooks/post-checkout", "mode": "100755"},
	}

	fileContents := map[string]string{
		"student_repo_template/README.md":               "# Hello",
		"student_repo_template/.gitignore":              "*.xcodeproj\n",
		"student_repo_template/.githooks/post-checkout": "#!/bin/bash\nxcodegen generate\n",
	}

	server := fakeGitLabServerPaginated(t, [][]map[string]interface{}{page1, page2}, fileContents)
	defer server.Close()

	client, err := gitlab.NewClient("test-token", gitlab.WithBaseURL(server.URL+"/api/v4"))
	require.NoError(t, err)

	files, err := fetchTemplateFiles(client, "42")
	require.NoError(t, err)
	require.Len(t, files, 3, "should fetch files from both pages")

	pathMap := make(map[string]templateFile)
	for _, f := range files {
		pathMap[f.Path] = f
	}

	assert.Contains(t, pathMap, "README.md")
	assert.Contains(t, pathMap, ".gitignore")
	assert.Contains(t, pathMap, ".githooks/post-checkout")
	assert.True(t, pathMap[".githooks/post-checkout"].ExecuteFilemode)
}

func TestIsAlreadyExistsError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "non-gitlab error",
			err:      fmt.Errorf("network timeout"),
			expected: false,
		},
		{
			name:     "409 Conflict",
			err:      makeGitLabError(http.StatusConflict, ""),
			expected: true,
		},
		{
			name:     "409 Conflict with message",
			err:      makeGitLabError(http.StatusConflict, "resource conflict"),
			expected: true,
		},
		{
			name:     "400 with already exists message",
			err:      makeGitLabError(http.StatusBadRequest, "A file with this name already exists"),
			expected: true,
		},
		{
			name:     "400 with already a member message",
			err:      makeGitLabError(http.StatusBadRequest, "Source user already a member of this group"),
			expected: true,
		},
		{
			name:     "400 with already been taken message",
			err:      makeGitLabError(http.StatusBadRequest, "Path has already been taken"),
			expected: true,
		},
		{
			name:     "400 with unrelated message",
			err:      makeGitLabError(http.StatusBadRequest, "invalid parameter value"),
			expected: false,
		},
		{
			name:     "500 server error not matched",
			err:      makeGitLabError(http.StatusInternalServerError, "internal server error"),
			expected: false,
		},
		{
			name:     "wrapped gitlab error still matched via errors.As",
			err:      fmt.Errorf("create project: %w", makeGitLabError(http.StatusConflict, "")),
			expected: true,
		},
		{
			name:     "wrapped non-gitlab error not matched",
			err:      fmt.Errorf("create project: %w", fmt.Errorf("something already exists")),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, isAlreadyExistsError(tt.err))
		})
	}
}

func TestCreateOrGetProject(t *testing.T) {
	t.Run("creates new project", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPost && r.URL.Path == "/api/v4/projects" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusCreated)
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"id":   42,
					"name": "test-project",
				})
				return
			}
			http.NotFound(w, r)
		}))
		defer server.Close()

		client, err := gitlab.NewClient("test-token", gitlab.WithBaseURL(server.URL+"/api/v4"))
		require.NoError(t, err)

		opts := &gitlab.CreateProjectOptions{
			Name:        gitlab.Ptr("test-project"),
			NamespaceID: gitlab.Ptr(int64(1)),
		}
		project, err := createOrGetProject(client, opts, "group/path")
		require.NoError(t, err)
		assert.Equal(t, "test-project", project.Name)
	})

	t.Run("returns existing project on conflict", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPost && r.URL.Path == "/api/v4/projects" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusConflict)
				_, _ = fmt.Fprint(w, `{"message":"conflict"}`)
				return
			}
			// GetProject: go-gitlab URL-encodes path slashes, Go decodes them
			if r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/api/v4/projects/") {
				w.Header().Set("Content-Type", "application/json")
				_, _ = fmt.Fprint(w, `{"id":99,"name":"existing-project"}`)
				return
			}
			http.NotFound(w, r)
		}))
		defer server.Close()

		client, err := gitlab.NewClient("test-token", gitlab.WithBaseURL(server.URL+"/api/v4"))
		require.NoError(t, err)

		opts := &gitlab.CreateProjectOptions{
			Name:        gitlab.Ptr("test-project"),
			NamespaceID: gitlab.Ptr(int64(1)),
		}
		project, err := createOrGetProject(client, opts, "group/path")
		require.NoError(t, err)
		assert.Equal(t, "existing-project", project.Name)
	})

	t.Run("propagates non-conflict error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			_, _ = fmt.Fprint(w, `{"message":"forbidden"}`)
		}))
		defer server.Close()

		client, err := gitlab.NewClient("test-token", gitlab.WithBaseURL(server.URL+"/api/v4"))
		require.NoError(t, err)

		opts := &gitlab.CreateProjectOptions{
			Name:        gitlab.Ptr("test-project"),
			NamespaceID: gitlab.Ptr(int64(1)),
		}
		_, err = createOrGetProject(client, opts, "group/path")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "create project")
	})
}

func TestCreateDemoProject(t *testing.T) {
	var (
		projectCreated     atomic.Bool
		branchProtected    atomic.Bool
		boardCreated       atomic.Bool
		sharedWithGroup    atomic.Bool
		dailyIssuesCreated atomic.Int32
		commitBody         map[string]interface{}
		createProjectBody  map[string]interface{}
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		w.Header().Set("Content-Type", "application/json")

		// CreateProject
		if r.Method == http.MethodPost && path == "/api/v4/projects" {
			projectCreated.Store(true)
			_ = json.NewDecoder(r.Body).Decode(&createProjectBody)
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"id": 300, "name": "demo",
			})
			return
		}

		// Template tree listing (teaching material repo)
		if path == "/api/v4/projects/100/repository/tree" {
			dirPath := r.URL.Query().Get("path")
			if dirPath == "daily_issues" {
				_ = json.NewEncoder(w).Encode([]map[string]interface{}{
					{"id": "i1", "name": "day1_intro.md", "type": "blob", "path": "daily_issues/day1_intro.md", "mode": "100644"},
				})
				return
			}
			_ = json.NewEncoder(w).Encode([]map[string]interface{}{
				{"id": "a1", "name": "README.md", "type": "blob", "path": "student_repo_template/README.md", "mode": "100644"},
			})
			return
		}

		// Raw file content
		if strings.HasPrefix(path, "/api/v4/projects/100/repository/files/") && strings.HasSuffix(path, "/raw") {
			filePath := strings.TrimPrefix(path, "/api/v4/projects/100/repository/files/")
			filePath = strings.TrimSuffix(filePath, "/raw")
			if strings.HasPrefix(filePath, "daily_issues") {
				_, _ = fmt.Fprint(w, "# Day 1: Introduction\n\nWelcome to the course.")
				return
			}
			w.Header().Set("Content-Type", "application/octet-stream")
			_, _ = fmt.Fprint(w, "# {{.StudentName}}'s Demo")
			return
		}

		// CreateCommit
		if path == "/api/v4/projects/300/repository/commits" && r.Method == http.MethodPost {
			_ = json.NewDecoder(r.Body).Decode(&commitBody)
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": "commit123"})
			return
		}

		// UnprotectBranch (called before re-protecting)
		if path == "/api/v4/projects/300/protected_branches/main" && r.Method == http.MethodDelete {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		// ProtectBranch
		if path == "/api/v4/projects/300/protected_branches" && r.Method == http.MethodPost {
			branchProtected.Store(true)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"name": "main"})
			return
		}

		// ListIssueBoards
		if path == "/api/v4/projects/300/boards" && r.Method == http.MethodGet {
			_ = json.NewEncoder(w).Encode([]interface{}{})
			return
		}

		// CreateIssueBoard
		if path == "/api/v4/projects/300/boards" && r.Method == http.MethodPost {
			boardCreated.Store(true)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"id": 1, "name": "Issue Board", "lists": []interface{}{},
			})
			return
		}

		// CreateIssueBoardList
		if strings.HasPrefix(path, "/api/v4/projects/300/boards/") && strings.HasSuffix(path, "/lists") && r.Method == http.MethodPost {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": 1})
			return
		}

		// Approval configuration
		if path == "/api/v4/projects/300/approvals" && r.Method == http.MethodPost {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{})
			return
		}

		// ShareProjectWithGroup
		if strings.HasSuffix(path, "/share") && r.Method == http.MethodPost {
			sharedWithGroup.Store(true)
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": 1})
			return
		}

		// ListProjectIssues (for daily issues idempotency check)
		if path == "/api/v4/projects/300/issues" && r.Method == http.MethodGet {
			_ = json.NewEncoder(w).Encode([]interface{}{})
			return
		}

		// CreateIssue (daily issues)
		if path == "/api/v4/projects/300/issues" && r.Method == http.MethodPost {
			dailyIssuesCreated.Add(1)
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": 1, "title": "test"})
			return
		}

		http.NotFound(w, r)
	}))
	defer server.Close()

	client, err := gitlab.NewClient("test-token", gitlab.WithBaseURL(server.URL+"/api/v4"))
	require.NoError(t, err)

	originalSingleton := InfrastructureServiceSingleton
	defer func() { InfrastructureServiceSingleton = originalSingleton }()

	InfrastructureServiceSingleton = &InfrastructureService{
		gitlabClient:              client,
		teachingMaterialProjectID: "100",
	}

	err = createDemoProject(client, 1, "ase/ipraktikum/introcourse", 50)
	require.NoError(t, err)

	// Verify all setup steps executed
	assert.True(t, projectCreated.Load(), "project should be created")
	assert.True(t, branchProtected.Load(), "main branch should be protected")
	assert.True(t, boardCreated.Load(), "issue board should be created")
	assert.True(t, sharedWithGroup.Load(), "demo should be shared with tutors group")

	// Verify CI/CD config path points to shared repo
	assert.Equal(t, ".gitlab-ci.yml@ase/ipraktikum/introcourse/ci-cd", createProjectBody["ci_config_path"])

	// Verify commit payload: template vars substituted correctly
	require.NotNil(t, commitBody, "commit should have been created")
	assert.Equal(t, "main", commitBody["branch"])
	assert.Equal(t, "Initialize repository from course template", commitBody["commit_message"])

	actions := commitBody["actions"].([]interface{})
	require.Len(t, actions, 1)
	action := actions[0].(map[string]interface{})
	assert.Equal(t, "README.md", action["file_path"])
	assert.Contains(t, action["content"], "Demo")
	assert.NotContains(t, action["content"], "{{.StudentName}}")

	// Verify daily issues were created
	assert.Equal(t, int32(1), dailyIssuesCreated.Load(), "should create 1 daily issue")
}

func TestFetchTemplateFilesPartialFailure(t *testing.T) {
	// Verify that a single file fetch error fails the entire operation
	// (no partial results — all or nothing)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		if path == "/api/v4/projects/42/repository/tree" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode([]map[string]interface{}{
				{"id": "a1", "name": "good.txt", "type": "blob", "path": "student_repo_template/good.txt", "mode": "100644"},
				{"id": "a2", "name": "bad.txt", "type": "blob", "path": "student_repo_template/bad.txt", "mode": "100644"},
			})
			return
		}

		prefix := "/api/v4/projects/42/repository/files/"
		suffix := "/raw"
		if strings.HasPrefix(path, prefix) && strings.HasSuffix(path, suffix) {
			filePath := strings.TrimPrefix(path, prefix)
			filePath = strings.TrimSuffix(filePath, suffix)
			if filePath == "student_repo_template/good.txt" {
				_, _ = fmt.Fprint(w, "good content")
				return
			}
			// bad.txt returns 404
			w.WriteHeader(http.StatusNotFound)
			_, _ = fmt.Fprint(w, `{"message":"404 File Not Found"}`)
			return
		}

		http.NotFound(w, r)
	}))
	defer server.Close()

	client, err := gitlab.NewClient("test-token", gitlab.WithBaseURL(server.URL+"/api/v4"))
	require.NoError(t, err)

	_, err = fetchTemplateFiles(client, "42")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "fetch file")
	assert.Contains(t, err.Error(), "bad.txt")
}

func TestCreateDemoProjectIdempotent(t *testing.T) {
	// Verify createDemoProject succeeds when all resources already exist
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		w.Header().Set("Content-Type", "application/json")

		// CreateProject returns 409 Conflict (project already exists)
		if r.Method == http.MethodPost && path == "/api/v4/projects" {
			w.WriteHeader(http.StatusConflict)
			_, _ = fmt.Fprint(w, `{"message":"conflict"}`)
			return
		}

		// GetProject (fetches existing project after conflict)
		if r.Method == http.MethodGet && strings.HasPrefix(path, "/api/v4/projects/") && !strings.Contains(path, "/repository") && !strings.Contains(path, "/boards") && !strings.Contains(path, "/protected_branches") && !strings.Contains(path, "/approval_rules") && !strings.Contains(path, "/issues") {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"id": 300, "name": "demo",
			})
			return
		}

		// Template tree listing
		if path == "/api/v4/projects/100/repository/tree" {
			dirPath := r.URL.Query().Get("path")
			if dirPath == "daily_issues" {
				_ = json.NewEncoder(w).Encode([]map[string]interface{}{
					{"id": "i1", "name": "day1_intro.md", "type": "blob", "path": "daily_issues/day1_intro.md", "mode": "100644"},
				})
				return
			}
			_ = json.NewEncoder(w).Encode([]map[string]interface{}{
				{"id": "a1", "name": "README.md", "type": "blob", "path": "student_repo_template/README.md", "mode": "100644"},
			})
			return
		}

		// Raw file content
		if strings.HasPrefix(path, "/api/v4/projects/100/repository/files/") && strings.HasSuffix(path, "/raw") {
			filePath := strings.TrimPrefix(path, "/api/v4/projects/100/repository/files/")
			filePath = strings.TrimSuffix(filePath, "/raw")
			if strings.HasPrefix(filePath, "daily_issues") {
				_, _ = fmt.Fprint(w, "# Day 1: Introduction\n\nWelcome.")
				return
			}
			w.Header().Set("Content-Type", "application/octet-stream")
			_, _ = fmt.Fprint(w, "# Demo content")
			return
		}

		// CreateCommit returns "already exists" (files already pushed)
		if path == "/api/v4/projects/300/repository/commits" && r.Method == http.MethodPost {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = fmt.Fprint(w, `{"message":"A file with this name already exists"}`)
			return
		}

		// UnprotectBranch (idempotent: may not exist)
		if path == "/api/v4/projects/300/protected_branches/main" && r.Method == http.MethodDelete {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		// ProtectBranch succeeds after unprotect
		if path == "/api/v4/projects/300/protected_branches" && r.Method == http.MethodPost {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"name": "main", "push_access_levels": []map[string]interface{}{{"access_level": 40}},
			})
			return
		}

		// ListIssueBoards returns existing board with both lists
		if path == "/api/v4/projects/300/boards" && r.Method == http.MethodGet {
			_ = json.NewEncoder(w).Encode([]map[string]interface{}{
				{
					"id": 1, "name": "Issue Board",
					"lists": []map[string]interface{}{
						{"id": 10, "label": map[string]interface{}{"id": inProgressLabelID}},
						{"id": 11, "label": map[string]interface{}{"id": inReviewLabelID}},
					},
				},
			})
			return
		}

		// Approval configuration (idempotent — always succeeds)
		if path == "/api/v4/projects/300/approvals" && r.Method == http.MethodPost {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{})
			return
		}

		// ShareProjectWithGroup returns "already shared" (idempotent)
		if strings.HasSuffix(path, "/share") && r.Method == http.MethodPost {
			w.WriteHeader(http.StatusConflict)
			_, _ = fmt.Fprint(w, `{"message":"already a member"}`)
			return
		}

		// ListProjectIssues — issue already exists (idempotent)
		if path == "/api/v4/projects/300/issues" && r.Method == http.MethodGet {
			_ = json.NewEncoder(w).Encode([]map[string]interface{}{
				{"id": 1, "title": "Day 1: Introduction"},
			})
			return
		}

		http.NotFound(w, r)
	}))
	defer server.Close()

	client, err := gitlab.NewClient("test-token", gitlab.WithBaseURL(server.URL+"/api/v4"))
	require.NoError(t, err)

	originalSingleton := InfrastructureServiceSingleton
	defer func() { InfrastructureServiceSingleton = originalSingleton }()

	InfrastructureServiceSingleton = &InfrastructureService{
		gitlabClient:              client,
		teachingMaterialProjectID: "100",
	}

	// Should succeed even though everything already exists
	err = createDemoProject(client, 1, "ase/ipraktikum/introcourse", 50)
	assert.NoError(t, err)
}

func TestParseIssueContent(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		wantTitle   string
		wantDesc    string
	}{
		{
			name:      "standard heading and description",
			content:   "# Day 1: Git Basics\n\nLearn git commands.\n\n## Tasks\n- Clone repo",
			wantTitle: "Day 1: Git Basics",
			wantDesc:  "Learn git commands.\n\n## Tasks\n- Clone repo",
		},
		{
			name:      "heading with leading whitespace lines",
			content:   "\n\n# My Title\n\nBody here",
			wantTitle: "My Title",
			wantDesc:  "Body here",
		},
		{
			name:      "no heading returns empty title",
			content:   "Just a paragraph\nwith no heading",
			wantTitle: "",
			wantDesc:  "Just a paragraph\nwith no heading",
		},
		{
			name:      "heading only, no body",
			content:   "# Only Title",
			wantTitle: "Only Title",
			wantDesc:  "",
		},
		{
			name:      "empty content",
			content:   "",
			wantTitle: "",
			wantDesc:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			title, desc := parseIssueContent(tt.content)
			assert.Equal(t, tt.wantTitle, title)
			assert.Equal(t, tt.wantDesc, desc)
		})
	}
}

func TestFetchIssueTemplates(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		if path == "/api/v4/projects/42/repository/tree" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode([]map[string]interface{}{
				{"id": "i1", "name": "day1_git_basics.md", "type": "blob", "path": "daily_issues/day1_git_basics.md", "mode": "100644"},
				{"id": "i2", "name": "day2_swiftui.md", "type": "blob", "path": "daily_issues/day2_swiftui.md", "mode": "100644"},
				{"id": "i3", "name": "not_markdown.txt", "type": "blob", "path": "daily_issues/not_markdown.txt", "mode": "100644"},
			})
			return
		}

		prefix := "/api/v4/projects/42/repository/files/"
		suffix := "/raw"
		if strings.HasPrefix(path, prefix) && strings.HasSuffix(path, suffix) {
			filePath := strings.TrimPrefix(path, prefix)
			filePath = strings.TrimSuffix(filePath, suffix)
			contents := map[string]string{
				"daily_issues/day1_git_basics.md": "# Day 1: Git Basics\n\nLearn git.\n\n## Tasks\n- Clone",
				"daily_issues/day2_swiftui.md":    "# Day 2: SwiftUI\n\nBuild your first view.",
			}
			if content, ok := contents[filePath]; ok {
				_, _ = fmt.Fprint(w, content)
				return
			}
		}

		http.NotFound(w, r)
	}))
	defer server.Close()

	client, err := gitlab.NewClient("test-token", gitlab.WithBaseURL(server.URL+"/api/v4"))
	require.NoError(t, err)

	issues, err := fetchIssueTemplates(client, "42")
	require.NoError(t, err)
	require.Len(t, issues, 2, "should skip non-.md files")

	// Verify sorted by filename (day1 before day2)
	assert.Equal(t, "Day 1: Git Basics", issues[0].Title)
	assert.Contains(t, issues[0].Description, "Learn git.")
	assert.Equal(t, "Day 2: SwiftUI", issues[1].Title)
	assert.Contains(t, issues[1].Description, "Build your first view.")
}

func TestFetchIssueTemplatesEmptyDir(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v4/projects/42/repository/tree" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode([]interface{}{})
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	client, err := gitlab.NewClient("test-token", gitlab.WithBaseURL(server.URL+"/api/v4"))
	require.NoError(t, err)

	// Empty directory should return empty slice (not error), since daily issues are optional
	issues, err := fetchIssueTemplates(client, "42")
	assert.NoError(t, err)
	assert.Empty(t, issues)
}

func TestCreateDailyIssues(t *testing.T) {
	var createdIssues []string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		w.Header().Set("Content-Type", "application/json")

		// Issue template tree listing
		if path == "/api/v4/projects/100/repository/tree" {
			_ = json.NewEncoder(w).Encode([]map[string]interface{}{
				{"id": "i1", "name": "day1.md", "type": "blob", "path": "daily_issues/day1.md", "mode": "100644"},
				{"id": "i2", "name": "day2.md", "type": "blob", "path": "daily_issues/day2.md", "mode": "100644"},
			})
			return
		}

		// Raw file content
		prefix := "/api/v4/projects/100/repository/files/"
		suffix := "/raw"
		if strings.HasPrefix(path, prefix) && strings.HasSuffix(path, suffix) {
			contents := map[string]string{
				"daily_issues/day1.md": "# Day 1: Setup\n\nSet up your environment.",
				"daily_issues/day2.md": "# Day 2: Basics\n\nLearn the basics.",
			}
			filePath := strings.TrimPrefix(path, prefix)
			filePath = strings.TrimSuffix(filePath, suffix)
			if content, ok := contents[filePath]; ok {
				_, _ = fmt.Fprint(w, content)
				return
			}
		}

		// ListProjectIssues (existing issues — Day 1 already exists)
		if path == "/api/v4/projects/200/issues" && r.Method == http.MethodGet {
			_ = json.NewEncoder(w).Encode([]map[string]interface{}{
				{"id": 1, "title": "Day 1: Setup"},
			})
			return
		}

		// CreateIssue
		if path == "/api/v4/projects/200/issues" && r.Method == http.MethodPost {
			var body map[string]interface{}
			_ = json.NewDecoder(r.Body).Decode(&body)
			createdIssues = append(createdIssues, body["title"].(string))
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"id": 2, "title": body["title"],
			})
			return
		}

		http.NotFound(w, r)
	}))
	defer server.Close()

	client, err := gitlab.NewClient("test-token", gitlab.WithBaseURL(server.URL+"/api/v4"))
	require.NoError(t, err)

	originalSingleton := InfrastructureServiceSingleton
	defer func() { InfrastructureServiceSingleton = originalSingleton }()

	InfrastructureServiceSingleton = &InfrastructureService{
		gitlabClient:              client,
		teachingMaterialProjectID: "100",
	}

	err = createDailyIssues(client, 200, "test-repo")
	require.NoError(t, err)

	// Only Day 2 should be created (Day 1 already exists)
	require.Len(t, createdIssues, 1)
	assert.Equal(t, "Day 2: Basics", createdIssues[0])
}

func TestIssueCacheThreadSafety(t *testing.T) {
	var fetchCount atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v4/projects/42/repository/tree" {
			fetchCount.Add(1)
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode([]map[string]interface{}{
				{"id": "i1", "name": "day1.md", "type": "blob", "path": "daily_issues/day1.md", "mode": "100644"},
			})
			return
		}

		if strings.Contains(r.URL.Path, "/repository/files/") && strings.HasSuffix(r.URL.Path, "/raw") {
			_, _ = fmt.Fprint(w, "# Day 1\n\nContent")
			return
		}

		http.NotFound(w, r)
	}))
	defer server.Close()

	client, err := gitlab.NewClient("test-token", gitlab.WithBaseURL(server.URL+"/api/v4"))
	require.NoError(t, err)

	cache := &issueCache{}

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			issues, err := cache.get(client, "42")
			assert.NoError(t, err)
			assert.Len(t, issues, 1)
		}()
	}
	wg.Wait()

	assert.Equal(t, int32(1), fetchCount.Load(), "issue tree should be fetched exactly once")
}

func TestCreateCICDProject(t *testing.T) {
	t.Run("creates project and pushes CI/CD files", func(t *testing.T) {
		var createProjectBody map[string]interface{}
		var commitBody map[string]interface{}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")

			// CreateProject
			if r.Method == http.MethodPost && r.URL.Path == "/api/v4/projects" {
				_ = json.NewDecoder(r.Body).Decode(&createProjectBody)
				w.WriteHeader(http.StatusCreated)
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"id": 500, "name": "ci-cd",
				})
				return
			}
			// ListTree for ci_cd/
			if r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/repository/tree") {
				_ = json.NewEncoder(w).Encode([]map[string]interface{}{
					{"name": ".gitlab-ci.yml", "path": "ci_cd/.gitlab-ci.yml", "type": "blob", "mode": "100644"},
				})
				return
			}
			// GetRawFile
			if r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/repository/files/") {
				w.Header().Set("Content-Type", "application/octet-stream")
				_, _ = fmt.Fprint(w, "stages:\n  - lint\n")
				return
			}
			// CreateCommit for CI/CD files
			if r.Method == http.MethodPost && r.URL.Path == "/api/v4/projects/500/repository/commits" {
				_ = json.NewDecoder(r.Body).Decode(&commitBody)
				w.WriteHeader(http.StatusCreated)
				_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": "abc123"})
				return
			}
			http.NotFound(w, r)
		}))
		defer server.Close()

		client, err := gitlab.NewClient("test-token", gitlab.WithBaseURL(server.URL+"/api/v4"))
		require.NoError(t, err)

		origSvc := InfrastructureServiceSingleton
		InfrastructureServiceSingleton = &InfrastructureService{
			teachingMaterialProjectID: "test-teaching-project",
		}
		defer func() { InfrastructureServiceSingleton = origSvc }()

		err = createCICDProject(client, 1, "ase/ipraktikum/introcourse")
		require.NoError(t, err)

		// Verify project creation
		assert.Equal(t, "ci-cd", createProjectBody["name"])
		assert.Equal(t, true, createProjectBody["initialize_with_readme"])

		// Verify CI/CD files were pushed
		require.NotNil(t, commitBody, "should push CI/CD files via commit")
		assert.Equal(t, "Initialize CI/CD pipeline from course template", commitBody["commit_message"])
		actions, ok := commitBody["actions"].([]interface{})
		require.True(t, ok, "commit should have actions")
		require.Len(t, actions, 1)
		action := actions[0].(map[string]interface{})
		assert.Equal(t, ".gitlab-ci.yml", action["file_path"])
		assert.Equal(t, "stages:\n  - lint\n", action["content"])
	})

	t.Run("idempotent on existing project with existing files", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")

			if r.Method == http.MethodPost && r.URL.Path == "/api/v4/projects" {
				w.WriteHeader(http.StatusConflict)
				_, _ = fmt.Fprint(w, `{"message":"conflict"}`)
				return
			}
			if r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/api/v4/projects/ase") {
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"id": 500, "name": "ci-cd",
				})
				return
			}
			// ListTree for ci_cd/
			if r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/repository/tree") {
				_ = json.NewEncoder(w).Encode([]map[string]interface{}{
					{"name": ".gitlab-ci.yml", "path": "ci_cd/.gitlab-ci.yml", "type": "blob", "mode": "100644"},
				})
				return
			}
			// GetRawFile
			if r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/repository/files/") {
				w.Header().Set("Content-Type", "application/octet-stream")
				_, _ = fmt.Fprint(w, "stages:\n  - lint\n")
				return
			}
			// CreateCommit — files already exist
			if r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/repository/commits") {
				w.WriteHeader(http.StatusBadRequest)
				_, _ = fmt.Fprint(w, `{"message":"A file with this name already exists"}`)
				return
			}
			http.NotFound(w, r)
		}))
		defer server.Close()

		client, err := gitlab.NewClient("test-token", gitlab.WithBaseURL(server.URL+"/api/v4"))
		require.NoError(t, err)

		origSvc := InfrastructureServiceSingleton
		InfrastructureServiceSingleton = &InfrastructureService{
			teachingMaterialProjectID: "test-teaching-project",
		}
		defer func() { InfrastructureServiceSingleton = origSvc }()

		err = createCICDProject(client, 1, "ase/ipraktikum/introcourse")
		assert.NoError(t, err)
	})

	t.Run("succeeds with no CI/CD files in teaching material", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")

			if r.Method == http.MethodPost && r.URL.Path == "/api/v4/projects" {
				w.WriteHeader(http.StatusCreated)
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"id": 500, "name": "ci-cd",
				})
				return
			}
			// ListTree for ci_cd/ — 404 (directory doesn't exist)
			if r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/repository/tree") {
				w.WriteHeader(http.StatusNotFound)
				_, _ = fmt.Fprint(w, `{"message":"404 Tree Not Found"}`)
				return
			}
			http.NotFound(w, r)
		}))
		defer server.Close()

		client, err := gitlab.NewClient("test-token", gitlab.WithBaseURL(server.URL+"/api/v4"))
		require.NoError(t, err)

		origSvc := InfrastructureServiceSingleton
		InfrastructureServiceSingleton = &InfrastructureService{
			teachingMaterialProjectID: "test-teaching-project",
		}
		defer func() { InfrastructureServiceSingleton = origSvc }()

		err = createCICDProject(client, 1, "ase/ipraktikum/introcourse")
		assert.NoError(t, err)
	})
}

func TestFetchCICDFiles(t *testing.T) {
	t.Run("fetches files from ci_cd directory", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")

			if strings.Contains(r.URL.Path, "/repository/tree") {
				_ = json.NewEncoder(w).Encode([]map[string]interface{}{
					{"name": ".gitlab-ci.yml", "path": "ci_cd/.gitlab-ci.yml", "type": "blob", "mode": "100644"},
					{"name": "lint.sh", "path": "ci_cd/scripts/lint.sh", "type": "blob", "mode": "100755"},
				})
				return
			}
			if strings.Contains(r.URL.Path, "/repository/files/") {
				w.Header().Set("Content-Type", "application/octet-stream")
				if strings.Contains(r.URL.Path, "gitlab-ci") {
					_, _ = fmt.Fprint(w, "stages:\n  - lint\n")
				} else {
					_, _ = fmt.Fprint(w, "#!/bin/bash\nswiftlint\n")
				}
				return
			}
			http.NotFound(w, r)
		}))
		defer server.Close()

		client, err := gitlab.NewClient("test-token", gitlab.WithBaseURL(server.URL+"/api/v4"))
		require.NoError(t, err)

		files, err := fetchCICDFiles(client, "test-project")
		require.NoError(t, err)
		require.Len(t, files, 2)

		assert.Equal(t, ".gitlab-ci.yml", files[0].Path)
		assert.Equal(t, "stages:\n  - lint\n", files[0].Content)
		assert.False(t, files[0].ExecuteFilemode)

		assert.Equal(t, "scripts/lint.sh", files[1].Path)
		assert.True(t, files[1].ExecuteFilemode)
	})

	t.Run("returns empty slice when directory missing", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			_, _ = fmt.Fprint(w, `{"message":"404 Tree Not Found"}`)
		}))
		defer server.Close()

		client, err := gitlab.NewClient("test-token", gitlab.WithBaseURL(server.URL+"/api/v4"))
		require.NoError(t, err)

		files, err := fetchCICDFiles(client, "test-project")
		assert.NoError(t, err)
		assert.Nil(t, files)
	})
}

func TestEnsureApprovalRule(t *testing.T) {
	t.Run("creates rule when none exists", func(t *testing.T) {
		var ruleCreated bool
		var ruleBody map[string]interface{}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")

			// GetProjectApprovalRules — no rules yet
			if r.URL.Path == "/api/v4/projects/300/approval_rules" && r.Method == http.MethodGet {
				_ = json.NewEncoder(w).Encode([]interface{}{})
				return
			}
			// CreateProjectApprovalRule
			if r.URL.Path == "/api/v4/projects/300/approval_rules" && r.Method == http.MethodPost {
				ruleCreated = true
				_ = json.NewDecoder(r.Body).Decode(&ruleBody)
				w.WriteHeader(http.StatusCreated)
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"id": 1, "name": "Tutor Approval",
				})
				return
			}
			http.NotFound(w, r)
		}))
		defer server.Close()

		client, err := gitlab.NewClient("test-token", gitlab.WithBaseURL(server.URL+"/api/v4"))
		require.NoError(t, err)

		err = ensureApprovalRule(client, 300, "test-repo", 42)
		require.NoError(t, err)
		assert.True(t, ruleCreated, "should create approval rule")
		assert.Equal(t, "Tutor Approval", ruleBody["name"])
	})

	t.Run("skips when rule already exists", func(t *testing.T) {
		var ruleCreated bool

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")

			if r.URL.Path == "/api/v4/projects/300/approval_rules" && r.Method == http.MethodGet {
				_ = json.NewEncoder(w).Encode([]map[string]interface{}{
					{"id": 1, "name": "Tutor Approval", "approvals_required": 1},
				})
				return
			}
			if r.URL.Path == "/api/v4/projects/300/approval_rules" && r.Method == http.MethodPost {
				ruleCreated = true
				return
			}
			http.NotFound(w, r)
		}))
		defer server.Close()

		client, err := gitlab.NewClient("test-token", gitlab.WithBaseURL(server.URL+"/api/v4"))
		require.NoError(t, err)

		err = ensureApprovalRule(client, 300, "test-repo", 42)
		assert.NoError(t, err)
		assert.False(t, ruleCreated, "should not create duplicate rule")
	})
}

func TestEnsureApprovalConfiguration(t *testing.T) {
	var configBody map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.URL.Path == "/api/v4/projects/300/approvals" && r.Method == http.MethodPost {
			_ = json.NewDecoder(r.Body).Decode(&configBody)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{})
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	client, err := gitlab.NewClient("test-token", gitlab.WithBaseURL(server.URL+"/api/v4"))
	require.NoError(t, err)

	err = ensureApprovalConfiguration(client, 300, "test-repo")
	require.NoError(t, err)

	assert.Equal(t, true, configBody["reset_approvals_on_push"])
	assert.Equal(t, false, configBody["merge_requests_author_approval"])
	assert.Equal(t, true, configBody["merge_requests_disable_committers_approval"])
}
