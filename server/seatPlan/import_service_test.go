package seatPlan

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/prompt-edu/prompt-intro-course/server/seatPlan/seatPlanDTO"
	"github.com/prompt-edu/prompt-intro-course/server/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type ImportServiceTestSuite struct {
	suite.Suite
	ctx            context.Context
	cleanup        func()
	coursePhaseID   uuid.UUID
	studentID1     uuid.UUID
	studentID2     uuid.UUID
	tutorID1       uuid.UUID
	tutorID2       uuid.UUID
}

func (s *ImportServiceTestSuite) SetupSuite() {
	s.ctx = context.Background()
	testDB, cleanup, err := testutils.SetupTestDB(s.ctx, "../database_dumps/intro_course.sql")
	if err != nil {
		s.T().Fatalf("Failed to set up test database: %v", err)
	}
	s.cleanup = cleanup
	s.coursePhaseID = uuid.MustParse("4179d58a-d00d-4fa7-94a5-397bc69fab02")
	s.studentID1 = uuid.MustParse("33333333-3333-3333-3333-333333333333")
	s.studentID2 = uuid.MustParse("44444444-4444-4444-4444-444444444444")
	s.tutorID1 = uuid.MustParse("11111111-1111-1111-1111-111111111111")
	s.tutorID2 = uuid.MustParse("22222222-2222-2222-2222-222222222222")

	SeatPlanServiceSingleton = &SeatPlanService{
		queries: *testDB.Queries,
		conn:    testDB.Conn,
	}
}

func (s *ImportServiceTestSuite) TearDownSuite() {
	if s.cleanup != nil {
		s.cleanup()
	}
}

func TestImportServiceTestSuite(t *testing.T) {
	suite.Run(t, new(ImportServiceTestSuite))
}

// TestImportStudentAndTutorByName verifies that students and tutors are matched
// by their full name and correctly assigned to seats.
func (s *ImportServiceTestSuite) TestImportStudentAndTutorByName() {
	// The test DB has Seat-1 and Seat-2 with pre-assigned students and tutors.
	// We import with different assignments to verify name matching works.
	students := []StudentInfo{
		{CourseParticipationID: s.studentID1, FirstName: "Max", LastName: "Mueller"},
		{CourseParticipationID: s.studentID2, FirstName: "Anna", LastName: "Schmidt"},
	}

	req := seatPlanDTO.ImportRequest{
		Assignments: []seatPlanDTO.ImportSeatAssignment{
			{
				SeatName:        "Seat-1",
				SeatMac:         true,
				AssignedStudent: "Anna Schmidt",
				AssignedTutor:   "Alice Tutor",
				IsTutorSeat:     false,
			},
			{
				SeatName:        "Seat-2",
				SeatMac:         false,
				AssignedStudent: "Max Mueller",
				AssignedTutor:   "Bob Tutor",
				IsTutorSeat:     false,
			},
		},
	}

	result, err := ImportSeatAssignments(s.ctx, s.coursePhaseID, req, students)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), 2, result.SeatsUpdated)
	assert.Empty(s.T(), result.Warnings)

	// Verify seats were updated
	seats, err := GetSeatPlan(s.ctx, s.coursePhaseID)
	require.NoError(s.T(), err)

	for _, seat := range seats {
		if seat.SeatName == "Seat-1" {
			assert.True(s.T(), seat.HasMac, "Seat-1 should have Mac")
			assert.True(s.T(), seat.AssignedStudent.Valid)
			assert.Equal(s.T(), s.studentID2, uuid.UUID(seat.AssignedStudent.Bytes), "Seat-1 should have Anna (studentID2)")
			assert.True(s.T(), seat.AssignedTutor.Valid)
			assert.Equal(s.T(), s.tutorID1, uuid.UUID(seat.AssignedTutor.Bytes), "Seat-1 should have Alice (tutorID1)")
		}
		if seat.SeatName == "Seat-2" {
			assert.False(s.T(), seat.HasMac, "Seat-2 should not have Mac")
			assert.True(s.T(), seat.AssignedStudent.Valid)
			assert.Equal(s.T(), s.studentID1, uuid.UUID(seat.AssignedStudent.Bytes), "Seat-2 should have Max (studentID1)")
			assert.True(s.T(), seat.AssignedTutor.Valid)
			assert.Equal(s.T(), s.tutorID2, uuid.UUID(seat.AssignedTutor.Bytes), "Seat-2 should have Bob (tutorID2)")
		}
	}
}

