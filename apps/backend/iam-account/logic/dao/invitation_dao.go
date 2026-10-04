// invitation_dao.go 是 t_invitation 表的数据访问层。
package dao

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// InvitationPO 是 t_invitation 表的持久化对象。
type InvitationPO struct {
	ID           int64
	AccountID    int64
	InviterID    int64
	InviteeEmail sql.NullString
	InviteePhone sql.NullString
	Role         string
	Token        string
	Status       string // pending|accepted|expired|revoked
	ExpireAt     time.Time
	AcceptedBy   sql.NullInt64
	AcceptedAt   sql.NullTime
	CreateTime   time.Time
	UpdateTime   time.Time
}

// InvitationDao 邀请令牌访问接口。
type InvitationDao interface {
	Insert(ctx context.Context, tx Executor, po *InvitationPO) error
	GetByToken(ctx context.Context, token string) (*InvitationPO, error)
	MarkAccepted(ctx context.Context, tx Executor, id int64, acceptedBy int64) error
}

// InvitationMySQLDao MySQL 实现。
type InvitationMySQLDao struct {
	db *sql.DB
}

// NewInvitationMySQLDao 构造。
func NewInvitationMySQLDao(db *sql.DB) *InvitationMySQLDao {
	return &InvitationMySQLDao{db: db}
}

// Insert 插入。
func (d *InvitationMySQLDao) Insert(ctx context.Context, tx Executor, po *InvitationPO) error {
	if tx == nil {
		tx = d.db
	}
	const q = `INSERT INTO t_invitation
      (id, account_id, inviter_id, invitee_email, invitee_phone, role, token, status, expire_at)
      VALUES (?,?,?,?,?,?,?,?,?)`
	_, err := tx.ExecContext(ctx, q, po.ID, po.AccountID, po.InviterID, po.InviteeEmail, po.InviteePhone,
		po.Role, po.Token, po.Status, po.ExpireAt)
	if err != nil {
		return fmt.Errorf("insert t_invitation: %w", err)
	}
	return nil
}

// GetByToken 按 token 查询。
func (d *InvitationMySQLDao) GetByToken(ctx context.Context, token string) (*InvitationPO, error) {
	const q = `SELECT id, account_id, inviter_id, invitee_email, invitee_phone, role, token, status,
                      expire_at, accepted_by, accepted_at, create_time, update_time
               FROM t_invitation WHERE token = ?`
	var po InvitationPO
	err := d.db.QueryRowContext(ctx, q, token).Scan(&po.ID, &po.AccountID, &po.InviterID,
		&po.InviteeEmail, &po.InviteePhone, &po.Role, &po.Token, &po.Status,
		&po.ExpireAt, &po.AcceptedBy, &po.AcceptedAt, &po.CreateTime, &po.UpdateTime)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("query t_invitation by token: %w", err)
	}
	return &po, nil
}

// MarkAccepted 更新为 accepted 状态。
func (d *InvitationMySQLDao) MarkAccepted(ctx context.Context, tx Executor, id int64, acceptedBy int64) error {
	if tx == nil {
		tx = d.db
	}
	const q = `UPDATE t_invitation
	           SET status = 'accepted', accepted_by = ?, accepted_at = CURRENT_TIMESTAMP
	           WHERE id = ? AND status = 'pending'`
	res, err := tx.ExecContext(ctx, q, acceptedBy, id)
	if err != nil {
		return fmt.Errorf("update t_invitation.status=accepted: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return errors.New("invitation not in pending status")
	}
	return nil
}
