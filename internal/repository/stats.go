package repository

import (
	"time"

	"Backendjh/internal/model"
)

// TrendPoint 某天的发布量
type TrendPoint struct {
	Date  string `json:"date"` // YYYY-MM-DD
	Total int    `json:"total"`
	Found int    `json:"found"`
	Lost  int    `json:"lost"`
}

// ItemStat 各状态 / 类型数量
type ItemStat struct {
	Status int   `json:"status"`
	Count  int64 `json:"count"`
}

// OverviewStats 全局统计
type OverviewStats struct {
	ItemsTotal       int64      `json:"items_total"`
	FoundTotal       int64      `json:"found_total"`
	LostTotal        int64      `json:"lost_total"`
	UsersTotal       int64      `json:"users_total"`
	Announcements    int64      `json:"announcements_total"`
	ClaimsTotal      int64      `json:"claims_total"`
	StatusStats      []ItemStat `json:"status_stats"`
	ItemTypeStats    []ItemStat `json:"item_type_stats"`
	ClaimStatusStats []ItemStat `json:"claim_status_stats"`
}

func Overview() (*OverviewStats, error) {
	o := &OverviewStats{}
	itemCount := func() int64 {
		var n int64
		model.DB.Model(&model.Item{}).Count(&n)
		return n
	}
	o.ItemsTotal = itemCount()
	o.FoundTotal = itemCountWhere("type = ?", model.ItemTypeFound)
	o.LostTotal = itemCountWhere("type = ?", model.ItemTypeLost)
	model.DB.Model(&model.User{}).Count(&o.UsersTotal)
	model.DB.Model(&model.Announcement{}).Count(&o.Announcements)
	model.DB.Model(&model.Claim{}).Count(&o.ClaimsTotal)

	// 物品状态分布
	model.DB.Model(&model.Item{}).
		Select("status, COUNT(*) AS count").
		Group("status").Scan(&o.StatusStats)
	// 物品类型分布
	model.DB.Model(&model.Item{}).
		Select("CASE type WHEN 'found' THEN 1 WHEN 'lost' THEN 0 END AS status, COUNT(*) AS count").
		Group("type").Scan(&o.ItemTypeStats)
	// 认领状态分布
	model.DB.Model(&model.Claim{}).
		Select("status, COUNT(*) AS count").
		Group("status").Scan(&o.ClaimStatusStats)
	return o, nil
}

func itemCountWhere(cond string, args ...any) int64 {
	var n int64
	model.DB.Model(&model.Item{}).Where(cond, args...).Count(&n)
	return n
}

// Trend 近 days 天每日发布量（缺失日期补 0）
func Trend(days int) ([]TrendPoint, error) {
	if days <= 0 {
		days = 30
	}
	start := time.Now().AddDate(0, 0, -(days - 1)).Format("2006-01-02")
	type row struct {
		Date  string
		Total int
		Found int
		Lost  int
	}
	var rows []row
	err := model.DB.Raw(
		`SELECT DATE_FORMAT(created_at,'%Y-%m-%d') AS date,
		        COUNT(*) AS total,
		        SUM(CASE WHEN type='found' THEN 1 ELSE 0 END) AS found,
		        SUM(CASE WHEN type='lost' THEN 1 ELSE 0 END) AS lost
		 FROM items WHERE created_at >= ?
		 GROUP BY DATE_FORMAT(created_at,'%Y-%m-%d') ORDER BY date`,
		start+" 00:00:00",
	).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	byDate := map[string]*TrendPoint{}
	for i := range rows {
		byDate[rows[i].Date] = &TrendPoint{Date: rows[i].Date, Total: rows[i].Total, Found: rows[i].Found, Lost: rows[i].Lost}
	}
	points := make([]TrendPoint, 0, days)
	for i := 0; i < days; i++ {
		d := time.Now().AddDate(0, 0, -(days - 1 - i)).Format("2006-01-02")
		if p, ok := byDate[d]; ok {
			points = append(points, *p)
		} else {
			points = append(points, TrendPoint{Date: d})
		}
	}
	return points, nil
}
