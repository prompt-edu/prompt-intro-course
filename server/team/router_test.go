package team

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/prompt-edu/prompt-intro-course/server/team/teamDTO"
	"github.com/prompt-edu/prompt-intro-course/server/testutils"
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
	ctx := context.Background()
	testDB, cleanup, err := testutils.SetupTestDB(ctx, "../database_dumps/intro_course.sql")
	if err != nil {
		suite.T().Fatalf("Failed to set up test database: %v", err)
	}
	suite.cleanup = cleanup
	TeamServiceSingleton = &TeamService{
		queries: *testDB.Queries,
		conn:    testDB.Conn,
	}

	suite.router = gin.Default()
	api := suite.router.Group("/intro-course/api/course_phase/:coursePhaseID")
	authMiddleware := func(allowedRoles ...string) gin.HandlerFunc {
		return testutils.MockAuthMiddlewareWithParticipation(allowedRoles, samStudentID)
	}
	setupTeamRouter(api, authMiddleware)
}

func (suite *TeamRouterTestSuite) TearDownSuite() {
	if suite.cleanup != nil {
		suite.cleanup()
	}
}

func TestTeamRouterTestSuite(t *testing.T) {
	suite.Run(t, new(TeamRouterTestSuite))
}

func (suite *TeamRouterTestSuite) get(path string) *httptest.ResponseRecorder {
	req, _ := http.NewRequest("GET", "/intro-course/api/course_phase/"+path, nil)
	resp := httptest.NewRecorder()
	suite.router.ServeHTTP(resp, req)
	return resp
}

func (suite *TeamRouterTestSuite) TestGetTeamsWrapsTeamsInDtoKey() {
	resp := suite.get(testCoursePhaseID.String() + "/team")
	require.Equal(suite.T(), http.StatusOK, resp.Code)

	var body map[string][]promptTypes.Team
	require.NoError(suite.T(), json.Unmarshal(resp.Body.Bytes(), &body))
	require.Len(suite.T(), body["teams"], 2)
	assert.Equal(suite.T(), aliceTutorID, body["teams"][0].ID)
}

func (suite *TeamRouterTestSuite) TestGetTeamsInvalidCoursePhase() {
	resp := suite.get("not-a-uuid/team")
	assert.Equal(suite.T(), http.StatusBadRequest, resp.Code)
}

func (suite *TeamRouterTestSuite) TestGetAllocations() {
	resp := suite.get(testCoursePhaseID.String() + "/allocation")
	require.Equal(suite.T(), http.StatusOK, resp.Code)

	var allocations []teamDTO.AllocationWithParticipation
	require.NoError(suite.T(), json.Unmarshal(resp.Body.Bytes(), &allocations))
	assert.Len(suite.T(), allocations, 2)
}

func (suite *TeamRouterTestSuite) TestGetAllocationsForPhaseWithoutSeatsIsEmptyArray() {
	resp := suite.get(uuid.New().String() + "/allocation")
	require.Equal(suite.T(), http.StatusOK, resp.Code)
	assert.JSONEq(suite.T(), "[]", resp.Body.String())
}

func (suite *TeamRouterTestSuite) TestGetAllocation() {
	resp := suite.get(testCoursePhaseID.String() + "/allocation/" + samStudentID.String())
	require.Equal(suite.T(), http.StatusOK, resp.Code)

	var body map[string]string
	require.NoError(suite.T(), json.Unmarshal(resp.Body.Bytes(), &body))
	assert.Equal(suite.T(), aliceTutorID.String(), body["teamAllocation"])
}

func (suite *TeamRouterTestSuite) TestGetAllocationUnseatedStudent() {
	resp := suite.get(testCoursePhaseID.String() + "/allocation/" + uuid.New().String())
	assert.Equal(suite.T(), http.StatusNotFound, resp.Code)
}

func (suite *TeamRouterTestSuite) TestGetAllocationInvalidParticipation() {
	resp := suite.get(testCoursePhaseID.String() + "/allocation/not-a-uuid")
	assert.Equal(suite.T(), http.StatusBadRequest, resp.Code)
}
