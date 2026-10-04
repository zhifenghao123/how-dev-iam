-- t_verify_code.sql（DDL）
-- ============================================================
-- 表名：t_verify_code
-- 表描述：邮箱/手机验证码。scene 区分场景（register / reset_pwd / bind / invite / mfa）。
--   - code_hash：存 hash 后的验证码，避免明文落库
--   - used / try_count：防重放 / 防爆破
--   - expire_at：过期时间，超时的记录由校验逻辑丢弃
-- ============================================================
CREATE TABLE IF NOT EXISTS t_verify_code (
    id           BIGINT       NOT NULL,
    channel      VARCHAR(16)  NOT NULL COMMENT 'email | sms',
    target       VARCHAR(128) NOT NULL COMMENT '邮箱/手机号',
    scene        VARCHAR(32)  NOT NULL COMMENT 'register|reset_pwd|bind|invite|mfa',
    code_hash    VARCHAR(255) NOT NULL COMMENT '验证码 hash',
    expire_at    DATETIME     NOT NULL,
    used         TINYINT      NOT NULL DEFAULT 0,
    used_at      DATETIME              DEFAULT NULL,
    try_count    INT          NOT NULL DEFAULT 0,
    client_ip    VARCHAR(64)           DEFAULT NULL,
    create_time  DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    KEY idx_verify_target_scene (target, scene, used, expire_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='验证码';
