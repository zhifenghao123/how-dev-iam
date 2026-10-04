// verify_code_dao.go 是 t_verify_code 表的数据访问层。
package dao

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// VerifyCodePO 是 t_verify_code 表的持久化对象。
type VerifyCodePO struct {
	ID        int64
	Channel   string // email | sms
	Target    string
	Scene     string
	CodeHash  string
	ExpireAt  time.Time
	Used      bool
	UsedAt    sql.NullTime
	TryCount  int
	ClientIP  sql.NullString
	CreateAt  time.Time
}

// VerifyCodeDao 验证码访问接口。
type VerifyCodeDao interface {
	Insert(ctx context.Context, po *VerifyCodePO) error
	GetLatestActive(ctx context.Context, target, scene string) (*VerifyCodePO, error)
	MarkUsed(ctx context.Context, id int64) error
	IncrementTry(ctx context.Context, id int64) error
	// LatestCreatedAt 用于发送频率限制：查询最近一次生成时间；无则返回 (time.Time{}, nil).
	LatestCreatedAt(ctx context.Context, target, scene string) (time.Time, error)
}

// VerifyCodeMySQLDao MySQL 实现。
type VerifyCodeMySQLDao struct {
	db *sql.DB
}

// NewVerifyCodeMySQLDao 构造。
func NewVerifyCodeMySQLDao(db *sql.DB) *VerifyCodeMySQLDao {
	return &VerifyCodeMySQLDao{db: db}
}

// Insert 插入。
func (d *VerifyCodeMySQLDao) Insert(ctx context.Context, po *VerifyCodePO) error {
	const q = `INSERT INTO t_verify_code
      (id, channel, target, scene, code_hash, expire_at, client_ip)
      VALUES (?,?,?,?,?,?,?)`
	_, err := d.db.ExecContext(ctx, q, po.ID, po.Channel, po.Target, po.Scene, po.CodeHash,
		po.ExpireAt, po.ClientIP)
	if err != nil {
		return fmt.Errorf("insert t_verify_code: %w", err)
	}
	return nil
}

// GetLatestActive 取最近一条 未使用 且 未过期 的验证码。
func (d *VerifyCodeMySQLDao) GetLatestActive(ctx context.Context, target, scene string) (*VerifyCodePO, error) {
	const q = `SELECT id, channel, target, scene, code_hash, expire_at, used, used_at,
                      try_count, client_ip, create_time
               FROM t_verify_code
               WHERE target = ? AND scene = ? AND used = 0 AND expire_at > CURRENT_TIMESTAMP
               ORDER BY id DESC LIMIT 1`
	var po VerifyCodePO
	var used int
	err := d.db.QueryRowContext(ctx, q, target, scene).Scan(
		&po.ID, &po.Channel, &po.Target, &po.Scene, &po.CodeHash, &po.ExpireAt, &used, &po.UsedAt,
		&po.TryCount, &po.ClientIP, &po.CreateAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("query t_verify_code: %w", err)
	}
	po.Used = used == 1
	return &po, nil
}

// MarkUsed 标记为已使用。
func (d *VerifyCodeMySQLDao) MarkUsed(ctx context.Context, id int64) error {
	const q = `UPDATE t_verify_code SET used = 1, used_at = CURRENT_TIMESTAMP WHERE id = ?`
	_, err := d.db.ExecContext(ctx, q, id)
	if err != nil {
		return fmt.Errorf("update t_verify_code.used=1: %w", err)
	}
	return nil
}

// IncrementTry 校验失败时 try_count+1。
func (d *VerifyCodeMySQLDao) IncrementTry(ctx context.Context, id int64) error {
	const q = `UPDATE t_verify_code SET try_count = try_count + 1 WHERE id = ?`
	_, err := d.db.ExecContext(ctx, q, id)
	if err != nil {
		return fmt.Errorf("update t_verify_code.try_count: %w", err)
	}
	return nil
}

// LatestCreatedAt 查询最近一次生成时间。
func (d *VerifyCodeMySQLDao) LatestCreatedAt(ctx context.Context, target, scene string) (time.Time, error) {
	const q = `SELECT create_time FROM t_verify_code
	           WHERE target = ? AND scene = ? ORDER BY id DESC LIMIT 1`
	var t time.Time
	err := d.db.QueryRowContext(ctx, q, target, scene).Scan(&t)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return time.Time{}, nil
		}
		return time.Time{}, fmt.Errorf("query t_verify_code.latest: %w", err)
	}
	return t, nil
}
