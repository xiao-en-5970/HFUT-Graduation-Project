package dao

import (
	"context"

	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/dao/model"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/common/pgsql"
)

type ServiceTokenAuditStore struct{}

// Create 写入一条审计记录。调用方通常异步执行（middleware 已拆 goroutine）。
//
// 不返回 ID——审计写失败不应影响主流程，调用方拿到 err 只 log 即可。
func (s *ServiceTokenAuditStore) Create(ctx context.Context, m *model.ServiceTokenAudit) error {
	return pgsql.DB.WithContext(ctx).Create(m).Error
}
