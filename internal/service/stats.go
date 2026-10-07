package service

import (
	"Backendjh/internal/model"
	"Backendjh/internal/pkg/errcode"
	"Backendjh/internal/repository"
)

type StatsService struct{}

func NewStatsService() *StatsService { return &StatsService{} }

// Overview 全局统计（管理员）
func (s *StatsService) Overview(role string) (*repository.OverviewStats, *errcode.Error) {
	if role != model.RoleSysAdmin && role != model.RoleFinderAdmin && role != "admin" {
		return nil, errcode.Forbidden
	}
	o, err := repository.Overview()
	if err != nil {
		return nil, errcode.InternalError
	}
	return o, nil
}

// Trend 发布趋势（管理员）
func (s *StatsService) Trend(role string, days int) ([]repository.TrendPoint, *errcode.Error) {
	if role != model.RoleSysAdmin && role != model.RoleFinderAdmin && role != "admin" {
		return nil, errcode.Forbidden
	}
	pts, err := repository.Trend(days)
	if err != nil {
		return nil, errcode.InternalError
	}
	return pts, nil
}
