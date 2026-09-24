package coreRequests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"

	"github.com/google/uuid"
)

// SendEnsureCustomKeycloakGroup repairs a missing tutor group before assigning
// members. Course metadata can outlive the corresponding Keycloak group.
func SendEnsureCustomKeycloakGroup(authHeader string, courseID uuid.UUID, groupName string) error {
	body, err := json.Marshal(struct {
		GroupName string `json:"groupName"`
	}{GroupName: groupName})
	if err != nil {
		return err
	}

	resp, err := sendRequest("PUT", "/api/keycloak/"+courseID.String()+"/group", authHeader, bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		responseBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("core server returned %s: %s", resp.Status, responseBody)
	}
	return nil
}
