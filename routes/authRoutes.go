package routes

import (
	"civicsync-be/controllers"
	"civicsync-be/middlewares"

	"github.com/gin-gonic/gin"
)

// AuthRoutes sets up the authentication routes
func AuthRoutes(r *gin.Engine) {
	auth := r.Group("/api/auth")
	{
		auth.POST("/register", controllers.RegisterUser)
		auth.POST("/login", controllers.LoginUser)
		auth.POST("/logout", controllers.LogoutUser)
		auth.GET("/me", middlewares.AuthMiddleware(), controllers.GetMe)
	}
}
