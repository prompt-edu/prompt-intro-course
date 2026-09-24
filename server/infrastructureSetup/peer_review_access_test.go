package infrastructureSetup

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

func TestPeerReviewAccessUsesDeveloperAndOptionalMainRule(t *testing.T) {
	const groupPath = "ase/ipraktikum/ios2627/Introcourse/tutor/peer-reviewers"
	var shared bool
	var shareBody map[string]any
	var ruleBody map[string]any
	var ruleCreated int

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v4/groups/"+groupPath:
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"message":"404 Group Not Found"}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v4/groups":
			var body map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			assert.Equal(t, "peer-reviewers", body["path"])
			assert.Equal(t, float64(10), body["parent_id"])
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 20, "parent_id": 10, "full_path": groupPath})
		case r.Method == http.MethodGet && r.URL.Path == "/api/v4/projects/300":
			groups := []map[string]any{}
			if shared {
				groups = append(groups, map[string]any{"group_id": 20, "group_access_level": 30})
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 300, "shared_with_groups": groups})
		case r.Method == http.MethodPost && r.URL.Path == "/api/v4/projects/300/share":
			require.NoError(t, json.NewDecoder(r.Body).Decode(&shareBody))
			shared = true
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]any{"group_id": 20, "group_access": 30})
		case r.Method == http.MethodGet && r.URL.Path == "/api/v4/projects/300/protected_branches/main":
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 40, "name": "main"})
		case r.Method == http.MethodGet && r.URL.Path == "/api/v4/projects/300/approval_rules":
			_ = json.NewEncoder(w).Encode([]any{})
		case r.Method == http.MethodPost && r.URL.Path == "/api/v4/projects/300/approval_rules":
			ruleCreated++
			require.NoError(t, json.NewDecoder(r.Body).Decode(&ruleBody))
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id": 1, "name": "Peer Review", "rule_type": "regular", "approvals_required": 0,
				"groups":             []map[string]any{{"id": 20}},
				"protected_branches": []map[string]any{{"id": 40, "name": "main"}},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	client, err := gitlab.NewClient("test-token", gitlab.WithBaseURL(server.URL+"/api/v4"))
	require.NoError(t, err)

	group, err := getOrCreatePeerReviewGroup(client, 10, "ase/ipraktikum/ios2627/Introcourse/tutor")
	require.NoError(t, err)
	assert.Equal(t, int64(20), group.ID)
	require.NoError(t, ensureProjectSharedWithPeerGroup(client, 300, group.ID))
	require.NoError(t, ensurePeerReviewRule(client, 300, "student-repo", group.ID))
	assert.Equal(t, float64(30), shareBody["group_access"])
	assert.Equal(t, float64(20), shareBody["group_id"])
	assert.Equal(t, float64(0), ruleBody["approvals_required"])
	assert.Equal(t, []any{float64(20)}, ruleBody["group_ids"])
	assert.Equal(t, []any{float64(40)}, ruleBody["protected_branch_ids"])
	assert.Equal(t, 1, ruleCreated)
}

func TestPeerReviewShareUpgradesReporterToDeveloper(t *testing.T) {
	access := 20
	deletes := 0
	shares := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v4/projects/300":
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 300, "shared_with_groups": []any{map[string]any{"group_id": 20, "group_access_level": access}}})
		case r.Method == http.MethodDelete && r.URL.Path == "/api/v4/projects/300/share/20":
			deletes++
			access = 0
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodPost && r.URL.Path == "/api/v4/projects/300/share":
			var body map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			access = int(body["group_access"].(float64))
			shares++
			w.WriteHeader(http.StatusCreated)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	client, err := gitlab.NewClient("test-token", gitlab.WithBaseURL(server.URL+"/api/v4"))
	require.NoError(t, err)
	require.NoError(t, ensureProjectSharedWithPeerGroup(client, 300, 20))
	require.NoError(t, ensureProjectSharedWithPeerGroup(client, 300, 20))
	assert.Equal(t, 30, access)
	assert.Equal(t, 1, deletes)
	assert.Equal(t, 1, shares)
}

func TestStudentMainMergeAccessExcludesPeerDevelopers(t *testing.T) {
	mergedByDeveloperRole := true
	updates := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path != "/api/v4/projects/300/protected_branches/main" {
			http.NotFound(w, r)
			return
		}
		if r.Method == http.MethodPatch {
			var body struct {
				AllowedToMerge []map[string]any `json:"allowed_to_merge"`
			}
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			require.Len(t, body.AllowedToMerge, 3)
			assert.Equal(t, float64(2), body.AllowedToMerge[0]["id"])
			assert.Equal(t, true, body.AllowedToMerge[0]["_destroy"])
			assert.Equal(t, float64(9), body.AllowedToMerge[1]["user_id"])
			assert.Equal(t, float64(40), body.AllowedToMerge[2]["access_level"])
			mergedByDeveloperRole = false
			updates++
		} else {
			require.Equal(t, http.MethodGet, r.Method)
		}
		mergeLevels := []any{map[string]any{"id": 2, "access_level": 30}}
		if !mergedByDeveloperRole {
			mergeLevels = []any{map[string]any{"id": 3, "user_id": 9, "access_level": 40}, map[string]any{"id": 4, "access_level": 40}}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"name": "main", "push_access_levels": []any{map[string]any{"id": 1, "access_level": 0}},
			"merge_access_levels": mergeLevels,
		})
	}))
	defer server.Close()
	client, err := gitlab.NewClient("test-token", gitlab.WithBaseURL(server.URL+"/api/v4"))
	require.NoError(t, err)
	require.NoError(t, restrictStudentMainMergeAccess(client, 300, 9))
	require.NoError(t, restrictStudentMainMergeAccess(client, 300, 9))
	assert.Equal(t, 1, updates)
}

func TestStudentProjectMemberUpgrade(t *testing.T) {
	access := 20
	edits := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path != "/api/v4/projects/300/members/9" {
			http.NotFound(w, r)
			return
		}
		if r.Method == http.MethodPut {
			var body map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			access = int(body["access_level"].(float64))
			edits++
		} else {
			require.Equal(t, http.MethodGet, r.Method)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"id": 9, "access_level": access})
	}))
	defer server.Close()
	client, err := gitlab.NewClient("test-token", gitlab.WithBaseURL(server.URL+"/api/v4"))
	require.NoError(t, err)
	require.NoError(t, ensureStudentProjectMember(client, 300, 9))
	require.NoError(t, ensureStudentProjectMember(client, 300, 9))
	assert.Equal(t, 30, access)
	assert.Equal(t, 1, edits)
}
