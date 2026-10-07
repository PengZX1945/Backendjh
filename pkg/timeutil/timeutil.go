// Package timeutil 统一时间串的读写格式。
//
// 接口文档里所有时间字段都是字符串（如 created_time、happen_time），前端按
// "YYYY-MM-DD HH:mm:ss" 解析。把布局收在这里，避免各层各写一份格式串导致
// 「有的带 T、有的带时区」这类隐性不一致 —— 它还会连带影响按时间倒序的排序结果。
package timeutil

import "time"

// Layout 是全站唯一的持久化时间布局。
//
// 选这个布局还有一个副作用：它按字典序排序的结果与按时间先后排序一致，
// 因此 happen_time / created_time 可以直接用字符串比较来排序（见仓储层 Order）。
const Layout = "2006-01-02 15:04:05"

// Now 返回当前时间的标准串。
func Now() string {
	return time.Now().Format(Layout)
}

// Format 把时间按标准布局格式化。
func Format(value time.Time) string {
	return value.Format(Layout)
}
