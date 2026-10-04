-- t_account.sql（DDL）
-- ============================================================
-- 表名：t_account
-- 表描述：账户空间（Account）。分两种类型：
--   - personal：个人账户，注册时自动创建，1 User ↔ 1 Personal Account
--   - organization：企业账户，由某 User 创建，可含多个 Member
-- 业务数据、订阅、计费都以 account_id 为主 scope。
-- ============================================================
CREATE TABLE IF NOT EXISTS t_account (
    id             BIGINT       NOT NULL COMMENT '账户ID(雪花)',
    type           VARCHAR(16)  NOT NULL COMMENT 'personal | organization',
    code           VARCHAR(64)  NOT NULL COMMENT '账户编码(人类可读,全局唯一)',
    name           VARCHAR(128) NOT NULL COMMENT '账户显示名',
    owner_user_id  BIGINT       NOT NULL COMMENT '所有者 user_id',
    logo_url       VARCHAR(512)          DEFAULT NULL,
    status         VARCHAR(16)  NOT NULL DEFAULT 'active' COMMENT 'active|frozen|dissolved|deleted',
    verified       TINYINT      NOT NULL DEFAULT 0 COMMENT '实名/企业认证',
    plan           VARCHAR(32)  NOT NULL DEFAULT 'free' COMMENT '订阅套餐',
    member_limit   INT          NOT NULL DEFAULT 5 COMMENT '成员上限(个人=1,企业按套餐)',
    create_time    DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    update_time    DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted        TINYINT      NOT NULL DEFAULT 0,
    PRIMARY KEY (id),
    UNIQUE KEY uk_account_code (code, deleted),
    KEY idx_account_owner (owner_user_id),
    KEY idx_account_type_status (type, status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='账户空间';
