package team

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
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

// fakeCore serves core's participation list and counts the requests it gets.
func fakeCore(t *testing.T, status int, participations []coreRequests.Participation) *atomic.Int32 {
	var hits atomic.Int32
	core := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(coreRequests.ParticipationsResponse{Participations: participations})
	}))
	t.Cleanup(core.Close)
	t.Setenv("SERVER_CORE_HOST", core.URL)
	return &hits
}

type TeamServiceTestSuite struct {
	suite.Suite
	ctx     context.Context
	conn    *pgxpool.Pool
	cleanup func()
}

func (suite *TeamServiceTestSuite) SetupSuite() {
	suite.ctx = context.Background()
	testDB, cleanup, err := testutils.SetupTestDB(suite.ctx, "../database_dumps/intro_course.sql")
	if err != nil {
		suite.T().Fatalf("Failed to set up test database: %v", err)
	}
	suite.cleanup = cleanup
	suite.conn = testDB.Conn
	TeamServiceSingleton = &TeamService{
		queries: *testDB.Queries,
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

func (suite *TeamServiceTestSuite) exec(sql string, args ...any) {
	_, err := suite.conn.Exec(suite.ctx, sql, args...)
	require.NoError(suite.T(), err)
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

func (suite *TeamServiceTestSuite) TestDoubleSeatedStudentBelongsToOneTeam() {
	coursePhaseID := uuid.New()
	firstTutor, secondTutor, student := uuid.New(), uuid.New(), uuid.New()
	suite.exec(`INSERT INTO tutor (course_phase_id, id, first_name, last_name, email, matriculation_number, university_login)
		VALUES ($1, $2, 'Amy', 'First', 'a@example.com', '1', 'amy'), ($1, $3, 'Zed', 'Second', 'z@example.com', '2', 'zed')`,
		coursePhaseID, firstTutor, secondTutor)
	// A-0 is a tutor seat and never counts; of the student seats, A-1 sorts before B-1.
	suite.exec(`INSERT INTO seat (course_phase_id, seat_name, assigned_student, assigned_tutor, is_tutor_seat)
		VALUES ($1, 'A-0', $2, $3, true), ($1, 'A-1', $2, $4, false), ($1, 'B-1', $2, $3, false)`,
		coursePhaseID, student, firstTutor, secondTutor)

	teams, err := GetTeams(suite.ctx, coursePhaseID)
	require.NoError(suite.T(), err)
	require.Len(suite.T(), teams, 2)
	assert.Empty(suite.T(), teams[0].Members, "Amy's seats are a tutor seat and a later duplicate")
	require.Len(suite.T(), teams[1].Members, 1)
	assert.Equal(suite.T(), student, teams[1].Members[0].ID)

	allocations, err := GetAllocations(suite.ctx, coursePhaseID)
	require.NoError(suite.T(), err)
	require.Len(suite.T(), allocations, 1)
	assert.Equal(suite.T(), secondTutor, allocations[0].TeamAllocation)

	teamID, err := GetAllocation(suite.ctx, coursePhaseID, student)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), secondTutor, teamID)
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
	tutor, student := uuid.New(), uuid.New()
	suite.exec(`INSERT INTO tutor (course_phase_id, id, first_name, last_name, email, matriculation_number, university_login)
		VALUES ($1, $2, 'Tia', 'Tutor', 't@example.com', '3', 'tia')`, coursePhaseID, tutor)
	suite.exec(`INSERT INTO seat (course_phase_id, seat_name, assigned_student, assigned_tutor) VALUES ($1, 'S-1', $2, $3)`,
		coursePhaseID, student, tutor)

	var gotAuthHeader, gotPath string
	core := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuthHeader, gotPath = r.Header.Get("Authorization"), r.URL.Path
		_ = json.NewEncoder(w).Encode(coreRequests.ParticipationsResponse{
			Participations: []coreRequests.Participation{
				{CourseParticipationID: student.String(), Student: coreRequests.StudentData{FirstName: "Nora", LastName: "New"}},
				{CourseParticipationID: "not-a-uuid", Student: coreRequests.StudentData{FirstName: "Broken"}},
			},
		})
	}))
	defer core.Close()
	suite.T().Setenv("SERVER_CORE_HOST", core.URL)

	require.NoError(suite.T(), RefreshParticipantNames(suite.ctx, "Bearer staff", coursePhaseID))
	assert.Equal(suite.T(), "Bearer staff", gotAuthHeader)
	assert.Equal(suite.T(), "/api/course_phases/"+coursePhaseID.String()+"/participations", gotPath)

	teams, err := GetTeams(suite.ctx, coursePhaseID)
	require.NoError(suite.T(), err)
	require.Len(suite.T(), teams, 1)
	require.Len(suite.T(), teams[0].Members, 1)
	assert.Equal(suite.T(), "Nora", teams[0].Members[0].FirstName)
	assert.Equal(suite.T(), "New", teams[0].Members[0].LastName)
}

func (suite *TeamServiceTestSuite) TestRefreshParticipantNamesReportsCoreFailure() {
	before, err := GetTeams(suite.ctx, testCoursePhaseID)
	require.NoError(suite.T(), err)

	fakeCore(suite.T(), http.StatusForbidden, nil)
	assert.Error(suite.T(), RefreshParticipantNames(suite.ctx, "Bearer student", testCoursePhaseID))

	after, err := GetTeams(suite.ctx, testCoursePhaseID)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), before, after, "cached names survive a failed refresh")
}

func (suite *TeamServiceTestSuite) TestRefreshParticipantNamesIfStaleThrottles() {
	coursePhaseID := uuid.New()

	failing := fakeCore(suite.T(), http.StatusBadGateway, nil)
	assert.Error(suite.T(), RefreshParticipantNamesIfStale(suite.ctx, "Bearer staff", coursePhaseID))
	assert.EqualValues(suite.T(), 1, failing.Load())

	// A failed refresh is not remembered, so the next read retries.
	working := fakeCore(suite.T(), http.StatusOK, nil)
	require.NoError(suite.T(), RefreshParticipantNamesIfStale(suite.ctx, "Bearer staff", coursePhaseID))
	require.NoError(suite.T(), RefreshParticipantNamesIfStale(suite.ctx, "Bearer staff", coursePhaseID))
	assert.EqualValues(suite.T(), 1, working.Load())
}
