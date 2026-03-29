package peerAssignment

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	db "github.com/prompt-edu/prompt-intro-course/server/db/sqlc"
	"github.com/prompt-edu/prompt-intro-course/server/peerAssignment/peerAssignmentDTO"
	"github.com/prompt-edu/prompt-intro-course/server/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type PeerAssignmentServiceTestSuite struct {
	suite.Suite
	ctx          context.Context
	cleanup      func()
	service      PeerAssignmentService
	coursePhaseID uuid.UUID
	studentID1   uuid.UUID
	studentID2   uuid.UUID
}

func (suite *PeerAssignmentServiceTestSuite) SetupSuite() {
	suite.ctx = context.Background()
	testDB, cleanup, err := testutils.SetupTestDB(suite.ctx, "../database_dumps/intro_course.sql")
	if err != nil {
		suite.T().Fatalf("Failed to set up test database: %v", err)
	}

	suite.cleanup = cleanup
	suite.coursePhaseID = uuid.MustParse("4179d58a-d00d-4fa7-94a5-397bc69fab02")
	suite.studentID1 = uuid.MustParse("33333333-3333-3333-3333-333333333333")
	suite.studentID2 = uuid.MustParse("44444444-4444-4444-4444-444444444444")
	suite.service = PeerAssignmentService{
		queries: *testDB.Queries,
		conn:    testDB.Conn,
	}
	PeerAssignmentServiceSingleton = &suite.service
}

func (suite *PeerAssignmentServiceTestSuite) TearDownSuite() {
	if suite.cleanup != nil {
		suite.cleanup()
	}
}

func TestPeerAssignmentServiceTestSuite(t *testing.T) {
	suite.Run(t, new(PeerAssignmentServiceTestSuite))
}

// --- Basic CRUD Tests ---

func (suite *PeerAssignmentServiceTestSuite) TestGetAllPeerAssignmentsEmpty() {
	// Use a phase with no assignments
	assignments, err := GetAllPeerAssignments(suite.ctx, uuid.New())
	assert.NoError(suite.T(), err)
	assert.Empty(suite.T(), assignments)
}

func (suite *PeerAssignmentServiceTestSuite) TestGeneratePeerAssignments() {
	// Use existing test data: 2 students across 2 tutors (1 student each)
	// Each tutor group has only 1 student, so no pairs can be formed
	assignments, err := GeneratePeerAssignments(suite.ctx, suite.coursePhaseID)
	assert.NoError(suite.T(), err)
	// Each tutor has 1 student → cannot form pairs → empty result
	assert.Empty(suite.T(), assignments)
}

func (suite *PeerAssignmentServiceTestSuite) TestUpdatePeerAssignments() {
	phaseID := uuid.New()
	// Set up some seats so we have a valid context
	_, err := suite.service.conn.Exec(suite.ctx,
		`INSERT INTO seat (course_phase_id, seat_name) VALUES ($1, 'Test-1'), ($1, 'Test-2')`,
		phaseID)
	assert.NoError(suite.T(), err)

	assignments := []peerAssignmentDTO.PeerAssignment{
		{StudentID: suite.studentID1, PeerID: suite.studentID2},
		{StudentID: suite.studentID2, PeerID: suite.studentID1},
	}

	err = UpdatePeerAssignments(suite.ctx, phaseID, assignments)
	assert.NoError(suite.T(), err)

	// Verify they were stored
	result, err := GetAllPeerAssignments(suite.ctx, phaseID)
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), result, 2)
}

func (suite *PeerAssignmentServiceTestSuite) TestUpdatePeerAssignmentsRejectsSelfReview() {
	phaseID := uuid.New()

	assignments := []peerAssignmentDTO.PeerAssignment{
		{StudentID: suite.studentID1, PeerID: suite.studentID1}, // self-review
		{StudentID: suite.studentID1, PeerID: suite.studentID2},
	}

	err := UpdatePeerAssignments(suite.ctx, phaseID, assignments)
	assert.Error(suite.T(), err)
	assert.Equal(suite.T(), "self-review assignment not allowed", err.Error())
}

