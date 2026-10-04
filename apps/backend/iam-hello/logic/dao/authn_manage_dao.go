// authn_manage_dao.go 是 authn 管理模块的数据访问层。
//
// 当前阶段提供"支持的认证方式列表"持久化能力。底层使用 SQLite 共享内存库
// （见 sqlite.go），服务启动时执行外部 sql 脚本完成建表与种子数据写入：
//
//   - DDL: logic/db/ddl/t_authn_type.sql
//   - DML: logic/db/dml/t_authn_type.sql
//
// sql 文件由调用方以 fs.FS 形式注入（生产用 db.DDLFS() / db.DMLFS()），
// 既保留物理 sql 供 DBA 直接使用，又让二进制自包含。
package dao

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
)

// AuthnTypePO 是 t_authn_type 表的持久化对象（Persistent Object）。
//
// 与 service 层的 AuthnType 故意保持解耦：dao 只做"原始字段读写"，
// 由 service 层负责到领域模型的映射，避免 dao -> service 的反向依赖。
type AuthnTypePO struct {
	Code        string
	Name        string
	Description string
}

// AuthnTypeDao 是 authn 类型的存取接口。service 仅依赖该接口，便于单测 mock。
type AuthnTypeDao interface {
	// ListAll 返回所有支持的认证方式（按 sort 升序）。
	ListAll(ctx context.Context) ([]AuthnTypePO, error)
}

// authn 类型相关的 sql 脚本文件名。
//
// 按表名直接命名，文件所在的 ddl/ 与 dml/ 子目录已隐含语义，无需在文件名重复。
const authnTypeSQLFile = "t_authn_type.sql"

// AuthnTypeSQLiteDao 是 AuthnTypeDao 的 SQLite 实现。
type AuthnTypeSQLiteDao struct {
	db *sql.DB
}

// NewAuthnTypeSQLiteDao 构造 SQLite 实现。
//
// 行为：
//  1. 从 ddlFS 读取 t_authn_type.sql 并执行（CREATE TABLE IF NOT EXISTS，幂等）；
//  2. 从 dmlFS 读取 t_authn_type.sql 并执行（INSERT OR IGNORE，幂等：
//     多次启动不会重复插入，也不会覆盖运行期人为修改）。
//
// 参数：
//   - db：调用方注入；
//   - ddlFS：建表脚本所在 fs.FS（生产用 db.DDLFS()，单测可注入 fstest.MapFS）；
//   - dmlFS：种子脚本所在 fs.FS（生产用 db.DMLFS()）。
func NewAuthnTypeSQLiteDao(ctx context.Context, db *sql.DB, ddlFS, dmlFS fs.FS) (*AuthnTypeSQLiteDao, error) {
	if db == nil {
		return nil, fmt.Errorf("dao.NewAuthnTypeSQLiteDao: db is nil")
	}
	if ddlFS == nil {
		return nil, fmt.Errorf("dao.NewAuthnTypeSQLiteDao: ddlFS is nil")
	}
	if dmlFS == nil {
		return nil, fmt.Errorf("dao.NewAuthnTypeSQLiteDao: dmlFS is nil")
	}
	if err := ExecSQLScript(ctx, db, ddlFS, authnTypeSQLFile); err != nil {
		return nil, fmt.Errorf("apply ddl: %w", err)
	}
	if err := ExecSQLScript(ctx, db, dmlFS, authnTypeSQLFile); err != nil {
		return nil, fmt.Errorf("apply dml: %w", err)
	}
	return &AuthnTypeSQLiteDao{db: db}, nil
}

// ListAll 返回 t_authn_type 表中的所有记录，按 sort 升序。
func (d *AuthnTypeSQLiteDao) ListAll(ctx context.Context) ([]AuthnTypePO, error) {
	const query = `SELECT code, name, description FROM t_authn_type ORDER BY sort ASC, code ASC;`
	rows, err := d.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query t_authn_type: %w", err)
	}
	defer rows.Close()

	out := make([]AuthnTypePO, 0, 8)
	for rows.Next() {
		var po AuthnTypePO
		if err := rows.Scan(&po.Code, &po.Name, &po.Description); err != nil {
			return nil, fmt.Errorf("scan t_authn_type: %w", err)
		}
		out = append(out, po)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate t_authn_type: %w", err)
	}
	return out, nil
}