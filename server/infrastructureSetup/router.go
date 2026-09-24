package infrastructureSetup

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/prompt-edu/prompt-intro-course/server/coreRequests"
	"github.com/prompt-edu/prompt-intro-course/server/gitlabutil"
	"github.com/prompt-edu/prompt-intro-course/server/infrastructureSetup/infrastructureDTO"
	"github.com/prompt-edu/prompt-intro-course/server/utils"
	promptSDK "github.com/prompt-edu/prompt-sdk"
	log "github.com/sirupsen/logrus"
)

func setupInfrastructureRouter(router *gin.RouterGroup, authMiddleware func(allowedRoles ...string) gin.HandlerFunc) {
	infrastructureRouter := router.Group("/infrastructure")

	// Infrastructure setup routes
	infrastructureRouter.POST("/gitlab/course-setup", authMiddleware(promptSDK.PromptAdmin, promptSDK.CourseLecturer), createCourseSetup)
	infrastructureRouter.GET("/gitlab/course-setup", authMiddleware(promptSDK.PromptAdmin, promptSDK.CourseLecturer), getCourseSetup)
	infrastructureRouter.POST("/gitlab/demo/reset", authMiddleware(promptSDK.PromptAdmin, promptSDK.CourseLecturer), resetDemo)
	infrastructureRouter.POST("/gitlab/student-setup/:courseParticipationID", authMiddleware(promptSDK.PromptAdmin, promptSDK.CourseLecturer), setupStudentInfrastructure)

	// Infrastructure status routes
	infrastructureRouter.GET("/gitlab/student-setup", authMiddleware(promptSDK.PromptAdmin, promptSDK.CourseLecturer), getAllStudentGitlabStatus)

	// Route for manually overwriting the status (i.e. if instructor manually created or fixed the repo)
	infrastructureRouter.PUT("/gitlab/student-setup/:courseParticipationID/manual", authMiddleware(promptSDK.PromptAdmin, promptSDK.CourseLecturer), manuallyOverwriteStudentGitlabStatus)
}

func resetDemo(c *gin.Context) {
	coursePhaseID, err := uuid.Parse(c.Param("coursePhaseID"))
	if err != nil {
		handleError(c, http.StatusBadRequest, err)
		return
	}
	var request resetDemoRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		handleError(c, http.StatusBadRequest, err)
		return
	}
	semesterTag, ok := requireSemesterTagForPhase(c, coursePhaseID, request.SemesterTag)
	if !ok {
		return
	}
	request.SemesterTag = semesterTag
	result, err := ResetDemo(c.Request.Context(), coursePhaseID, request)
	if err != nil {
		handleError(c, http.StatusConflict, err)
		return
	}
	c.JSON(http.StatusCreated, result)
}

func getCourseSetup(c *gin.Context) {
	coursePhaseID, err := uuid.Parse(c.Param("coursePhaseID"))
	if err != nil {
		handleError(c, http.StatusBadRequest, err)
		return
	}
	semesterTag, ok := requireSemesterTagForPhase(c, coursePhaseID, c.Query("semesterTag"))
	if !ok {
		return
	}
	status, err := CourseInfrastructureStatus(c.Request.Context(), coursePhaseID, semesterTag)
	if err != nil {
		handleError(c, http.StatusBadGateway, err)
		return
	}
	c.JSON(http.StatusOK, status)
}

