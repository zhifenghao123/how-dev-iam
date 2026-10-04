// membership_dao.go 是 t_membership 表的数据访问层。
package dao

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// MembershipPO 是 t_membership 表的持久化对象。
type MembershipPO struct {
	ID          int64
	AccountID   int64
	UserID      int64
	Role        string // owner|admin|member|viewer
	Status      string // pending|active|disabled|left
	DisplayName sql.NullString
	JoinSource  string
	JoinTime    time.Time
	LeaveTime   sql.NullTime
	InvitedBy   sql.NullInt64
	CreateTime  time.Time
	UpdateTime  time.Time
}

// MembershipDao 成员关系访问接口。
type MembershipDao interface {
	Insert(ctx context.Context, tx Executor, po *MembershipPO) error
	GetByID(ctx context.Context, id int64) (*MembershipPO, error)
	GetByAccountAndUser(ctx context.Context, accountID, userID int64) (*MembershipPO, error)
	ListByUser(ctx context.Context, userID int64) ([]MembershipPO, error)
	ListByAccount(ctx context.Context, accountID int64, offset, limit int) ([]MembershipPO, error)
	CountByAccount(ctx context.Context, accountID int64) (int, error)
	MarkLeft(ctx context.Context, id int64) error
}

// MembershipMySQLDao MySQL 实现。
type MembershipMySQLDao struct {
	db *sql.DB
}

// NewMembershipMySQLDao 构造。
func NewMembershipMySQLDao(db *sql.DB) *MembershipMySQLDao {
	return &MembershipMySQLDao{db: db}
}

const membershipColumns = `id, account_id, user_id, role, status, display_name,
    join_source, join_time, leave_time, invited_by, create_time, update_time`

func scanMembership(row interface {
	Scan(dest ...any) error
}) (*MembershipPO, error) {
	var po MembershipPO
	err := row.Scan(&po.ID, &po.AccountID, &po.UserID, &po.Role, &po.Status, &po.DisplayName,
		&po.JoinSource, &po.JoinTime, &po.LeaveTime, &po.InvitedBy, &po.CreateTime, &po.UpdateTime)
	if err != nil {
		return nil, err
	}
	return &po, nil
}

// Insert 插入。
func (d *MembershipMySQLDao) Insert(ctx context.Context, tx Executor, po *MembershipPO) error {
	if tx == nil {
		tx = d.db
	}
	const q = `INSERT INTO t_membership
      (id, account_id, user_id, role, status, display_name, join_source, invited_by)
      VALUES (?,?,?,?,?,?,?,?)`
	_, err := tx.ExecContext(ctx, q, po.ID, po.AccountID, po.UserID, po.Role, po.Status,
		po.DisplayName, po.JoinSource, po.InvitedBy)
	if err != nil {
		return fmt.Errorf("insert t_membership: %w", err)
	}
	return nil
}

// GetByID 按 id 查询。
func (d *MembershipMySQLDao) GetByID(ctx context.Context, id int64) (*MembershipPO, error) {
	q := `SELECT ` + membershipColumns + ` FROM t_membership WHERE id = ?`
	po, err := scanMembership(d.db.QueryRowContext(ctx, q, id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("query t_membership by id: %w", err)
	}
	return po, nil
}

// GetByAccountAndUser 按 (account_id, user_id) 查询唯一成员关系。
func (d *MembershipMySQLDao) GetByAccountAndUser(ctx context.Context, accountID, userID int64) (*MembershipPO, error) {
	q := `SELECT ` + membershipColumns + ` FROM t_membership WHERE account_id = ? AND user_id = ?`
	po, err := scanMembership(d.db.QueryRowContext(ctx, q, accountID, userID))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("query t_membership by account_user: %w", err)
	}
	return po, nil
}

// ListByUser 列出某 User 的所有 Membership（status=active）。
func (d *MembershipMySQLDao) ListByUser(ctx context.Context, userID int64) ([]MembershipPO, error) {
	q := `SELECT ` + membershipColumns + ` FROM t_membership
	      WHERE user_id = ? AND status = 'active' ORDER BY join_time ASC`
	rows, err := d.db.QueryContext(ctx, q, userID)
	if err != nil {
		return nil, fmt.Errorf("list t_membership by user: %w", err)
	}
	defer rows.Close()

	out := make([]MembershipPO, 0, 4)
	for rows.Next() {
		po, err := scanMembership(rows)
		if err != nil {
			return nil, fmt.Errorf("scan t_membership: %w", err)
		}
		out = append(out, *po)
	}
	return out, rows.Err()
}

// ListByAccount 列出某 Account 下的 Membership，分页。
func (d *MembershipMySQLDao) ListByAccount(ctx context.Context, accountID int64, offset, limit int) ([]MembershipPO, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	q := `SELECT ` + membershipColumns + ` FROM t_membership
	      WHERE account_id = ? AND status IN ('pending','active')
	      ORDER BY join_time ASC LIMIT ? OFFSET ?`
	rows, err := d.db.QueryContext(ctx, q, accountID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list t_membership by account: %w", err)
	}
	defer rows.Close()

	out := make([]MembershipPO, 0, limit)
	for rows.Next() {
		po, err := scanMembership(rows)
		if err != nil {
			return nil, fmt.Errorf("scan t_membership: %w", err)
		}
		out = append(out, *po)
	}
	return out, rows.Err()
}

// CountByAccount 统计某 Account 下的活跃成员数（含 pending）。
func (d *MembershipMySQLDao) CountByAccount(ctx context.Context, accountID int64) (int, error) {
	const q = `SELECT COUNT(1) FROM t_membership
	           WHERE account_id = ? AND status IN ('pending','active')`
	var n int
	if err := d.db.QueryRowContext(ctx, q, accountID).Scan(&n); err != nil {
		return 0, fmt.Errorf("count t_membership by account: %w", err)
	}
	return n, nil
}

// MarkLeft 标记成员离开。
func (d *MembershipMySQLDao) MarkLeft(ctx context.Context, id int64) error {
	const q = `UPDATE t_membership SET status = 'left', leave_time = CURRENT_TIMESTAMP WHERE id = ?`
	_, err := d.db.ExecContext(ctx, q, id)
	if err != nil {
		return fmt.Errorf("update t_membership.status=left: %w", err)
	}
	return nil
}
