package copy

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	promptSDK "github.com/prompt-edu/prompt-sdk"
	db "github.com/prompt-edu/prompt-intro-course/server/db/sqlc"
)

// AuditCopyAction names the copy route in the audit log, so the entry says what
// was copied instead of the derived "Created copy".
const AuditCopyAction = "Copied intro course phase"

func InitCopyModule(routerGroup *gin.RouterGroup, queries db.Queries, conn *pgxpool.Pool) {
	setupCopyRouter(routerGroup, promptSDK.AuthenticationMiddleware)
	CopyServiceSingleton = &CopyService{
		queries: queries,
		conn:    conn,
	}
}
