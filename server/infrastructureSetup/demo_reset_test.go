package infrastructureSetup

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

func TestDemoTemplateActionsReplaceOnlyChanges(t *testing.T) {
	actions, err := demoTemplateActions(map[string]demoRootFile{
		"README.md":            {content: "For Demo\n"},
		".githooks/pre-commit": {content: "old", executable: true},
		"old.txt":              {content: "remove"},
	}, []templateFile{
		{Path: "README.md", Content: "For {{.StudentName}}\n"},
		{Path: ".githooks/pre-commit", Content: "new", ExecuteFilemode: true},
		{Path: "new.txt", Content: "add"},
	}, demoTemplateVars("ase/ipraktikum/ios2627/Introcourse"))
	require.NoError(t, err)
	require.Len(t, actions, 3)
	byPath := make(map[string]*gitlab.CommitActionOptions)
	for _, action := range actions {
		byPath[*action.FilePath] = action
	}
	require.NotContains(t, byPath, "README.md")
	require.Equal(t, gitlab.FileUpdate, *byPath[".githooks/pre-commit"].Action)
	require.True(t, *byPath[".githooks/pre-commit"].ExecuteFilemode)
	require.Equal(t, gitlab.FileCreate, *byPath["new.txt"].Action)
	require.Equal(t, gitlab.FileDelete, *byPath["old.txt"].Action)
}

func TestDemoTemplateActionsRejectDuplicatePaths(t *testing.T) {
	_, err := demoTemplateActions(nil, []templateFile{{Path: "README.md"}, {Path: "README.md"}}, demoTemplateVars("ase/ipraktikum/ios2627/Introcourse"))
	require.ErrorContains(t, err, "duplicate")
}

func TestDemoTemplateActionsAllowsUnchangedTemplate(t *testing.T) {
	actions, err := demoTemplateActions(map[string]demoRootFile{
		"README.md": {content: "For Demo\n"},
	}, []templateFile{{Path: "README.md", Content: "For {{.StudentName}}\n"}}, demoTemplateVars("ase/ipraktikum/ios2627/Introcourse"))
	require.NoError(t, err)
	require.Empty(t, actions)
}

func TestDemoResetKeepsCourseBundleIdentifier(t *testing.T) {
	vars := demoTemplateVars("ase/ipraktikum/ios2627/Introcourse")
	actions, err := demoTemplateActions(nil, []templateFile{{
		Path: "project.yml", Content: "PRODUCT_BUNDLE_IDENTIFIER: {{.BundleIdentifier}}",
	}}, vars)
	require.NoError(t, err)
	require.Len(t, actions, 1)
	require.Equal(t, "PRODUCT_BUNDLE_IDENTIFIER: de.tum.cit.aet.ios2627.demo.introcourseapp", *actions[0].Content)
}

func TestResetDemoIssuesDoesNotTouchCleanDailyIssue(t *testing.T) {
	updates := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/api/v4/projects/42/issues" && r.Method == http.MethodGet:
			_, _ = w.Write([]byte(`[{"id":1,"iid":1,"title":"Day 1","description":"Do the work","state":"opened","labels":[],"assignees":[]}]`))
		case r.URL.Path == "/api/graphql":
			var request struct {
				Query string `json:"query"`
			}
			require.NoError(t, json.NewDecoder(r.Body).Decode(&request))
			if strings.Contains(request.Query, "workItems(first:") {
				_, _ = w.Write([]byte(`{"data":{"project":{"workItems":{"nodes":[{"iid":"1","widgets":[{"status":{"id":"gid://gitlab/Status/1"}}]}],"pageInfo":{"hasNextPage":false}}}}}`))
			} else {
				_, _ = w.Write([]byte(`{"data":{"project":{"group":{"lifecycles":{"nodes":[{"name":"Default","statuses":[{"name":"Open","id":"gid://gitlab/Status/1"},{"name":"In Progress","id":"gid://gitlab/Status/2"},{"name":"In Review","id":"gid://gitlab/Status/3"},{"name":"Blocked","id":"gid://gitlab/Status/4"},{"name":"Done","id":"gid://gitlab/Status/5"}]}]}},"boards":{"nodes":[]}}}}`))
			}
		default:
			updates++
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	defer server.Close()
	client, err := gitlab.NewClient("test-token", gitlab.WithBaseURL(server.URL+"/api/v4"))
	require.NoError(t, err)
	require.NoError(t, resetDemoIssues(client, 42, "course/demo", []issueTemplate{{Title: "Day 1", Description: "Do the work"}}))
	require.Zero(t, updates)
}
