-- t_group.sql（DDL）
-- ============================================================
-- 表名：t_group
-- 表描述：企业内用户组（横向逻辑分组），与部门解耦，用于灵活授权。
--         type=static 手工维护成员；type=dynamic 由 rule_expr 匹配。
-- ============================================================
CREATE TABLE IF NOT EXISTS t_group (
    id           BIGINT       NOT NULL,
    account_id   BIGINT       NOT NULL,
    code         VARCHAR(64)  NOT NULL,
    name         VARCHAR(128) NOT NULL,
    description  VARCHAR(512)          DEFAULT NULL,
    type         VARCHAR(16)  NOT NULL DEFAULT 'static' COMMENT 'static|dynamic',
    rule_expr    TEXT                  DEFAULT NULL COMMENT '动态组匹配规则',
    create_time  DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    update_time  DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted      TINYINT      NOT NULL DEFAULT 0,
    PRIMARY KEY (id),
    UNIQUE KEY uk_group_account_code (account_id, code, deleted)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='企业内用户组';