func (suite *PeerAssignmentServiceTestSuite) TestDeletePeerAssignments() {
	phaseID := uuid.New()

	// Insert some assignments
	assignments := []peerAssignmentDTO.PeerAssignment{
		{StudentID: suite.studentID1, PeerID: suite.studentID2},
		{StudentID: suite.studentID2, PeerID: suite.studentID1},
	}
	err := UpdatePeerAssignments(suite.ctx, phaseID, assignments)
	assert.NoError(suite.T(), err)

	// Delete them
	err = DeletePeerAssignments(suite.ctx, phaseID)
	assert.NoError(suite.T(), err)

	// Verify empty
	result, err := GetAllPeerAssignments(suite.ctx, phaseID)
	assert.NoError(suite.T(), err)
	assert.Empty(suite.T(), result)
}

func (suite *PeerAssignmentServiceTestSuite) TestGetOwnPeerAssignment() {
	phaseID := uuid.New()

	// Insert bidirectional assignment + dev profiles for join
	_, err := suite.service.conn.Exec(suite.ctx,
		`INSERT INTO developer_profile (course_phase_id, course_participation_id, gitlab_username, apple_id, has_macbook)
		 VALUES ($1, $2, 'peer1git', 'peer1@apple.com', true),
		        ($1, $3, 'peer2git', 'peer2@apple.com', true)`,
		phaseID, suite.studentID1, suite.studentID2)
	assert.NoError(suite.T(), err)

	assignments := []peerAssignmentDTO.PeerAssignment{
		{StudentID: suite.studentID1, PeerID: suite.studentID2},
		{StudentID: suite.studentID2, PeerID: suite.studentID1},
	}
	err = UpdatePeerAssignments(suite.ctx, phaseID, assignments)
	assert.NoError(suite.T(), err)

	// Get own peer assignment for student1
	own, err := GetOwnPeerAssignment(suite.ctx, phaseID, suite.studentID1)
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), own.PeersIReview, 1)
	assert.Len(suite.T(), own.PeersWhoReviewMe, 1)
	assert.Equal(suite.T(), "peer2git", own.PeersIReview[0].GitlabUsername)
	assert.Equal(suite.T(), "peer2git", own.PeersWhoReviewMe[0].GitlabUsername)
}

func (suite *PeerAssignmentServiceTestSuite) TestGetOwnPeerAssignmentNoPeers() {
	own, err := GetOwnPeerAssignment(suite.ctx, uuid.New(), uuid.New())
	assert.NoError(suite.T(), err)
	assert.Empty(suite.T(), own.PeersIReview)
	assert.Empty(suite.T(), own.PeersWhoReviewMe)
}

// --- createPeerGroups unit tests (groups of 3-4) ---

func (suite *PeerAssignmentServiceTestSuite) TestCreatePeerGroupsSingleStudent() {
	groups := createPeerGroups([]uuid.UUID{uuid.New()})
	assert.Empty(suite.T(), groups)
}

func (suite *PeerAssignmentServiceTestSuite) TestCreatePeerGroupsTwoStudents() {
	groups := createPeerGroups([]uuid.UUID{uuid.New(), uuid.New()})
	assert.Len(suite.T(), groups, 1)
	assert.Len(suite.T(), groups[0], 2) // best-effort pair
}

func (suite *PeerAssignmentServiceTestSuite) TestCreatePeerGroupsTriple() {
	students := make([]uuid.UUID, 3)
	for i := range students { students[i] = uuid.New() }
	groups := createPeerGroups(students)
	assert.Len(suite.T(), groups, 1)
	assert.Len(suite.T(), groups[0], 3)
}

