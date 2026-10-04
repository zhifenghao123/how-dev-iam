// authn_manage_service.go 是 authn 管理模块的 service 层。
//
// 当前阶段仅提供"查询支持的认证方式"能力；后续接入实际认证流程时，
// 在本文件追加方法即可（保持 controller 与 service 的分层不变）。
package service

import (
	"context"
	"fmt"
	"log"

	"github.com/zhifenghao123/how-dev-iam/iam-hello/logic/dao"
	dbres "github.com/zhifenghao123/how-dev-iam/iam-hello/logic/db"
)

// AuthnType 描述一种受支持的认证方式。
//
// 设计为可序列化的纯数据结构，方便直接作为 controller 层 JSON 响应返回。
type AuthnType struct {
	// Code 是机器可读的稳定枚举值（如 "password"），上下游用其做路由 / 鉴权决策。
	Code string `json:"code"`
	// Name 是面向人类的展示名（如 "用户名密码"），可用于前端下拉。
	Name string `json:"name"`
	// Description 是补充说明，便于接入方理解该认证方式的语义。
	Description string `json:"description,omitempty"`
}

// AuthnManageService 是认证管理模块的业务服务。
//
// 字段说明：
//   - typeDao：authn 类型数据源；面向接口编程，便于单测 mock。
type AuthnManageService struct {
	typeDao dao.AuthnTypeDao
}

// NewAuthnManageService 构造 AuthnManageService。依赖通过参数显式注入。
func NewAuthnManageService(typeDao dao.AuthnTypeDao) *AuthnManageService {
	return &AuthnManageService{typeDao: typeDao}
}

// NewDefaultAuthnManageService 使用默认 dao 实现装配 service。
//
// 默认装配链：
//  1. 以 "authn" 为命名空间打开一份共享内存 SQLite；
//  2. 用该 db 构造 AuthnTypeSQLiteDao（构造时执行 DDL + 种子数据写入）；
//  3. 把 dao 注入 AuthnManageService。
//
// 失败时 fail-fast：直接 panic（启动期异常），让 main 因 init 失败立即暴露。
// 单测请改用 NewAuthnManageService 显式注入 mock。
func NewDefaultAuthnManageService() *AuthnManageService {
	ctx := context.Background()
	db, err := dao.OpenSharedMemorySQLite("authn")
	if err != nil {
		log.Panicf("[authn] open sqlite failed: %v", err)
	}
	typeDao, err := dao.NewAuthnTypeSQLiteDao(ctx, db, dbres.DDLFS(), dbres.DMLFS())
	if err != nil {
		log.Panicf("[authn] init authn_type dao failed: %v", err)
	}
	return NewAuthnManageService(typeDao)
}

// SupportTypes 返回当前支持的认证方式列表。
//
// 数据从 dao 层（SQLite authn_type 表）读取并映射为 service 层领域模型。
func (s *AuthnManageService) SupportTypes(ctx context.Context) ([]AuthnType, error) {
	pos, err := s.typeDao.ListAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("list authn types: %w", err)
	}
	out := make([]AuthnType, 0, len(pos))
	for _, po := range pos {
		out = append(out, AuthnType{
			Code:        po.Code,
			Name:        po.Name,
			Description: po.Description,
		})
	}
	return out, nil
}
