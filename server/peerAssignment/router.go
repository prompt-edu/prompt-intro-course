package peerAssignment

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/prompt-edu/prompt-intro-course/server/peerAssignment/peerAssignmentDTO"
	promptSDK "github.com/prompt-edu/prompt-sdk"
	log "github.com/sirupsen/logrus"
)

func setupPeerAssignmentRouter(router *gin.RouterGroup, authMiddleware func(allowedRoles ...string) gin.HandlerFunc) {
	peerRouter := router.Group("/peer-assignments")

	peerRouter.GET("", authMiddleware(promptSDK.PromptAdmin, promptSDK.CourseLecturer), getPeerAssignments)
	peerRouter.POST("/generate", authMiddleware(promptSDK.PromptAdmin, promptSDK.CourseLecturer), generatePeerAssignments)
	peerRouter.PUT("", authMiddleware(promptSDK.PromptAdmin, promptSDK.CourseLecturer), updatePeerAssignments)
	peerRouter.DELETE("", authMiddleware(promptSDK.PromptAdmin, promptSDK.CourseLecturer), deletePeerAssignments)
	peerRouter.POST("/sync-gitlab", authMiddleware(promptSDK.PromptAdmin, promptSDK.CourseLecturer), syncPeerAssignmentsToGitlab)

	peerRouter.GET("/own", authMiddleware(promptSDK.CourseStudent), getOwnPeerAssignment)
}

// getPeerAssignments godoc
// @Summary Get all peer assignments
// @Description Returns all peer assignments for the course phase.
// @Tags peer-assignments
// @Produce json
// @Param coursePhaseID path string true "Course Phase UUID"
// @Success 200 {array} peerAssignmentDTO.PeerAssignment
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security ApiKeyAuth
// @Router /course_phase/{coursePhaseID}/peer-assignments [get]
func getPeerAssignments(c *gin.Context) {
	coursePhaseID, err := uuid.Parse(c.Param("coursePhaseID"))
	if err != nil {
		handleError(c, http.StatusBadRequest, err)
		return
	}

	assignments, err := GetAllPeerAssignments(c, coursePhaseID)
	if err != nil {
		handleError(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, assignments)
}

// generatePeerAssignments godoc
// @Summary Generate peer assignments
// @Description Auto-generates peer pairs within tutor groups.
// @Tags peer-assignments
// @Produce json
// @Param coursePhaseID path string true "Course Phase UUID"
// @Success 201 {array} peerAssignmentDTO.PeerAssignment
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security ApiKeyAuth
// @Router /course_phase/{coursePhaseID}/peer-assignments/generate [post]
func generatePeerAssignments(c *gin.Context) {
	coursePhaseID, err := uuid.Parse(c.Param("coursePhaseID"))
	if err != nil {
		handleError(c, http.StatusBadRequest, err)
		return
	}

	assignments, err := GeneratePeerAssignments(c, coursePhaseID)
	if err != nil {
		handleError(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusCreated, assignments)
}

// updatePeerAssignments godoc
// @Summary Update peer assignments
// @Description Replaces all peer assignments with the provided set.
// @Tags peer-assignments
// @Accept json
// @Produce json
// @Param coursePhaseID path string true "Course Phase UUID"
// @Param request body []peerAssignmentDTO.PeerAssignment true "Peer assignments"
// @Success 200
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security ApiKeyAuth
// @Router /course_phase/{coursePhaseID}/peer-assignments [put]
func updatePeerAssignments(c *gin.Context) {
	coursePhaseID, err := uuid.Parse(c.Param("coursePhaseID"))
	if err != nil {
		handleError(c, http.StatusBadRequest, err)
		return
	}

	var assignments []peerAssignmentDTO.PeerAssignment
	if err := c.BindJSON(&assignments); err != nil {
		handleError(c, http.StatusBadRequest, err)
		return
	}

	err = UpdatePeerAssignments(c, coursePhaseID, assignments)
	if err != nil {
		handleError(c, http.StatusInternalServerError, err)
		return
	}

	c.Status(http.StatusOK)
}

// deletePeerAssignments godoc
// @Summary Delete all peer assignments
// @Description Removes all peer assignments for the course phase.
// @Tags peer-assignments
// @Param coursePhaseID path string true "Course Phase UUID"
// @Success 200
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security ApiKeyAuth
// @Router /course_phase/{coursePhaseID}/peer-assignments [delete]
func deletePeerAssignments(c *gin.Context) {
	coursePhaseID, err := uuid.Parse(c.Param("coursePhaseID"))
	if err != nil {
		handleError(c, http.StatusBadRequest, err)
		return
	}

	err = DeletePeerAssignments(c, coursePhaseID)
	if err != nil {
		handleError(c, http.StatusInternalServerError, err)
		return
	}

	c.Status(http.StatusOK)
}

// syncPeerAssignmentsToGitlab godoc
// @Summary Sync peer assignments to GitLab
// @Description Adds peers as Reporter members and creates approval rules on GitLab.
// @Tags peer-assignments
// @Accept json
// @Produce json
// @Param coursePhaseID path string true "Course Phase UUID"
// @Param request body peerAssignmentDTO.SyncRequest true "Sync request with semester tag"
// @Success 200 {array} peerAssignmentDTO.SyncResult
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security ApiKeyAuth
// @Router /course_phase/{coursePhaseID}/peer-assignments/sync-gitlab [post]
func syncPeerAssignmentsToGitlab(c *gin.Context) {
	coursePhaseID, err := uuid.Parse(c.Param("coursePhaseID"))
	if err != nil {
		handleError(c, http.StatusBadRequest, err)
		return
	}

	var req peerAssignmentDTO.SyncRequest
	if err := c.BindJSON(&req); err != nil {
		handleError(c, http.StatusBadRequest, err)
		return
	}

	semesterTag := strings.ToUpper(req.SemesterTag)
	results, err := SyncPeerAssignmentsToGitlab(c, coursePhaseID, semesterTag)
	if err != nil {
		handleError(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, results)
}

// getOwnPeerAssignment godoc
// @Summary Get own peer assignment
// @Description Returns the peer assignments for the authenticated student.
// @Tags peer-assignments
// @Produce json
// @Param coursePhaseID path string true "Course Phase UUID"
// @Success 200 {object} peerAssignmentDTO.OwnPeerAssignment
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security ApiKeyAuth
// @Router /course_phase/{coursePhaseID}/peer-assignments/own [get]
func getOwnPeerAssignment(c *gin.Context) {
	coursePhaseID, err := uuid.Parse(c.Param("coursePhaseID"))
	if err != nil {
		handleError(c, http.StatusBadRequest, err)
		return
	}

	courseParticipationID, ok := c.Get("courseParticipationID")
	if !ok {
		log.Error("Error getting courseParticipationID from context")
		handleError(c, http.StatusInternalServerError, errors.New("error getting courseParticipationID from context"))
		return
	}

	assignment, err := GetOwnPeerAssignment(c, coursePhaseID, courseParticipationID.(uuid.UUID))
	if err != nil {
		handleError(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, assignment)
}

func handleError(c *gin.Context, statusCode int, err error) {
	c.JSON(statusCode, gin.H{"error": err.Error()})
}
