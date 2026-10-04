-- t_invitation.sql（DDL）
-- ============================================================
-- 表名：t_invitation
-- 表描述：企业邀请成员的令牌记录。企业管理员生成邀请链接，
--         被邀请人点击链接完成注册/登录后接受，服务端消费 token 建立 Membership。
-- ============================================================
CREATE TABLE IF NOT EXISTS t_invitation (
    id            BIGINT       NOT NULL,
    account_id    BIGINT       NOT NULL COMMENT '归属企业 Account',
    inviter_id    BIGINT       NOT NULL COMMENT '邀请人 user_id',
    invitee_email VARCHAR(128)          DEFAULT NULL COMMENT '被邀请人邮箱',
    invitee_phone VARCHAR(32)           DEFAULT NULL COMMENT '被邀请人手机号',
    role          VARCHAR(32)  NOT NULL DEFAULT 'member' COMMENT 'admin|member|viewer',
    token         VARCHAR(64)  NOT NULL COMMENT '邀请令牌(URL 中携带)',
    status        VARCHAR(16)  NOT NULL DEFAULT 'pending' COMMENT 'pending|accepted|expired|revoked',
    expire_at     DATETIME     NOT NULL,
    accepted_by   BIGINT                DEFAULT NULL COMMENT '接受者 user_id',
    accepted_at   DATETIME              DEFAULT NULL,
    create_time   DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    update_time   DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY uk_invitation_token (token),
    KEY idx_invitation_account_status (account_id, status),
    KEY idx_invitation_email (invitee_email),
    KEY idx_invitation_phone (invitee_phone)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='邀请令牌';
