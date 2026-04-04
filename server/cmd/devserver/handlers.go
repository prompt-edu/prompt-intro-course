package main

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	db "github.com/prompt-edu/prompt-intro-course/server/db/sqlc"
	log "github.com/sirupsen/logrus"
)

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

func parsePhaseID(c *gin.Context) (uuid.UUID, bool) {
	phaseID, err := uuid.Parse(c.Param("coursePhaseID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return uuid.Nil, false
	}
	return phaseID, true
}

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
		phaseID, ok := parsePhaseID(c)
		if !ok {
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

func createSeatPlanHandler(q *db.Queries, conn *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		phaseID, ok := parsePhaseID(c)
		if !ok {
			return
		}
		var body struct {
			Seats []string `json:"seats"`
		}
		if err := c.BindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if err := q.CreateSeatPlan(c, db.CreateSeatPlanParams{
			CoursePhaseID: phaseID,
			Seats:         body.Seats,
		}); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.Status(http.StatusCreated)
	}
}

func updateSeatPlanHandler(q *db.Queries, conn *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		phaseID, ok := parsePhaseID(c)
		if !ok {
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
		defer func() { _ = tx.Rollback(c) }()
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
				uid, err := uuid.Parse(*s.AssignedStudent)
				if err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": "invalid assignedStudent UUID"})
					return
				}
				params.AssignedStudent = pgtype.UUID{Bytes: uid, Valid: true}
			}
			if s.AssignedTutor != nil {
				uid, err := uuid.Parse(*s.AssignedTutor)
				if err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": "invalid assignedTutor UUID"})
					return
				}
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

func deleteSeatPlanHandler(q *db.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		phaseID, ok := parsePhaseID(c)
		if !ok {
			return
		}
		if err := q.DeleteSeatPlan(c, phaseID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.Status(http.StatusOK)
	}
}

type SeatAssignmentDTO struct {
	SeatName                     string  `json:"seatName"`
	HasMac                       bool    `json:"hasMac"`
	DeviceID                     *string `json:"deviceID"`
	StudentCourseParticipationID string  `json:"studentCourseParticipationID"`
	TutorFirstName               string  `json:"tutorFirstName"`
	TutorLastName                string  `json:"tutorLastName"`
	TutorEmail                   string  `json:"tutorEmail"`
}

func getOwnSeatAssignmentHandler(q *db.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		phaseID, ok := parsePhaseID(c)
		if !ok {
			return
		}
		cpID, _ := c.Get("courseParticipationID")
		participationID := cpID.(uuid.UUID)

		row, err := q.GetOwnSeatAssignment(c, db.GetOwnSeatAssignmentParams{
			CoursePhaseID:   phaseID,
			AssignedStudent: pgtype.UUID{Bytes: participationID, Valid: true},
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
	ID                  string `json:"id"`
	FirstName           string `json:"firstName"`
	LastName            string `json:"lastName"`
	Email               string `json:"email"`
	MatriculationNumber string `json:"matriculationNumber,omitempty"`
	UniversityLogin     string `json:"universityLogin,omitempty"`
	GitlabUsername      string `json:"gitlabUsername,omitempty"`
}

func getTutorsHandler(q *db.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		phaseID, ok := parsePhaseID(c)
		if !ok {
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
				ID:                  uuidToString(pgtype.UUID{Bytes: r.ID, Valid: true}),
				FirstName:           r.FirstName,
				LastName:            r.LastName,
				Email:               r.Email,
				MatriculationNumber: r.MatriculationNumber,
				UniversityLogin:     r.UniversityLogin,
				GitlabUsername:      stringOrEmpty(r.GitlabUsername),
			}
		}
		c.JSON(http.StatusOK, tutors)
	}
}

func importTutorsHandler(q *db.Queries, conn *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		phaseID, ok := parsePhaseID(c)
		if !ok {
			return
		}
		var tutors []struct {
			ID                  string `json:"id"`
			FirstName           string `json:"firstName"`
			LastName            string `json:"lastName"`
			Email               string `json:"email"`
			MatriculationNumber string `json:"matriculationNumber"`
			UniversityLogin     string `json:"universityLogin"`
		}
		if err := c.BindJSON(&tutors); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		tx, err := conn.Begin(c)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer func() { _ = tx.Rollback(c) }()
		qtx := q.WithTx(tx)
		for _, t := range tutors {
			tid, err := uuid.Parse(t.ID)
			if err != nil {
				tid = uuid.New()
			}
			if err := qtx.CreateTutor(c, db.CreateTutorParams{
				CoursePhaseID:       phaseID,
				ID:                  tid,
				FirstName:           t.FirstName,
				LastName:            t.LastName,
				Email:               t.Email,
				MatriculationNumber: t.MatriculationNumber,
				UniversityLogin:     t.UniversityLogin,
			}); err != nil {
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

func updateTutorGitlabUsernameHandler(q *db.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		phaseID, ok := parsePhaseID(c)
		if !ok {
			return
		}
		var body struct {
			TutorID        string `json:"tutorId"`
			GitlabUsername string `json:"gitlabUsername"`
		}
		if err := c.BindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		tid, err := uuid.Parse(body.TutorID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tutor ID"})
			return
		}
		if err := q.UpdateTutorGitlabUsername(c, db.UpdateTutorGitlabUsernameParams{
			ID:             tid,
			CoursePhaseID:  phaseID,
			GitlabUsername: pgtype.Text{String: body.GitlabUsername, Valid: body.GitlabUsername != ""},
		}); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.Status(http.StatusOK)
	}
}

// ── Developer Profiles ───────────────────────────────────────────────

type DeveloperProfileDTO struct {
	CourseParticipationID string  `json:"courseParticipationID"`
	GitLabUsername        string  `json:"gitLabUsername"`
	AppleID              string  `json:"appleId"`
	HasMacBook           bool    `json:"hasMacBook"`
	IPhoneUDID           *string `json:"iPhoneUDID,omitempty"`
	IPadUDID             *string `json:"iPadUDID,omitempty"`
	AppleWatchUDID       *string `json:"appleWatchUDID,omitempty"`
}

func getAllDeveloperProfilesHandler(q *db.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		phaseID, ok := parsePhaseID(c)
		if !ok {
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
				AppleID:              r.AppleID,
				HasMacBook:           r.HasMacbook,
			}
			if r.IphoneUdid.Valid {
				profiles[i].IPhoneUDID = &r.IphoneUdid.String
			}
			if r.IpadUdid.Valid {
				profiles[i].IPadUDID = &r.IpadUdid.String
			}
			if r.AppleWatchUdid.Valid {
				profiles[i].AppleWatchUDID = &r.AppleWatchUdid.String
			}
		}
		c.JSON(http.StatusOK, profiles)
	}
}

func upsertDeveloperProfileHandler(q *db.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		phaseID, ok := parsePhaseID(c)
		if !ok {
			return
		}
		var body DeveloperProfileDTO
		if err := c.BindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		cpID, err := uuid.Parse(body.CourseParticipationID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid courseParticipationID"})
			return
		}
		params := db.CreateOrUpdateDeveloperProfileParams{
			CourseParticipationID: cpID,
			CoursePhaseID:         phaseID,
			GitlabUsername:        body.GitLabUsername,
			AppleID:              body.AppleID,
			HasMacbook:           body.HasMacBook,
		}
		if body.IPhoneUDID != nil {
			params.IphoneUdid = pgtype.Text{String: *body.IPhoneUDID, Valid: true}
		}
		if body.IPadUDID != nil {
			params.IpadUdid = pgtype.Text{String: *body.IPadUDID, Valid: true}
		}
		if body.AppleWatchUDID != nil {
			params.AppleWatchUdid = pgtype.Text{String: *body.AppleWatchUDID, Valid: true}
		}
		if err := q.CreateOrUpdateDeveloperProfile(c, params); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.Status(http.StatusOK)
	}
}

