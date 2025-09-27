package routes

import (
	"civicsync-be/controllers"
	"civicsync-be/middlewares"
	"github.com/gin-gonic/gin"
)

// IssueRoutes sets up the issue routes
func IssueRoutes(r *gin.Engine) {
	issue := r.Group("/api/issue")
	{
		issue.POST("/create", middlewares.AuthMiddleware(), controllers.CreateIssue)
		issue.GET("/:id", middlewares.AuthMiddleware(), controllers.GetIssue)
		issue.PUT("/:id", middlewares.AuthMiddleware(), controllers.UpdateIssue)
		issue.DELETE("/:id", middlewares.AuthMiddleware(), controllers.DeleteIssue)
	}
}
