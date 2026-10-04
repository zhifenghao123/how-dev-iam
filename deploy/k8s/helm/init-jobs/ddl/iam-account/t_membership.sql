-- t_membership.sql（DDL）
-- ============================================================
-- 表名：t_membership
-- 表描述：用户与账户的成员关系（M:N）。
--   一个 User 可加入多个 Account（个人 + 若干企业），
--   在不同 Account 下有不同 role（owner / admin / member / viewer）。
--   "子用户"本质就是一条 Membership。
-- ============================================================
CREATE TABLE IF NOT EXISTS t_membership (
    id            BIGINT       NOT NULL,
    account_id    BIGINT       NOT NULL,
    user_id       BIGINT       NOT NULL,
    role          VARCHAR(32)  NOT NULL DEFAULT 'member' COMMENT 'owner|admin|member|viewer',
    status        VARCHAR(16)  NOT NULL DEFAULT 'active' COMMENT 'pending|active|disabled|left',
    display_name  VARCHAR(64)           DEFAULT NULL COMMENT '在此账户内的显示名',
    join_source   VARCHAR(32)  NOT NULL DEFAULT 'invited' COMMENT 'owner_init|invited|org_created',
    join_time     DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    leave_time    DATETIME              DEFAULT NULL,
    invited_by    BIGINT                DEFAULT NULL COMMENT '邀请人 user_id',
    create_time   DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    update_time   DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY uk_membership (account_id, user_id),
    KEY idx_membership_user (user_id, status),
    KEY idx_membership_account_role (account_id, role, status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='成员关系(User×Account)';
