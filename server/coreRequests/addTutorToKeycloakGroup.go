package coreRequests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"

	"github.com/google/uuid"
	"github.com/prompt-edu/prompt-intro-course/server/coreRequests/coreRequestDTOs"
	log "github.com/sirupsen/logrus"
)

func SendAddStudentsToKeycloakGroup(authHeader string, courseID uuid.UUID, studentIDs []uuid.UUID, groupName string) error {
	path := "/api/keycloak/" + courseID.String() + "/group/" + groupName + "/students"

	// Create the payload
	payload := coreRequestDTOs.AddStudentsToGroup{
		StudentsToAdd: studentIDs,
	}

	// Marshal payload to JSON
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	// Send the request with the payload attached
	resp, err := sendRequest("PUT", path, authHeader, bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			log.Warn("Failed to close response body:", closeErr)
		}
	}()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			log.Error("Failed to read error response body: ", readErr)
		}
		log.Errorf("Received non-2xx response from core server: status=%s body=%s", resp.Status, string(respBody))
		return fmt.Errorf("core server returned %s: %s", resp.Status, string(respBody))
	}

	return nil
}
