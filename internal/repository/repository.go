// Package repository 是数据访问层，唯一与 GORM 打交道的地方。
//
// 两条纪律：
//  1. 上层（service）不 import gorm。仓储把 gorm.ErrRecordNotFound 统一翻译成
//     本包的 ErrNotFound，避免 ORM 细节沿着调用链往上渗。
//  2. 不返回裸 *gorm.DB。查询条件必须走本包定义好的 filter 结构体，
//     这样「能按什么条件查」是显式的，也不会把 SQL 拼装权散到上层。
package repository

import (
	"errors"

	"gorm.io/gorm"
)

// ErrNotFound 表示目标记录不存在。
// 所有 Find* 方法在查无结果时返回它，供上层翻译成业务错误码 4。
var ErrNotFound = errors.New("记录不存在")

// translate 把 GORM 的「未找到」翻译成本包的 ErrNotFound，其余错误原样返回。
func translate(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound
	}
	return err
}
