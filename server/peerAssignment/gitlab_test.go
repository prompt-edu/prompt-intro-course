package peerAssignment

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

func TestSharedPeerReviewGroupID(t *testing.T) {
	const tutorPath = "ase/iPraktikum/ios2627/Introcourse/tutor"
	const peerPath = tutorPath + "/peer-reviewers"

	tests := []struct {
		name        string
		groupStatus int
		groupPath   string
		shares      []map[string]any
		wantID      int64
		wantError   bool
	}{
		{
			name:        "shared peer group without approval rule is group based",
			groupStatus: http.StatusOK,
			groupPath:   peerPath,
			shares:      []map[string]any{{"group_id": 42, "group_access_level": 20}},
			wantID:      42,
		},
		{
			name:        "group exists but is not shared uses legacy access",
			groupStatus: http.StatusOK,
			groupPath:   peerPath,
			shares:      []map[string]any{{"group_id": 99, "group_access_level": 20}},
		},
		{
			name:        "missing peer group uses legacy access",
			groupStatus: http.StatusNotFound,
		},
		{
			name:        "unexpected group path is not accepted",
			groupStatus: http.StatusOK,
			groupPath:   "ase/iPraktikum/ios2627/Introcourse/other/peer-reviewers",
			wantError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			approvalRuleCalls := 0
			projectCalls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch {
				case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/api/v4/groups/"):
					w.WriteHeader(tt.groupStatus)
					if tt.groupStatus == http.StatusOK {
						require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"id": 42, "full_path": tt.groupPath}))
					}
				case r.Method == http.MethodGet && r.URL.Path == "/api/v4/projects/7":
					projectCalls++
					require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"id": 7, "shared_with_groups": tt.shares}))
				case strings.Contains(r.URL.Path, "approval_rules"):
					approvalRuleCalls++
					http.Error(w, "approval rules must not classify access", http.StatusInternalServerError)
				default:
					http.Error(w, "unexpected request", http.StatusNotFound)
				}
			}))
			defer server.Close()

			client, err := gitlab.NewClient("test-token", gitlab.WithBaseURL(server.URL+"/api/v4"))
			require.NoError(t, err)
			gotID, err := sharedPeerReviewGroupID(client, 7, tutorPath)
			if tt.wantError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.wantID, gotID)
			}
			require.Zero(t, approvalRuleCalls)
			if tt.groupStatus == http.StatusNotFound || tt.wantError {
				require.Zero(t, projectCalls)
			} else {
				require.Equal(t, 1, projectCalls)
			}
		})
	}
}
