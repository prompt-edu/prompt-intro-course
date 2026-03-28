package peerAssignment

import (
	"context"
	"errors"
	"math/rand/v2"

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

func GetAllPeerAssignments(ctx context.Context, coursePhaseID uuid.UUID) ([]peerAssignmentDTO.PeerAssignment, error) {
	ctxWithTimeout, cancel := db.GetTimeoutContext(ctx)
	defer cancel()

	assignments, err := PeerAssignmentServiceSingleton.queries.GetPeerAssignments(ctxWithTimeout, coursePhaseID)
	if err != nil {
		log.WithFields(log.Fields{
			"coursePhaseID": coursePhaseID,
			"error":        err,
		}).Error("Failed to get peer assignments")
		return nil, errors.New("failed to get peer assignments")
	}

	return peerAssignmentDTO.GetPeerAssignmentDTOsFromDBModels(assignments), nil
}

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
			"error":                err,
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
			"error":                err,
		}).Error("Failed to get reviewers for student")
		return peerAssignmentDTO.OwnPeerAssignment{}, errors.New("failed to get peer assignment")
	}

	return peerAssignmentDTO.OwnPeerAssignment{
		PeersIReview:     peerAssignmentDTO.GetPeerInfosFromPeersRows(peersIReview),
		PeersWhoReviewMe: peerAssignmentDTO.GetPeerInfosFromReviewersRows(peersWhoReviewMe),
	}, nil
}

// GeneratePeerAssignments creates peer pairs within each tutor group.
// For odd-sized groups, the last group becomes a triple.
func GeneratePeerAssignments(ctx context.Context, coursePhaseID uuid.UUID) ([]peerAssignmentDTO.PeerAssignment, error) {
	svc := PeerAssignmentServiceSingleton

	// 1. Get all seats to determine tutor groups
	seats, err := svc.queries.GetSeatPlan(ctx, coursePhaseID)
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
	tx, err := svc.conn.Begin(ctx)
	if err != nil {
		log.WithError(err).Error("Failed to begin transaction for peer assignment generation")
		return nil, errors.New("failed to generate peer assignments")
	}
	defer promptSDK.DeferDBRollback(tx, ctx)
	qtx := svc.queries.WithTx(tx)

	// Clear existing assignments
	if err := qtx.DeletePeerAssignments(ctx, coursePhaseID); err != nil {
		log.WithError(err).Error("Failed to clear existing peer assignments")
		return nil, errors.New("failed to generate peer assignments")
	}

	var allAssignments []peerAssignmentDTO.PeerAssignment

	for _, students := range tutorGroups {
		if len(students) < 2 {
			continue // cannot form a pair with fewer than 2 students
		}

		// Shuffle for randomness
		rand.Shuffle(len(students), func(i, j int) {
			students[i], students[j] = students[j], students[i]
		})

		groups := createPeerGroups(students)

		for _, group := range groups {
			// Insert bidirectional assignments for each pair/triple
			for i := 0; i < len(group); i++ {
				for j := 0; j < len(group); j++ {
					if i == j {
						continue
					}
					err := qtx.CreatePeerAssignment(ctx, db.CreatePeerAssignmentParams{
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

	if err := tx.Commit(ctx); err != nil {
		log.WithError(err).Error("Failed to commit peer assignment generation")
		return nil, errors.New("failed to generate peer assignments")
	}

	return allAssignments, nil
}

// createPeerGroups splits students into groups of 3 (triples) or 4 (quads),
// ensuring each student has 2-3 peers. Uses 3a + 4b = n to partition.
func createPeerGroups(students []uuid.UUID) [][]uuid.UUID {
	n := len(students)
	if n < 2 {
		return nil
	}
	if n == 2 {
		// Best effort: pair gives only 1 peer each
		return [][]uuid.UUID{{students[0], students[1]}}
	}

	// Determine how many quads we need to absorb the remainder after triples
	numQuads := 0
	switch n % 3 {
	case 1:
		numQuads = 1 // 3a + 4(1) covers remainder 1
	case 2:
		if n >= 8 {
			numQuads = 2 // 3a + 4(2) covers remainder 2
		}
		// n=5: fall through with 0 quads, handle remainder below
	}
	numTriples := (n - 4*numQuads) / 3

	var groups [][]uuid.UUID
	idx := 0
	for i := 0; i < numTriples; i++ {
		groups = append(groups, []uuid.UUID{students[idx], students[idx+1], students[idx+2]})
		idx += 3
	}
	for i := 0; i < numQuads; i++ {
		groups = append(groups, []uuid.UUID{students[idx], students[idx+1], students[idx+2], students[idx+3]})
		idx += 4
	}
	// n=5 edge case: 1 triple + 2 remaining → best-effort pair
	if idx < n {
		remaining := make([]uuid.UUID, n-idx)
		copy(remaining, students[idx:])
		groups = append(groups, remaining)
	}
	return groups
}

// UpdatePeerAssignments replaces all peer assignments with the provided set.
func UpdatePeerAssignments(ctx context.Context, coursePhaseID uuid.UUID, assignments []peerAssignmentDTO.PeerAssignment) error {
	svc := PeerAssignmentServiceSingleton

	tx, err := svc.conn.Begin(ctx)
	if err != nil {
		log.WithError(err).Error("Failed to begin transaction for peer assignment update")
		return errors.New("failed to update peer assignments")
	}
	defer promptSDK.DeferDBRollback(tx, ctx)
	qtx := svc.queries.WithTx(tx)

	// Clear existing
	if err := qtx.DeletePeerAssignments(ctx, coursePhaseID); err != nil {
		log.WithError(err).Error("Failed to clear existing peer assignments")
		return errors.New("failed to update peer assignments")
	}

	// Insert new assignments
	for _, a := range assignments {
		if a.StudentID == a.PeerID {
			continue // skip self-review
		}
		err := qtx.CreatePeerAssignment(ctx, db.CreatePeerAssignmentParams{
			CoursePhaseID: coursePhaseID,
			StudentID:     a.StudentID,
			PeerID:        a.PeerID,
		})
		if err != nil {
			log.WithError(err).Error("Failed to update peer assignment")
			return errors.New("failed to update peer assignments")
		}
	}

	if err := tx.Commit(ctx); err != nil {
		log.WithError(err).Error("Failed to commit peer assignment update")
		return errors.New("failed to update peer assignments")
	}

	return nil
}

func DeletePeerAssignments(ctx context.Context, coursePhaseID uuid.UUID) error {
	ctxWithTimeout, cancel := db.GetTimeoutContext(ctx)
	defer cancel()

	err := PeerAssignmentServiceSingleton.queries.DeletePeerAssignments(ctxWithTimeout, coursePhaseID)
	if err != nil {
		log.WithFields(log.Fields{
			"coursePhaseID": coursePhaseID,
			"error":        err,
		}).Error("Failed to delete peer assignments")
		return errors.New("failed to delete peer assignments")
	}
	return nil
}
