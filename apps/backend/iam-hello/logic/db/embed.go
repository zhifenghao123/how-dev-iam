// Package db 把 logic/db/ 下的 sql 资源在编译期嵌入二进制。
//
// 目录约定：
//   - logic/db/ddl/<table>.sql：表的 DDL（建表语句）；
//   - logic/db/dml/<table>.sql：表的 DML（种子 / 初始化数据）。
//
// 设计要点：
//   - Go 的 //go:embed 指令路径是"相对当前 Go 文件目录"且不允许跨目录向上引用，
//     因此把"嵌入器"直接放在它服务的资源目录旁边，是最稳妥也最内聚的姿势：
//     dao 通过 db.DDLFS() / db.DMLFS() 拿到 fs.FS 即可，不必感知物理路径。
//   - ddl / dml 各自暴露一份 fs.FS，dao 在两段使用点（建表 vs 灌种子）
//     从职责上也对应清楚，错误信息能各自聚焦。
//   - 物理 sql 文件保留为可读文本，DBA 可直接在生产数据库执行；
//     运行时 dao 通过本包暴露的 fs.FS 读取，二进制自包含、部署不必带资源目录。
//
// 注意：本包只做"资源读取器"，不应包含任何业务逻辑。
package db

import (
	"embed"
	"fmt"
	"io/fs"
)

// ddlFS / dmlFS 分别嵌入 ddl/、dml/ 子目录下的全部 .sql 文件。
//
//go:embed ddl/*.sql
var ddlFS embed.FS

//go:embed dml/*.sql
var dmlFS embed.FS

// DDLFS 返回一个仅含 ddl/ 子目录下 .sql 的 fs.FS。
//
// 文件名层级被剥离，调用方只需传裸文件名（如 "t_authn_type.sql"）。
func DDLFS() fs.FS {
	sub, err := fs.Sub(ddlFS, "ddl")
	if err != nil {
		// 编译期保证 "ddl" 子目录存在；运行期到此说明构建链路异常，fail-fast。
		panic(fmt.Errorf("logic/db: sub fs %q: %w", "ddl", err))
	}
	return sub
}

// DMLFS 返回一个仅含 dml/ 子目录下 .sql 的 fs.FS。
func DMLFS() fs.FS {
	sub, err := fs.Sub(dmlFS, "dml")
	if err != nil {
		panic(fmt.Errorf("logic/db: sub fs %q: %w", "dml", err))
	}
	return sub
}

// ReadDDL 读取嵌入的 ddl/<name> 全部内容。典型用法：
//
//	stmt, _ := db.ReadDDL("t_authn_type.sql")
func ReadDDL(name string) (string, error) {
	b, err := fs.ReadFile(DDLFS(), name)
	if err != nil {
		return "", fmt.Errorf("logic/db: read ddl %q: %w", name, err)
	}
	return string(b), nil
}

// ReadDML 读取嵌入的 dml/<name> 全部内容。
func ReadDML(name string) (string, error) {
	b, err := fs.ReadFile(DMLFS(), name)
	if err != nil {
		return "", fmt.Errorf("logic/db: read dml %q: %w", name, err)
	}
	return string(b), nil
}
