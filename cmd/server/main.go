package main

import (
	"log"

	"Backendjh/internal/config"
	"Backendjh/internal/handler"
	"Backendjh/internal/middleware"
	"Backendjh/internal/model"
	"Backendjh/internal/router"
	"Backendjh/internal/service"

	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.Load()
	if err := model.InitDB(cfg.DSN); err != nil {
		log.Fatalf("init db failed: %v", err)
	}

	us := service.NewUserService(cfg.JWTSecret)
	uh := handler.NewUserHandler(us)

	ups := service.NewUploadService()
	uph := handler.NewUploadHandler(ups)

	r := gin.Default()
	r.Use(middleware.ErrorHandler())
	r.Use(middleware.CORS())
	r.Static("/uploads", "./uploads")

	router.Setup(r, uh, uph, cfg.JWTSecret)

	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatalf("server exited: %v", err)
	}
}
