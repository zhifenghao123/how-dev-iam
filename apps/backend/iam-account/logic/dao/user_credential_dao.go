// user_credential_dao.go 是 t_user_credential 表的数据访问层。
package dao

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// UserCredentialPO 是 t_user_credential 的持久化对象。
type UserCredentialPO struct {
	UserID            int64
	PasswordHash      string
	PasswordAlgo      string
	PasswordUpdatedAt time.Time
	FailedAttempts    int
}

// UserCredentialDao 用户凭据访问接口。
type UserCredentialDao interface {
	Insert(ctx context.Context, tx Executor, po *UserCredentialPO) error
	GetByUserID(ctx context.Context, userID int64) (*UserCredentialPO, error)
	UpdatePassword(ctx context.Context, userID int64, hash, algo string) error
}

// UserCredentialMySQLDao MySQL 实现。
type UserCredentialMySQLDao struct {
	db *sql.DB
}

// NewUserCredentialMySQLDao 构造。
func NewUserCredentialMySQLDao(db *sql.DB) *UserCredentialMySQLDao {
	return &UserCredentialMySQLDao{db: db}
}

// Insert 插入新凭据。
func (d *UserCredentialMySQLDao) Insert(ctx context.Context, tx Executor, po *UserCredentialPO) error {
	if tx == nil {
		tx = d.db
	}
	const q = `INSERT INTO t_user_credential
      (user_id, password_hash, password_algo) VALUES (?,?,?)`
	_, err := tx.ExecContext(ctx, q, po.UserID, po.PasswordHash, po.PasswordAlgo)
	if err != nil {
		return fmt.Errorf("insert t_user_credential: %w", err)
	}
	return nil
}

// GetByUserID 按 user_id 查询。
func (d *UserCredentialMySQLDao) GetByUserID(ctx context.Context, userID int64) (*UserCredentialPO, error) {
	const q = `SELECT user_id, password_hash, password_algo, password_updated_at, failed_attempts
	           FROM t_user_credential WHERE user_id = ?`
	var po UserCredentialPO
	err := d.db.QueryRowContext(ctx, q, userID).
		Scan(&po.UserID, &po.PasswordHash, &po.PasswordAlgo, &po.PasswordUpdatedAt, &po.FailedAttempts)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("query t_user_credential: %w", err)
	}
	return &po, nil
}

// UpdatePassword 更新密码 hash / 算法。
func (d *UserCredentialMySQLDao) UpdatePassword(ctx context.Context, userID int64, hash, algo string) error {
	const q = `UPDATE t_user_credential
	           SET password_hash = ?, password_algo = ?, password_updated_at = CURRENT_TIMESTAMP,
	               failed_attempts = 0, locked_until = NULL
	           WHERE user_id = ?`
	_, err := d.db.ExecContext(ctx, q, hash, algo, userID)
	if err != nil {
		return fmt.Errorf("update t_user_credential: %w", err)
	}
	return nil
}