func (suite *PeerAssignmentServiceTestSuite) TestCreatePeerGroupsQuad() {
	// 4%3=1 → 1 quad
	students := make([]uuid.UUID, 4)
	for i := range students { students[i] = uuid.New() }
	groups := createPeerGroups(students)
	assert.Len(suite.T(), groups, 1)
	assert.Len(suite.T(), groups[0], 4)
}

func (suite *PeerAssignmentServiceTestSuite) TestCreatePeerGroupsFive() {
	// 5%3=2, n<8 → 1 triple + 1 pair (best effort)
	students := make([]uuid.UUID, 5)
	for i := range students { students[i] = uuid.New() }
	groups := createPeerGroups(students)
	assert.Len(suite.T(), groups, 2)
	assert.Len(suite.T(), groups[0], 3)
	assert.Len(suite.T(), groups[1], 2)
}

func (suite *PeerAssignmentServiceTestSuite) TestCreatePeerGroupsSix() {
	// 6%3=0 → 2 triples
	students := make([]uuid.UUID, 6)
	for i := range students { students[i] = uuid.New() }
	groups := createPeerGroups(students)
	assert.Len(suite.T(), groups, 2)
	for _, g := range groups { assert.Len(suite.T(), g, 3) }
}

func (suite *PeerAssignmentServiceTestSuite) TestCreatePeerGroupsSeven() {
	// 7%3=1 → 1 triple + 1 quad
	students := make([]uuid.UUID, 7)
	for i := range students { students[i] = uuid.New() }
	groups := createPeerGroups(students)
	assert.Len(suite.T(), groups, 2)
	assert.Len(suite.T(), groups[0], 3)
	assert.Len(suite.T(), groups[1], 4)
}

func (suite *PeerAssignmentServiceTestSuite) TestCreatePeerGroupsEight() {
	// 8%3=2, n>=8 → 2 quads
	students := make([]uuid.UUID, 8)
	for i := range students { students[i] = uuid.New() }
	groups := createPeerGroups(students)
	assert.Len(suite.T(), groups, 2)
	for _, g := range groups { assert.Len(suite.T(), g, 4) }
}

func (suite *PeerAssignmentServiceTestSuite) TestCreatePeerGroupsNine() {
	// 9%3=0 → 3 triples
	students := make([]uuid.UUID, 9)
	for i := range students { students[i] = uuid.New() }
	groups := createPeerGroups(students)
	assert.Len(suite.T(), groups, 3)
	for _, g := range groups { assert.Len(suite.T(), g, 3) }
}

func (suite *PeerAssignmentServiceTestSuite) TestCreatePeerGroupsTen() {
	// 10%3=1 → 2 triples + 1 quad
	students := make([]uuid.UUID, 10)
	for i := range students { students[i] = uuid.New() }
	groups := createPeerGroups(students)
	assert.Len(suite.T(), groups, 3)
	assert.Len(suite.T(), groups[0], 3)
	assert.Len(suite.T(), groups[1], 3)
	assert.Len(suite.T(), groups[2], 4)
}

func (suite *PeerAssignmentServiceTestSuite) TestCreatePeerGroupsEleven() {
	// 11%3=2, n>=8 → 1 triple + 2 quads
	students := make([]uuid.UUID, 11)
	for i := range students { students[i] = uuid.New() }
	groups := createPeerGroups(students)
	assert.Len(suite.T(), groups, 3)
	assert.Len(suite.T(), groups[0], 3)
	assert.Len(suite.T(), groups[1], 4)
	assert.Len(suite.T(), groups[2], 4)
}

// --- Large-scale integration test: 56 students, 6 tutors ---

