package team

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/prompt-edu/prompt-intro-course/server/team/teamDTO"
	promptSDK "github.com/prompt-edu/prompt-sdk"
	"github.com/prompt-edu/prompt-sdk/keycloakTokenVerifier"
	"github.com/prompt-edu/prompt-sdk/promptTypes"
	log "github.com/sirupsen/logrus"
)

// The routes serve the inter-phase communication protocol: the phase level "teams" output
// and the participation level "teamAllocation" output, in the shape of the team allocation phase.
func setupTeamRouter(router *gin.RouterGroup, authMiddleware func(allowedRoles ...string) gin.HandlerFunc) {
	readRoles := []string{promptSDK.PromptAdmin, promptSDK.CourseLecturer, promptSDK.CourseEditor, promptSDK.CourseStudent}
	// Confines a tutor (editor registered as a tutor of this phase) to their own team.
	scopingMW := promptSDK.TutorScopingMiddleware(tutorTeamResolver{})

	teamRouter := router.Group("/team")
	teamRouter.GET("", authMiddleware(readRoles...), scopingMW, getTeams)

	allocationRouter := router.Group("/allocation")
	allocationRouter.GET("", authMiddleware(readRoles...), scopingMW, getAllocations)
	allocationRouter.GET("/:courseParticipationID", authMiddleware(readRoles...), scopingMW, getAllocation)
}

// getTeams godoc
// @Summary Get tutor teams
// @Description Returns one team per tutor with the students seated in that tutor's seats. The team ID is the tutor's ID. A tutor only sees their own team.
// @Tags team
// @Produce json
// @Param coursePhaseID path string true "Course Phase UUID"
// @Success 200 {object} map[string][]promptTypes.Team
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security ApiKeyAuth
// @Router /course_phase/{coursePhaseID}/team [get]
func getTeams(c *gin.Context) {
	coursePhaseID, err := uuid.Parse(c.Param("coursePhaseID"))
	if err != nil {
		log.Error("Error parsing coursePhaseID: ", err)
		handleError(c, http.StatusBadRequest, err)
		return
	}

	// Students cannot read core's participation list, so they get the names cached by
	// the seat import or the last staff read instead.
	if canReadCoreParticipations(c) {
		if err := RefreshParticipantNamesIfStale(c, c.GetHeader("Authorization"), coursePhaseID); err != nil {
			log.WithError(err).Warn("Failed to refresh participant names, serving cached names")
		}
	}

	teams, err := GetTeams(c, coursePhaseID)
	if err != nil {
		handleError(c, http.StatusInternalServerError, err)
		return
	}

	if tutorTeamID, scoped := promptSDK.GetTutorTeamID(c); scoped {
		teams = filterTeamsByID(teams, tutorTeamID)
	}
	c.JSON(http.StatusOK, gin.H{"teams": teams})
}

// getAllocations godoc
// @Summary Get team allocations
// @Description Returns the team (tutor) ID of every seated student. A tutor only sees their own team's students.
// @Tags team
// @Produce json
// @Param coursePhaseID path string true "Course Phase UUID"
// @Success 200 {array} teamDTO.AllocationWithParticipation
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security ApiKeyAuth
// @Router /course_phase/{coursePhaseID}/allocation [get]
func getAllocations(c *gin.Context) {
	coursePhaseID, err := uuid.Parse(c.Param("coursePhaseID"))
	if err != nil {
		log.Error("Error parsing coursePhaseID: ", err)
		handleError(c, http.StatusBadRequest, err)
		return
	}

	allocations, err := GetAllocations(c, coursePhaseID)
	if err != nil {
		handleError(c, http.StatusInternalServerError, err)
		return
	}

	if tutorTeamID, scoped := promptSDK.GetTutorTeamID(c); scoped {
		allocations = filterAllocationsByTeam(allocations, tutorTeamID)
	}
	c.JSON(http.StatusOK, allocations)
}

// getAllocation godoc
// @Summary Get team allocation of a participant
// @Description Returns the team (tutor) ID of the student's seat.
// @Tags team
// @Produce json
// @Param coursePhaseID path string true "Course Phase UUID"
// @Param courseParticipationID path string true "Course Participation UUID"
// @Success 200 {object} teamDTO.Allocation
// @Failure 400 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security ApiKeyAuth
// @Router /course_phase/{coursePhaseID}/allocation/{courseParticipationID} [get]
func getAllocation(c *gin.Context) {
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

	teamID, err := GetAllocation(c, coursePhaseID, courseParticipationID)
	if errors.Is(err, ErrAllocationNotFound) {
		handleError(c, http.StatusNotFound, err)
		return
	}
	if err != nil {
		handleError(c, http.StatusInternalServerError, err)
		return
	}

	if tutorTeamID, scoped := promptSDK.GetTutorTeamID(c); scoped && teamID != tutorTeamID {
		c.JSON(http.StatusForbidden, gin.H{"error": "access restricted to assigned team"})
		return
	}
	c.JSON(http.StatusOK, teamDTO.Allocation{TeamAllocation: teamID})
}

func filterTeamsByID(teams []promptTypes.Team, teamID uuid.UUID) []promptTypes.Team {
	for _, team := range teams {
		if team.ID == teamID {
			return []promptTypes.Team{team}
		}
	}
	return []promptTypes.Team{}
}

func filterAllocationsByTeam(allocations []teamDTO.AllocationWithParticipation, teamID uuid.UUID) []teamDTO.AllocationWithParticipation {
	result := make([]teamDTO.AllocationWithParticipation, 0)
	for _, allocation := range allocations {
		if allocation.TeamAllocation == teamID {
			result = append(result, allocation)
		}
	}
	return result
}

// canReadCoreParticipations mirrors the roles core admits to a phase's participation list.
func canReadCoreParticipations(c *gin.Context) bool {
	tokenUser, ok := keycloakTokenVerifier.GetTokenUser(c)
	if !ok {
		return false
	}
	return tokenUser.Roles[promptSDK.PromptAdmin] || tokenUser.IsLecturer || tokenUser.IsEditor
}

func handleError(c *gin.Context, statusCode int, err error) {
	c.JSON(statusCode, gin.H{"error": err.Error()})
}
