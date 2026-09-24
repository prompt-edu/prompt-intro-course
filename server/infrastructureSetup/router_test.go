package infrastructureSetup

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/prompt-edu/prompt-intro-course/server/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

func TestSemesterTagIsBoundToCoursePhase(t *testing.T) {
	gin.SetMode(gin.TestMode)
	phaseID := uuid.MustParse("4179d58a-d00d-4fa7-94a5-397bc69fab02")
	courseID := uuid.MustParse("5179d58a-d00d-4fa7-94a5-397bc69fab03")
	coreRequests := 0
	core := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		coreRequests++
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/course_phases/" + phaseID.String():
			_ = json.NewEncoder(w).Encode(map[string]any{"id": phaseID, "courseID": courseID})
		case "/api/courses/" + courseID.String():
			_ = json.NewEncoder(w).Encode(map[string]any{"id": courseID, "semesterTag": "ws2627"})
		default:
			http.NotFound(w, r)
		}
	}))
	defer core.Close()
	t.Setenv("SERVER_CORE_HOST", core.URL)

	for _, test := range []struct {
		name       string
		method     string
		path       string
		body       string
		wantStatus int
	}{
		{"reset", http.MethodPost, "/gitlab/demo/reset", `{"semesterTag":"SS2728","expectedProjectID":1,"expectedSourceSHA":"` + strings.Repeat("a", 40) + `"}`, http.StatusBadRequest},
		{"course setup", http.MethodPost, "/gitlab/course-setup", `{"semesterTag":"SS2728"}`, http.StatusBadRequest},
		{"course status", http.MethodGet, "/gitlab/course-setup?semesterTag=SS2728", "", http.StatusBadRequest},
		{"student setup", http.MethodPost, "/gitlab/student-setup/" + uuid.NewString(), `{"semesterTag":"SS2728"}`, http.StatusBadRequest},
	} {
		t.Run(test.name, func(t *testing.T) {
			router := gin.New()
			group := router.Group("/course_phase/:coursePhaseID")
			group.POST("/gitlab/demo/reset", resetDemo)
			group.POST("/gitlab/course-setup", createCourseSetup)
			group.GET("/gitlab/course-setup", getCourseSetup)
			group.POST("/gitlab/student-setup/:courseParticipationID", setupStudentInfrastructure)

			req := httptest.NewRequest(test.method, "/course_phase/"+phaseID.String()+test.path, bytes.NewBufferString(test.body))
			req.Header.Set("Authorization", "Bearer test-token")
			req.Header.Set("Content-Type", "application/json")
			response := httptest.NewRecorder()
			router.ServeHTTP(response, req)
			assert.Equal(t, test.wantStatus, response.Code)
			assert.Contains(t, response.Body.String(), "does not belong to this course phase")
		})
	}
	assert.Equal(t, 8, coreRequests)

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("Authorization", "Bearer test-token")
	response := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(response)
	c.Request = request
	tag, ok := requireSemesterTagForPhase(c, phaseID, " WS2627 ")
	assert.True(t, ok)
	assert.Equal(t, "IOS2627", tag)
	assert.Equal(t, 10, coreRequests)
}

func TestSemesterTagCheckFailsClosedWhenCoreIsUnavailable(t *testing.T) {
	gin.SetMode(gin.TestMode)
	core := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}))
	defer core.Close()
	t.Setenv("SERVER_CORE_HOST", core.URL)

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	_, ok := requireSemesterTagForPhase(c, uuid.New(), "WS2627")
	assert.False(t, ok)
	assert.Equal(t, http.StatusBadGateway, c.Writer.Status())
}

type InfrastructureRouterTestSuite struct {
	suite.Suite
	ctx           context.Context
	router        *gin.Engine
	cleanup       func()
	coursePhaseID uuid.UUID
}

func (suite *InfrastructureRouterTestSuite) SetupSuite() {
	gin.SetMode(gin.TestMode)
	suite.ctx = context.Background()
	testDB, cleanup, err := testutils.SetupTestDB(suite.ctx, "../database_dumps/intro_course.sql")
	if err != nil {
		suite.T().Fatalf("Failed to set up test database: %v", err)
	}
	suite.cleanup = cleanup
	suite.coursePhaseID = uuid.MustParse("4179d58a-d00d-4fa7-94a5-397bc69fab02")

	service := InfrastructureService{
		queries: *testDB.Queries,
		conn:    testDB.Conn,
	}
	InfrastructureServiceSingleton = &service

	suite.router = gin.Default()
	api := suite.router.Group("/intro-course/api/course_phase/:coursePhaseID")
	authMiddleware := func(allowedRoles ...string) gin.HandlerFunc {
		return testutils.DefaultMockAuthMiddleware()
	}
	setupInfrastructureRouter(api, authMiddleware)
}

func (suite *InfrastructureRouterTestSuite) TearDownSuite() {
	if suite.cleanup != nil {
		suite.cleanup()
	}
}

func TestInfrastructureRouterTestSuite(t *testing.T) {
	suite.Run(t, new(InfrastructureRouterTestSuite))
}

func (suite *InfrastructureRouterTestSuite) TestGetAllStudentGitlabStatus() {
	req, _ := http.NewRequest("GET", "/intro-course/api/course_phase/"+suite.coursePhaseID.String()+"/infrastructure/gitlab/student-setup", nil)
	resp := httptest.NewRecorder()

	suite.router.ServeHTTP(resp, req)

	assert.Equal(suite.T(), http.StatusOK, resp.Code)
	var statuses []map[string]interface{}
	assert.NoError(suite.T(), json.Unmarshal(resp.Body.Bytes(), &statuses))
	assert.True(suite.T(), len(statuses) >= 1)
}

func (suite *InfrastructureRouterTestSuite) TestManuallyOverwriteStudentGitlabStatus() {
	courseParticipationID := uuid.MustParse("33333333-3333-3333-3333-333333333333")
	req, _ := http.NewRequest("PUT", "/intro-course/api/course_phase/"+suite.coursePhaseID.String()+"/infrastructure/gitlab/student-setup/"+courseParticipationID.String()+"/manual", nil)
	resp := httptest.NewRecorder()

	suite.router.ServeHTTP(resp, req)

	assert.Equal(suite.T(), http.StatusOK, resp.Code)
}
