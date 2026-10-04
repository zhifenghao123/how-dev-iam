-- t_account_sso.sql（DDL）
-- ============================================================
-- 表名：t_account_sso
-- 表描述：企业 SSO 配置（本 IAM 作为 SP 对接企业自有 IdP）。
--         config JSON 承载协议差异（issuer / metadata_url / client_id / ...）。
-- ============================================================
CREATE TABLE IF NOT EXISTS t_account_sso (
    id           BIGINT       NOT NULL,
    account_id   BIGINT       NOT NULL COMMENT '归属企业 Account',
    protocol     VARCHAR(16)  NOT NULL COMMENT 'oidc | saml | ldap',
    name         VARCHAR(128) NOT NULL,
    config       JSON         NOT NULL COMMENT '协议配置',
    domain       VARCHAR(128)          DEFAULT NULL COMMENT '自动分发登录域(例如 @xxx.com)',
    enabled      TINYINT      NOT NULL DEFAULT 1,
    create_time  DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    update_time  DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted      TINYINT      NOT NULL DEFAULT 0,
    PRIMARY KEY (id),
    KEY idx_sso_account (account_id),
    KEY idx_sso_domain (domain)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='企业SSO配置';
