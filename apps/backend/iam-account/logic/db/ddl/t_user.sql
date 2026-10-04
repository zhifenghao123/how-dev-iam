-- t_user.sql（DDL）
-- ============================================================
-- 表名：t_user
-- 表描述：自然人账号（Identity），平台所有登录都以此表为主体。
-- 数据稳定性：中等（用户注册 / 状态变更）。
-- ============================================================
CREATE TABLE IF NOT EXISTS t_user (
    id                BIGINT       NOT NULL COMMENT '雪花ID',
    username          VARCHAR(64)           DEFAULT NULL COMMENT '登录名(可选,全平台唯一)',
    email             VARCHAR(128)          DEFAULT NULL COMMENT '邮箱,全平台唯一',
    email_verified    TINYINT      NOT NULL DEFAULT 0 COMMENT '邮箱是否验证',
    phone             VARCHAR(32)           DEFAULT NULL COMMENT '手机号E.164,全平台唯一',
    phone_verified    TINYINT      NOT NULL DEFAULT 0 COMMENT '手机号是否验证',
    nickname          VARCHAR(64)           DEFAULT NULL COMMENT '昵称',
    real_name         VARCHAR(64)           DEFAULT NULL COMMENT '真实姓名',
    avatar_url        VARCHAR(512)          DEFAULT NULL COMMENT '头像URL',
    gender            TINYINT      NOT NULL DEFAULT 0 COMMENT '0未知 1男 2女',
    status            VARCHAR(16)  NOT NULL DEFAULT 'active' COMMENT 'pending|active|locked|disabled|deleted',
    register_source   VARCHAR(32)  NOT NULL DEFAULT 'self' COMMENT 'self|invited|org_created',
    must_change_pwd   TINYINT      NOT NULL DEFAULT 0 COMMENT '下次登录强制改密',
    is_platform_admin TINYINT      NOT NULL DEFAULT 0 COMMENT '平台超管',
    last_login_at     DATETIME              DEFAULT NULL COMMENT '最后登录时间',
    last_login_ip     VARCHAR(64)           DEFAULT NULL COMMENT '最后登录IP',
    create_time       DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    update_time       DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted           TINYINT      NOT NULL DEFAULT 0 COMMENT '软删 0未删 1已删',
    PRIMARY KEY (id),
    UNIQUE KEY uk_user_username (username, deleted),
    UNIQUE KEY uk_user_email    (email, deleted),
    UNIQUE KEY uk_user_phone    (phone, deleted),
    KEY idx_user_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='自然人账号';
