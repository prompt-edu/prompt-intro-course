package team

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/prompt-edu/prompt-intro-course/server/coreRequests"
	"github.com/prompt-edu/prompt-intro-course/server/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

var (
	testCoursePhaseID = uuid.MustParse("4179d58a-d00d-4fa7-94a5-397bc69fab02")
	aliceTutorID      = uuid.MustParse("11111111-1111-1111-1111-111111111111")
	bobTutorID        = uuid.MustParse("22222222-2222-2222-2222-222222222222")
	samStudentID      = uuid.MustParse("33333333-3333-3333-3333-333333333333")
	unnamedStudentID  = uuid.MustParse("44444444-4444-4444-4444-444444444444")
)

type TeamServiceTestSuite struct {
	suite.Suite
	ctx     context.Context
	cleanup func()
}

func (suite *TeamServiceTestSuite) SetupSuite() {
	suite.ctx = context.Background()
	testDB, cleanup, err := testutils.SetupTestDB(suite.ctx, "../database_dumps/intro_course.sql")
	if err != nil {
		suite.T().Fatalf("Failed to set up test database: %v", err)
	}
	suite.cleanup = cleanup
	TeamServiceSingleton = &TeamService{
		queries: *testDB.Queries,
		conn:    testDB.Conn,
	}
}

func (suite *TeamServiceTestSuite) TearDownSuite() {
	if suite.cleanup != nil {
		suite.cleanup()
	}
}

func TestTeamServiceTestSuite(t *testing.T) {
	suite.Run(t, new(TeamServiceTestSuite))
}

func (suite *TeamServiceTestSuite) TestGetTeamsBuildsOneTeamPerTutor() {
	teams, err := GetTeams(suite.ctx, testCoursePhaseID)
	require.NoError(suite.T(), err)
	require.Len(suite.T(), teams, 2)

	alice := teams[0]
	assert.Equal(suite.T(), aliceTutorID, alice.ID)
	assert.Equal(suite.T(), "Alice Tutor", alice.Name)
	require.Len(suite.T(), alice.Tutors, 1)
	assert.Equal(suite.T(), aliceTutorID, alice.Tutors[0].ID)
	assert.Equal(suite.T(), "Alice", alice.Tutors[0].FirstName)
	require.Len(suite.T(), alice.Members, 1)
	assert.Equal(suite.T(), samStudentID, alice.Members[0].ID)
	assert.Equal(suite.T(), "Sam", alice.Members[0].FirstName)
	assert.Equal(suite.T(), "Student", alice.Members[0].LastName)

	bob := teams[1]
	assert.Equal(suite.T(), bobTutorID, bob.ID)
	require.Len(suite.T(), bob.Members, 1)
	// No cached name yet: the member is still listed so the allocation stays resolvable.
	assert.Equal(suite.T(), unnamedStudentID, bob.Members[0].ID)
	assert.Empty(suite.T(), bob.Members[0].FirstName)
}

func (suite *TeamServiceTestSuite) TestGetTeamsForPhaseWithoutTutors() {
	teams, err := GetTeams(suite.ctx, uuid.New())
	require.NoError(suite.T(), err)
	assert.NotNil(suite.T(), teams)
	assert.Empty(suite.T(), teams)
}

func (suite *TeamServiceTestSuite) TestGetAllocations() {
	allocations, err := GetAllocations(suite.ctx, testCoursePhaseID)
	require.NoError(suite.T(), err)

	byParticipant := make(map[uuid.UUID]uuid.UUID)
	for _, allocation := range allocations {
		byParticipant[allocation.CourseParticipationID] = allocation.TeamAllocation
	}
	assert.Equal(suite.T(), map[uuid.UUID]uuid.UUID{
		samStudentID:     aliceTutorID,
		unnamedStudentID: bobTutorID,
	}, byParticipant)
}

func (suite *TeamServiceTestSuite) TestGetAllocation() {
	teamID, err := GetAllocation(suite.ctx, testCoursePhaseID, samStudentID)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), aliceTutorID, teamID)

	_, err = GetAllocation(suite.ctx, testCoursePhaseID, uuid.New())
	assert.ErrorIs(suite.T(), err, ErrAllocationNotFound)
}

func (suite *TeamServiceTestSuite) TestRefreshParticipantNames() {
	coursePhaseID := uuid.New()
	var gotAuthHeader string
	core := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuthHeader = r.Header.Get("Authorization")
		assert.Equal(suite.T(), "/api/course_phases/"+coursePhaseID.String()+"/participations", r.URL.Path)
		_ = json.NewEncoder(w).Encode(coreRequests.ParticipationsResponse{
			Participations: []coreRequests.Participation{
				{CourseParticipationID: samStudentID.String(), Student: coreRequests.StudentData{FirstName: "Samantha", LastName: "Renamed"}},
				{CourseParticipationID: "not-a-uuid", Student: coreRequests.StudentData{FirstName: "Broken"}},
			},
		})
	}))
	defer core.Close()
	suite.T().Setenv("SERVER_CORE_HOST", core.URL)

	require.NoError(suite.T(), RefreshParticipantNames(suite.ctx, "Bearer staff", coursePhaseID))
	assert.Equal(suite.T(), "Bearer staff", gotAuthHeader)

	teams, err := GetTeams(suite.ctx, coursePhaseID)
	require.NoError(suite.T(), err)
	assert.Empty(suite.T(), teams, "the refresh must not create teams")

	var firstName string
	err = TeamServiceSingleton.conn.QueryRow(suite.ctx,
		"SELECT first_name FROM participant_name WHERE course_phase_id = $1 AND course_participation_id = $2",
		coursePhaseID, samStudentID).Scan(&firstName)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), "Samantha", firstName)
}

func (suite *TeamServiceTestSuite) TestRefreshParticipantNamesReportsCoreFailure() {
	core := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer core.Close()
	suite.T().Setenv("SERVER_CORE_HOST", core.URL)

	assert.Error(suite.T(), RefreshParticipantNames(suite.ctx, "Bearer student", testCoursePhaseID))

	teams, err := GetTeams(suite.ctx, testCoursePhaseID)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), "Sam", teams[0].Members[0].FirstName, "cached names survive a failed refresh")
}
