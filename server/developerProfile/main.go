package developerProfile

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	db "github.com/prompt-edu/prompt-intro-course/server/db/sqlc"
	"github.com/prompt-edu/prompt-intro-course/server/gitlabutil"
	promptSDK "github.com/prompt-edu/prompt-sdk"
	log "github.com/sirupsen/logrus"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

func InitDeveloperProfileModule(routerGroup *gin.RouterGroup, queries db.Queries, conn *pgxpool.Pool, gitlabAccessToken string) {
	setupDeveloperProfileRouter(routerGroup, promptSDK.AuthenticationMiddleware)
	var gitlabClient *gitlab.Client
	if gitlabAccessToken != "" {
		client, err := gitlab.NewClient(gitlabAccessToken, gitlab.WithBaseURL(gitlabutil.GitLabBaseURL))
		if err != nil {
			log.WithError(err).Error("GitLab validation is unavailable")
		} else {
			gitlabClient = client
		}
	}
	DeveloperProfileServiceSingleton = &DeveloperProfileService{
		queries:      queries,
		conn:         conn,
		gitlabClient: gitlabClient,
	}
}
