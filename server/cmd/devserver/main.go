// Development server with no-auth middleware for E2E screenshots.
// Usage: go run ./cmd/devserver
package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	db "github.com/prompt-edu/prompt-intro-course/server/db/sqlc"
	"github.com/prompt-edu/prompt-intro-course/server/infrastructureSetup"
	"github.com/prompt-edu/prompt-intro-course/server/peerAssignment"
	"github.com/prompt-edu/prompt-intro-course/server/utils"
	log "github.com/sirupsen/logrus"
)

// noAuthMiddleware bypasses all auth checks, setting admin roles.
func noAuthMiddleware(allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRoles := map[string]bool{
			"prompt_admin":    true,
			"course_lecturer": true,
			"course_student":  true,
		}
		c.Set("userRoles", userRoles)
		c.Set("userEmail", "admin@example.com")
		c.Set("matriculationNumber", "0000000")
		c.Set("universityLogin", "admin")
		c.Set("firstName", "Admin")
		c.Set("lastName", "User")
		// Default student for own-assignment: student 1 (Max Mueller)
		c.Set("courseParticipationID", uuid.MustParse("b0000000-0000-0000-0000-000000000001"))
		c.Next()
	}
}

// devCORS returns middleware that allows the local dev frontends.
func devCORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		allowed := []string{"http://localhost:3005", "http://localhost:3000"}
		for _, a := range allowed {
			if strings.EqualFold(origin, a) {
				c.Header("Access-Control-Allow-Origin", a)
				break
			}
		}
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		c.Header("Access-Control-Allow-Credentials", "true")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

func main() {
	dbUser := utils.GetEnv("DB_USER", "postgres")
	dbPassword := utils.GetEnv("DB_PASSWORD", "postgres")
	dbHost := utils.GetEnv("DB_HOST", "localhost")
	dbPort := utils.GetEnv("DB_PORT", "5433")
	dbName := utils.GetEnv("DB_NAME", "intro_course")
	sslMode := utils.GetEnv("SSL_MODE", "disable")
	tz := utils.GetEnv("DB_TIMEZONE", "Europe/Berlin")

	databaseURL := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s&TimeZone=%s",
		dbUser, dbPassword, dbHost, dbPort, dbName, sslMode, tz)

	conn, err := pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		log.Fatalf("DB connection failed: %v", err)
	}
	defer conn.Close()

	query := db.New(conn)

	// Load students CSV (optional — mock participations use this)
	studentsCSV := utils.GetEnv("STUDENTS_CSV", "students.csv")
	studentLoader := NewStudentLoader(studentsCSV)
	if loadErr := studentLoader.Load(); loadErr != nil {
		log.Warnf("Could not load students CSV %q: %v — mock participations will be empty", studentsCSV, loadErr)
	} else {
		log.Infof("Loaded %d students from %s", len(studentLoader.Students), studentsCSV)
	}

	router := gin.Default()
	router.Use(devCORS())

	api := router.Group("intro-course/api/course_phase/:coursePhaseID")

	// Seat plan routes (no auth)
	seatGroup := api.Group("/seat_plan")
	seatGroup.GET("", noAuthMiddleware(), getSeatPlanHandler(query))
	seatGroup.POST("", noAuthMiddleware(), createSeatPlanHandler(query, conn))
	seatGroup.PUT("", noAuthMiddleware(), updateSeatPlanHandler(query, conn))
	seatGroup.DELETE("", noAuthMiddleware(), deleteSeatPlanHandler(query))
	seatGroup.GET("/own-assignment", noAuthMiddleware(), getOwnSeatAssignmentHandler(query))

	// Tutor routes (no auth)
	tutorGroup := api.Group("/tutor")
	tutorGroup.GET("", noAuthMiddleware(), getTutorsHandler(query))
	tutorGroup.POST("/import", noAuthMiddleware(), importTutorsHandler(query, conn))
	tutorGroup.PUT("/gitlab-username", noAuthMiddleware(), updateTutorGitlabUsernameHandler(query))

	// Developer profile routes (no auth)
	devGroup := api.Group("/developer-profile")
	devGroup.GET("/all", noAuthMiddleware(), getAllDeveloperProfilesHandler(query))
	devGroup.PUT("", noAuthMiddleware(), upsertDeveloperProfileHandler(query))

	// Device routes (no auth)
	deviceGroup := api.Group("/devices")
	deviceGroup.GET("", noAuthMiddleware(), getDevicesHandler(query))

	// Real infrastructure and peer assignment modules with no-auth override
	gitlabAccessToken := utils.GetEnv("GITLAB_ACCESS_TOKEN", "")
	teachingMaterialProjectID := utils.GetEnv("GITLAB_TEACHING_MATERIAL_PROJECT_ID", "")
	infrastructureSetup.InitInfrastructureModule(api, *query, conn, gitlabAccessToken, teachingMaterialProjectID, noAuthMiddleware)
	peerAssignment.InitPeerAssignmentModule(api, *query, conn, gitlabAccessToken, noAuthMiddleware)

	// Export/Import routes
	api.GET("/export", noAuthMiddleware(), exportHandler(query))
	api.POST("/import", noAuthMiddleware(), importHandler(query, conn))

	// Mock core platform participations endpoints
	mockGroup := router.Group("/api/v2")
	mockGroup.GET("/courses/:courseId/course_phases/:phaseId/participations", noAuthMiddleware(), mockParticipationsHandler(query, studentLoader))
	mockGroup.GET("/courses/:courseId/participations/students", noAuthMiddleware(), mockStudentsHandler(studentLoader))
	mockGroup.GET("/courses/:courseId/participations/self", noAuthMiddleware(), mockSelfHandler(studentLoader))

	serverAddress := utils.GetEnv("SERVER_ADDRESS", "localhost:8082")
	log.Infof("Dev server (no-auth) started on %s", serverAddress)
	if err := router.Run(serverAddress); err != nil {
		log.Fatalf("Failed to start: %v", err)
	}
}
