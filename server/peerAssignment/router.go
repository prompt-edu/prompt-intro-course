package peerAssignment

import (
	"errors"
	"net/http"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	promptSDK "github.com/prompt-edu/prompt-sdk"
	"github.com/prompt-edu/prompt-intro-course/server/peerAssignment/peerAssignmentDTO"
	log "github.com/sirupsen/logrus"
)

var semesterTagPattern = regexp.MustCompile(`^[A-Z0-9]+$`)

func setupPeerAssignmentRouter(router *gin.RouterGroup, authMiddleware func(allowedRoles ...string) gin.HandlerFunc) {
	peerRouter := router.Group("/peer_assignments")

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
// @Tags peer-assignment
// @Produce json
// @Param coursePhaseID path string true "Course Phase UUID"
// @Success 200 {array} peerAssignmentDTO.PeerAssignment
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security ApiKeyAuth
// @Router /course_phase/{coursePhaseID}/peer_assignments [get]
func getPeerAssignments(c *gin.Context) {
	coursePhaseID, err := uuid.Parse(c.Param("coursePhaseID"))
	if err != nil {
		log.Error("Error parsing coursePhaseID: ", err)
		handleError(c, http.StatusBadRequest, err)
		return
	}

	assignments, err := GetAllPeerAssignments(c, coursePhaseID)
	if err != nil {
		log.Error("Error getting peer assignments: ", err)
		handleError(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, assignments)
}

// generatePeerAssignments godoc
// @Summary Generate peer assignments
// @Description Auto-generates peer groups within tutor groups.
// @Tags peer-assignment
// @Produce json
// @Param coursePhaseID path string true "Course Phase UUID"
// @Success 201 {array} peerAssignmentDTO.PeerAssignment
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security ApiKeyAuth
// @Router /course_phase/{coursePhaseID}/peer_assignments/generate [post]
func generatePeerAssignments(c *gin.Context) {
	coursePhaseID, err := uuid.Parse(c.Param("coursePhaseID"))
	if err != nil {
		log.Error("Error parsing coursePhaseID: ", err)
		handleError(c, http.StatusBadRequest, err)
		return
	}

	assignments, err := GeneratePeerAssignments(c, coursePhaseID)
	if err != nil {
		log.Error("Error generating peer assignments: ", err)
		handleError(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusCreated, assignments)
}

// updatePeerAssignments godoc
// @Summary Update peer assignments
// @Description Replaces all peer assignments with the provided set.
// @Tags peer-assignment
// @Accept json
// @Produce json
// @Param coursePhaseID path string true "Course Phase UUID"
// @Param request body []peerAssignmentDTO.PeerAssignment true "Peer assignments"
// @Success 200
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security ApiKeyAuth
// @Router /course_phase/{coursePhaseID}/peer_assignments [put]
func updatePeerAssignments(c *gin.Context) {
	coursePhaseID, err := uuid.Parse(c.Param("coursePhaseID"))
	if err != nil {
		log.Error("Error parsing coursePhaseID: ", err)
		handleError(c, http.StatusBadRequest, err)
		return
	}

	var assignments []peerAssignmentDTO.PeerAssignment
	if err := c.BindJSON(&assignments); err != nil {
		handleError(c, http.StatusBadRequest, err)
		return
	}

	if len(assignments) == 0 {
		handleError(c, http.StatusBadRequest, errors.New("assignments list must not be empty"))
		return
	}

	// Limit payload to prevent abuse — a course phase cannot have more than 1000 assignments
	if len(assignments) > 1000 {
		handleError(c, http.StatusBadRequest, errors.New("too many assignments"))
		return
	}

	for _, a := range assignments {
		if a.StudentID == a.PeerID {
			handleError(c, http.StatusBadRequest, errors.New("self-review assignment not allowed"))
			return
		}
	}

	err = UpdatePeerAssignments(c, coursePhaseID, assignments)
	if err != nil {
		log.Error("Error updating peer assignments: ", err)
		handleError(c, http.StatusInternalServerError, err)
		return
	}

	c.Status(http.StatusOK)
}

// deletePeerAssignments godoc
// @Summary Delete all peer assignments
// @Description Removes all peer assignments for the course phase.
// @Tags peer-assignment
// @Param coursePhaseID path string true "Course Phase UUID"
// @Success 200
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security ApiKeyAuth
// @Router /course_phase/{coursePhaseID}/peer_assignments [delete]
func deletePeerAssignments(c *gin.Context) {
	coursePhaseID, err := uuid.Parse(c.Param("coursePhaseID"))
	if err != nil {
		log.Error("Error parsing coursePhaseID: ", err)
		handleError(c, http.StatusBadRequest, err)
		return
	}

	err = DeletePeerAssignments(c, coursePhaseID)
	if err != nil {
		log.Error("Error deleting peer assignments: ", err)
		handleError(c, http.StatusInternalServerError, err)
		return
	}

	c.Status(http.StatusOK)
}

// syncPeerAssignmentsToGitlab godoc
// @Summary Sync peer assignments to GitLab
// @Description Adds peers as Reporter members and creates approval rules on GitLab.
// @Tags peer-assignment
// @Accept json
// @Produce json
// @Param coursePhaseID path string true "Course Phase UUID"
// @Param request body peerAssignmentDTO.SyncRequest true "Sync request with semester tag"
// @Success 200 {array} peerAssignmentDTO.SyncResult
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security ApiKeyAuth
// @Router /course_phase/{coursePhaseID}/peer_assignments/sync-gitlab [post]
func syncPeerAssignmentsToGitlab(c *gin.Context) {
	coursePhaseID, err := uuid.Parse(c.Param("coursePhaseID"))
	if err != nil {
		log.Error("Error parsing coursePhaseID: ", err)
		handleError(c, http.StatusBadRequest, err)
		return
	}

	var req peerAssignmentDTO.SyncRequest
	if err := c.BindJSON(&req); err != nil {
		handleError(c, http.StatusBadRequest, err)
		return
	}

	semesterTag := strings.ToUpper(req.SemesterTag)
	if !semesterTagPattern.MatchString(semesterTag) {
		handleError(c, http.StatusBadRequest, errors.New("invalid semester tag format"))
		return
	}

	results, err := SyncPeerAssignmentsToGitlab(c, coursePhaseID, semesterTag)
	if err != nil {
		log.Error("Error syncing peer assignments to GitLab: ", err)
		handleError(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, results)
}

// getOwnPeerAssignment godoc
// @Summary Get own peer assignment
// @Description Returns the peer assignments for the authenticated student.
// @Tags peer-assignment
// @Produce json
// @Param coursePhaseID path string true "Course Phase UUID"
// @Success 200 {object} peerAssignmentDTO.OwnPeerAssignment
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security ApiKeyAuth
// @Router /course_phase/{coursePhaseID}/peer_assignments/own [get]
func getOwnPeerAssignment(c *gin.Context) {
	coursePhaseID, err := uuid.Parse(c.Param("coursePhaseID"))
	if err != nil {
		log.Error("Error parsing coursePhaseID: ", err)
		handleError(c, http.StatusBadRequest, err)
		return
	}

	cpID, ok := c.Get("courseParticipationID")
	if !ok {
		log.Error("Error getting courseParticipationID from context")
		handleError(c, http.StatusInternalServerError, errors.New("error getting courseParticipationID from context"))
		return
	}
	courseParticipationID, ok := cpID.(uuid.UUID)
	if !ok {
		log.Error("courseParticipationID is not a valid UUID")
		handleError(c, http.StatusInternalServerError, errors.New("invalid courseParticipationID type"))
		return
	}

	assignment, err := GetOwnPeerAssignment(c, coursePhaseID, courseParticipationID)
	if err != nil {
		log.Error("Error getting own peer assignment: ", err)
		handleError(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, assignment)
}

func handleError(c *gin.Context, statusCode int, err error) {
	c.JSON(statusCode, gin.H{"error": err.Error()})
}
