package team

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prompt-edu/prompt-intro-course/server/coreRequests"
	db "github.com/prompt-edu/prompt-intro-course/server/db/sqlc"
	"github.com/prompt-edu/prompt-intro-course/server/team/teamDTO"
	"github.com/prompt-edu/prompt-sdk/promptTypes"
	log "github.com/sirupsen/logrus"
)

var ErrAllocationNotFound = errors.New("no team allocation found for this participant")

type TeamService struct {
	queries db.Queries
	conn    *pgxpool.Pool
}

var TeamServiceSingleton *TeamService

// GetTeams returns one team per tutor, holding the students seated in that tutor's seats.
func GetTeams(ctx context.Context, coursePhaseID uuid.UUID) ([]promptTypes.Team, error) {
	ctxWithTimeout, cancel := db.GetTimeoutContext(ctx)
	defer cancel()

	dbTeams, err := TeamServiceSingleton.queries.GetTutorTeams(ctxWithTimeout, coursePhaseID)
	if err != nil {
		log.WithError(err).WithField("coursePhaseID", coursePhaseID).Error("Failed to get tutor teams")
		return nil, errors.New("failed to get teams")
	}

	teams, err := teamDTO.GetTeamDTOsFromDBModels(dbTeams)
	if err != nil {
		log.WithError(err).WithField("coursePhaseID", coursePhaseID).Error("Failed to parse tutor team members")
		return nil, errors.New("failed to get teams")
	}
	return teams, nil
}

func GetAllocations(ctx context.Context, coursePhaseID uuid.UUID) ([]teamDTO.AllocationWithParticipation, error) {
	ctxWithTimeout, cancel := db.GetTimeoutContext(ctx)
	defer cancel()

	allocations, err := TeamServiceSingleton.queries.GetTeamAllocations(ctxWithTimeout, coursePhaseID)
	if err != nil {
		log.WithError(err).WithField("coursePhaseID", coursePhaseID).Error("Failed to get team allocations")
		return nil, errors.New("failed to get team allocations")
	}
	return teamDTO.GetAllocationDTOsFromDBModels(allocations), nil
}

// GetAllocation returns the team (tutor) ID of a participant's seat.
func GetAllocation(ctx context.Context, coursePhaseID, courseParticipationID uuid.UUID) (uuid.UUID, error) {
	ctxWithTimeout, cancel := db.GetTimeoutContext(ctx)
	defer cancel()

	teamID, err := TeamServiceSingleton.queries.GetTeamAllocation(ctxWithTimeout, db.GetTeamAllocationParams{
		CoursePhaseID:         coursePhaseID,
		CourseParticipationID: courseParticipationID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, ErrAllocationNotFound
	}
	if err != nil {
		log.WithError(err).WithFields(log.Fields{
			"coursePhaseID":         coursePhaseID,
			"courseParticipationID": courseParticipationID,
		}).Error("Failed to get team allocation")
		return uuid.Nil, errors.New("failed to get team allocation")
	}
	return teamID, nil
}

// RefreshParticipantNames copies the participants' names from core into the local cache
// the teams are served from. Core only lists participations to admins, lecturers and
// editors, so authHeader must belong to one of them.
func RefreshParticipantNames(ctx context.Context, authHeader string, coursePhaseID uuid.UUID) error {
	participations, err := coreRequests.GetCoursePhaseParticipations(authHeader, coursePhaseID)
	if err != nil {
		return fmt.Errorf("fetch participations from core: %w", err)
	}

	params := db.UpsertParticipantNamesParams{CoursePhaseID: coursePhaseID}
	for _, participation := range participations {
		courseParticipationID, err := uuid.Parse(participation.CourseParticipationID)
		if err != nil {
			log.WithError(err).Warn("Skipping participation with invalid courseParticipationID")
			continue
		}
		params.CourseParticipationIds = append(params.CourseParticipationIds, courseParticipationID)
		params.FirstNames = append(params.FirstNames, participation.Student.FirstName)
		params.LastNames = append(params.LastNames, participation.Student.LastName)
	}
	if len(params.CourseParticipationIds) == 0 {
		return nil
	}

	ctxWithTimeout, cancel := db.GetTimeoutContext(ctx)
	defer cancel()

	if err := TeamServiceSingleton.queries.UpsertParticipantNames(ctxWithTimeout, params); err != nil {
		return fmt.Errorf("store participant names: %w", err)
	}
	return nil
}
