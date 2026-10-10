package peerAssignment

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	db "github.com/prompt-edu/prompt-intro-course/server/db/sqlc"
	"github.com/prompt-edu/prompt-intro-course/server/peerAssignment/peerAssignmentDTO"
	promptSDK "github.com/prompt-edu/prompt-sdk"
	log "github.com/sirupsen/logrus"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

type PeerAssignmentService struct {
	queries      db.Queries
	conn         *pgxpool.Pool
	gitlabClient *gitlab.Client
}

var PeerAssignmentServiceSingleton *PeerAssignmentService

// GetAllPeerAssignments returns all peer assignments for a course phase.
func GetAllPeerAssignments(ctx context.Context, coursePhaseID uuid.UUID) ([]peerAssignmentDTO.PeerAssignment, error) {
	ctxWithTimeout, cancel := db.GetTimeoutContext(ctx)
	defer cancel()

	assignments, err := PeerAssignmentServiceSingleton.queries.GetPeerAssignments(ctxWithTimeout, coursePhaseID)
	if err != nil {
		log.WithFields(log.Fields{
			"coursePhaseID": coursePhaseID,
			"error":         err,
		}).Error("Failed to get peer assignments")
		return nil, errors.New("failed to get peer assignments")
	}

	return peerAssignmentDTO.GetPeerAssignmentDTOsFromDBModels(assignments), nil
}

// GetOwnPeerAssignment returns the peer assignments for a specific student, including
// both the peers they review and the peers who review them.
func GetOwnPeerAssignment(ctx context.Context, coursePhaseID uuid.UUID, courseParticipationID uuid.UUID) (peerAssignmentDTO.OwnPeerAssignment, error) {
	ctxWithTimeout, cancel := db.GetTimeoutContext(ctx)
	defer cancel()

	peersIReview, err := PeerAssignmentServiceSingleton.queries.GetPeersForStudent(ctxWithTimeout, db.GetPeersForStudentParams{
		CoursePhaseID: coursePhaseID,
		StudentID:     courseParticipationID,
	})
	if err != nil {
		log.WithFields(log.Fields{
			"coursePhaseID":         coursePhaseID,
			"courseParticipationID": courseParticipationID,
			"error":                 err,
		}).Error("Failed to get peers for student")
		return peerAssignmentDTO.OwnPeerAssignment{}, errors.New("failed to get peer assignment")
	}

	peersWhoReviewMe, err := PeerAssignmentServiceSingleton.queries.GetReviewersForStudent(ctxWithTimeout, db.GetReviewersForStudentParams{
		CoursePhaseID: coursePhaseID,
		PeerID:        courseParticipationID,
	})
	if err != nil {
		log.WithFields(log.Fields{
			"coursePhaseID":         coursePhaseID,
			"courseParticipationID": courseParticipationID,
			"error":                 err,
		}).Error("Failed to get reviewers for student")
		return peerAssignmentDTO.OwnPeerAssignment{}, errors.New("failed to get peer assignment")
	}

	return peerAssignmentDTO.OwnPeerAssignment{
		PeersIReview:     peerAssignmentDTO.GetPeerInfosFromPeersRows(peersIReview),
		PeersWhoReviewMe: peerAssignmentDTO.GetPeerInfosFromReviewersRows(peersWhoReviewMe),
	}, nil
}