// createCourseSetup godoc
// @Summary Create course infrastructure
// @Description Creates the GitLab course infrastructure for the course phase.
// @Tags infrastructure
// @Accept json
// @Produce json
// @Param coursePhaseID path string true "Course Phase UUID"
// @Param request body infrastructureDTO.CreateCourseInfrastructureRequest true "Course infrastructure request"
// @Success 201
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security ApiKeyAuth
// @Router /course_phase/{coursePhaseID}/infrastructure/gitlab/course-setup [post]
func createCourseSetup(c *gin.Context) {
	coursePhaseID, err := uuid.Parse(c.Param("coursePhaseID"))
	if err != nil {
		log.Error("Error parsing coursePhaseID: ", err)
		handleError(c, http.StatusBadRequest, err)
		return
	}

	// get semester tag (= top level group name)
	var infrastructureRequest infrastructureDTO.CreateCourseInfrastructureRequest
	if err := c.BindJSON(&infrastructureRequest); err != nil {
		handleError(c, http.StatusBadRequest, err)
		return
	}

	semesterTag, ok := requireSemesterTagForPhase(c, coursePhaseID, infrastructureRequest.SemesterTag)
	if !ok {
		return
	}

	if err := CreateCourseInfrastructure(c.Request.Context(), coursePhaseID, semesterTag); err != nil {
		handleError(c, http.StatusInternalServerError, err)
		return
	}

	c.Status(http.StatusCreated)
}

// setupStudentInfrastructure godoc
// @Summary Create student infrastructure
// @Description Creates the GitLab repository for a student.
// @Tags infrastructure
// @Accept json
// @Produce json
// @Param coursePhaseID path string true "Course Phase UUID"
// @Param courseParticipationID path string true "Course Participation UUID"
// @Param request body infrastructureDTO.CreateStudentRepo true "Student repository request"
// @Success 201
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security ApiKeyAuth
// @Router /course_phase/{coursePhaseID}/infrastructure/gitlab/student-setup/{courseParticipationID} [post]
func setupStudentInfrastructure(c *gin.Context) {
	coursePhaseID, err := uuid.Parse(c.Param("coursePhaseID"))
	if err != nil {
		log.Error("Error parsing coursePhaseID: ", err)
		handleError(c, http.StatusBadRequest, err)
		return
	}

	courseParticipationID, err := uuid.Parse(c.Param("courseParticipationID"))
	if err != nil {
		log.Error("Error parsing courseParticipationID: ", err)
		handleError(c, http.StatusBadRequest, err)
		return
	}

	// get semester tag (= top level group name)
	var infrastructureRequest infrastructureDTO.CreateStudentRepo
	if err := c.BindJSON(&infrastructureRequest); err != nil {
		handleError(c, http.StatusBadRequest, err)
		return
	}

	semesterTag, ok := requireSemesterTagForPhase(c, coursePhaseID, infrastructureRequest.SemesterTag)
	if !ok {
		return
	}
	participations, err := coreRequests.GetCoursePhaseParticipations(c.GetHeader("Authorization"), coursePhaseID)
	if err != nil {
		handleError(c, http.StatusBadGateway, fmt.Errorf("check participation status: %w", err))
		return
	}
	allowed := false
	for _, participation := range participations {
		if participation.CourseParticipationID == courseParticipationID.String() {
			allowed = participation.PassStatus != "" && !strings.EqualFold(participation.PassStatus, "failed")
			break
		}
	}
	if !allowed {
		handleError(c, http.StatusForbidden, fmt.Errorf("student repository cannot be created for a failed or missing course participation"))
		return
	}

	err = CreateStudentInfrastructure(c, coursePhaseID, courseParticipationID, semesterTag, infrastructureRequest.RepoName, infrastructureRequest.StudentName, infrastructureRequest.SubmissionDeadline)
	if err != nil {
		handleError(c, http.StatusInternalServerError, err)
		return
	}

	c.Status(http.StatusCreated)

}

