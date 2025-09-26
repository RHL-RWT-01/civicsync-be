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
		auth.POST("/register", authController.RegisterUser)
		auth.POST("/login", authController.LoginUser)
		auth.GET("/me", middlewares.AuthMiddleware(), authController.GetMe)
	}
}
