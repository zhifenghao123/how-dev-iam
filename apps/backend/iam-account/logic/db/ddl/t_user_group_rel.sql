-- t_user_group_rel.sql（DDL）
-- ============================================================
-- 表名：t_user_group_rel
-- 表描述：成员-用户组关系（M:N）。关联 membership_id。
-- ============================================================
CREATE TABLE IF NOT EXISTS t_user_group_rel (
    id            BIGINT NOT NULL,
    account_id    BIGINT NOT NULL,
    membership_id BIGINT NOT NULL,
    group_id      BIGINT NOT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uk_user_group (membership_id, group_id),
    KEY idx_user_group_group (group_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='成员-用户组';
