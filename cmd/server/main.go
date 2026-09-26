package main

import (
	"log"

	"Backendjh/internal/handler"
	"Backendjh/internal/router"
	"Backendjh/internal/service"
	"Backendjh/internal/config"
	"Backendjh/internal/middleware"
	"Backendjh/internal/model"

	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.Load()
	if err := model.InitDB(cfg.DSN); err != nil {
		log.Fatalf("init db failed: %v", err)
	}

	us := service.NewUserService(cfg.JWTSecret)
	uh := handler.NewUserHandler(us)

	r := gin.Default()
	r.Use(middleware.ErrorHandler())

	router.Setup(r, uh, cfg.JWTSecret)

	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatalf("server exited: %v", err)
	}
}
