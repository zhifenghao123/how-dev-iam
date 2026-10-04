// account_org_profile_dao.go 是 t_account_org_profile 表的数据访问层。
package dao

import (
	"context"
	"database/sql"
	"fmt"
)

// AccountOrgProfilePO 是 t_account_org_profile 的持久化对象。
type AccountOrgProfilePO struct {
	AccountID       int64
	LegalName       string
	UnifiedCreditNo sql.NullString
	Industry        sql.NullString
	Scale           sql.NullString
	Country         string
	Province        sql.NullString
	City            sql.NullString
	Address         sql.NullString
	ContactEmail    sql.NullString
	ContactPhone    sql.NullString
	LicenseURL      sql.NullString
	VerifyStatus    string
}

// AccountOrgProfileDao 企业账户扩展信息访问接口。
type AccountOrgProfileDao interface {
	Insert(ctx context.Context, tx Executor, po *AccountOrgProfilePO) error
	GetByAccountID(ctx context.Context, accountID int64) (*AccountOrgProfilePO, error)
}

// AccountOrgProfileMySQLDao MySQL 实现。
type AccountOrgProfileMySQLDao struct {
	db *sql.DB
}

// NewAccountOrgProfileMySQLDao 构造。
func NewAccountOrgProfileMySQLDao(db *sql.DB) *AccountOrgProfileMySQLDao {
	return &AccountOrgProfileMySQLDao{db: db}
}

// Insert 插入。
func (d *AccountOrgProfileMySQLDao) Insert(ctx context.Context, tx Executor, po *AccountOrgProfilePO) error {
	if tx == nil {
		tx = d.db
	}
	const q = `INSERT INTO t_account_org_profile
        (account_id, legal_name, unified_credit_no, industry, scale, country,
         province, city, address, contact_email, contact_phone, license_url, verify_status)
        VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)`
	country := po.Country
	if country == "" {
		country = "CN"
	}
	verifyStatus := po.VerifyStatus
	if verifyStatus == "" {
		verifyStatus = "unverified"
	}
	_, err := tx.ExecContext(ctx, q,
		po.AccountID, po.LegalName, po.UnifiedCreditNo, po.Industry, po.Scale, country,
		po.Province, po.City, po.Address, po.ContactEmail, po.ContactPhone, po.LicenseURL, verifyStatus)
	if err != nil {
		return fmt.Errorf("insert t_account_org_profile: %w", err)
	}
	return nil
}

// GetByAccountID 按 account_id 查询。
func (d *AccountOrgProfileMySQLDao) GetByAccountID(ctx context.Context, accountID int64) (*AccountOrgProfilePO, error) {
	const q = `SELECT account_id, legal_name, unified_credit_no, industry, scale, country,
                      province, city, address, contact_email, contact_phone, license_url, verify_status
               FROM t_account_org_profile WHERE account_id = ?`
	var po AccountOrgProfilePO
	err := d.db.QueryRowContext(ctx, q, accountID).Scan(
		&po.AccountID, &po.LegalName, &po.UnifiedCreditNo, &po.Industry, &po.Scale, &po.Country,
		&po.Province, &po.City, &po.Address, &po.ContactEmail, &po.ContactPhone, &po.LicenseURL, &po.VerifyStatus,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("query t_account_org_profile: %w", err)
	}
	return &po, nil
}
