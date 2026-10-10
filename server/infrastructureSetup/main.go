package infrastructureSetup

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	db "github.com/prompt-edu/prompt-intro-course/server/db/sqlc"
	"github.com/prompt-edu/prompt-intro-course/server/gitlabutil"
	promptSDK "github.com/prompt-edu/prompt-sdk"
	log "github.com/sirupsen/logrus"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

func InitInfrastructureModule(routerGroup *gin.RouterGroup, queries db.Queries, conn *pgxpool.Pool, gitlabAccessToken, teachingMaterialProjectID string, authMiddlewareOverride ...func(allowedRoles ...string) gin.HandlerFunc) {
	authMiddleware := promptSDK.AuthenticationMiddleware
	if len(authMiddlewareOverride) > 0 && authMiddlewareOverride[0] != nil {
		authMiddleware = authMiddlewareOverride[0]
	}
	setupInfrastructureRouter(routerGroup, authMiddleware)

	var gitlabClient *gitlab.Client
	if gitlabAccessToken != "" {
		client, err := gitlab.NewClient(gitlabAccessToken, gitlab.WithBaseURL(gitlabutil.GitLabBaseURL))
		if err != nil {
			log.Errorf("Failed to create GitLab client: %v — GitLab operations will fail", err)
		} else {
			log.Info("GitLab client initialized")
			gitlabClient = client
		}
	} else {
		log.Warn("GITLAB_ACCESS_TOKEN not set — GitLab operations will fail")
	}

	service := &InfrastructureService{
		queries:                   queries,
		conn:                      conn,
		gitlabClient:              gitlabClient,
		teachingMaterialProjectID: teachingMaterialProjectID,
	}
	InfrastructureServiceSingleton = service

	// Each setup request now reads one immutable teaching-material commit.
	// Startup must not pin an old revision in a process-lifetime cache.
	if teachingMaterialProjectID == "" {
		log.Warn("GITLAB_TEACHING_MATERIAL_PROJECT_ID not set — student repo setup will fail")
	}
}
