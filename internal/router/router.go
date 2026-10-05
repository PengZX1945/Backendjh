package router

import (
	"Backendjh/internal/handler"
	"Backendjh/internal/middleware"

	"github.com/gin-gonic/gin"
)

func Setup(r *gin.Engine, uh *handler.UserHandler, uph *handler.UploadHandler, ih *handler.ItemHandler, jwtSecret string) {
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


	itempub := r.Group("/api/items")
	itempub.GET("/list/:type/", ih.List)

	itemopt := r.Group("/api/items", middleware.OptionalAuth(jwtSecret))
	itemopt.GET("/:item_id", ih.Detail)

	itempri := r.Group("/api/items", middleware.Auth(jwtSecret))
	itempri.POST("/:type", ih.Publish)
	itempri.POST("/:type/close", ih.Close)
	itempri.PUT("/:item_id", ih.Update)
	itempri.DELETE("/:item_id", ih.Delete)

	my := r.Group("/api/my", middleware.Auth(jwtSecret))
	my.GET("/items", ih.MyItem)
}
