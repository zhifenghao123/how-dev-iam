// user_dao.go 是 t_user 表的数据访问层。
package dao

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// UserPO 是 t_user 表的持久化对象。
type UserPO struct {
	ID              int64
	Username        sql.NullString
	Email           sql.NullString
	EmailVerified   bool
	Phone           sql.NullString
	PhoneVerified   bool
	Nickname        sql.NullString
	RealName        sql.NullString
	AvatarURL       sql.NullString
	Gender          int
	Status          string
	RegisterSource  string
	MustChangePwd   bool
	IsPlatformAdmin bool
	LastLoginAt     sql.NullTime
	LastLoginIP     sql.NullString
	CreateTime      time.Time
	UpdateTime      time.Time
}

// UserDao 用户主表访问接口。
type UserDao interface {
	// Insert 插入新用户。id / 时间字段由调用方或数据库默认值负责。
	Insert(ctx context.Context, tx Executor, po *UserPO) error
	// GetByID 按 id 查询（软删过滤）。未找到返回 (nil, nil)。
	GetByID(ctx context.Context, id int64) (*UserPO, error)
	// GetByEmail 按邮箱查询（软删过滤）。
	GetByEmail(ctx context.Context, email string) (*UserPO, error)
	// GetByPhone 按手机号查询（软删过滤）。
	GetByPhone(ctx context.Context, phone string) (*UserPO, error)
	// UpdateStatus 更新 status。
	UpdateStatus(ctx context.Context, id int64, status string) error
}

// Executor 抽象 *sql.DB / *sql.Tx，便于在事务内外复用同一个 dao 方法。
type Executor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// UserMySQLDao 是 UserDao 的 MySQL 实现。
type UserMySQLDao struct {
	db *sql.DB
}

// NewUserMySQLDao 构造 MySQL 实现。
func NewUserMySQLDao(db *sql.DB) *UserMySQLDao {
	return &UserMySQLDao{db: db}
}

const userColumns = `id, username, email, email_verified, phone, phone_verified,
    nickname, real_name, avatar_url, gender, status, register_source,
    must_change_pwd, is_platform_admin, last_login_at, last_login_ip,
    create_time, update_time`

func scanUser(row interface {
	Scan(dest ...any) error
}) (*UserPO, error) {
	var po UserPO
	var emailVerified, phoneVerified, mustChangePwd, isPlatformAdmin int
	err := row.Scan(&po.ID, &po.Username, &po.Email, &emailVerified, &po.Phone, &phoneVerified,
		&po.Nickname, &po.RealName, &po.AvatarURL, &po.Gender, &po.Status, &po.RegisterSource,
		&mustChangePwd, &isPlatformAdmin, &po.LastLoginAt, &po.LastLoginIP,
		&po.CreateTime, &po.UpdateTime)
	if err != nil {
		return nil, err
	}
	po.EmailVerified = emailVerified == 1
	po.PhoneVerified = phoneVerified == 1
	po.MustChangePwd = mustChangePwd == 1
	po.IsPlatformAdmin = isPlatformAdmin == 1
	return &po, nil
}

// Insert 插入新用户。
func (d *UserMySQLDao) Insert(ctx context.Context, tx Executor, po *UserPO) error {
	if tx == nil {
		tx = d.db
	}
	const q = `INSERT INTO t_user
      (id, username, email, email_verified, phone, phone_verified,
       nickname, real_name, avatar_url, gender, status, register_source,
       must_change_pwd, is_platform_admin)
      VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?)`
	_, err := tx.ExecContext(ctx, q,
		po.ID, po.Username, po.Email, boolInt(po.EmailVerified), po.Phone, boolInt(po.PhoneVerified),
		po.Nickname, po.RealName, po.AvatarURL, po.Gender, po.Status, po.RegisterSource,
		boolInt(po.MustChangePwd), boolInt(po.IsPlatformAdmin))
	if err != nil {
		return fmt.Errorf("insert t_user: %w", err)
	}
	return nil
}

// GetByID 按 id 查询（软删过滤）。
func (d *UserMySQLDao) GetByID(ctx context.Context, id int64) (*UserPO, error) {
	q := `SELECT ` + userColumns + ` FROM t_user WHERE id = ? AND deleted = 0`
	po, err := scanUser(d.db.QueryRowContext(ctx, q, id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("query t_user by id: %w", err)
	}
	return po, nil
}

// GetByEmail 按邮箱查询。
func (d *UserMySQLDao) GetByEmail(ctx context.Context, email string) (*UserPO, error) {
	q := `SELECT ` + userColumns + ` FROM t_user WHERE email = ? AND deleted = 0`
	po, err := scanUser(d.db.QueryRowContext(ctx, q, email))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("query t_user by email: %w", err)
	}
	return po, nil
}

// GetByPhone 按手机号查询。
func (d *UserMySQLDao) GetByPhone(ctx context.Context, phone string) (*UserPO, error) {
	q := `SELECT ` + userColumns + ` FROM t_user WHERE phone = ? AND deleted = 0`
	po, err := scanUser(d.db.QueryRowContext(ctx, q, phone))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("query t_user by phone: %w", err)
	}
	return po, nil
}

// UpdateStatus 更新 status。
func (d *UserMySQLDao) UpdateStatus(ctx context.Context, id int64, status string) error {
	_, err := d.db.ExecContext(ctx, `UPDATE t_user SET status = ? WHERE id = ?`, status, id)
	if err != nil {
		return fmt.Errorf("update t_user.status: %w", err)
	}
	return nil
}

// boolInt 把 bool 转成 TINYINT(0/1)。
func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
