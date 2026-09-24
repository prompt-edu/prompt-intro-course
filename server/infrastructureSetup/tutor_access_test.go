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
