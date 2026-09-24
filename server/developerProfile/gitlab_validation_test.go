package developerProfile

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

func TestCheckGitLabUsername(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v4/users" || r.URL.Query().Get("username") != "student123" {
			http.Error(w, "unexpected request", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"id":42,"username":"student123","name":"Student Name","web_url":"https://gitlab.lrz.de/student123"}]`))
	}))
	defer server.Close()
	client, err := gitlab.NewClient("token", gitlab.WithBaseURL(server.URL+"/api/v4"))
	require.NoError(t, err)

	found := checkGitLabUsername(client, "student123")
	require.Equal(t, "found", found.Status)
	require.Equal(t, "Student Name", found.GitLabName)
	require.Equal(t, "https://gitlab.lrz.de/student123", found.GitLabURL)
	require.Equal(t, "missing", checkGitLabUsername(client, "").Status)
	require.Equal(t, "invalid_format", checkGitLabUsername(client, "https://gitlab.lrz.de/student123").Status)
	require.Equal(t, "invalid_format", checkGitLabUsername(client, "student@tum.de").Status)
}
