// sqlite.go 提供 dao 层共用的 SQLite 连接打开工具。
//
// 选型：modernc.org/sqlite —— 纯 Go 实现，零 CGO，跨平台编译开箱即用。
// 该驱动在 database/sql 中注册的 driver 名为 "sqlite"。
//
// ===== file::memory: 与 cache=shared 的要点 =====
//
// database/sql 维护"连接池"。直接使用 ":memory:" 时，每条新建连接都会
// 拿到一个独立的内存数据库——A 连接建的表 B 连接看不到。
//
// 标准做法：
//  1. DSN 使用 "file:<name>?mode=memory&cache=shared"，多连接共享同一份内存库；
//  2. 同时把连接池上限收紧（MaxOpenConns=1 / MaxIdleConns=1），
//     避免在并发下因驱动层细节引起的"看似 shared 实则各自一份"的边界问题，
//     也避免内存库被空闲连接全部回收后整库丢失。
//
// 这两点合起来才能保证同一个 *sql.DB 句柄内"建表 + 写种子 + 查询"始终面向同一份数据。
package dao

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"strings"

	// 仅注册 sqlite driver，不直接使用其包名。
	_ "modernc.org/sqlite"
)

// OpenSharedMemorySQLite 打开一个进程内共享的内存 SQLite 数据库。
//
// namespace 用于隔离不同模块（例如 "authn" / "authz"），同名 namespace
// 在同一进程内会复用同一份内存库；不同 namespace 互不可见。
//
// 调用方持有返回的 *sql.DB，并负责在进程退出时 Close。
func OpenSharedMemorySQLite(namespace string) (*sql.DB, error) {
	if namespace == "" {
		return nil, fmt.Errorf("dao.OpenSharedMemorySQLite: namespace is empty")
	}
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", namespace)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite (%s): %w", dsn, err)
	}
	// 单连接策略：避免内存库在并发场景下被认为是多份；
	// 业务量是 O(读支持列表)，单连接性能足够。
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping sqlite (%s): %w", dsn, err)
	}
	return db, nil
}

// ExecSQLScript 从 fsys 中读取脚本文件并按 ";" 拆分语句逐条执行。
//
// 适用场景：业务表的 DDL（建表）与 DML（种子数据）脚本。脚本规则：
//   - 以 ";" 结尾分隔多条语句；
//   - 支持 "--" 起始的整行注释（拆分前已剔除）；
//   - 空白语句自动忽略。
//
// 之所以自己拆分而不是把整段交给驱动：modernc.org/sqlite 的 ExecContext
// 一次只执行一条语句，多语句脚本会报 "syntax error near \";\""。
// 自行简单拆分能覆盖 DDL/DML 等"无字符串字面量含分号"的常见场景，
// 后续若 sql 中确需出现分号字面量，再升级为更完善的语句切分器。
func ExecSQLScript(ctx context.Context, db *sql.DB, fsys fs.FS, name string) error {
	if db == nil {
		return fmt.Errorf("dao.ExecSQLScript: db is nil")
	}
	if fsys == nil {
		return fmt.Errorf("dao.ExecSQLScript(%s): fs is nil", name)
	}
	raw, err := fs.ReadFile(fsys, name)
	if err != nil {
		return fmt.Errorf("dao.ExecSQLScript: read %q: %w", name, err)
	}

	stmts := splitSQLStatements(string(raw))
	for i, stmt := range stmts {
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("dao.ExecSQLScript: %s stmt#%d %q: %w", name, i+1, firstLine(stmt), err)
		}
	}
	return nil
}

// splitSQLStatements 按 ";" 切分语句，并剥离 "--" 整行注释与首尾空白。
func splitSQLStatements(script string) []string {
	// 1) 去除注释行
	var sb strings.Builder
	for _, line := range strings.Split(script, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "--") {
			continue
		}
		sb.WriteString(line)
		sb.WriteByte('\n')
	}
	// 2) 按 ";" 分段
	parts := strings.Split(sb.String(), ";")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		s := strings.TrimSpace(p)
		if s == "" {
			continue
		}
		out = append(out, s)
	}
	return out
}

// firstLine 返回字符串首行（用于错误信息预览）。
func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return strings.TrimSpace(s[:i])
	}
	return strings.TrimSpace(s)
}