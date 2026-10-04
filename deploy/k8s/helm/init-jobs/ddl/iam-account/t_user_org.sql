-- t_user_org.sql（DDL）
-- ============================================================
-- 表名：t_user_org
-- 表描述：成员-部门关系（M:N）。关联 membership_id，天然限定在同一 account 内。
-- ============================================================
CREATE TABLE IF NOT EXISTS t_user_org (
    id             BIGINT   NOT NULL,
    account_id     BIGINT   NOT NULL COMMENT '冗余,便于按企业查询',
    membership_id  BIGINT   NOT NULL,
    org_id         BIGINT   NOT NULL,
    is_primary     TINYINT  NOT NULL DEFAULT 0 COMMENT '主部门',
    position       VARCHAR(64) DEFAULT NULL COMMENT '岗位',
    join_time      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY uk_user_org (membership_id, org_id),
    KEY idx_user_org_org (org_id),
    KEY idx_user_org_account (account_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='成员-部门关系';
