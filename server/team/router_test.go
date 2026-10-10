package team

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/prompt-edu/prompt-intro-course/server/coreRequests"
	"github.com/prompt-edu/prompt-intro-course/server/team/teamDTO"
	"github.com/prompt-edu/prompt-intro-course/server/testutils"
	"github.com/prompt-edu/prompt-sdk/keycloakTokenVerifier"
	"github.com/prompt-edu/prompt-sdk/promptTypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type TeamRouterTestSuite struct {
	suite.Suite
	router  *gin.Engine
	cleanup func()
}

func (suite *TeamRouterTestSuite) SetupSuite() {
	gin.SetMode(gin.TestMode)
	testDB, cleanup, err := testutils.SetupTestDB(context.Background(), "../database_dumps/intro_course.sql")
	if err != nil {
		suite.T().Fatalf("Failed to set up test database: %v", err)
	}
	suite.cleanup = cleanup
	TeamServiceSingleton = &TeamService{
		queries: *testDB.Queries,
	}

	suite.router = newRouter(func(allowedRoles ...string) gin.HandlerFunc {
		return testutils.MockAuthMiddlewareWithParticipation(allowedRoles, samStudentID)
	})
}

func (suite *TeamRouterTestSuite) SetupTest() {
	TeamServiceSingleton.lastNameRefresh.Clear()
}

func (suite *TeamRouterTestSuite) TearDownSuite() {
	if suite.cleanup != nil {
		suite.cleanup()
	}
}

func TestTeamRouterTestSuite(t *testing.T) {
	suite.Run(t, new(TeamRouterTestSuite))
}

func newRouter(authMiddleware func(allowedRoles ...string) gin.HandlerFunc) *gin.Engine {
	router := gin.New()
	setupTeamRouter(router.Group("/intro-course/api/course_phase/:coursePhaseID"), authMiddleware)
	return router
}

// routerAs authenticates every request as tokenUser, the way the SDK middleware leaves it.
func routerAs(tokenUser keycloakTokenVerifier.TokenUser) *gin.Engine {
	return newRouter(func(allowedRoles ...string) gin.HandlerFunc {
		return func(c *gin.Context) {
			keycloakTokenVerifier.SetTokenUser(c, tokenUser)
			c.Next()
		}
	})
}

func get(router *gin.Engine, path string) *httptest.ResponseRecorder {
	req, _ := http.NewRequest("GET", "/intro-course/api/course_phase/"+path, nil)
	req.Header.Set("Authorization", "Bearer caller")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	return resp
}

func decodeTeams(t *testing.T, resp *httptest.ResponseRecorder) []promptTypes.Team {
	require.Equal(t, http.StatusOK, resp.Code)
	var body map[string][]promptTypes.Team
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &body))
	return body["teams"]
}

func (suite *TeamRouterTestSuite) TestGetTeamsWrapsTeamsInDtoKey() {
	teams := decodeTeams(suite.T(), get(suite.router, testCoursePhaseID.String()+"/team"))
	require.Len(suite.T(), teams, 2)
	assert.Equal(suite.T(), aliceTutorID, teams[0].ID)
}

func (suite *TeamRouterTestSuite) TestGetTeamsInvalidCoursePhase() {
	resp := get(suite.router, "not-a-uuid/team")
	assert.Equal(suite.T(), http.StatusBadRequest, resp.Code)
}

func (suite *TeamRouterTestSuite) TestStaffReadRefreshesNamesFromCore() {
	hits := fakeCore(suite.T(), http.StatusOK, []coreRequests.Participation{
		{CourseParticipationID: unnamedStudentID.String(), Student: coreRequests.StudentData{FirstName: "Una", LastName: "Named"}},
	})

	teams := decodeTeams(suite.T(), get(routerAs(keycloakTokenVerifier.TokenUser{IsLecturer: true}), testCoursePhaseID.String()+"/team"))
	assert.EqualValues(suite.T(), 1, hits.Load())
	require.Len(suite.T(), teams[1].Members, 1)
	assert.Equal(suite.T(), "Una", teams[1].Members[0].FirstName)
}

