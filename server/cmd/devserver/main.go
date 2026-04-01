// Development server with no-auth middleware for E2E screenshots.
// Usage: go run ./cmd/devserver
package main

import (
	"context"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	db "github.com/prompt-edu/prompt-intro-course/server/db/sqlc"
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

	router := gin.Default()
	router.Use(utils.CORS())

	api := router.Group("intro-course/api/course_phase/:coursePhaseID")

	// Seat plan routes (no auth)
	seatGroup := api.Group("/seat_plan")
	seatGroup.GET("", noAuthMiddleware(), getSeatPlanHandler(query))
	seatGroup.PUT("", noAuthMiddleware(), updateSeatPlanHandler(query, conn))
	seatGroup.GET("/own-assignment", noAuthMiddleware(), getOwnSeatAssignmentHandler(query))

	// Tutor routes (no auth)
	tutorGroup := api.Group("/tutor")
	tutorGroup.GET("", noAuthMiddleware(), getTutorsHandler(query))

	// Peer assignment routes (no auth)
	peerGroup := api.Group("/peer_assignments")
	peerGroup.GET("", noAuthMiddleware(), getPeerAssignmentsHandler(query))
	peerGroup.GET("/own", noAuthMiddleware(), getOwnPeerAssignmentsHandler(query))

	// Developer profile routes (no auth)
	devGroup := api.Group("/developer-profile")
	devGroup.GET("/all", noAuthMiddleware(), getAllDeveloperProfilesHandler(query))

	serverAddress := utils.GetEnv("SERVER_ADDRESS", "localhost:8082")
	log.Infof("Dev server (no-auth) started on %s", serverAddress)
	if err := router.Run(serverAddress); err != nil {
		log.Fatalf("Failed to start: %v", err)
	}
}
