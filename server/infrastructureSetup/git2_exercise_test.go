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

func TestCleanDemoAcceptsOnlyUntouchedPracticeBranches(t *testing.T) {
	const mainSHA = "initial-main"
	contents := map[string]string{}
	sourceContents := map[string]string{}
	branches := []*gitlab.Branch{{Name: "main", Commit: &gitlab.Commit{ID: mainSHA}}}
	for _, spec := range git2ExerciseBranches {
		branches = append(branches, &gitlab.Branch{
			Name: spec.name, Protected: true,
			Commit: &gitlab.Commit{ID: spec.dir + "-commit", ParentIDs: []string{mainSHA}},
		})
		for _, file := range spec.files {
			contents[spec.name+"/IntrocourseApp/"+file] = spec.dir + ":" + file
			sourceContents[spec.name+"/IntrocourseApp/"+file] = spec.dir + ":" + file
		}
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		path := r.URL.Path
		switch {
		case strings.HasPrefix(path, "/api/v4/projects/100/repository/files/"):
			name := strings.TrimSuffix(strings.TrimPrefix(path, "/api/v4/projects/100/repository/files/"), "/raw")
			parts := strings.Split(name, "/")
			value := sourceContents["exercise/git2-"+parts[1]+"/IntrocourseApp/"+parts[len(parts)-1]]
			_, _ = w.Write([]byte(value))
		case strings.HasPrefix(path, "/api/v4/projects/42/repository/files/"):
			name := strings.TrimSuffix(strings.TrimPrefix(path, "/api/v4/projects/42/repository/files/"), "/raw")
			_, _ = w.Write([]byte(contents[r.URL.Query().Get("ref")+"/"+name]))
		case strings.HasPrefix(path, "/api/v4/projects/42/protected_branches/"):
			name := strings.TrimPrefix(path, "/api/v4/projects/42/protected_branches/")
			_ = json.NewEncoder(w).Encode(map[string]any{"name": name, "push_access_levels": []map[string]any{{"access_level": 0}}, "merge_access_levels": []map[string]any{{"access_level": 0}}})
		case strings.HasPrefix(path, "/api/v4/projects/42/repository/commits/") && strings.HasSuffix(path, "/diff"):
			name := strings.TrimSuffix(strings.TrimPrefix(path, "/api/v4/projects/42/repository/commits/"), "/diff")
			folder := strings.TrimSuffix(name, "-commit")
			files := []map[string]any{{"new_path": "IntrocourseApp/ContentView.swift"}}
			if folder == "navigation" {
				files = append(files, map[string]any{"new_path": "IntrocourseApp/BirdListView.swift", "new_file": true})
			} else {
				files = append(files, map[string]any{"new_path": "IntrocourseApp/BirdRow.swift", "new_file": true})
			}
			_ = json.NewEncoder(w).Encode(files)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	git, err := gitlab.NewClient("test-token", gitlab.WithBaseURL(server.URL+"/api/v4"))
	require.NoError(t, err)
	clean, err := demoGit2ExerciseIsClean(git, "100", "source", 42, mainSHA, branches)
	require.NoError(t, err)
	require.True(t, clean)

	contents["exercise/git2-rows/IntrocourseApp/BirdRow.swift"] = "changed"
	clean, err = demoGit2ExerciseIsClean(git, "100", "source", 42, mainSHA, branches)
	require.NoError(t, err)
	require.False(t, clean)

	branches = append(branches, &gitlab.Branch{Name: "practice/leftover"})
	clean, err = demoGit2ExerciseIsClean(git, "100", "source", 42, mainSHA, branches)
	require.NoError(t, err)
	require.False(t, clean)
}
