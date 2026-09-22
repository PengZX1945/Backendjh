package main

import (
	"log"

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
	r := gin.Default()
	r.Use(middleware.ErrorHandler())
	//健康检查接口
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{"msg": "pong"})
	})

	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatalf("server exited: %v", err)
	}
}
