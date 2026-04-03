package peerAssignment

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	promptSDK "github.com/prompt-edu/prompt-sdk"
	db "github.com/prompt-edu/prompt-intro-course/server/db/sqlc"
	"github.com/prompt-edu/prompt-intro-course/server/gitlabutil"
	log "github.com/sirupsen/logrus"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

func InitPeerAssignmentModule(routerGroup *gin.RouterGroup, queries db.Queries, conn *pgxpool.Pool, gitlabAccessToken string) {
	setupPeerAssignmentRouter(routerGroup, promptSDK.AuthenticationMiddleware)

	var gitlabClient *gitlab.Client
	if gitlabAccessToken != "" {
		client, err := gitlab.NewClient(gitlabAccessToken, gitlab.WithBaseURL(gitlabutil.GitLabBaseURL))
		if err != nil {
			log.Errorf("Failed to create GitLab client: %v — GitLab operations will fail", err)
		} else {
			log.Info("GitLab client initialized for peer assignment module")
			gitlabClient = client
		}
	} else {
		log.Warn("GITLAB_ACCESS_TOKEN not set — peer assignment GitLab operations will fail")
	}

	PeerAssignmentServiceSingleton = &PeerAssignmentService{
		queries:      queries,
		conn:         conn,
		gitlabClient: gitlabClient,
	}
}
