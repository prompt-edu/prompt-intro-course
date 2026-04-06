package coreRequests

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
)

type StudentData struct {
	FirstName           string `json:"firstName"`
	LastName            string `json:"lastName"`
	MatriculationNumber string `json:"matriculationNumber"`
}

type Participation struct {
	CourseParticipationID string      `json:"courseParticipationID"`
	Student               StudentData `json:"student"`
}

type ParticipationsResponse struct {
	Participations []Participation `json:"participations"`
}

// GetCoursePhaseParticipations fetches all participations for a course phase
// from the core platform, forwarding the caller's Authorization header.
func GetCoursePhaseParticipations(authHeader string, coursePhaseID uuid.UUID) ([]Participation, error) {
	path := "/api/course_phases/" + coursePhaseID.String() + "/participations"

	resp, err := sendRequest("GET", path, authHeader, nil)
	if err != nil {
		return nil, fmt.Errorf("request participations: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			log.Warn("Failed to close response body:", closeErr)
		}
	}()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("core returned %s: %s", resp.Status, string(body))
	}

	var result ParticipationsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode participations: %w", err)
	}

	return result.Participations, nil
}
