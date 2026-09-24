package infrastructureSetup

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type fakeStatusBoard struct {
	created bool
	hidden  bool
	lists   map[string]int
}

func (fake *fakeStatusBoard) handle(t *testing.T, w http.ResponseWriter, r *http.Request) bool {
	if r.URL.Path == "/api/v4/projects/300" && r.Method == http.MethodGet {
		_ = json.NewEncoder(w).Encode(map[string]any{"id": 300, "name": "demo", "path_with_namespace": "ase/ipraktikum/introcourse/demo",
			"visibility": "private", "ci_config_path": ".gitlab-ci.yml@ase/ipraktikum/introcourse/ci-cd", "merge_method": "merge", "squash_option": "default_on",
			"remove_source_branch_after_merge": true, "only_allow_merge_if_pipeline_succeeds": true, "only_allow_merge_if_all_discussions_are_resolved": true})
		return true
	}
	if r.URL.Path == "/api/v4/projects/300/boards" {
		switch r.Method {
		case http.MethodGet:
			if fake.created {
				_ = json.NewEncoder(w).Encode([]map[string]any{{"id": 1, "name": courseBoardName, "hide_backlog_list": fake.hidden, "hide_closed_list": fake.hidden}})
			} else {
				_, _ = w.Write([]byte(`[]`))
			}
		case http.MethodPost:
			fake.created = true
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 1, "name": courseBoardName})
		}
		return true
	}
	if r.URL.Path == "/api/v4/projects/300/boards/1" && r.Method == http.MethodPut {
		fake.hidden = true
		_ = json.NewEncoder(w).Encode(map[string]any{"id": 1, "name": courseBoardName, "hide_backlog_list": true, "hide_closed_list": true})
		return true
	}
	if r.URL.Path != "/api/graphql" {
		return false
	}
	var body struct {
		Query     string         `json:"query"`
		Variables map[string]any `json:"variables"`
	}
	require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
	if strings.Contains(body.Query, "boardListCreate") {
		if fake.lists == nil {
			fake.lists = make(map[string]int)
		}
		fake.lists[body.Variables["status"].(string)] = int(body.Variables["position"].(float64))
		_, _ = w.Write([]byte(`{"data":{"boardListCreate":{"errors":[]}}}`))
		return true
	}
	statuses := []map[string]string{}
	for _, item := range []struct{ name, id string }{{"Open", "80"}, {"In Progress", "81"}, {"In Review", "85"}, {"Blocked", "87"}, {"Done", "82"}} {
		statuses = append(statuses, map[string]string{"name": item.name, "id": "gid://gitlab/WorkItems::Statuses::Custom::Status/" + item.id})
	}
	lists := []map[string]any{}
	for id, position := range fake.lists {
		lists = append(lists, map[string]any{"position": position, "status": map[string]any{"id": id}})
	}
	boards := []map[string]any{}
	if fake.created {
		boards = append(boards, map[string]any{"id": "gid://gitlab/Board/1", "name": courseBoardName, "lists": map[string]any{"nodes": lists}})
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"project": map[string]any{
		"group":  map[string]any{"lifecycles": map[string]any{"nodes": []map[string]any{{"name": "Default", "statuses": statuses}}}},
		"boards": map[string]any{"nodes": boards},
	}}})
	return true
}
