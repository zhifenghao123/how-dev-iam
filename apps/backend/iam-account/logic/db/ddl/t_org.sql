-- t_org.sql（DDL）
-- ============================================================
-- 表名：t_org
-- 表描述：企业组织/部门树，scope by account_id。个人 Account 不使用此表。
--   - parent_id + path 混合结构：写简单、查子树快
-- ============================================================
CREATE TABLE IF NOT EXISTS t_org (
    id           BIGINT        NOT NULL,
    account_id   BIGINT        NOT NULL COMMENT '所属企业 Account',
    parent_id    BIGINT        NOT NULL DEFAULT 0,
    code         VARCHAR(64)   NOT NULL,
    name         VARCHAR(128)  NOT NULL,
    type         VARCHAR(16)   NOT NULL DEFAULT 'dept' COMMENT 'company|dept|team',
    path         VARCHAR(1024) NOT NULL DEFAULT '' COMMENT '物化路径 /1/12/123',
    level        INT           NOT NULL DEFAULT 1,
    sort_order   INT           NOT NULL DEFAULT 0,
    leader_mid   BIGINT                 DEFAULT NULL COMMENT '负责人 membership_id',
    status       TINYINT       NOT NULL DEFAULT 1,
    create_time  DATETIME      NOT NULL DEFAULT CURRENT_TIMESTAMP,
    update_time  DATETIME      NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted      TINYINT       NOT NULL DEFAULT 0,
    PRIMARY KEY (id),
    UNIQUE KEY uk_org_account_code (account_id, code, deleted),
    KEY idx_org_account_parent (account_id, parent_id),
    KEY idx_org_path (account_id, path(255))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='企业组织/部门';
