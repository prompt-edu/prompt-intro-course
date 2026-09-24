package coreRequests

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestSendAddStudentsToKeycloakGroup(t *testing.T) {
	courseID := uuid.New()
	tutorIDs := []uuid.UUID{uuid.New(), uuid.New()}
	tests := []struct {
		name      string
		response  any
		wantError string
	}{
		{
			name: "all tutors added",
			response: map[string]any{
				"succeededToAddStudentIDs": tutorIDs,
				"failedToAddStudentIDs":    []uuid.UUID{},
			},
		},
		{
			name: "partial failure",
			response: map[string]any{
				"succeededToAddStudentIDs": tutorIDs[:1],
				"failedToAddStudentIDs":    tutorIDs[1:],
			},
			wantError: "could not add 1 of 2 tutors",
		},
		{
			name: "missing success confirmation",
			response: map[string]any{
				"succeededToAddStudentIDs": tutorIDs[:1],
				"failedToAddStudentIDs":    []uuid.UUID{},
			},
			wantError: "did not confirm tutor",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPut || r.URL.Path != "/api/keycloak/"+courseID.String()+"/group/introCourseTutors/students" {
					t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
				}
				if r.Header.Get("Authorization") != "Bearer test-token" {
					t.Errorf("authorization header missing")
				}
				var request struct {
					StudentsToAdd []uuid.UUID `json:"studentsToAdd"`
				}
				if err := json.NewDecoder(r.Body).Decode(&request); err != nil || len(request.StudentsToAdd) != 2 {
					t.Errorf("unexpected request body: %+v, %v", request, err)
				}
				_ = json.NewEncoder(w).Encode(tc.response)
			}))
			defer server.Close()
			t.Setenv("SERVER_CORE_HOST", server.URL)

			err := SendAddStudentsToKeycloakGroup("Bearer test-token", courseID, tutorIDs, "introCourseTutors")
			if tc.wantError == "" && err != nil {
				t.Fatal(err)
			}
			if tc.wantError != "" && (err == nil || !strings.Contains(err.Error(), tc.wantError)) {
				t.Fatalf("wanted error containing %q, got %v", tc.wantError, err)
			}
		})
	}
}
