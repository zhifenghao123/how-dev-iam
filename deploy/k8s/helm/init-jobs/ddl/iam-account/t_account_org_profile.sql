-- t_account_org_profile.sql（DDL）
-- ============================================================
-- 表名：t_account_org_profile
-- 表描述：企业账户扩展信息（一对一挂到 t_account, type=organization 时使用）。
-- ============================================================
CREATE TABLE IF NOT EXISTS t_account_org_profile (
    account_id        BIGINT       NOT NULL COMMENT '关联 t_account.id',
    legal_name        VARCHAR(255) NOT NULL COMMENT '企业注册名',
    unified_credit_no VARCHAR(64)           DEFAULT NULL COMMENT '统一社会信用代码',
    industry          VARCHAR(64)           DEFAULT NULL,
    scale             VARCHAR(32)           DEFAULT NULL COMMENT '规模: 1-49 / 50-499 / 500+',
    country           VARCHAR(32)  NOT NULL DEFAULT 'CN',
    province          VARCHAR(64)           DEFAULT NULL,
    city              VARCHAR(64)           DEFAULT NULL,
    address           VARCHAR(255)          DEFAULT NULL,
    contact_email     VARCHAR(128)          DEFAULT NULL,
    contact_phone     VARCHAR(32)           DEFAULT NULL,
    license_url       VARCHAR(512)          DEFAULT NULL COMMENT '营业执照图片URL',
    verify_status     VARCHAR(16)  NOT NULL DEFAULT 'unverified' COMMENT 'unverified|reviewing|verified|rejected',
    verify_time       DATETIME              DEFAULT NULL,
    verify_by         BIGINT                DEFAULT NULL,
    reject_reason     VARCHAR(512)          DEFAULT NULL,
    create_time       DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    update_time       DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (account_id),
    UNIQUE KEY uk_account_org_credit (unified_credit_no)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='企业账户扩展信息';
