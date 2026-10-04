// account_dao.go 是 t_account 表的数据访问层。
package dao

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// AccountPO 是 t_account 表的持久化对象。
type AccountPO struct {
	ID          int64
	Type        string // personal | organization
	Code        string
	Name        string
	OwnerUserID int64
	LogoURL     sql.NullString
	Status      string
	Verified    bool
	Plan        string
	MemberLimit int
	CreateTime  time.Time
	UpdateTime  time.Time
}

// AccountDao 账户空间访问接口。
type AccountDao interface {
	Insert(ctx context.Context, tx Executor, po *AccountPO) error
	GetByID(ctx context.Context, id int64) (*AccountPO, error)
	ListByOwner(ctx context.Context, ownerUserID int64) ([]AccountPO, error)
}

// AccountMySQLDao MySQL 实现。
type AccountMySQLDao struct {
	db *sql.DB
}

// NewAccountMySQLDao 构造。
func NewAccountMySQLDao(db *sql.DB) *AccountMySQLDao {
	return &AccountMySQLDao{db: db}
}

const accountColumns = `id, type, code, name, owner_user_id, logo_url, status, verified,
    plan, member_limit, create_time, update_time`

func scanAccount(row interface {
	Scan(dest ...any) error
}) (*AccountPO, error) {
	var po AccountPO
	var verified int
	err := row.Scan(&po.ID, &po.Type, &po.Code, &po.Name, &po.OwnerUserID, &po.LogoURL,
		&po.Status, &verified, &po.Plan, &po.MemberLimit, &po.CreateTime, &po.UpdateTime)
	if err != nil {
		return nil, err
	}
	po.Verified = verified == 1
	return &po, nil
}

// Insert 插入新账户。
func (d *AccountMySQLDao) Insert(ctx context.Context, tx Executor, po *AccountPO) error {
	if tx == nil {
		tx = d.db
	}
	const q = `INSERT INTO t_account
      (id, type, code, name, owner_user_id, logo_url, status, verified, plan, member_limit)
      VALUES (?,?,?,?,?,?,?,?,?,?)`
	_, err := tx.ExecContext(ctx, q, po.ID, po.Type, po.Code, po.Name, po.OwnerUserID, po.LogoURL,
		po.Status, boolInt(po.Verified), po.Plan, po.MemberLimit)
	if err != nil {
		return fmt.Errorf("insert t_account: %w", err)
	}
	return nil
}

// GetByID 按 id 查询。
func (d *AccountMySQLDao) GetByID(ctx context.Context, id int64) (*AccountPO, error) {
	q := `SELECT ` + accountColumns + ` FROM t_account WHERE id = ? AND deleted = 0`
	po, err := scanAccount(d.db.QueryRowContext(ctx, q, id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("query t_account: %w", err)
	}
	return po, nil
}

// ListByOwner 列出某 owner 名下所有 Account。
func (d *AccountMySQLDao) ListByOwner(ctx context.Context, ownerUserID int64) ([]AccountPO, error) {
	q := `SELECT ` + accountColumns + ` FROM t_account
	      WHERE owner_user_id = ? AND deleted = 0 ORDER BY create_time ASC`
	rows, err := d.db.QueryContext(ctx, q, ownerUserID)
	if err != nil {
		return nil, fmt.Errorf("list t_account by owner: %w", err)
	}
	defer rows.Close()

	out := make([]AccountPO, 0, 4)
	for rows.Next() {
		po, err := scanAccount(rows)
		if err != nil {
			return nil, fmt.Errorf("scan t_account: %w", err)
		}
		out = append(out, *po)
	}
	return out, rows.Err()
}
