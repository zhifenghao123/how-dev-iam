// Package dao 是 iam-account 的持久化层。
//
// mysql.go 只负责打开 MySQL 连接池并 Ping 一次；建表由 K8s init-job 完成，
// 服务启动时不再执行 DDL（DDL 文件仍保留在 logic/db/ddl/*.sql 供 init-job / DBA 使用）。
package dao

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"github.com/zhifenghao123/how-dev-iam/iam-account/logic/config"
)

// OpenMySQL 打开 MySQL 连接池，并做一次 Ping 校验可达性。
//
// 调用方负责在进程退出时 Close。
func OpenMySQL(cfg config.MySQLConfig) (*sql.DB, error) {
	if cfg.DSN == "" {
		return nil, fmt.Errorf("dao.OpenMySQL: dsn is empty")
	}
	db, err := sql.Open("mysql", cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("dao.OpenMySQL: open: %w", err)
	}
	if cfg.MaxOpenConns > 0 {
		db.SetMaxOpenConns(cfg.MaxOpenConns)
	}
	if cfg.MaxIdleConns > 0 {
		db.SetMaxIdleConns(cfg.MaxIdleConns)
	}
	if cfg.ConnMaxLifeSecs > 0 {
		db.SetConnMaxLifetime(time.Duration(cfg.ConnMaxLifeSecs) * time.Second)
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("dao.OpenMySQL: ping: %w", err)
	}
	return db, nil
}
