package tutor

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	promptSDK "github.com/prompt-edu/prompt-sdk"
	db "github.com/prompt-edu/prompt-intro-course/server/db/sqlc"
)

// name has to be the same constant as in the corresponding micro frontend
const KEYCLOAK_GROUP_NAME = "introCourseTutors"

func InitTutorModule(routerGroup *gin.RouterGroup, queries db.Queries, conn *pgxpool.Pool) {
	setupTutorRouter(routerGroup, promptSDK.AuthenticationMiddleware)
	TutorServiceSingleton = &TutorService{
		queries: queries,
		conn:    conn,
	}
}
