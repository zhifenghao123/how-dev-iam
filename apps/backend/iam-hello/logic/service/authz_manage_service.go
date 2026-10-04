// authz_manage_service.go 是 authz 管理模块的 service 层。
//
// 当前阶段仅提供"查询支持的授权方式"能力；后续接入策略评估 / 决策时，
// 在本文件追加方法即可。
package service

import (
	"context"
	"fmt"
	"log"

	"github.com/zhifenghao123/how-dev-iam/iam-hello/logic/dao"
	dbres "github.com/zhifenghao123/how-dev-iam/iam-hello/logic/db"
)

// AuthzType 描述一种受支持的授权方式。
//
// 字段语义与 AuthnType 一致：Code 机器可读、Name 人类可读、Description 备注。
type AuthzType struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// AuthzManageService 是授权管理模块的业务服务。
type AuthzManageService struct {
	typeDao dao.AuthzTypeDao
}

// NewAuthzManageService 构造 AuthzManageService。依赖通过参数显式注入。
func NewAuthzManageService(typeDao dao.AuthzTypeDao) *AuthzManageService {
	return &AuthzManageService{typeDao: typeDao}
}

// NewDefaultAuthzManageService 使用默认 dao 实现装配 service。
//
// 默认装配链：
//  1. 以 "authz" 为命名空间打开一份共享内存 SQLite；
//  2. 用该 db 构造 AuthzTypeSQLiteDao（构造时执行 DDL + 种子数据写入）；
//  3. 把 dao 注入 AuthzManageService。
func NewDefaultAuthzManageService() *AuthzManageService {
	ctx := context.Background()
	db, err := dao.OpenSharedMemorySQLite("authz")
	if err != nil {
		log.Panicf("[authz] open sqlite failed: %v", err)
	}
	typeDao, err := dao.NewAuthzTypeSQLiteDao(ctx, db, dbres.DDLFS(), dbres.DMLFS())
	if err != nil {
		log.Panicf("[authz] init authz_type dao failed: %v", err)
	}
	return NewAuthzManageService(typeDao)
}

// SupportTypes 返回当前支持的授权方式列表。
//
// 数据从 dao 层（SQLite authz_type 表）读取并映射为 service 层领域模型。
func (s *AuthzManageService) SupportTypes(ctx context.Context) ([]AuthzType, error) {
	pos, err := s.typeDao.ListAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("list authz types: %w", err)
	}
	out := make([]AuthzType, 0, len(pos))
	for _, po := range pos {
		out = append(out, AuthzType{
			Code:        po.Code,
			Name:        po.Name,
			Description: po.Description,
		})
	}
	return out, nil
}