// ── Devices ──────────────────────────────────────────────────────────

func getDevicesHandler(q *db.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		phaseID, ok := parsePhaseID(c)
		if !ok {
			return
		}
		rows, err := q.GetDevicesForCoursePhase(c, phaseID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, rows)
	}
}

// ── Export / Import ──────────────────────────────────────────────────

type ExportData struct {
	Seats            []SeatDTO            `json:"seats"`
	Tutors           []TutorDTO           `json:"tutors"`
	DeveloperProfiles []DeveloperProfileDTO `json:"developerProfiles"`
	PeerAssignments  []PeerAssignmentDTO  `json:"peerAssignments"`
}

type PeerAssignmentDTO struct {
	StudentID string `json:"studentID"`
	PeerID    string `json:"peerID"`
}

func exportHandler(q *db.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		phaseID, ok := parsePhaseID(c)
		if !ok {
			return
		}

		// Seats
		seatRows, err := q.GetSeatPlan(c, phaseID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		seats := make([]SeatDTO, len(seatRows))
		for i, r := range seatRows {
			seats[i] = SeatDTO{SeatName: r.SeatName, HasMac: r.HasMac, IsTutorSeat: r.IsTutorSeat}
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

		// Tutors
		tutorRows, err := q.GetAllTutors(c, phaseID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		tutors := make([]TutorDTO, len(tutorRows))
		for i, r := range tutorRows {
			tutors[i] = TutorDTO{
				ID:                  uuidToString(pgtype.UUID{Bytes: r.ID, Valid: true}),
				FirstName:           r.FirstName,
				LastName:            r.LastName,
				Email:               r.Email,
				MatriculationNumber: r.MatriculationNumber,
				UniversityLogin:     r.UniversityLogin,
				GitlabUsername:      stringOrEmpty(r.GitlabUsername),
			}
		}

		// Developer profiles
		profileRows, err := q.GetAllDeveloperProfiles(c, phaseID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		profiles := make([]DeveloperProfileDTO, len(profileRows))
		for i, r := range profileRows {
			profiles[i] = DeveloperProfileDTO{
				CourseParticipationID: uuidToString(pgtype.UUID{Bytes: r.CourseParticipationID, Valid: true}),
				GitLabUsername:        r.GitlabUsername,
				AppleID:              r.AppleID,
				HasMacBook:           r.HasMacbook,
			}
			if r.IphoneUdid.Valid {
				profiles[i].IPhoneUDID = &r.IphoneUdid.String
			}
			if r.IpadUdid.Valid {
				profiles[i].IPadUDID = &r.IpadUdid.String
			}
			if r.AppleWatchUdid.Valid {
				profiles[i].AppleWatchUDID = &r.AppleWatchUdid.String
			}
		}

		// Peer assignments
		paRows, err := q.GetPeerAssignments(c, phaseID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		peerAssignments := make([]PeerAssignmentDTO, len(paRows))
		for i, r := range paRows {
			peerAssignments[i] = PeerAssignmentDTO{
				StudentID: uuidToString(pgtype.UUID{Bytes: r.StudentID, Valid: true}),
				PeerID:    uuidToString(pgtype.UUID{Bytes: r.PeerID, Valid: true}),
			}
		}

		c.JSON(http.StatusOK, ExportData{
			Seats:             seats,
			Tutors:            tutors,
			DeveloperProfiles: profiles,
			PeerAssignments:   peerAssignments,
		})
	}
}

type ImportRequest struct {
	TargetCoursePhaseID string     `json:"targetCoursePhaseID"`
	Data                ExportData `json:"data"`
}

func importHandler(q *db.Queries, conn *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		phaseID, ok := parsePhaseID(c)
		if !ok {
			return
		}
		var req ImportRequest
		if err := c.BindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		targetPhase := phaseID
		if req.TargetCoursePhaseID != "" {
			parsed, err := uuid.Parse(req.TargetCoursePhaseID)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid targetCoursePhaseID"})
				return
			}
			targetPhase = parsed
		}

		tx, err := conn.Begin(c)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer func() { _ = tx.Rollback(c) }()
		qtx := q.WithTx(tx)

		// Import tutors
		for _, t := range req.Data.Tutors {
			tid, err := uuid.Parse(t.ID)
			if err != nil {
				tid = uuid.New()
			}
			if err := qtx.CreateTutor(c, db.CreateTutorParams{
				CoursePhaseID:       targetPhase,
				ID:                  tid,
				FirstName:           t.FirstName,
				LastName:            t.LastName,
				Email:               t.Email,
				MatriculationNumber: t.MatriculationNumber,
				UniversityLogin:     t.UniversityLogin,
			}); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "import tutor: " + err.Error()})
				return
			}
			if t.GitlabUsername != "" {
				_ = qtx.UpdateTutorGitlabUsername(c, db.UpdateTutorGitlabUsernameParams{
					ID:             tid,
					CoursePhaseID:  targetPhase,
					GitlabUsername: pgtype.Text{String: t.GitlabUsername, Valid: true},
				})
			}
		}

		// Import developer profiles
		for _, p := range req.Data.DeveloperProfiles {
			cpID, err := uuid.Parse(p.CourseParticipationID)
			if err != nil {
				continue
			}
			params := db.CreateOrUpdateDeveloperProfileParams{
				CourseParticipationID: cpID,
				CoursePhaseID:         targetPhase,
				GitlabUsername:        p.GitLabUsername,
				AppleID:              p.AppleID,
				HasMacbook:           p.HasMacBook,
			}
			if p.IPhoneUDID != nil {
				params.IphoneUdid = pgtype.Text{String: *p.IPhoneUDID, Valid: true}
			}
			if p.IPadUDID != nil {
				params.IpadUdid = pgtype.Text{String: *p.IPadUDID, Valid: true}
			}
			if p.AppleWatchUDID != nil {
				params.AppleWatchUdid = pgtype.Text{String: *p.AppleWatchUDID, Valid: true}
			}
			if err := qtx.CreateOrUpdateDeveloperProfile(c, params); err != nil {
				log.Warnf("import dev profile %s: %v", p.CourseParticipationID, err)
			}
		}

		// Import peer assignments
		for _, pa := range req.Data.PeerAssignments {
			sid, err := uuid.Parse(pa.StudentID)
			if err != nil {
				continue
			}
			pid, err := uuid.Parse(pa.PeerID)
			if err != nil {
				continue
			}
			if err := qtx.CreatePeerAssignment(c, db.CreatePeerAssignmentParams{
				CoursePhaseID: targetPhase,
				StudentID:    sid,
				PeerID:       pid,
			}); err != nil {
				log.Warnf("import peer assignment: %v", err)
			}
		}

		if err := tx.Commit(c); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"imported": true})
	}
}

