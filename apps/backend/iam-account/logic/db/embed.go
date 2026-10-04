// Package db 把 logic/db/ 下的 sql 资源在编译期嵌入二进制。
//
// 目录约定：
//   - logic/db/ddl/<table>.sql：表的 DDL（建表语句）。
//
// 设计说明：
//   - 正式部署走 K8s init-job（deploy/k8s/helm/init-jobs）执行 DDL；
//     应用启动时 dao 层只做 db.Ping()，不再执行建表。
//   - 但本包依旧保留 embed，以便本地开发时用一个可选的 helper
//     （dao.ApplyDDL）一次性拉起测试数据库；以及方便 IDE 关联 sql 文件。
//   - 物理 sql 文件保留为可读文本，DBA / init-job 可直接在生产数据库执行。
package db

import (
	"embed"
	"fmt"
	"io/fs"
)

//go:embed ddl/*.sql
var ddlFS embed.FS

// DDLFS 返回一个仅含 ddl/ 子目录下 .sql 的 fs.FS。
//
// 文件名层级被剥离，调用方只需传裸文件名（如 "t_user.sql"）。
func DDLFS() fs.FS {
	sub, err := fs.Sub(ddlFS, "ddl")
	if err != nil {
		panic(fmt.Errorf("logic/db: sub fs %q: %w", "ddl", err))
	}
	return sub
}

// ReadDDL 读取嵌入的 ddl/<name> 全部内容。典型用法：
//
//	stmt, _ := db.ReadDDL("t_user.sql")
func ReadDDL(name string) (string, error) {
	b, err := fs.ReadFile(DDLFS(), name)
	if err != nil {
		return "", fmt.Errorf("logic/db: read ddl %q: %w", name, err)
	}
	return string(b), nil
}

// ListDDLFiles 返回 ddl 目录下所有 .sql 文件名（按字典序）。
// 便于测试代码或本地 helper 一次性遍历执行。
func ListDDLFiles() ([]string, error) {
	entries, err := fs.ReadDir(DDLFS(), ".")
	if err != nil {
		return nil, fmt.Errorf("logic/db: read ddl dir: %w", err)
	}
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if len(name) > 4 && name[len(name)-4:] == ".sql" {
			out = append(out, name)
		}
	}
	return out, nil
}