// GeneratePeerAssignments makes every student a peer of everyone else in their tutor group.
func GeneratePeerAssignments(ctx context.Context, coursePhaseID uuid.UUID) ([]peerAssignmentDTO.PeerAssignment, error) {
	ctxWithTimeout, cancel := db.GetTimeoutContext(ctx)
	defer cancel()

	svc := PeerAssignmentServiceSingleton

	// 1. Get all seats to determine tutor groups
	seats, err := svc.queries.GetSeatPlan(ctxWithTimeout, coursePhaseID)
	if err != nil {
		log.WithError(err).WithField("coursePhaseID", coursePhaseID).Error("Failed to get seat plan for peer assignment generation")
		return nil, errors.New("failed to generate peer assignments")
	}

	// 2. Group students by assigned tutor
	tutorGroups := make(map[uuid.UUID][]uuid.UUID) // tutorID -> []studentID
	for _, seat := range seats {
		if !seat.AssignedStudent.Valid || !seat.AssignedTutor.Valid {
			continue
		}
		tutorID := seat.AssignedTutor.Bytes
		studentID := seat.AssignedStudent.Bytes
		tutorGroups[tutorID] = append(tutorGroups[tutorID], studentID)
	}

	// 3. Generate pairs within each tutor group in a transaction
	tx, err := svc.conn.Begin(ctxWithTimeout)
	if err != nil {
		log.WithError(err).Error("Failed to begin transaction for peer assignment generation")
		return nil, errors.New("failed to generate peer assignments")
	}
	defer promptSDK.DeferDBRollback(tx, ctxWithTimeout)
	qtx := svc.queries.WithTx(tx)

	// Clear existing assignments
	if err := qtx.DeletePeerAssignments(ctxWithTimeout, coursePhaseID); err != nil {
		log.WithError(err).Error("Failed to clear existing peer assignments")
		return nil, errors.New("failed to generate peer assignments")
	}

	var allAssignments []peerAssignmentDTO.PeerAssignment

	for _, students := range tutorGroups {
		if len(students) < 2 {
			continue // cannot form a pair with fewer than 2 students
		}

		groups := createPeerGroups(students)

		for _, group := range groups {
			// Insert bidirectional assignments for each group
			for i := 0; i < len(group); i++ {
				for j := 0; j < len(group); j++ {
					if i == j {
						continue
					}
					err := qtx.CreatePeerAssignment(ctxWithTimeout, db.CreatePeerAssignmentParams{
						CoursePhaseID: coursePhaseID,
						StudentID:     group[i],
						PeerID:        group[j],
					})
					if err != nil {
						log.WithError(err).Error("Failed to insert peer assignment")
						return nil, errors.New("failed to generate peer assignments")
					}
					allAssignments = append(allAssignments, peerAssignmentDTO.PeerAssignment{
						StudentID: group[i],
						PeerID:    group[j],
					})
				}
			}
		}
	}

	if err := tx.Commit(ctxWithTimeout); err != nil {
		log.WithError(err).Error("Failed to commit peer assignment generation")
		return nil, errors.New("failed to generate peer assignments")
	}

	return allAssignments, nil
}

// createPeerGroups keeps the entire tutor group together.
func createPeerGroups(students []uuid.UUID) [][]uuid.UUID {
	if len(students) < 2 {
		return nil
	}
	return [][]uuid.UUID{students}
}

// UpdatePeerAssignments replaces all peer assignments with the provided set.
func UpdatePeerAssignments(ctx context.Context, coursePhaseID uuid.UUID, assignments []peerAssignmentDTO.PeerAssignment) error {
	ctxWithTimeout, cancel := db.GetTimeoutContext(ctx)
	defer cancel()

	svc := PeerAssignmentServiceSingleton

	tx, err := svc.conn.Begin(ctxWithTimeout)
	if err != nil {
		log.WithError(err).Error("Failed to begin transaction for peer assignment update")
		return errors.New("failed to update peer assignments")
	}
	defer promptSDK.DeferDBRollback(tx, ctxWithTimeout)
	qtx := svc.queries.WithTx(tx)

	// Clear existing
	if err := qtx.DeletePeerAssignments(ctxWithTimeout, coursePhaseID); err != nil {
		log.WithError(err).Error("Failed to clear existing peer assignments")
		return errors.New("failed to update peer assignments")
	}

	// Insert new assignments
	for _, a := range assignments {
		if a.StudentID == a.PeerID {
			return errors.New("self-review assignment not allowed")
		}
		err := qtx.CreatePeerAssignment(ctxWithTimeout, db.CreatePeerAssignmentParams{
			CoursePhaseID: coursePhaseID,
			StudentID:     a.StudentID,
			PeerID:        a.PeerID,
		})
		if err != nil {
			log.WithError(err).Error("Failed to update peer assignment")
			return errors.New("failed to update peer assignments")
		}
	}

	if err := tx.Commit(ctxWithTimeout); err != nil {
		log.WithError(err).Error("Failed to commit peer assignment update")
		return errors.New("failed to update peer assignments")
	}

	return nil
}

// DeletePeerAssignments removes all peer assignments for a course phase.
func DeletePeerAssignments(ctx context.Context, coursePhaseID uuid.UUID) error {
	ctxWithTimeout, cancel := db.GetTimeoutContext(ctx)
	defer cancel()

	err := PeerAssignmentServiceSingleton.queries.DeletePeerAssignments(ctxWithTimeout, coursePhaseID)
	if err != nil {
		log.WithFields(log.Fields{
			"coursePhaseID": coursePhaseID,
			"error":         err,
		}).Error("Failed to delete peer assignments")
		return errors.New("failed to delete peer assignments")
	}
	return nil
}
