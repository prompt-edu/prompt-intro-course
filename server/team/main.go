package team

import (
	"github.com/gin-gonic/gin"
	db "github.com/prompt-edu/prompt-intro-course/server/db/sqlc"
	promptSDK "github.com/prompt-edu/prompt-sdk"
)

func InitTeamModule(routerGroup *gin.RouterGroup, queries db.Queries) {
	setupTeamRouter(routerGroup, promptSDK.AuthenticationMiddleware)
	TeamServiceSingleton = &TeamService{
		queries: queries,
	}
}