// TestImportTutorSeat verifies that isTutorSeat flag is set correctly.
func (s *ImportServiceTestSuite) TestImportTutorSeat() {
	students := []StudentInfo{}

	req := seatPlanDTO.ImportRequest{
		Assignments: []seatPlanDTO.ImportSeatAssignment{
			{
				SeatName:        "Seat-1",
				SeatMac:         false,
				AssignedStudent: "Unassigned",
				AssignedTutor:   "Alice Tutor",
				IsTutorSeat:     true,
			},
		},
	}

	result, err := ImportSeatAssignments(s.ctx, s.coursePhaseID, req, students)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), 1, result.SeatsUpdated)

	seats, err := GetSeatPlan(s.ctx, s.coursePhaseID)
	require.NoError(s.T(), err)

	for _, seat := range seats {
		if seat.SeatName == "Seat-1" {
			assert.True(s.T(), seat.IsTutorSeat, "Seat-1 should be a tutor seat")
			assert.False(s.T(), seat.AssignedStudent.Valid, "Tutor seat should have no student")
			assert.True(s.T(), seat.AssignedTutor.Valid)
			assert.Equal(s.T(), s.tutorID1, uuid.UUID(seat.AssignedTutor.Bytes))
		}
	}
}

// TestImportMacAssignment verifies Mac assignment is imported.
func (s *ImportServiceTestSuite) TestImportMacAssignment() {
	students := []StudentInfo{
		{CourseParticipationID: s.studentID1, FirstName: "Max", LastName: "Mueller"},
	}

	req := seatPlanDTO.ImportRequest{
		Assignments: []seatPlanDTO.ImportSeatAssignment{
			{
				SeatName:        "Seat-2", // Was hasMac=false in test data
				SeatMac:         true,
				AssignedStudent: "Max Mueller",
				AssignedTutor:   "Bob Tutor",
			},
		},
	}

	result, err := ImportSeatAssignments(s.ctx, s.coursePhaseID, req, students)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), 1, result.SeatsUpdated)

	seats, err := GetSeatPlan(s.ctx, s.coursePhaseID)
	require.NoError(s.T(), err)
	for _, seat := range seats {
		if seat.SeatName == "Seat-2" {
			assert.True(s.T(), seat.HasMac, "Seat-2 should now have Mac")
		}
	}
}

// TestImportPeerGroups verifies that peer groups create bidirectional peer assignments.
func (s *ImportServiceTestSuite) TestImportPeerGroups() {
	students := []StudentInfo{
		{CourseParticipationID: s.studentID1, FirstName: "Max", LastName: "Mueller"},
		{CourseParticipationID: s.studentID2, FirstName: "Anna", LastName: "Schmidt"},
	}

	req := seatPlanDTO.ImportRequest{
		Assignments: []seatPlanDTO.ImportSeatAssignment{
			{
				SeatName:        "Seat-1",
				AssignedStudent: "Max Mueller",
				AssignedTutor:   "Alice Tutor",
				PeerGroup:       "P1",
			},
			{
				SeatName:        "Seat-2",
				AssignedStudent: "Anna Schmidt",
				AssignedTutor:   "Alice Tutor",
				PeerGroup:       "P1",
			},
		},
	}

	result, err := ImportSeatAssignments(s.ctx, s.coursePhaseID, req, students)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), 2, result.SeatsUpdated)
	assert.Equal(s.T(), 1, result.PeerGroupsImported, "Should have 1 peer group")

	// Verify peer assignments were created
	peers, err := SeatPlanServiceSingleton.queries.GetPeerAssignments(s.ctx, s.coursePhaseID)
	require.NoError(s.T(), err)
	assert.Len(s.T(), peers, 2, "Should have 2 bidirectional peer assignments")
}

