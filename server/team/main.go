package team

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	db "github.com/prompt-edu/prompt-intro-course/server/db/sqlc"
	promptSDK "github.com/prompt-edu/prompt-sdk"
)

func InitTeamModule(routerGroup *gin.RouterGroup, queries db.Queries, conn *pgxpool.Pool) {
	setupTeamRouter(routerGroup, promptSDK.AuthenticationMiddleware)
	TeamServiceSingleton = &TeamService{
		queries: queries,
		conn:    conn,
	}
}
