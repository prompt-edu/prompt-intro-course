package peerAssignment

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/prompt-edu/prompt-intro-course/server/peerAssignment/peerAssignmentDTO"
	"github.com/prompt-edu/prompt-intro-course/server/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type PeerAssignmentRouterTestSuite struct {
	suite.Suite
	ctx           context.Context
	router        *gin.Engine
	cleanup       func()
	coursePhaseID  uuid.UUID
	studentID     uuid.UUID
}

func (suite *PeerAssignmentRouterTestSuite) SetupSuite() {
	gin.SetMode(gin.TestMode)
	suite.ctx = context.Background()
	testDB, cleanup, err := testutils.SetupTestDB(suite.ctx, "../database_dumps/intro_course.sql")
	if err != nil {
		suite.T().Fatalf("Failed to set up test database: %v", err)
	}
	suite.cleanup = cleanup
	suite.coursePhaseID = uuid.MustParse("4179d58a-d00d-4fa7-94a5-397bc69fab02")
	suite.studentID = uuid.MustParse("33333333-3333-3333-3333-333333333333")

	peerService := PeerAssignmentService{
		queries: *testDB.Queries,
		conn:    testDB.Conn,
	}
	PeerAssignmentServiceSingleton = &peerService

	suite.router = gin.Default()
	api := suite.router.Group("/intro-course/api/course_phase/:coursePhaseID")
	authMiddleware := func(allowedRoles ...string) gin.HandlerFunc {
		return testutils.MockAuthMiddlewareWithParticipation(allowedRoles, suite.studentID)
	}
	setupPeerAssignmentRouter(api, authMiddleware)
}

func (suite *PeerAssignmentRouterTestSuite) TearDownSuite() {
	if suite.cleanup != nil {
		suite.cleanup()
	}
}

func TestPeerAssignmentRouterTestSuite(t *testing.T) {
	suite.Run(t, new(PeerAssignmentRouterTestSuite))
}

func (suite *PeerAssignmentRouterTestSuite) basePath() string {
	return "/intro-course/api/course_phase/" + suite.coursePhaseID.String() + "/peer_assignments"
}

// --- GET /peer_assignments ---

func (suite *PeerAssignmentRouterTestSuite) TestGetPeerAssignmentsSuccess() {
	req, _ := http.NewRequest("GET", suite.basePath(), nil)
	resp := httptest.NewRecorder()
	suite.router.ServeHTTP(resp, req)

	assert.Equal(suite.T(), http.StatusOK, resp.Code)
	var assignments []peerAssignmentDTO.PeerAssignment
	assert.NoError(suite.T(), json.Unmarshal(resp.Body.Bytes(), &assignments))
}

func (suite *PeerAssignmentRouterTestSuite) TestGetPeerAssignmentsInvalidUUID() {
	req, _ := http.NewRequest("GET", "/intro-course/api/course_phase/not-a-uuid/peer_assignments", nil)
	resp := httptest.NewRecorder()
	suite.router.ServeHTTP(resp, req)

	assert.Equal(suite.T(), http.StatusBadRequest, resp.Code)
}

// --- POST /peer_assignments/generate ---

func (suite *PeerAssignmentRouterTestSuite) TestGeneratePeerAssignmentsSuccess() {
	req, _ := http.NewRequest("POST", suite.basePath()+"/generate", nil)
	resp := httptest.NewRecorder()
	suite.router.ServeHTTP(resp, req)

	assert.Equal(suite.T(), http.StatusCreated, resp.Code)
}

func (suite *PeerAssignmentRouterTestSuite) TestGeneratePeerAssignmentsInvalidUUID() {
	req, _ := http.NewRequest("POST", "/intro-course/api/course_phase/bad-uuid/peer_assignments/generate", nil)
	resp := httptest.NewRecorder()
	suite.router.ServeHTTP(resp, req)

	assert.Equal(suite.T(), http.StatusBadRequest, resp.Code)
}

// --- PUT /peer_assignments ---