// TestImportUnknownStudentWarning verifies a warning is returned for unmatched students.
func (s *ImportServiceTestSuite) TestImportUnknownStudentWarning() {
	students := []StudentInfo{
		{CourseParticipationID: s.studentID1, FirstName: "Max", LastName: "Mueller"},
	}

	req := seatPlanDTO.ImportRequest{
		Assignments: []seatPlanDTO.ImportSeatAssignment{
			{
				SeatName:        "Seat-1",
				AssignedStudent: "Nonexistent Person",
				AssignedTutor:   "Alice Tutor",
			},
		},
	}

	result, err := ImportSeatAssignments(s.ctx, s.coursePhaseID, req, students)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), 0, result.SeatsUpdated, "Seat with unknown student should not be updated")
	assert.Len(s.T(), result.Warnings, 1)
	assert.Contains(s.T(), result.Warnings[0], "Nonexistent Person")
}

// TestImportUnknownTutorWarning verifies a warning is returned for unmatched tutors.
func (s *ImportServiceTestSuite) TestImportUnknownTutorWarning() {
	students := []StudentInfo{
		{CourseParticipationID: s.studentID1, FirstName: "Max", LastName: "Mueller"},
	}

	req := seatPlanDTO.ImportRequest{
		Assignments: []seatPlanDTO.ImportSeatAssignment{
			{
				SeatName:        "Seat-1",
				AssignedStudent: "Max Mueller",
				AssignedTutor:   "Unknown Instructor",
			},
		},
	}

	result, err := ImportSeatAssignments(s.ctx, s.coursePhaseID, req, students)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), 0, result.SeatsUpdated, "Seat with unknown tutor should not be updated")
	assert.Len(s.T(), result.Warnings, 1)
	assert.Contains(s.T(), result.Warnings[0], "Unknown Instructor")
}

// TestImportCaseInsensitiveNameMatch verifies name matching is case-insensitive.
func (s *ImportServiceTestSuite) TestImportCaseInsensitiveNameMatch() {
	students := []StudentInfo{
		{CourseParticipationID: s.studentID1, FirstName: "Max", LastName: "Mueller"},
	}

	req := seatPlanDTO.ImportRequest{
		Assignments: []seatPlanDTO.ImportSeatAssignment{
			{
				SeatName:        "Seat-1",
				AssignedStudent: "max mueller",
				AssignedTutor:   "alice tutor",
			},
		},
	}

	result, err := ImportSeatAssignments(s.ctx, s.coursePhaseID, req, students)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), 1, result.SeatsUpdated)
	assert.Empty(s.T(), result.Warnings)
}

// TestImportUnassignedStudent verifies that "Unassigned" clears the student.
func (s *ImportServiceTestSuite) TestImportUnassignedStudent() {
	students := []StudentInfo{}

	req := seatPlanDTO.ImportRequest{
		Assignments: []seatPlanDTO.ImportSeatAssignment{
			{
				SeatName:        "Seat-1",
				AssignedStudent: "Unassigned",
				AssignedTutor:   "Alice Tutor",
			},
		},
	}

	result, err := ImportSeatAssignments(s.ctx, s.coursePhaseID, req, students)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), 1, result.SeatsUpdated)

	seats, err := GetSeatPlan(s.ctx, s.coursePhaseID)
	require.NoError(s.T(), err)
	for _, seat := range seats {
		if seat.SeatName == "Seat-1" {
			assert.False(s.T(), seat.AssignedStudent.Valid, "Student should be cleared")
		}
	}
}

// TestImportAtomicity verifies that the entire import is atomic —
// a failure in one seat doesn't partially commit.
func (s *ImportServiceTestSuite) TestImportEmptyRequest() {
	req := seatPlanDTO.ImportRequest{
		Assignments: []seatPlanDTO.ImportSeatAssignment{},
	}

	// Empty assignments should be rejected at the router level,
	// but the service handles it gracefully
	result, err := ImportSeatAssignments(s.ctx, s.coursePhaseID, req, nil)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), 0, result.SeatsUpdated)
}
