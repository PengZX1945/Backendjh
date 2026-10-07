package router

import (
	"Backendjh/internal/handler"
	"Backendjh/internal/middleware"

	"github.com/gin-gonic/gin"
)

func Setup(r *gin.Engine, uh *handler.UserHandler, uph *handler.UploadHandler, ih *handler.ItemHandler, ch *handler.ClaimHandler, jwtSecret string) {
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

	claim := r.Group("/api/claims", middleware.Auth(jwtSecret))
	claim.POST("/", ch.Submit)
	claim.GET("/:claim_id", ch.Detail)
	claim.PUT("/:claim_id", ch.Update)
	claim.DELETE("/:claim_id/", ch.Delete)

	admin := r.Group("/api/admin", middleware.Auth(jwtSecret))
	admin.GET("/claims", ch.ListClaims)
	admin.POST("/claims/:claim_id/:option", ch.Review)
	admin.GET("/items/pending/:type/", ih.PendingList)
	admin.GET("/items/", ih.AdminItemList)
	admin.POST("/items/:item_id/:option", ih.Review)
	admin.PUT("/items/:item_id/", ih.AdminClose)
	admin.GET("/users", uh.AdminUserList)
	admin.PUT("/users/:user_id/role", uh.UpdateRole)

	my := r.Group("/api/my", middleware.Auth(jwtSecret))
	my.GET("/items", ih.MyItem)
	my.GET("/claims/", ch.MyClaims)
}
