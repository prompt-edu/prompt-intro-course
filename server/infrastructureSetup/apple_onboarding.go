package infrastructureSetup

import (
	"fmt"
	"net/http"
	"net/mail"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/prompt-edu/prompt-intro-course/server/appleTeam"
	"github.com/prompt-edu/prompt-intro-course/server/coreRequests"
	"github.com/prompt-edu/prompt-intro-course/server/developerProfile"
	"github.com/prompt-edu/prompt-intro-course/server/developerProfile/developerProfileDTO"
)

// The Apple key is team-wide. Resolve every write from a saved profile and a
// currently eligible course participation; never accept an address or UDID
// supplied by the caller of this endpoint.
func appleOnboardingProfile(c *gin.Context) (developerProfileDTO.DeveloperProfile, coreRequests.Participation, bool) {
	phaseID, phaseErr := uuid.Parse(c.Param("coursePhaseID"))
	participationID, participationErr := uuid.Parse(c.Param("courseParticipationID"))
	if phaseErr != nil || participationErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid course participation"})
		return developerProfileDTO.DeveloperProfile{}, coreRequests.Participation{}, false
	}
	participations, err := coreRequests.GetCoursePhaseParticipations(c.GetHeader("Authorization"), phaseID)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "Could not verify course participation"})
		return developerProfileDTO.DeveloperProfile{}, coreRequests.Participation{}, false
	}
	for _, participation := range participations {
		if participation.CourseParticipationID != participationID.String() {
			continue
		}
		if participation.PassStatus == "" || strings.EqualFold(participation.PassStatus, "failed") {
			c.JSON(http.StatusForbidden, gin.H{"error": "Apple onboarding is unavailable for this participation"})
			return developerProfileDTO.DeveloperProfile{}, coreRequests.Participation{}, false
		}
		profile, err := developerProfile.GetOwnDeveloperProfile(c.Request.Context(), phaseID, participationID)
		if err != nil || profile.CourseParticipationID == uuid.Nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Save the developer profile first"})
			return developerProfileDTO.DeveloperProfile{}, coreRequests.Participation{}, false
		}
		return profile, participation, true
	}
	c.JSON(http.StatusNotFound, gin.H{"error": "Course participation not found"})
	return developerProfileDTO.DeveloperProfile{}, coreRequests.Participation{}, false
}

func inviteAppleDeveloper(c *gin.Context) {
	profile, participation, ok := appleOnboardingProfile(c)
	if !ok {
		return
	}
	email := strings.TrimSpace(profile.AppleID)
	parsed, err := mail.ParseAddress(email)
	if email == "" || err != nil || parsed.Address != email {
		c.JSON(http.StatusConflict, gin.H{"error": "Save a valid Apple Account email in the developer profile first"})
		return
	}
	client, err := appleTeam.NewFromEnvironment()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Apple team connection is not configured"})
		return
	}
	created, err := client.InviteDeveloper(c.Request.Context(), email,
		strings.TrimSpace(participation.Student.FirstName), strings.TrimSpace(participation.Student.LastName))
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"created": created})
}

func registerAppleDevice(c *gin.Context) {
	profile, participation, ok := appleOnboardingProfile(c)
	if !ok {
		return
	}
	kind := c.Param("kind")
	var udid string
	switch kind {
	case "iphone":
		udid = profile.IPhoneUDID.String
	case "ipad":
		udid = profile.IPadUDID.String
	case "watch":
		udid = profile.AppleWatchUDID.String
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unknown device type"})
		return
	}
	udid = strings.TrimSpace(udid)
	if udid == "" {
		c.JSON(http.StatusConflict, gin.H{"error": "Save this device UDID in the developer profile first"})
		return
	}
	client, err := appleTeam.NewFromEnvironment()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Apple team connection is not configured"})
		return
	}
	// A readable device name identifies the owner in Apple Developer without
	// storing additional data in PROMPT.
	name := fmt.Sprintf("Intro Course %s %s %s", participation.Student.FirstName, participation.Student.LastName, kind)
	created, err := client.RegisterDevice(c.Request.Context(), udid, name, kind)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"created": created})
}
