package seatPlan

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	db "github.com/prompt-edu/prompt-intro-course/server/db/sqlc"
	"github.com/prompt-edu/prompt-intro-course/server/seatPlan/seatPlanDTO"
	promptSDK "github.com/prompt-edu/prompt-sdk"
	log "github.com/sirupsen/logrus"
)

// StudentInfo holds the data needed to match a student by name.
type StudentInfo struct {
	CourseParticipationID uuid.UUID
	FirstName            string
	LastName             string
}

// ImportSeatAssignments resolves names to IDs, updates all seats and peer
// assignments in a single atomic transaction.
func ImportSeatAssignments(
	ctx context.Context,
	coursePhaseID uuid.UUID,
	req seatPlanDTO.ImportRequest,
	students []StudentInfo,
) (*seatPlanDTO.ImportResult, error) {
	svc := SeatPlanServiceSingleton

	// Build name lookup maps (case-insensitive)
	studentNameToID := make(map[string]uuid.UUID)
	for _, s := range students {
		name := strings.ToLower(s.FirstName + " " + s.LastName)
		studentNameToID[name] = s.CourseParticipationID
	}

	tutors, err := svc.queries.GetAllTutors(ctx, coursePhaseID)
	if err != nil {
		return nil, fmt.Errorf("get tutors: %w", err)
	}
	tutorNameToID := make(map[string]uuid.UUID)
	for _, t := range tutors {
		name := strings.ToLower(t.FirstName + " " + t.LastName)
		tutorNameToID[name] = t.ID
	}

	// Resolve assignments
	type resolvedSeat struct {
		seatName string
		hasMac   bool
		student  pgtype.UUID
		tutor    pgtype.UUID
		isTutor  bool
	}

	var resolved []resolvedSeat
	peerGroups := make(map[string][]uuid.UUID) // group label -> student IDs
	var warnings []string
	var details []seatPlanDTO.ImportResultEntry

	for _, a := range req.Assignments {
		r := resolvedSeat{
			seatName: a.SeatName,
			hasMac:   a.SeatMac,
			isTutor:  a.IsTutorSeat,
		}

		// Match student
		studentName := strings.TrimSpace(a.AssignedStudent)
		if studentName != "" && strings.ToLower(studentName) != "unassigned" {
			if sid, ok := studentNameToID[strings.ToLower(studentName)]; ok {
				r.student = pgtype.UUID{Bytes: sid, Valid: true}
			} else {
				w := fmt.Sprintf("Student %q not found", studentName)
				warnings = append(warnings, w)
				details = append(details, seatPlanDTO.ImportResultEntry{
					SeatName: a.SeatName, Success: false, Warning: w,
				})
				continue
			}
		}

		// Match tutor
		tutorName := strings.TrimSpace(a.AssignedTutor)
		if tutorName != "" && strings.ToLower(tutorName) != "unknown tutor" {
			if tid, ok := tutorNameToID[strings.ToLower(tutorName)]; ok {
				r.tutor = pgtype.UUID{Bytes: tid, Valid: true}
			} else {
				w := fmt.Sprintf("Tutor %q not found", tutorName)
				warnings = append(warnings, w)
				details = append(details, seatPlanDTO.ImportResultEntry{
					SeatName: a.SeatName, Success: false, Warning: w,
				})
				continue
			}
		}

		// Collect peer group
		pg := strings.TrimSpace(a.PeerGroup)
		if pg != "" && r.student.Valid {
			peerGroups[pg] = append(peerGroups[pg], r.student.Bytes)
		}

		resolved = append(resolved, r)
		details = append(details, seatPlanDTO.ImportResultEntry{
			SeatName: a.SeatName, Success: true,
		})
	}

	// Execute in a single transaction
	tx, err := svc.conn.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer promptSDK.DeferDBRollback(tx, ctx)
	qtx := svc.queries.WithTx(tx)

	// Update seats
	seatsUpdated := 0
	for _, r := range resolved {
		err := qtx.UpdateSeat(ctx, db.UpdateSeatParams{
			CoursePhaseID:   coursePhaseID,
			SeatName:        r.seatName,
			HasMac:          r.hasMac,
			AssignedStudent: r.student,
			AssignedTutor:   r.tutor,
			IsTutorSeat:     r.isTutor,
		})
		if err != nil {
			log.WithFields(log.Fields{
				"seat":  r.seatName,
				"error": err,
			}).Warn("Failed to update seat during import")
			warnings = append(warnings, fmt.Sprintf("Failed to update seat %q: %s", r.seatName, err))
		} else {
			seatsUpdated++
		}
	}

	// Delete existing peer assignments and create new ones
	peerGroupsImported := 0
	if len(peerGroups) > 0 {
		if err := qtx.DeletePeerAssignments(ctx, coursePhaseID); err != nil {
			warnings = append(warnings, fmt.Sprintf("Failed to clear peer assignments: %s", err))
		} else {
			for _, members := range peerGroups {
				if len(members) < 2 {
					continue
				}
				peerGroupsImported++
				for i := 0; i < len(members); i++ {
					for j := 0; j < len(members); j++ {
						if i == j {
							continue
						}
						if err := qtx.CreatePeerAssignment(ctx, db.CreatePeerAssignmentParams{
							CoursePhaseID: coursePhaseID,
							StudentID:    members[i],
							PeerID:       members[j],
						}); err != nil {
							warnings = append(warnings, fmt.Sprintf("Failed to create peer assignment: %s", err))
						}
					}
				}
			}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	return &seatPlanDTO.ImportResult{
		SeatsUpdated:       seatsUpdated,
		PeerGroupsImported: peerGroupsImported,
		Warnings:           warnings,
		Details:            details,
	}, nil
}
