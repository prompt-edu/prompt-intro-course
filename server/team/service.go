package team

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/prompt-edu/prompt-intro-course/server/coreRequests"
	db "github.com/prompt-edu/prompt-intro-course/server/db/sqlc"
	"github.com/prompt-edu/prompt-intro-course/server/team/teamDTO"
	"github.com/prompt-edu/prompt-sdk/promptTypes"
	log "github.com/sirupsen/logrus"
)

var ErrAllocationNotFound = errors.New("no team allocation found for this participant")

// nameRefreshInterval bounds how often staff reads of the teams copy names from core.
const nameRefreshInterval = 5 * time.Minute

type TeamService struct {
	queries db.Queries

	// lastNameRefresh maps a course phase ID to the time its names were last copied from core.
	lastNameRefresh sync.Map
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

// RefreshParticipantNamesIfStale runs RefreshParticipantNames at most once per
// nameRefreshInterval and course phase.
func RefreshParticipantNamesIfStale(ctx context.Context, authHeader string, coursePhaseID uuid.UUID) error {
	if last, ok := TeamServiceSingleton.lastNameRefresh.Load(coursePhaseID); ok && time.Since(last.(time.Time)) < nameRefreshInterval {
		return nil
	}
	if err := RefreshParticipantNames(ctx, authHeader, coursePhaseID); err != nil {
		return err
	}
	TeamServiceSingleton.lastNameRefresh.Store(coursePhaseID, time.Now())
	return nil
}

// RefreshParticipantNames copies the participants' names from core into the local cache
// the teams are served from. Core only lists participations to admins, lecturers and
// editors, so authHeader must belong to one of them.
func RefreshParticipantNames(ctx context.Context, authHeader string, coursePhaseID uuid.UUID) error {
	participations, err := coreRequests.GetCoursePhaseParticipations(authHeader, coursePhaseID)
	if err != nil {
		return fmt.Errorf("fetch participations from core: %w", err)
	}

	participants := make([]promptTypes.Person, 0, len(participations))
	for _, participation := range participations {
		courseParticipationID, err := uuid.Parse(participation.CourseParticipationID)
		if err != nil {
			log.WithError(err).Warn("Skipping participation with invalid courseParticipationID")
			continue
		}
		participants = append(participants, promptTypes.Person{
			ID:        courseParticipationID,
			FirstName: participation.Student.FirstName,
			LastName:  participation.Student.LastName,
		})
	}
	return StoreParticipantNames(ctx, TeamServiceSingleton.queries, coursePhaseID, participants)
}

// StoreParticipantNames upserts names into the participant name cache. Rows are written
// in ID order so concurrent upserts lock them in the same order.
func StoreParticipantNames(ctx context.Context, queries db.Queries, coursePhaseID uuid.UUID, participants []promptTypes.Person) error {
	sorted := make([]promptTypes.Person, 0, len(participants))
	for _, participant := range participants {
		if participant.ID != uuid.Nil {
			sorted = append(sorted, participant)
		}
	}
	if len(sorted) == 0 {
		return nil
	}
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].ID.String() < sorted[j].ID.String() })

	params := db.UpsertParticipantNamesParams{CoursePhaseID: coursePhaseID}
	for _, participant := range sorted {
		params.CourseParticipationIds = append(params.CourseParticipationIds, participant.ID)
		params.FirstNames = append(params.FirstNames, participant.FirstName)
		params.LastNames = append(params.LastNames, participant.LastName)
	}

	ctxWithTimeout, cancel := db.GetTimeoutContext(ctx)
	defer cancel()

	if err := queries.UpsertParticipantNames(ctxWithTimeout, params); err != nil {
		return fmt.Errorf("store participant names: %w", err)
	}
	return nil
}

// tutorTeamResolver maps a tutor's university login to their team, which carries the tutor's ID.
type tutorTeamResolver struct{}

func (tutorTeamResolver) ResolveTutorTeam(ctx context.Context, coursePhaseID uuid.UUID, universityLogin string) (uuid.UUID, error) {
	return TeamServiceSingleton.queries.GetTutorTeamByUniversityLogin(ctx, db.GetTutorTeamByUniversityLoginParams{
		CoursePhaseID:   coursePhaseID,
		UniversityLogin: universityLogin,
	})
}
