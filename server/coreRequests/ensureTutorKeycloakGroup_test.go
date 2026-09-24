package coreRequests

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestSendEnsureCustomKeycloakGroup(t *testing.T) {
	courseID := uuid.New()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/api/keycloak/"+courseID.String()+"/group" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Errorf("authorization header = %q", got)
		}
		var payload struct {
			GroupName string `json:"groupName"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil || payload.GroupName != "introCourseTutors" {
			t.Errorf("unexpected payload: %+v, %v", payload, err)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	t.Setenv("SERVER_CORE_HOST", server.URL)

	if err := SendEnsureCustomKeycloakGroup("Bearer test-token", courseID, "introCourseTutors"); err != nil {
		t.Fatal(err)
	}
}

func TestSendEnsureCustomKeycloakGroupReportsFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"missing course group"}`))
	}))
	defer server.Close()
	t.Setenv("SERVER_CORE_HOST", server.URL)

	err := SendEnsureCustomKeycloakGroup("Bearer test-token", uuid.New(), "introCourseTutors")
	if err == nil || !strings.Contains(err.Error(), "missing course group") {
		t.Fatalf("expected course-group error, got %v", err)
	}
}
