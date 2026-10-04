package infrastructureSetup

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

func TestEnsureTutorGroupAccess(t *testing.T) {
	for _, alreadyShared := range []bool{false, true} {
		t.Run(map[bool]string{false: "shares_missing_group", true: "keeps_existing_share"}[alreadyShared], func(t *testing.T) {
			shareCalls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch {
				case r.Method == http.MethodGet && r.URL.Path == "/api/v4/groups/100":
					group := map[string]interface{}{"id": 100}
					if alreadyShared {
						group["shared_with_groups"] = []map[string]interface{}{{"group_id": 200, "group_access_level": 30}}
					}
					_ = json.NewEncoder(w).Encode(group)
				case r.Method == http.MethodPost && r.URL.Path == "/api/v4/groups/100/share":
					shareCalls++
					var request struct {
						GroupID     int64 `json:"group_id"`
						GroupAccess int64 `json:"group_access"`
					}
					require.NoError(t, json.NewDecoder(r.Body).Decode(&request))
					require.Equal(t, int64(200), request.GroupID)
					require.Equal(t, int64(30), request.GroupAccess)
					w.WriteHeader(http.StatusCreated)
					_, _ = w.Write([]byte(`{"id":1}`))
				default:
					http.Error(w, "unexpected request", http.StatusNotFound)
				}
			}))
			defer server.Close()

			client, err := gitlab.NewClient("test-token", gitlab.WithBaseURL(server.URL+"/api/v4"))
			require.NoError(t, err)
			require.NoError(t, ensureTutorGroupAccess(client, 100, 200))
			if alreadyShared {
				require.Zero(t, shareCalls)
			} else {
				require.Equal(t, 1, shareCalls)
			}
		})
	}
}

func handleDemoTutorAccessFixture(w http.ResponseWriter, r *http.Request) bool {
	switch {
	case r.Method == http.MethodGet && r.URL.Path == "/api/v4/projects/300":
		_, _ = w.Write([]byte(`{"id":300,"name":"demo","path_with_namespace":"ase/ipraktikum/introcourse/demo","visibility":"private","ci_config_path":".gitlab-ci.yml@ase/ipraktikum/introcourse/ci-cd","merge_method":"merge","squash_option":"default_on","remove_source_branch_after_merge":true,"only_allow_merge_if_pipeline_succeeds":true,"only_allow_merge_if_all_discussions_are_resolved":true,"shared_with_groups":[{"group_id":50,"group_access_level":30}]}`))
	case r.Method == http.MethodPatch && r.URL.Path == "/api/v4/projects/300/protected_branches/main":
		_, _ = w.Write([]byte(`{"name":"main","push_access_levels":[{"access_level":0}],"merge_access_levels":[{"access_level":40},{"group_id":50,"access_level":40}]}`))
	default:
		return false
	}
	return true
}

func TestEnsureDemoTutorMergeAccess(t *testing.T) {
	shared, mergeGranted := false, false
	shareCalls, patchCalls := 0, 0
	branch := &gitlab.ProtectedBranch{
		Name:              "main",
		PushAccessLevels:  []*gitlab.BranchAccessDescription{{AccessLevel: gitlab.NoPermissions}},
		MergeAccessLevels: []*gitlab.BranchAccessDescription{{AccessLevel: gitlab.MaintainerPermissions}},
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v4/projects/300":
			project := map[string]any{"id": 300}
			if shared {
				project["shared_with_groups"] = []map[string]any{{"group_id": 50, "group_access_level": 30}}
			}
			_ = json.NewEncoder(w).Encode(project)
		case r.Method == http.MethodPost && r.URL.Path == "/api/v4/projects/300/share":
			var body map[string]int64
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			require.Equal(t, int64(50), body["group_id"])
			require.Equal(t, int64(30), body["group_access"])
			shared = true
			shareCalls++
			_, _ = w.Write([]byte(`{}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v4/projects/300/protected_branches/main":
			_ = json.NewEncoder(w).Encode(branch)
		case r.Method == http.MethodPatch && r.URL.Path == "/api/v4/projects/300/protected_branches/main":
			require.True(t, shared, "the group must be shared before granting branch access")
			var body map[string]json.RawMessage
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			require.Len(t, body, 1, "do not change push or approval permissions")
			require.JSONEq(t, `[{"group_id":50}]`, string(body["allowed_to_merge"]))
			branch.MergeAccessLevels = append(branch.MergeAccessLevels, &gitlab.BranchAccessDescription{GroupID: 50, AccessLevel: gitlab.MaintainerPermissions})
			mergeGranted = true
			patchCalls++
			_ = json.NewEncoder(w).Encode(branch)
		default:
			http.Error(w, "unexpected request", http.StatusNotFound)
		}
	}))
	defer server.Close()
	client, err := gitlab.NewClient("test-token", gitlab.WithBaseURL(server.URL+"/api/v4"))
	require.NoError(t, err)
	require.NoError(t, ensureDemoTutorMergeAccess(client, 300, 50))
	require.NoError(t, ensureDemoTutorMergeAccess(client, 300, 50))
	require.True(t, mergeGranted)
	require.Equal(t, 1, shareCalls)
	require.Equal(t, 1, patchCalls)
	// Extra access must not be mistaken for a valid demo rule.
	branch.MergeAccessLevels = append(branch.MergeAccessLevels, &gitlab.BranchAccessDescription{AccessLevel: gitlab.DeveloperPermissions})
	require.False(t, demoMainBranchProtectionMatches(branch, 50))
	require.Error(t, ensureDemoTutorMergeAccess(client, 300, 50))
	require.Equal(t, 1, patchCalls)
	branch.MergeAccessLevels = branch.MergeAccessLevels[:2]
	branch.PushAccessLevels[0].AccessLevel = gitlab.DeveloperPermissions
	require.False(t, demoMainBranchProtectionMatches(branch, 50))
}
