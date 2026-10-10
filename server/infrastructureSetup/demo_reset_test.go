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

func TestRewriteDemoMainRestoresTutorProtection(t *testing.T) {
	for _, test := range []struct {
		name        string
		commitFails bool
		unsafePush  bool
		wrongGroup  bool
	}{
		{name: "success"},
		{name: "commit_rejected", commitFails: true},
		{name: "reject_developer_push", unsafePush: true},
		{name: "reject_other_merge_group", wrongGroup: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			branch := &gitlab.ProtectedBranch{
				Name:             "main",
				PushAccessLevels: []*gitlab.BranchAccessDescription{{ID: 1, AccessLevel: gitlab.NoPermissions}},
				MergeAccessLevels: []*gitlab.BranchAccessDescription{
					{ID: 2, AccessLevel: gitlab.MaintainerPermissions},
					{ID: 3, GroupID: 50, AccessLevel: gitlab.MaintainerPermissions},
				},
			}
			if test.unsafePush {
				branch.PushAccessLevels[0].AccessLevel = gitlab.DeveloperPermissions
			}
			if test.wrongGroup {
				branch.MergeAccessLevels[1].GroupID = 60
			}
			nextID, commitCalls := int64(4), 0
			applyPermissions := func(current []*gitlab.BranchAccessDescription, changes []*gitlab.BranchPermissionOptions) []*gitlab.BranchAccessDescription {
				for _, change := range changes {
					if change.Destroy != nil && *change.Destroy {
						for index, access := range current {
							if change.ID != nil && access.ID == *change.ID {
								current = append(current[:index], current[index+1:]...)
								break
							}
						}
						continue
					}
					access := &gitlab.BranchAccessDescription{ID: nextID}
					nextID++
					if change.AccessLevel != nil {
						access.AccessLevel = *change.AccessLevel
					}
					if change.UserID != nil {
						access.UserID = *change.UserID
					}
					if change.GroupID != nil {
						access.GroupID = *change.GroupID
						access.AccessLevel = gitlab.MaintainerPermissions
					}
					current = append(current, access)
				}
				return current
			}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				switch {
				case r.Method == http.MethodGet && r.URL.Path == "/api/v4/projects/300/repository/tree":
					_, _ = w.Write([]byte(`[{"path":"README.md","type":"blob","mode":"100644"}]`))
				case r.Method == http.MethodGet && r.URL.Path == "/api/v4/projects/300/repository/files/README.md/raw":
					_, _ = w.Write([]byte("Original template"))
				case r.Method == http.MethodGet && r.URL.Path == "/api/v4/user":
					_, _ = w.Write([]byte(`{"id":99}`))
				case r.Method == http.MethodGet && r.URL.Path == "/api/v4/projects/300":
					_, _ = w.Write([]byte(`{"id":300,"shared_with_groups":[{"group_id":50,"group_access_level":30}]}`))
				case r.Method == http.MethodGet && r.URL.Path == "/api/v4/projects/300/protected_branches":
					_ = json.NewEncoder(w).Encode([]*gitlab.ProtectedBranch{branch})
				case r.Method == http.MethodGet && r.URL.Path == "/api/v4/projects/300/protected_branches/main":
					_ = json.NewEncoder(w).Encode(branch)
				case r.Method == http.MethodPatch && r.URL.Path == "/api/v4/projects/300/protected_branches/main":
					var options gitlab.UpdateProtectedBranchOptions
					require.NoError(t, json.NewDecoder(r.Body).Decode(&options))
					if options.AllowForcePush != nil {
						branch.AllowForcePush = *options.AllowForcePush
					}
					if options.AllowedToPush != nil {
						branch.PushAccessLevels = applyPermissions(branch.PushAccessLevels, *options.AllowedToPush)
					}
					if options.AllowedToMerge != nil {
						branch.MergeAccessLevels = applyPermissions(branch.MergeAccessLevels, *options.AllowedToMerge)
					}
					_ = json.NewEncoder(w).Encode(branch)
				case r.Method == http.MethodPost && r.URL.Path == "/api/v4/projects/300/repository/commits":
					commitCalls++
					require.True(t, branch.AllowForcePush)
					require.Len(t, branch.PushAccessLevels, 1)
					require.Equal(t, int64(99), branch.PushAccessLevels[0].UserID, "only the service user can rewrite the branch")
					if test.commitFails {
						http.Error(w, `{"message":"commit rejected"}`, http.StatusBadRequest)
					} else {
						_, _ = w.Write([]byte(`{"id":"new-main","parent_ids":["original-root"]}`))
					}
				default:
					t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
					http.Error(w, "unexpected request", http.StatusNotFound)
				}
			}))
			defer server.Close()
			client, err := gitlab.NewClient("test-token", gitlab.WithBaseURL(server.URL+"/api/v4"))
			require.NoError(t, err)
			err = rewriteDemoMain(client, 300, "original-root", []templateFile{{Path: "README.md", Content: "Current template"}}, demoTemplateVars("course/Introcourse"), 50)
			if test.unsafePush || test.wrongGroup {
				require.ErrorContains(t, err, "expected strict protection")
				require.Zero(t, commitCalls)
				return
			}
			if test.commitFails {
				require.ErrorContains(t, err, "commit rejected")
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, 1, commitCalls)
			require.True(t, demoMainBranchProtectionMatches(branch, 50), "restore direct-push denial, force-push denial, and tutor merge access even after failure")
		})
	}
}

func TestRetireDemoPracticeIssue(t *testing.T) {
	for _, test := range []struct {
		name         string
		deleteStatus int
		state        string
		wantClose    bool
		wantError    bool
	}{
		{"deleted", http.StatusNoContent, "opened", false, false},
		{"already_deleted", http.StatusNotFound, "opened", false, false},
		{"close_when_deletion_forbidden", http.StatusForbidden, "opened", true, false},
		{"keep_closed_history_when_deletion_forbidden", http.StatusForbidden, "closed", false, false},
		{"do_not_hide_server_errors", http.StatusBadGateway, "opened", false, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			closeCalls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, "/api/v4/projects/300/issues/13", r.URL.Path)
				switch r.Method {
				case http.MethodDelete:
					w.WriteHeader(test.deleteStatus)
				case http.MethodPut:
					closeCalls++
					var body struct {
						StateEvent string `json:"state_event"`
					}
					require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
					require.Equal(t, "close", body.StateEvent)
					_, _ = w.Write([]byte(`{"id":13,"iid":13,"state":"closed"}`))
				default:
					t.Errorf("unexpected request: %s", r.Method)
					http.Error(w, "unexpected request", http.StatusNotFound)
				}
			}))
			defer server.Close()
			client, err := gitlab.NewClient("test-token", gitlab.WithBaseURL(server.URL+"/api/v4"), gitlab.WithCustomRetryMax(0))
			require.NoError(t, err)
			err = retireDemoPracticeIssue(client, 300, &gitlab.Issue{IID: 13, State: test.state})
			if test.wantError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, boolToInt(test.wantClose), closeCalls)
		})
	}
}
