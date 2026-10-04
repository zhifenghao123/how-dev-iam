-- t_user_credential.sql（DDL）
-- ============================================================
-- 表名：t_user_credential
-- 表描述：用户凭据。密码/MFA/失败计数与 t_user 分表，做字段级安全隔离。
-- 数据稳定性：中等（修改密码 / 失败尝试）。
-- ============================================================
CREATE TABLE IF NOT EXISTS t_user_credential (
    user_id             BIGINT       NOT NULL COMMENT '关联 t_user.id',
    password_hash       VARCHAR(255) NOT NULL COMMENT '密码 hash',
    password_algo       VARCHAR(16)  NOT NULL DEFAULT 'bcrypt' COMMENT 'bcrypt|argon2',
    password_updated_at DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    password_expire_at  DATETIME              DEFAULT NULL COMMENT '密码过期时间',
    failed_attempts     INT          NOT NULL DEFAULT 0 COMMENT '连续失败次数',
    locked_until        DATETIME              DEFAULT NULL COMMENT '锁定至',
    mfa_enabled         TINYINT      NOT NULL DEFAULT 0,
    mfa_secret          VARCHAR(128)          DEFAULT NULL COMMENT 'TOTP 密钥(加密)',
    recovery_codes      TEXT                  DEFAULT NULL COMMENT '恢复码 JSON(hash)',
    history_hashes      TEXT                  DEFAULT NULL COMMENT '最近N次历史密码 hash',
    create_time         DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    update_time         DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用户凭据';
