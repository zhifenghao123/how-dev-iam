// authz_manage_dao.go 是 authz 管理模块的数据访问层。
//
// 与 authn_manage_dao.go 同构，底层同样使用 SQLite 共享内存库。
// 服务启动时通过外部 sql 脚本完成建表与种子数据写入：
//
//   - DDL: logic/db/ddl/t_authz_type.sql
//   - DML: logic/db/dml/t_authz_type.sql
package dao

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
)

// AuthzTypePO 是 t_authz_type 表的持久化对象。
type AuthzTypePO struct {
	Code        string
	Name        string
	Description string
}

// AuthzTypeDao 是 authz 类型的存取接口。
type AuthzTypeDao interface {
	// ListAll 返回所有支持的授权方式（按 sort 升序）。
	ListAll(ctx context.Context) ([]AuthzTypePO, error)
}

// authz 类型相关的 sql 脚本文件名（与表名同名，隐含在 ddl/ 与 dml/ 子目录中）。
const authzTypeSQLFile = "t_authz_type.sql"

// AuthzTypeSQLiteDao 是 AuthzTypeDao 的 SQLite 实现。
type AuthzTypeSQLiteDao struct {
	db *sql.DB
}

// NewAuthzTypeSQLiteDao 构造 SQLite 实现，执行外部 ddl 与 dml 脚本。
func NewAuthzTypeSQLiteDao(ctx context.Context, db *sql.DB, ddlFS, dmlFS fs.FS) (*AuthzTypeSQLiteDao, error) {
	if db == nil {
		return nil, fmt.Errorf("dao.NewAuthzTypeSQLiteDao: db is nil")
	}
	if ddlFS == nil {
		return nil, fmt.Errorf("dao.NewAuthzTypeSQLiteDao: ddlFS is nil")
	}
	if dmlFS == nil {
		return nil, fmt.Errorf("dao.NewAuthzTypeSQLiteDao: dmlFS is nil")
	}
	if err := ExecSQLScript(ctx, db, ddlFS, authzTypeSQLFile); err != nil {
		return nil, fmt.Errorf("apply ddl: %w", err)
	}
	if err := ExecSQLScript(ctx, db, dmlFS, authzTypeSQLFile); err != nil {
		return nil, fmt.Errorf("apply dml: %w", err)
	}
	return &AuthzTypeSQLiteDao{db: db}, nil
}

// ListAll 返回 t_authz_type 表中的所有记录，按 sort 升序。
func (d *AuthzTypeSQLiteDao) ListAll(ctx context.Context) ([]AuthzTypePO, error) {
	const query = `SELECT code, name, description FROM t_authz_type ORDER BY sort ASC, code ASC;`
	rows, err := d.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query t_authz_type: %w", err)
	}
	defer rows.Close()

	out := make([]AuthzTypePO, 0, 8)
	for rows.Next() {
		var po AuthzTypePO
		if err := rows.Scan(&po.Code, &po.Name, &po.Description); err != nil {
			return nil, fmt.Errorf("scan t_authz_type: %w", err)
		}
		out = append(out, po)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate t_authz_type: %w", err)
	}
	return out, nil
}