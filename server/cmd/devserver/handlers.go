package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	db "github.com/prompt-edu/prompt-intro-course/server/db/sqlc"
	log "github.com/sirupsen/logrus"
)

// ── Seat Plan ────────────────────────────────────────────────────────

type SeatDTO struct {
	SeatName        string  `json:"seatName"`
	HasMac          bool    `json:"hasMac"`
	DeviceID        *string `json:"deviceID"`
	AssignedStudent *string `json:"assignedStudent"`
	AssignedTutor   *string `json:"assignedTutor"`
	IsTutorSeat     bool    `json:"isTutorSeat"`
}

func getSeatPlanHandler(q *db.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		phaseID, err := uuid.Parse(c.Param("coursePhaseID"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		rows, err := q.GetSeatPlan(c, phaseID)
		if err != nil {
			log.Error("GetSeatPlan error: ", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		seats := make([]SeatDTO, len(rows))
		for i, r := range rows {
			seats[i] = SeatDTO{
				SeatName:    r.SeatName,
				HasMac:      r.HasMac,
				IsTutorSeat: r.IsTutorSeat,
			}
			if r.DeviceID.Valid {
				s := r.DeviceID.String
				seats[i].DeviceID = &s
			}
			if r.AssignedStudent.Valid {
				s := uuidToString(r.AssignedStudent)
				seats[i].AssignedStudent = &s
			}
			if r.AssignedTutor.Valid {
				s := uuidToString(r.AssignedTutor)
				seats[i].AssignedTutor = &s
			}
		}
		c.JSON(http.StatusOK, seats)
	}
}

func updateSeatPlanHandler(q *db.Queries, conn *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		phaseID, err := uuid.Parse(c.Param("coursePhaseID"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		var seats []SeatDTO
		if err := c.BindJSON(&seats); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		tx, err := conn.Begin(c)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer tx.Rollback(c)
		qtx := q.WithTx(tx)
		for _, s := range seats {
			params := db.UpdateSeatParams{
				CoursePhaseID: phaseID,
				SeatName:      s.SeatName,
				HasMac:        s.HasMac,
				IsTutorSeat:   s.IsTutorSeat,
			}
			if s.DeviceID != nil {
				params.DeviceID = pgtype.Text{String: *s.DeviceID, Valid: true}
			}
			if s.AssignedStudent != nil {
				uid, _ := uuid.Parse(*s.AssignedStudent)
				params.AssignedStudent = pgtype.UUID{Bytes: uid, Valid: true}
			}
			if s.AssignedTutor != nil {
				uid, _ := uuid.Parse(*s.AssignedTutor)
				params.AssignedTutor = pgtype.UUID{Bytes: uid, Valid: true}
			}
			if err := qtx.UpdateSeat(c, params); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
		}
		if err := tx.Commit(c); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.Status(http.StatusOK)
	}
}

type SeatAssignmentDTO struct {
	SeatName                      string  `json:"seatName"`
	HasMac                        bool    `json:"hasMac"`
	DeviceID                      *string `json:"deviceID"`
	StudentCourseParticipationID  string  `json:"studentCourseParticipationID"`
	TutorFirstName                string  `json:"tutorFirstName"`
	TutorLastName                 string  `json:"tutorLastName"`
	TutorEmail                    string  `json:"tutorEmail"`
}

func getOwnSeatAssignmentHandler(q *db.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		phaseID, err := uuid.Parse(c.Param("coursePhaseID"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		cpID, _ := c.Get("courseParticipationID")
		participationID := cpID.(uuid.UUID)

		row, err := q.GetOwnSeatAssignment(c, db.GetOwnSeatAssignmentParams{
			CoursePhaseID:      phaseID,
			AssignedStudent:    pgtype.UUID{Bytes: participationID, Valid: true},
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		dto := SeatAssignmentDTO{
			SeatName:                     row.SeatName,
			HasMac:                       row.HasMac,
			StudentCourseParticipationID: participationID.String(),
			TutorFirstName:               row.TutorFirstName,
			TutorLastName:                row.TutorLastName,
			TutorEmail:                   row.TutorEmail,
		}
		if row.DeviceID.Valid {
			dto.DeviceID = &row.DeviceID.String
		}
		c.JSON(http.StatusOK, dto)
	}
}

// ── Tutors ───────────────────────────────────────────────────────────

type TutorDTO struct {
	ID        string `json:"id"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Email     string `json:"email"`
}

func getTutorsHandler(q *db.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		phaseID, err := uuid.Parse(c.Param("coursePhaseID"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		rows, err := q.GetAllTutors(c, phaseID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		tutors := make([]TutorDTO, len(rows))
		for i, r := range rows {
			tutors[i] = TutorDTO{
				ID:        uuidToString(pgtype.UUID{Bytes: r.ID, Valid: true}),
				FirstName: r.FirstName,
				LastName:  r.LastName,
				Email:     r.Email,
			}
		}
		c.JSON(http.StatusOK, tutors)
	}
}

// ── Peer Assignments ─────────────────────────────────────────────────

type PeerAssignmentDTO struct {
	StudentID string `json:"studentID"`
	PeerID    string `json:"peerID"`
}

func getPeerAssignmentsHandler(q *db.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		phaseID, err := uuid.Parse(c.Param("coursePhaseID"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		rows, err := q.GetPeerAssignments(c, phaseID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		result := make([]PeerAssignmentDTO, len(rows))
		for i, r := range rows {
			result[i] = PeerAssignmentDTO{
				StudentID: uuidToString(pgtype.UUID{Bytes: r.StudentID, Valid: true}),
				PeerID:    uuidToString(pgtype.UUID{Bytes: r.PeerID, Valid: true}),
			}
		}
		c.JSON(http.StatusOK, result)
	}
}

type PeerInfoDTO struct {
	CourseParticipationID string `json:"courseParticipationID"`
	GitlabUsername        string `json:"gitlabUsername"`
	SeatName              string `json:"seatName"`
	TutorGitlabUsername   string `json:"tutorGitlabUsername"`
}

type OwnPeerAssignmentDTO struct {
	PeersIReview    []PeerInfoDTO `json:"peersIReview"`
	PeersWhoReviewMe []PeerInfoDTO `json:"peersWhoReviewMe"`
}

func getOwnPeerAssignmentsHandler(q *db.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		phaseID, err := uuid.Parse(c.Param("coursePhaseID"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		cpID, _ := c.Get("courseParticipationID")
		participationID := cpID.(uuid.UUID)

		peers, err := q.GetPeersForStudent(c, db.GetPeersForStudentParams{
			CoursePhaseID: phaseID,
			StudentID:     participationID,
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		reviewers, err := q.GetReviewersForStudent(c, db.GetReviewersForStudentParams{
			CoursePhaseID: phaseID,
			PeerID:        participationID,
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		dto := OwnPeerAssignmentDTO{
			PeersIReview:    make([]PeerInfoDTO, len(peers)),
			PeersWhoReviewMe: make([]PeerInfoDTO, len(reviewers)),
		}
		for i, p := range peers {
			dto.PeersIReview[i] = PeerInfoDTO{
				CourseParticipationID: uuidToString(pgtype.UUID{Bytes: p.PeerID, Valid: true}),
				GitlabUsername:        stringOrEmpty(p.GitlabUsername),
				SeatName:              stringOrEmpty(p.SeatName),
				TutorGitlabUsername:   stringOrEmpty2(p.TutorGitlabUsername),
			}
		}
		for i, r := range reviewers {
			dto.PeersWhoReviewMe[i] = PeerInfoDTO{
				CourseParticipationID: uuidToString(pgtype.UUID{Bytes: r.StudentID, Valid: true}),
				GitlabUsername:        stringOrEmpty(r.GitlabUsername),
				SeatName:              stringOrEmpty(r.SeatName),
				TutorGitlabUsername:   stringOrEmpty2(r.TutorGitlabUsername),
			}
		}
		c.JSON(http.StatusOK, dto)
	}
}

// ── Developer Profiles ───────────────────────────────────────────────

type DeveloperProfileDTO struct {
	CourseParticipationID string  `json:"courseParticipationID"`
	GitLabUsername        string  `json:"gitLabUsername"`
	AppleID               string  `json:"appleId"`
	HasMacBook            bool    `json:"hasMacBook"`
}

func getAllDeveloperProfilesHandler(q *db.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		phaseID, err := uuid.Parse(c.Param("coursePhaseID"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		rows, err := q.GetAllDeveloperProfiles(c, phaseID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		profiles := make([]DeveloperProfileDTO, len(rows))
		for i, r := range rows {
			profiles[i] = DeveloperProfileDTO{
				CourseParticipationID: uuidToString(pgtype.UUID{Bytes: r.CourseParticipationID, Valid: true}),
				GitLabUsername:        r.GitlabUsername,
				AppleID:               r.AppleID,
				HasMacBook:            r.HasMacbook,
			}
		}
		c.JSON(http.StatusOK, profiles)
	}
}

// ── Helpers ──────────────────────────────────────────────────────────

func uuidToString(u pgtype.UUID) string {
	if !u.Valid {
		return ""
	}
	return uuid.UUID(u.Bytes).String()
}

func stringOrEmpty(t pgtype.Text) string {
	if !t.Valid {
		return ""
	}
	return t.String
}

func stringOrEmpty2(t pgtype.Text) string {
	if !t.Valid {
		return ""
	}
	return t.String
}