// ── Mock Participations ──────────────────────────────────────────────

type StudentDTO struct {
	FirstName           string `json:"firstName"`
	LastName            string `json:"lastName"`
	Email               string `json:"email"`
	MatriculationNumber string `json:"matriculationNumber"`
	UniversityLogin     string `json:"universityLogin"`
	Gender              string `json:"gender"`
}

type ParticipationDTO struct {
	CourseParticipationID string     `json:"courseParticipationID"`
	Student               StudentDTO `json:"student"`
	PassStatus            string     `json:"passStatus"`
}

func mockParticipationsHandler(q *db.Queries, sl *StudentLoader) gin.HandlerFunc {
	return func(c *gin.Context) {
		result := make([]ParticipationDTO, 0, len(sl.Students))
		for _, s := range sl.Students {
			result = append(result, ParticipationDTO{
				CourseParticipationID: s.CourseParticipationID,
				Student: StudentDTO{
					FirstName:           s.FirstName,
					LastName:            s.LastName,
					Email:               s.Email,
					MatriculationNumber: s.MatriculationNumber,
					UniversityLogin:     s.UniversityLogin,
					Gender:              s.Gender,
				},
				PassStatus: s.PassStatus,
			})
		}
		c.JSON(http.StatusOK, result)
	}
}

func mockStudentsHandler(sl *StudentLoader) gin.HandlerFunc {
	return func(c *gin.Context) {
		result := make([]json.RawMessage, 0, len(sl.Students))
		for _, s := range sl.Students {
			b, _ := json.Marshal(map[string]interface{}{
				"courseParticipationID": s.CourseParticipationID,
				"firstName":            s.FirstName,
				"lastName":             s.LastName,
				"email":                s.Email,
				"matriculationNumber":  s.MatriculationNumber,
				"universityLogin":      s.UniversityLogin,
				"gender":               s.Gender,
			})
			result = append(result, b)
		}
		c.JSON(http.StatusOK, result)
	}
}

func mockSelfHandler(sl *StudentLoader) gin.HandlerFunc {
	return func(c *gin.Context) {
		if len(sl.Students) > 0 {
			s := sl.Students[0]
			c.JSON(http.StatusOK, gin.H{
				"courseParticipationID": s.CourseParticipationID,
				"firstName":            s.FirstName,
				"lastName":             s.LastName,
				"email":                s.Email,
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"courseParticipationID": "b0000000-0000-0000-0000-000000000001",
			"firstName":            "Admin",
			"lastName":             "User",
			"email":                "admin@example.com",
		})
	}
}
