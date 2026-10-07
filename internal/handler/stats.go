package handler

import (
	"strconv"

	"Backendjh/internal/pkg/response"
	"Backendjh/internal/service"

	"github.com/gin-gonic/gin"
)

type StatsHandler struct {
	svc *service.StatsService
}

func NewStatsHandler(s *service.StatsService) *StatsHandler {
	return &StatsHandler{svc: s}
}

// Overview 各项统计
func (h *StatsHandler) Overview(c *gin.Context) {
	o, e := h.svc.Overview(c.GetString("role"))
	if e != nil {
		response.Fail(c, e)
		return
	}
	response.OK(c, o)
}

// Trend 全校发布趋势（近 N 天）
func (h *StatsHandler) Trend(c *gin.Context) {
	days, _ := strconv.Atoi(c.DefaultQuery("days", "30"))
	pts, e := h.svc.Trend(c.GetString("role"), days)
	if e != nil {
		response.Fail(c, e)
		return
	}
	response.OK(c, gin.H{"days": pts})
}
