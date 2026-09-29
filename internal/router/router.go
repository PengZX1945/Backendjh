package router

import (
	"Backendjh/internal/handler"
	"Backendjh/internal/middleware"

	"github.com/gin-gonic/gin"
)

func Setup(r *gin.Engine, uh *handler.UserHandler, uph *handler.UploadHandler, jwtSecret string) {
	userpub := r.Group("/api/auth")
	userpub.POST("/register", uh.Register)
	userpub.POST("/login", uh.Login)

	userpri := r.Group("/api/auth", middleware.Auth(jwtSecret))
	userpri.POST("/logout", uh.Logout)
	userpri.GET("/profile", uh.GetProfile)
	userpri.PUT("/profile", uh.UpdateProfile)
	userpri.PUT("/password", uh.ChangePassword)

	upload := r.Group("/api/upload", middleware.Auth(jwtSecret))
	upload.POST("/", uph.UploadFile)
}