// getAllStudentGitlabStatus godoc
// @Summary Get student GitLab status
// @Description Returns GitLab setup status for all course participations.
// @Tags infrastructure
// @Produce json
// @Param coursePhaseID path string true "Course Phase UUID"
// @Success 200 {array} infrastructureDTO.GitlabStatus
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security ApiKeyAuth
// @Router /course_phase/{coursePhaseID}/infrastructure/gitlab/student-setup [get]
func getAllStudentGitlabStatus(c *gin.Context) {
	coursePhaseID, err := uuid.Parse(c.Param("coursePhaseID"))
	if err != nil {
		log.Error("Error parsing coursePhaseID: ", err)
		handleError(c, http.StatusBadRequest, err)
		return
	}

	studentInfrastructureStatus, err := GetAllStudentGitlabStatus(c, coursePhaseID)
	if err != nil {
		handleError(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, studentInfrastructureStatus)
}

// manuallyOverwriteStudentGitlabStatus godoc
// @Summary Manually overwrite student GitLab status
// @Description Marks GitLab setup as completed for a student.
// @Tags infrastructure
// @Produce json
// @Param coursePhaseID path string true "Course Phase UUID"
// @Param courseParticipationID path string true "Course Participation UUID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security ApiKeyAuth
// @Router /course_phase/{coursePhaseID}/infrastructure/gitlab/student-setup/{courseParticipationID}/manual [put]
func manuallyOverwriteStudentGitlabStatus(c *gin.Context) {
	coursePhaseID, err := uuid.Parse(c.Param("coursePhaseID"))
	if err != nil {
		log.Error("Error parsing coursePhaseID: ", err)
		handleError(c, http.StatusBadRequest, err)
		return
	}

	courseParticipationID, err := uuid.Parse(c.Param("courseParticipationID"))
	if err != nil {
		log.Error("Error parsing courseParticipationID: ", err)
		handleError(c, http.StatusBadRequest, err)
		return
	}

	err = ManuallyOverwriteStudentGitlabStatus(c, coursePhaseID, courseParticipationID)
	if err != nil {
		handleError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Successfully overwritten student gitlab status"})
}

func handleError(c *gin.Context, statusCode int, err error) {
	c.JSON(statusCode, gin.H{"error": err.Error()})
}

// The semester group is supplied by the client, but must belong to the course
// that owns this phase before it is used to read or mutate GitLab projects.
func requireSemesterTagForPhase(c *gin.Context, coursePhaseID uuid.UUID, suppliedTag string) (string, bool) {
	semesterTag := gitlabutil.CourseGroupName(suppliedTag)
	if semesterTag == "" {
		handleError(c, http.StatusBadRequest, fmt.Errorf("semesterTag is required"))
		return "", false
	}

	expectedTag, err := courseSemesterTag(c.Request.Context(), c.GetHeader("Authorization"), coursePhaseID)
	if err != nil {
		handleError(c, http.StatusBadGateway, fmt.Errorf("verify course semester: %w", err))
		return "", false
	}
	if semesterTag != expectedTag {
		handleError(c, http.StatusBadRequest, fmt.Errorf("semesterTag does not belong to this course phase"))
		return "", false
	}
	return semesterTag, true
}

func courseSemesterTag(ctx context.Context, authHeader string, coursePhaseID uuid.UUID) (string, error) {
	var phase struct {
		ID       uuid.UUID `json:"id"`
		CourseID uuid.UUID `json:"courseID"`
	}
	if err := getCoreResource(ctx, authHeader, "/api/course_phases/"+coursePhaseID.String(), &phase); err != nil {
		return "", fmt.Errorf("get course phase: %w", err)
	}
	if phase.ID != coursePhaseID || phase.CourseID == uuid.Nil {
		return "", fmt.Errorf("core returned an invalid course phase")
	}

	var course struct {
		ID          uuid.UUID `json:"id"`
		SemesterTag string    `json:"semesterTag"`
	}
	if err := getCoreResource(ctx, authHeader, "/api/courses/"+phase.CourseID.String(), &course); err != nil {
		return "", fmt.Errorf("get course: %w", err)
	}
	tag := gitlabutil.CourseGroupName(course.SemesterTag)
	if course.ID != phase.CourseID || tag == "" {
		return "", fmt.Errorf("core returned an invalid course semester")
	}
	return tag, nil
}

func getCoreResource(ctx context.Context, authHeader, path string, target any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(utils.GetCoreUrl(), "/")+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", authHeader)
	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("core returned %s", resp.Status)
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(target); err != nil {
		return fmt.Errorf("decode core response: %w", err)
	}
	return nil
}
