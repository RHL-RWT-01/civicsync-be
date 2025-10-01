package routes

import (
	"civicsync-be/controllers"
	"civicsync-be/middlewares"
	"time"
	"github.com/gin-gonic/gin"
)

// IssueRoutes sets up the issue routes
func IssueRoutes(r *gin.Engine) {
	issue := r.Group("/api/issue")
	{
		issue.POST("/create", middlewares.AuthMiddleware(), middlewares.IssueRateLimiter(2, time.Hour*24), controllers.CreateIssue)
		issue.GET("/:id", controllers.GetIssue)
		issue.GET("/issues", controllers.GetAllIssues)
		issue.GET("/user", middlewares.AuthMiddleware(), controllers.GetIssuesByUser)
		issue.PATCH("/update/:id", middlewares.AuthMiddleware(), controllers.UpdateIssue)
		issue.DELETE("/delete/:id", middlewares.AuthMiddleware(), controllers.DeleteIssue)
		issue.POST("/vote/:id", middlewares.AuthMiddleware(), controllers.HandleVoteOnIssue)
		issue.GET("/analytics", controllers.GetIssueAnalytics)
		issue.GET("/recent-issues", controllers.RecentIssues)
	}
}