func (suite *TeamRouterTestSuite) TestStudentReadServesCachedNamesWithoutCallingCore() {
	hits := fakeCore(suite.T(), http.StatusOK, nil)

	teams := decodeTeams(suite.T(), get(routerAs(keycloakTokenVerifier.TokenUser{IsStudentOfCourse: true}), testCoursePhaseID.String()+"/team"))
	assert.EqualValues(suite.T(), 0, hits.Load())
	assert.Equal(suite.T(), "Sam", teams[0].Members[0].FirstName)
}

func (suite *TeamRouterTestSuite) TestStaffReadSurvivesCoreFailure() {
	hits := fakeCore(suite.T(), http.StatusInternalServerError, nil)

	teams := decodeTeams(suite.T(), get(routerAs(keycloakTokenVerifier.TokenUser{IsEditor: true}), testCoursePhaseID.String()+"/team"))
	assert.EqualValues(suite.T(), 1, hits.Load())
	assert.Len(suite.T(), teams, 2)
}

func (suite *TeamRouterTestSuite) TestTutorIsScopedToOwnTeam() {
	fakeCore(suite.T(), http.StatusOK, nil)
	bob := routerAs(keycloakTokenVerifier.TokenUser{IsEditor: true, UniversityLogin: "Bob"})

	teams := decodeTeams(suite.T(), get(bob, testCoursePhaseID.String()+"/team"))
	require.Len(suite.T(), teams, 1)
	assert.Equal(suite.T(), bobTutorID, teams[0].ID)

	resp := get(bob, testCoursePhaseID.String()+"/allocation")
	require.Equal(suite.T(), http.StatusOK, resp.Code)
	var allocations []teamDTO.AllocationWithParticipation
	require.NoError(suite.T(), json.Unmarshal(resp.Body.Bytes(), &allocations))
	assert.Equal(suite.T(), []teamDTO.AllocationWithParticipation{{CourseParticipationID: unnamedStudentID, TeamAllocation: bobTutorID}}, allocations)

	assert.Equal(suite.T(), http.StatusOK, get(bob, testCoursePhaseID.String()+"/allocation/"+unnamedStudentID.String()).Code)
	assert.Equal(suite.T(), http.StatusForbidden, get(bob, testCoursePhaseID.String()+"/allocation/"+samStudentID.String()).Code)
}

func (suite *TeamRouterTestSuite) TestEditorWithoutTutorRecordIsNotScoped() {
	fakeCore(suite.T(), http.StatusOK, nil)
	editor := routerAs(keycloakTokenVerifier.TokenUser{IsEditor: true, UniversityLogin: "someone-else"})

	assert.Len(suite.T(), decodeTeams(suite.T(), get(editor, testCoursePhaseID.String()+"/team")), 2)
}

func (suite *TeamRouterTestSuite) TestGetAllocations() {
	resp := get(suite.router, testCoursePhaseID.String()+"/allocation")
	require.Equal(suite.T(), http.StatusOK, resp.Code)

	var allocations []teamDTO.AllocationWithParticipation
	require.NoError(suite.T(), json.Unmarshal(resp.Body.Bytes(), &allocations))
	assert.Len(suite.T(), allocations, 2)
}

func (suite *TeamRouterTestSuite) TestGetAllocationsForPhaseWithoutSeatsIsEmptyArray() {
	resp := get(suite.router, uuid.New().String()+"/allocation")
	require.Equal(suite.T(), http.StatusOK, resp.Code)
	assert.JSONEq(suite.T(), "[]", resp.Body.String())
}

func (suite *TeamRouterTestSuite) TestGetAllocation() {
	resp := get(suite.router, testCoursePhaseID.String()+"/allocation/"+samStudentID.String())
	require.Equal(suite.T(), http.StatusOK, resp.Code)

	var body map[string]string
	require.NoError(suite.T(), json.Unmarshal(resp.Body.Bytes(), &body))
	assert.Equal(suite.T(), aliceTutorID.String(), body["teamAllocation"])
}

func (suite *TeamRouterTestSuite) TestGetAllocationUnseatedStudent() {
	resp := get(suite.router, testCoursePhaseID.String()+"/allocation/"+uuid.New().String())
	assert.Equal(suite.T(), http.StatusNotFound, resp.Code)
}

func (suite *TeamRouterTestSuite) TestGetAllocationInvalidParticipation() {
	resp := get(suite.router, testCoursePhaseID.String()+"/allocation/not-a-uuid")
	assert.Equal(suite.T(), http.StatusBadRequest, resp.Code)
}