func (suite *PeerAssignmentServiceTestSuite) TestGeneratePeerAssignments56Students6Tutors() {
	phaseID := uuid.New()
	numTutors := 6
	numStudents := 56

	// Create 6 tutors
	tutorIDs := make([]uuid.UUID, numTutors)
	for i := 0; i < numTutors; i++ {
		tutorIDs[i] = uuid.New()
		_, err := suite.service.conn.Exec(suite.ctx,
			`INSERT INTO tutor (course_phase_id, id, first_name, last_name, email, matriculation_number, university_login, gitlab_username)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
			phaseID, tutorIDs[i],
			fmt.Sprintf("Tutor%d", i+1), fmt.Sprintf("Last%d", i+1),
			fmt.Sprintf("tutor%d@example.com", i+1),
			fmt.Sprintf("T%06d", i+1), fmt.Sprintf("tutor%d", i+1),
			fmt.Sprintf("tutor%dgit", i+1))
		suite.Require().NoError(err)
	}

	// Create 56 students distributed across tutors via seats
	// Distribution: tutors 0-1 get 10 students, tutors 2-5 get 9 students (56 = 2*10 + 4*9)
	studentIDs := make([]uuid.UUID, numStudents)
	tutorStudentCounts := make([]int, numTutors)
	for i := 0; i < numStudents; i++ {
		studentIDs[i] = uuid.New()
		tutorIdx := i % numTutors
		tutorStudentCounts[tutorIdx]++
		seatName := fmt.Sprintf("S-%d-%d", tutorIdx+1, tutorStudentCounts[tutorIdx])

		// Insert developer profile
		_, err := suite.service.conn.Exec(suite.ctx,
			`INSERT INTO developer_profile (course_phase_id, course_participation_id, gitlab_username, apple_id, has_macbook)
			 VALUES ($1, $2, $3, $4, $5)`,
			phaseID, studentIDs[i],
			fmt.Sprintf("student%dgit", i+1),
			fmt.Sprintf("student%d@apple.com", i+1),
			i%3 == 0) // every 3rd student has a MacBook
		suite.Require().NoError(err)

		// Insert seat assigned to tutor and student
		_, err = suite.service.conn.Exec(suite.ctx,
			`INSERT INTO seat (course_phase_id, seat_name, has_mac, assigned_student, assigned_tutor)
			 VALUES ($1, $2, $3, $4, $5)`,
			phaseID, seatName, i%4 == 0, // every 4th seat has a Mac
			pgtype.UUID{Bytes: studentIDs[i], Valid: true},
			pgtype.UUID{Bytes: tutorIDs[tutorIdx], Valid: true})
		suite.Require().NoError(err)
	}

	// Generate peer assignments
	assignments, err := GeneratePeerAssignments(suite.ctx, phaseID)
	suite.Require().NoError(err)

	// Verify: all assignments are bidirectional
	assignmentSet := make(map[[2]uuid.UUID]bool)
	for _, a := range assignments {
		assignmentSet[[2]uuid.UUID{a.StudentID, a.PeerID}] = true
	}
	for _, a := range assignments {
		assert.True(suite.T(), assignmentSet[[2]uuid.UUID{a.PeerID, a.StudentID}],
			"Missing reverse assignment for %s <-> %s", a.StudentID, a.PeerID)
	}

	// Verify: no self-review
	for _, a := range assignments {
		assert.NotEqual(suite.T(), a.StudentID, a.PeerID, "Self-review found")
	}

	// Verify: every assigned student has at least one peer
	studentsWithPeers := make(map[uuid.UUID]bool)
	for _, a := range assignments {
		studentsWithPeers[a.StudentID] = true
	}
	// Count students actually assigned to a tutor
	assignedStudents := 0
	for _, count := range tutorStudentCounts {
		if count >= 2 {
			assignedStudents += count
		}
	}
	assert.Equal(suite.T(), assignedStudents, len(studentsWithPeers),
		"Not all assigned students have peers")

	// Verify: pairs are within same tutor group
	// Build student->tutor mapping
	studentTutor := make(map[uuid.UUID]uuid.UUID)
	for i := 0; i < numStudents; i++ {
		tutorIdx := i % numTutors
		studentTutor[studentIDs[i]] = tutorIDs[tutorIdx]
	}
	for _, a := range assignments {
		assert.Equal(suite.T(), studentTutor[a.StudentID], studentTutor[a.PeerID],
			"Cross-tutor peer assignment: student %s (tutor %s) paired with peer %s (tutor %s)",
			a.StudentID, studentTutor[a.StudentID], a.PeerID, studentTutor[a.PeerID])
	}

	// Verify: each student has 2-3 peers (groups of 3 or 4)
	peerCount := make(map[uuid.UUID]int)
	for _, a := range assignments {
		peerCount[a.StudentID]++
	}
	for sid, count := range peerCount {
		assert.True(suite.T(), count >= 2 && count <= 3,
			"Student %s has %d peers (expected 2-3)", sid, count)
	}

	// Verify: assignment count is correct
	// Groups of 3 → 6 edges, groups of 4 → 12 edges
	expectedCount := 0
	for _, count := range tutorStudentCounts {
		if count < 2 {
			continue
		}
		numQuads := 0
		switch count % 3 {
		case 1:
			numQuads = 1
		case 2:
			if count >= 8 {
				numQuads = 2
			}
		}
		numTriples := (count - 4*numQuads) / 3
		remaining := count - 3*numTriples - 4*numQuads
		expectedCount += numTriples * 6
		expectedCount += numQuads * 12
		expectedCount += remaining * (remaining - 1) // best-effort pair or single
	}
	assert.Equal(suite.T(), expectedCount, len(assignments),
		"Unexpected assignment count")

	// Verify: can retrieve via GetAllPeerAssignments
	all, err := GetAllPeerAssignments(suite.ctx, phaseID)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), len(assignments), len(all))
}

// Test that re-generating clears old assignments
func (suite *PeerAssignmentServiceTestSuite) TestGenerateReplacesExisting() {
	phaseID := uuid.New()
	tutorID := uuid.New()

	// Set up 4 students under 1 tutor
	_, err := suite.service.conn.Exec(suite.ctx,
		`INSERT INTO tutor (course_phase_id, id, first_name, last_name, email, matriculation_number, university_login)
		 VALUES ($1, $2, 'TestTutor', 'Replace', 'tr@example.com', 'T999', 'tr')`,
		phaseID, tutorID)
	suite.Require().NoError(err)

	studentIDs := make([]uuid.UUID, 4)
	for i := 0; i < 4; i++ {
		studentIDs[i] = uuid.New()
		_, err := suite.service.conn.Exec(suite.ctx,
			`INSERT INTO seat (course_phase_id, seat_name, assigned_student, assigned_tutor)
			 VALUES ($1, $2, $3, $4)`,
			phaseID, fmt.Sprintf("R-%d", i),
			pgtype.UUID{Bytes: studentIDs[i], Valid: true},
			pgtype.UUID{Bytes: tutorID, Valid: true})
		suite.Require().NoError(err)
	}

	// Generate first time
	first, err := GeneratePeerAssignments(suite.ctx, phaseID)
	assert.NoError(suite.T(), err)
	assert.NotEmpty(suite.T(), first)

	// Generate second time (should replace, not duplicate)
	second, err := GeneratePeerAssignments(suite.ctx, phaseID)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), len(first), len(second),
		"Re-generation should produce same count (no duplicates)")

	// Verify DB count matches
	all, err := GetAllPeerAssignments(suite.ctx, phaseID)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), len(second), len(all))
}

// Test with exactly 4 students (quad) — each student reviews 3 peers
func (suite *PeerAssignmentServiceTestSuite) TestGenerateQuad() {
	phaseID := uuid.New()
	tutorID := uuid.New()

	_, err := suite.service.conn.Exec(suite.ctx,
		`INSERT INTO tutor (course_phase_id, id, first_name, last_name, email, matriculation_number, university_login)
		 VALUES ($1, $2, 'QuadTutor', 'Test', 'qt@example.com', 'T666', 'qt')`,
		phaseID, tutorID)
	suite.Require().NoError(err)

	studentIDs := make([]uuid.UUID, 4)
	for i := 0; i < 4; i++ {
		studentIDs[i] = uuid.New()
		_, err := suite.service.conn.Exec(suite.ctx,
			`INSERT INTO seat (course_phase_id, seat_name, assigned_student, assigned_tutor)
			 VALUES ($1, $2, $3, $4)`,
			phaseID, fmt.Sprintf("Q-%d", i),
			pgtype.UUID{Bytes: studentIDs[i], Valid: true},
			pgtype.UUID{Bytes: tutorID, Valid: true})
		suite.Require().NoError(err)
	}

	assignments, err := GeneratePeerAssignments(suite.ctx, phaseID)
	assert.NoError(suite.T(), err)
	// Quad: 4 students × 3 peers = 12 directed edges
	assert.Len(suite.T(), assignments, 12)

	reviewCount := make(map[uuid.UUID]int)
	for _, a := range assignments {
		reviewCount[a.StudentID]++
	}
	for _, id := range studentIDs {
		assert.Equal(suite.T(), 3, reviewCount[id],
			"Student in quad should review exactly 3 peers")
	}
}

// Test with exactly 3 students (triple)
func (suite *PeerAssignmentServiceTestSuite) TestGenerateTriple() {
	phaseID := uuid.New()
	tutorID := uuid.New()

	_, err := suite.service.conn.Exec(suite.ctx,
		`INSERT INTO tutor (course_phase_id, id, first_name, last_name, email, matriculation_number, university_login)
		 VALUES ($1, $2, 'TripleTutor', 'Test', 'tt@example.com', 'T888', 'tt')`,
		phaseID, tutorID)
	suite.Require().NoError(err)

	studentIDs := make([]uuid.UUID, 3)
	for i := 0; i < 3; i++ {
		studentIDs[i] = uuid.New()
		_, err := suite.service.conn.Exec(suite.ctx,
			`INSERT INTO seat (course_phase_id, seat_name, assigned_student, assigned_tutor)
			 VALUES ($1, $2, $3, $4)`,
			phaseID, fmt.Sprintf("T-%d", i),
			pgtype.UUID{Bytes: studentIDs[i], Valid: true},
			pgtype.UUID{Bytes: tutorID, Valid: true})
		suite.Require().NoError(err)
	}

	assignments, err := GeneratePeerAssignments(suite.ctx, phaseID)
	assert.NoError(suite.T(), err)
	// Triple: 3 students, each reviews 2 others = 6 directed assignments
	assert.Len(suite.T(), assignments, 6)

	// Verify each student reviews exactly 2 peers
	reviewCount := make(map[uuid.UUID]int)
	for _, a := range assignments {
		reviewCount[a.StudentID]++
	}
	for _, id := range studentIDs {
		assert.Equal(suite.T(), 2, reviewCount[id],
			"Student in triple should review exactly 2 peers")
	}
}

// Test with tutor group that has only 1 student (should be skipped)
func (suite *PeerAssignmentServiceTestSuite) TestGenerateSkipsSingleStudent() {
	phaseID := uuid.New()
	tutorID := uuid.New()

	_, err := suite.service.conn.Exec(suite.ctx,
		`INSERT INTO tutor (course_phase_id, id, first_name, last_name, email, matriculation_number, university_login)
		 VALUES ($1, $2, 'LoneTutor', 'Skip', 'ls@example.com', 'T777', 'ls')`,
		phaseID, tutorID)
	suite.Require().NoError(err)

	studentID := uuid.New()
	_, err = suite.service.conn.Exec(suite.ctx,
		`INSERT INTO seat (course_phase_id, seat_name, assigned_student, assigned_tutor)
		 VALUES ($1, 'Lone-1', $2, $3)`,
		phaseID,
		pgtype.UUID{Bytes: studentID, Valid: true},
		pgtype.UUID{Bytes: tutorID, Valid: true})
	suite.Require().NoError(err)

	assignments, err := GeneratePeerAssignments(suite.ctx, phaseID)
	assert.NoError(suite.T(), err)
	assert.Empty(suite.T(), assignments)
}

// Test update replaces existing
func (suite *PeerAssignmentServiceTestSuite) TestUpdateReplacesExisting() {
	phaseID := uuid.New()
	s1, s2, s3 := uuid.New(), uuid.New(), uuid.New()

	// Initial: s1 <-> s2
	err := UpdatePeerAssignments(suite.ctx, phaseID, []peerAssignmentDTO.PeerAssignment{
		{StudentID: s1, PeerID: s2},
		{StudentID: s2, PeerID: s1},
	})
	assert.NoError(suite.T(), err)

	// Replace with: s1 <-> s3
	err = UpdatePeerAssignments(suite.ctx, phaseID, []peerAssignmentDTO.PeerAssignment{
		{StudentID: s1, PeerID: s3},
		{StudentID: s3, PeerID: s1},
	})
	assert.NoError(suite.T(), err)

	result, err := GetAllPeerAssignments(suite.ctx, phaseID)
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), result, 2)

	// Verify s2 is no longer assigned
	for _, a := range result {
		assert.NotEqual(suite.T(), s2, a.StudentID)
		assert.NotEqual(suite.T(), s2, a.PeerID)
	}
}

// Test DB constraint: student_id != peer_id
func (suite *PeerAssignmentServiceTestSuite) TestDBConstraintNoSelfReview() {
	phaseID := uuid.New()
	studentID := uuid.New()

	// Directly try to insert a self-review (bypassing service validation)
	_, err := suite.service.conn.Exec(suite.ctx,
		`INSERT INTO peer_assignment (course_phase_id, student_id, peer_id) VALUES ($1, $2, $2)`,
		phaseID, studentID)
	assert.Error(suite.T(), err, "DB should reject self-review via CHECK constraint")
}

// Test idempotent insert (ON CONFLICT DO NOTHING)
func (suite *PeerAssignmentServiceTestSuite) TestIdempotentInsert() {
	phaseID := uuid.New()

	err := suite.service.queries.CreatePeerAssignment(suite.ctx, db.CreatePeerAssignmentParams{
		CoursePhaseID: phaseID,
		StudentID:     suite.studentID1,
		PeerID:        suite.studentID2,
	})
	assert.NoError(suite.T(), err)

	// Insert same again — should not error
	err = suite.service.queries.CreatePeerAssignment(suite.ctx, db.CreatePeerAssignmentParams{
		CoursePhaseID: phaseID,
		StudentID:     suite.studentID1,
		PeerID:        suite.studentID2,
	})
	assert.NoError(suite.T(), err)

	// Only one row should exist
	all, err := GetAllPeerAssignments(suite.ctx, phaseID)
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), all, 1)
}

// Isolation: different course phases don't interfere
func (suite *PeerAssignmentServiceTestSuite) TestPhaseIsolation() {
	phase1 := uuid.New()
	phase2 := uuid.New()

	err := UpdatePeerAssignments(suite.ctx, phase1, []peerAssignmentDTO.PeerAssignment{
		{StudentID: suite.studentID1, PeerID: suite.studentID2},
	})
	assert.NoError(suite.T(), err)

	err = UpdatePeerAssignments(suite.ctx, phase2, []peerAssignmentDTO.PeerAssignment{
		{StudentID: suite.studentID2, PeerID: suite.studentID1},
	})
	assert.NoError(suite.T(), err)

	// Delete phase1 — phase2 should be unaffected
	err = DeletePeerAssignments(suite.ctx, phase1)
	assert.NoError(suite.T(), err)

	p1, err := GetAllPeerAssignments(suite.ctx, phase1)
	assert.NoError(suite.T(), err)
	assert.Empty(suite.T(), p1)

	p2, err := GetAllPeerAssignments(suite.ctx, phase2)
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), p2, 1)
}
