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

func TestGit2ExerciseBranchesStayOffMainAndRetryWithoutCommits(t *testing.T) {
	files := map[string][]templateFile{
		"exercise/git2-navigation": {
			{Path: "IntrocourseApp/ContentView.swift", Content: "navigation"},
			{Path: "IntrocourseApp/BirdListView.swift", Content: "list"},
		},
		"exercise/git2-rows": {
			{Path: "IntrocourseApp/ContentView.swift", Content: "rows"},
			{Path: "IntrocourseApp/BirdRow.swift", Content: "row"},
		},
	}
	created := map[string]bool{}
	protected := map[string]bool{}
	commits := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		path := r.URL.Path
		switch {
		case r.Method == http.MethodGet && strings.HasPrefix(path, "/api/v4/projects/42/repository/branches/"):
			name := strings.TrimPrefix(path, "/api/v4/projects/42/repository/branches/")
			if !created[name] {
				http.NotFound(w, r)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"name": name})
		case r.Method == http.MethodPost && path == "/api/v4/projects/42/repository/commits":
			var body struct {
				Branch      string `json:"branch"`
				StartBranch string `json:"start_branch"`
				Actions     []struct {
					Action   string `json:"action"`
					FilePath string `json:"file_path"`
				} `json:"actions"`
			}
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			require.Equal(t, "main", body.StartBranch)
			require.False(t, created[body.Branch])
			require.Len(t, body.Actions, 2)
			require.Equal(t, "update", body.Actions[0].Action)
			require.Equal(t, "IntrocourseApp/ContentView.swift", body.Actions[0].FilePath)
			require.Equal(t, "create", body.Actions[1].Action)
			created[body.Branch] = true
			commits++
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id":"commit"}`))
		case r.Method == http.MethodGet && strings.HasPrefix(path, "/api/v4/projects/42/protected_branches/"):
			name := strings.TrimPrefix(path, "/api/v4/projects/42/protected_branches/")
			if !protected[name] {
				http.NotFound(w, r)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"name": name, "push_access_levels": []map[string]any{{"access_level": 0}}, "merge_access_levels": []map[string]any{{"access_level": 0}}})
		case r.Method == http.MethodPost && path == "/api/v4/projects/42/protected_branches":
			var body map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			name, _ := body["name"].(string)
			require.True(t, created[name])
			require.Equal(t, float64(0), body["push_access_level"])
			require.Equal(t, float64(0), body["merge_access_level"])
			protected[name] = true
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]any{"name": name, "push_access_levels": []map[string]any{{"access_level": 0}}, "merge_access_levels": []map[string]any{{"access_level": 0}}})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	git, err := gitlab.NewClient("test-token", gitlab.WithBaseURL(server.URL+"/api/v4"))
	require.NoError(t, err)
	require.NoError(t, ensureGit2ExerciseBranches(git, 42, "demo", files))
	require.NoError(t, ensureGit2ExerciseBranches(git, 42, "demo", files))
	require.Equal(t, 2, commits)
	require.Len(t, protected, 2)
	require.False(t, created["main"])
}
