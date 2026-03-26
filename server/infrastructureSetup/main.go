package infrastructureSetup

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	db "github.com/prompt-edu/prompt-intro-course/server/db/sqlc"
	promptSDK "github.com/prompt-edu/prompt-sdk"
	log "github.com/sirupsen/logrus"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

func InitInfrastructureModule(routerGroup *gin.RouterGroup, queries db.Queries, conn *pgxpool.Pool, gitlabAccessToken string) {
	setupInfrastructureRouter(routerGroup, promptSDK.AuthenticationMiddleware)

	var gitlabClient *gitlab.Client
	if gitlabAccessToken != "" {
		client, err := gitlab.NewClient(gitlabAccessToken, gitlab.WithBaseURL("https://gitlab.lrz.de/api/v4"))
		if err != nil {
			log.Errorf("Failed to create GitLab client: %v — GitLab operations will fail", err)
		} else {
			log.Info("GitLab client initialized")
			gitlabClient = client
		}
	} else {
		log.Warn("GITLAB_ACCESS_TOKEN not set — GitLab operations will fail")
	}

	InfrastructureServiceSingleton = &InfrastructureService{
		queries:      queries,
		conn:         conn,
		gitlabClient: gitlabClient,
	}
}