func (suite *PeerAssignmentRouterTestSuite) TestUpdatePeerAssignmentsSuccess() {
	s1, s2 := uuid.New(), uuid.New()
	body, _ := json.Marshal([]peerAssignmentDTO.PeerAssignment{
		{StudentID: s1, PeerID: s2},
		{StudentID: s2, PeerID: s1},
	})
	req, _ := http.NewRequest("PUT", suite.basePath(), bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	suite.router.ServeHTTP(resp, req)

	assert.Equal(suite.T(), http.StatusOK, resp.Code)
}

func (suite *PeerAssignmentRouterTestSuite) TestUpdatePeerAssignmentsInvalidBody() {
	req, _ := http.NewRequest("PUT", suite.basePath(), bytes.NewBuffer([]byte("not-json")))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	suite.router.ServeHTTP(resp, req)

	assert.Equal(suite.T(), http.StatusBadRequest, resp.Code)
}

func (suite *PeerAssignmentRouterTestSuite) TestUpdatePeerAssignmentsInvalidUUID() {
	body, _ := json.Marshal([]peerAssignmentDTO.PeerAssignment{})
	req, _ := http.NewRequest("PUT", "/intro-course/api/course_phase/bad/peer_assignments", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	suite.router.ServeHTTP(resp, req)

	assert.Equal(suite.T(), http.StatusBadRequest, resp.Code)
}

func (suite *PeerAssignmentRouterTestSuite) TestUpdatePeerAssignmentsEmptyPayload() {
	body, _ := json.Marshal([]peerAssignmentDTO.PeerAssignment{})
	req, _ := http.NewRequest("PUT", suite.basePath(), bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	suite.router.ServeHTTP(resp, req)

	assert.Equal(suite.T(), http.StatusBadRequest, resp.Code)
	assert.Contains(suite.T(), resp.Body.String(), "empty")
}

func (suite *PeerAssignmentRouterTestSuite) TestUpdatePeerAssignmentsTooManyAssignments() {
	assignments := make([]peerAssignmentDTO.PeerAssignment, 1001)
	for i := range assignments {
		assignments[i] = peerAssignmentDTO.PeerAssignment{StudentID: uuid.New(), PeerID: uuid.New()}
	}
	body, _ := json.Marshal(assignments)
	req, _ := http.NewRequest("PUT", suite.basePath(), bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	suite.router.ServeHTTP(resp, req)

	assert.Equal(suite.T(), http.StatusBadRequest, resp.Code)
	assert.Contains(suite.T(), resp.Body.String(), "too many")
}

func (suite *PeerAssignmentRouterTestSuite) TestUpdatePeerAssignmentsSelfReview() {
	sameID := uuid.New()
	assignments := []peerAssignmentDTO.PeerAssignment{{StudentID: sameID, PeerID: sameID}}
	body, _ := json.Marshal(assignments)
	req, _ := http.NewRequest("PUT", suite.basePath(), bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	suite.router.ServeHTTP(resp, req)

	assert.Equal(suite.T(), http.StatusBadRequest, resp.Code)
	assert.Contains(suite.T(), resp.Body.String(), "self-review")
}

// --- DELETE /peer_assignments ---

func (suite *PeerAssignmentRouterTestSuite) TestDeletePeerAssignmentsSuccess() {
	req, _ := http.NewRequest("DELETE", suite.basePath(), nil)
	resp := httptest.NewRecorder()
	suite.router.ServeHTTP(resp, req)

	assert.Equal(suite.T(), http.StatusOK, resp.Code)
}

func (suite *PeerAssignmentRouterTestSuite) TestDeletePeerAssignmentsInvalidUUID() {
	req, _ := http.NewRequest("DELETE", "/intro-course/api/course_phase/bad/peer_assignments", nil)
	resp := httptest.NewRecorder()
	suite.router.ServeHTTP(resp, req)

	assert.Equal(suite.T(), http.StatusBadRequest, resp.Code)
}

// --- GET /peer_assignments/own ---

func (suite *PeerAssignmentRouterTestSuite) TestGetOwnPeerAssignmentSuccess() {
	req, _ := http.NewRequest("GET", suite.basePath()+"/own", nil)
	resp := httptest.NewRecorder()
	suite.router.ServeHTTP(resp, req)

	assert.Equal(suite.T(), http.StatusOK, resp.Code)
	var own peerAssignmentDTO.OwnPeerAssignment
	assert.NoError(suite.T(), json.Unmarshal(resp.Body.Bytes(), &own))
}

func (suite *PeerAssignmentRouterTestSuite) TestGetOwnPeerAssignmentInvalidUUID() {
	req, _ := http.NewRequest("GET", "/intro-course/api/course_phase/bad/peer_assignments/own", nil)
	resp := httptest.NewRecorder()
	suite.router.ServeHTTP(resp, req)

	assert.Equal(suite.T(), http.StatusBadRequest, resp.Code)
}

// --- POST /peer_assignments/sync-gitlab ---

func (suite *PeerAssignmentRouterTestSuite) TestSyncGitlabMissingSemesterTag() {
	body, _ := json.Marshal(map[string]string{})
	req, _ := http.NewRequest("POST", suite.basePath()+"/sync-gitlab", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	suite.router.ServeHTTP(resp, req)

	// Should fail because semesterTag is required
	assert.Equal(suite.T(), http.StatusBadRequest, resp.Code)
}

func (suite *PeerAssignmentRouterTestSuite) TestSyncGitlabInvalidUUID() {
	body, _ := json.Marshal(peerAssignmentDTO.SyncRequest{SemesterTag: "SS2025"})
	req, _ := http.NewRequest("POST", "/intro-course/api/course_phase/bad/peer_assignments/sync-gitlab", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	suite.router.ServeHTTP(resp, req)

	assert.Equal(suite.T(), http.StatusBadRequest, resp.Code)
}

// --- Integration: Generate -> Get -> GetOwn -> Delete round trip ---

func (suite *PeerAssignmentRouterTestSuite) TestFullRoundTrip() {
	phaseID := uuid.New()
	tutorID := uuid.New()

	// Set up test data: 1 tutor, 4 students
	_, err := PeerAssignmentServiceSingleton.conn.Exec(suite.ctx,
		`INSERT INTO tutor (course_phase_id, id, first_name, last_name, email, matriculation_number, university_login, gitlab_username)
		 VALUES ($1, $2, 'RoundTrip', 'Tutor', 'rt@example.com', 'RT001', 'rt', 'rtgit')`,
		phaseID, tutorID)
	suite.Require().NoError(err)

	studentIDs := make([]uuid.UUID, 4)
	for i := 0; i < 4; i++ {
		studentIDs[i] = uuid.New()
		_, err := PeerAssignmentServiceSingleton.conn.Exec(suite.ctx,
			`INSERT INTO developer_profile (course_phase_id, course_participation_id, gitlab_username, apple_id, has_macbook)
			 VALUES ($1, $2, $3, $4, true)`,
			phaseID, studentIDs[i],
			fmt.Sprintf("rt_student%d", i), fmt.Sprintf("rt%d@apple.com", i))
		suite.Require().NoError(err)

		_, err = PeerAssignmentServiceSingleton.conn.Exec(suite.ctx,
			`INSERT INTO seat (course_phase_id, seat_name, assigned_student, assigned_tutor)
			 VALUES ($1, $2, $3, $4)`,
			phaseID, fmt.Sprintf("RT-%d", i),
			pgtype.UUID{Bytes: studentIDs[i], Valid: true},
			pgtype.UUID{Bytes: tutorID, Valid: true})
		suite.Require().NoError(err)
	}

	basePath := "/intro-course/api/course_phase/" + phaseID.String() + "/peer_assignments"

	// Step 1: Generate
	req, _ := http.NewRequest("POST", basePath+"/generate", nil)
	resp := httptest.NewRecorder()
	suite.router.ServeHTTP(resp, req)
	assert.Equal(suite.T(), http.StatusCreated, resp.Code)

	var generated []peerAssignmentDTO.PeerAssignment
	assert.NoError(suite.T(), json.Unmarshal(resp.Body.Bytes(), &generated))
	assert.NotEmpty(suite.T(), generated)
	// 4 students = 1 quad = 12 directed assignments (4 × 3 peers)
	assert.Len(suite.T(), generated, 12)

	// Step 2: Get all
	req, _ = http.NewRequest("GET", basePath, nil)
	resp = httptest.NewRecorder()
	suite.router.ServeHTTP(resp, req)
	assert.Equal(suite.T(), http.StatusOK, resp.Code)

	var all []peerAssignmentDTO.PeerAssignment
	assert.NoError(suite.T(), json.Unmarshal(resp.Body.Bytes(), &all))
	assert.Len(suite.T(), all, 12)

	// Step 3: Get own (using studentIDs[0] through mock middleware — need to re-setup)
	// The mock middleware uses suite.studentID which is a different ID.
	// We test /own with the default studentID which has no peers in this phase.
	req, _ = http.NewRequest("GET", basePath+"/own", nil)
	resp = httptest.NewRecorder()
	suite.router.ServeHTTP(resp, req)
	assert.Equal(suite.T(), http.StatusOK, resp.Code)

	// Step 4: Delete all
	req, _ = http.NewRequest("DELETE", basePath, nil)
	resp = httptest.NewRecorder()
	suite.router.ServeHTTP(resp, req)
	assert.Equal(suite.T(), http.StatusOK, resp.Code)

	// Step 5: Verify empty
	req, _ = http.NewRequest("GET", basePath, nil)
	resp = httptest.NewRecorder()
	suite.router.ServeHTTP(resp, req)
	assert.Equal(suite.T(), http.StatusOK, resp.Code)

	var empty []peerAssignmentDTO.PeerAssignment
	assert.NoError(suite.T(), json.Unmarshal(resp.Body.Bytes(), &empty))
	assert.Empty(suite.T(), empty)
}

// Test PUT + GET round trip
func (suite *PeerAssignmentRouterTestSuite) TestUpdateAndRetrieve() {
	phaseID := uuid.New()
	s1, s2 := uuid.New(), uuid.New()
	basePath := "/intro-course/api/course_phase/" + phaseID.String() + "/peer_assignments"

	// Update
	body, _ := json.Marshal([]peerAssignmentDTO.PeerAssignment{
		{StudentID: s1, PeerID: s2},
		{StudentID: s2, PeerID: s1},
	})
	req, _ := http.NewRequest("PUT", basePath, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	suite.router.ServeHTTP(resp, req)
	assert.Equal(suite.T(), http.StatusOK, resp.Code)

	// Retrieve
	req, _ = http.NewRequest("GET", basePath, nil)
	resp = httptest.NewRecorder()
	suite.router.ServeHTTP(resp, req)
	assert.Equal(suite.T(), http.StatusOK, resp.Code)

	var result []peerAssignmentDTO.PeerAssignment
	assert.NoError(suite.T(), json.Unmarshal(resp.Body.Bytes(), &result))
	assert.Len(suite.T(), result, 2)
}
