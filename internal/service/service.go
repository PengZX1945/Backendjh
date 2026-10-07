// Package service 承载业务规则。
//
// 分层约定：
//   - service 不 import gin，也不 import gorm —— 它只面对 repository 暴露的具名方法与
//     dto 定义的入参出参，因此业务规则可以脱离 HTTP 与 ORM 单独推敲；
//   - 可预期的失败一律返回 *apperr.Error，由统一响应中间件翻译；
//   - 仓储层的「查无记录」在这里翻译成业务错误码 4，不让 ORM 语义漏到上层。
package service

import (
	"errors"
	"strings"

	"lostfound/internal/repository"
	"lostfound/pkg/apperr"
	"lostfound/pkg/errcode"
)

// translate 把仓储层错误翻译成对外错误：查无记录 → 资源不存在(4)，其余原样上抛。
func translate(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, repository.ErrNotFound) {
		return apperr.New(errcode.NotFound)
	}
	return err
}

// requireText 校验必填文本。空串与纯空白都视为缺失，统一判参数错误(1)。
func requireText(values ...string) error {
	for _, value := range values {
		if strings.TrimSpace(value) == "" {
			return apperr.New(errcode.BadRequest)
		}
	}
	return nil
}
