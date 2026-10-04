-- t_audit_log.sql（DDL）
-- ============================================================
-- 表名：t_audit_log
-- 表描述：变更审计日志。谁、在哪个 Account、对什么对象、做了什么、结果如何。
-- ============================================================
CREATE TABLE IF NOT EXISTS t_audit_log (
    id           BIGINT       NOT NULL,
    account_id   BIGINT       NOT NULL DEFAULT 0 COMMENT '0=平台级操作',
    actor_id     BIGINT       NOT NULL COMMENT '操作者 user_id',
    action       VARCHAR(64)  NOT NULL COMMENT 'account.create|member.invite|...',
    target_type  VARCHAR(32)  NOT NULL,
    target_id    VARCHAR(64)  NOT NULL,
    before_val   JSON                  DEFAULT NULL,
    after_val    JSON                  DEFAULT NULL,
    request_id   VARCHAR(64)           DEFAULT NULL,
    client_ip    VARCHAR(64)           DEFAULT NULL,
    result       VARCHAR(16)  NOT NULL DEFAULT 'success',
    error_msg    VARCHAR(1024)         DEFAULT NULL,
    create_time  DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    KEY idx_audit_account_time (account_id, create_time),
    KEY idx_audit_actor (actor_id),
    KEY idx_audit_target (target_type, target_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='审计日志';
