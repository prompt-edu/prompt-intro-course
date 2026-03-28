package peerAssignmentDTO

import (
	"github.com/google/uuid"
	db "github.com/prompt-edu/prompt-intro-course/server/db/sqlc"
)

// PeerAssignment is the admin-facing DTO for a single directed assignment.
type PeerAssignment struct {
	StudentID uuid.UUID `json:"studentID"`
	PeerID    uuid.UUID `json:"peerID"`
}

// PeerInfo is the enriched peer info used in student-facing responses.
type PeerInfo struct {
	CourseParticipationID uuid.UUID `json:"courseParticipationID"`
	GitlabUsername        string    `json:"gitlabUsername"`
	SeatName              string    `json:"seatName"`
	TutorGitlabUsername   string    `json:"tutorGitlabUsername"`
}

// OwnPeerAssignment is the student-facing DTO combining both directions.
type OwnPeerAssignment struct {
	PeersIReview     []PeerInfo `json:"peersIReview"`
	PeersWhoReviewMe []PeerInfo `json:"peersWhoReviewMe"`
}

// SyncRequest carries the semester tag needed to locate GitLab projects.
type SyncRequest struct {
	SemesterTag string `json:"semesterTag" binding:"required"`
}

// SyncResult reports per-pair sync outcome.
type SyncResult struct {
	StudentID uuid.UUID `json:"studentID"`
	PeerID    uuid.UUID `json:"peerID"`
	Success   bool      `json:"success"`
	Error     string    `json:"error,omitempty"`
}

func GetPeerAssignmentDTOsFromDBModels(assignments []db.PeerAssignment) []PeerAssignment {
	dtos := make([]PeerAssignment, 0, len(assignments))
	for _, a := range assignments {
		dtos = append(dtos, PeerAssignment{
			StudentID: a.StudentID,
			PeerID:    a.PeerID,
		})
	}
	return dtos
}

func GetPeerInfoFromPeersForStudentRow(row db.GetPeersForStudentRow) PeerInfo {
	return PeerInfo{
		CourseParticipationID: row.PeerID,
		GitlabUsername:        row.GitlabUsername.String,
		SeatName:              row.SeatName.String,
		TutorGitlabUsername:   row.TutorGitlabUsername.String,
	}
}

func GetPeerInfoFromReviewersForStudentRow(row db.GetReviewersForStudentRow) PeerInfo {
	return PeerInfo{
		CourseParticipationID: row.StudentID,
		GitlabUsername:        row.GitlabUsername.String,
		SeatName:              row.SeatName.String,
		TutorGitlabUsername:   row.TutorGitlabUsername.String,
	}
}

func GetPeerInfosFromPeersRows(rows []db.GetPeersForStudentRow) []PeerInfo {
	infos := make([]PeerInfo, 0, len(rows))
	for _, r := range rows {
		infos = append(infos, GetPeerInfoFromPeersForStudentRow(r))
	}
	return infos
}

func GetPeerInfosFromReviewersRows(rows []db.GetReviewersForStudentRow) []PeerInfo {
	infos := make([]PeerInfo, 0, len(rows))
	for _, r := range rows {
		infos = append(infos, GetPeerInfoFromReviewersForStudentRow(r))
	}
	return infos
}
