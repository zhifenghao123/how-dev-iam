-- t_user_identity.sql（DDL）
-- ============================================================
-- 表名：t_user_identity
-- 表描述：用户外部身份绑定（GitHub / 微信 / Google / Apple ...）。
--         一个 User 可绑多个 provider；(provider, provider_uid) 唯一。
-- ============================================================
CREATE TABLE IF NOT EXISTS t_user_identity (
    id            BIGINT       NOT NULL,
    user_id       BIGINT       NOT NULL,
    provider      VARCHAR(32)  NOT NULL COMMENT 'github|wechat|google|apple|...',
    provider_uid  VARCHAR(255) NOT NULL,
    union_id      VARCHAR(255)          DEFAULT NULL,
    raw_profile   JSON                  DEFAULT NULL,
    linked_at     DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    create_time   DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    update_time   DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY uk_identity_provider_uid (provider, provider_uid),
    KEY idx_identity_user (user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用户外部身份绑定';
