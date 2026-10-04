-- t_outbox_event.sql（DDL）
-- ============================================================
-- 表名：t_outbox_event
-- 表描述：事件发件箱（Transactional Outbox）。业务变更 + 事件同事务写入，
--         由独立 worker 拉取并投递至 MQ，保证消息与业务变更最终一致。
-- ============================================================
CREATE TABLE IF NOT EXISTS t_outbox_event (
    id           BIGINT       NOT NULL,
    aggregate    VARCHAR(32)  NOT NULL COMMENT 'user|account|membership|org',
    aggregate_id VARCHAR(64)  NOT NULL,
    event_type   VARCHAR(64)  NOT NULL COMMENT 'user.created|account.created|...',
    payload      JSON         NOT NULL,
    status       VARCHAR(16)  NOT NULL DEFAULT 'pending' COMMENT 'pending|sent|failed',
    retry_count  INT          NOT NULL DEFAULT 0,
    next_retry_at DATETIME             DEFAULT NULL,
    create_time  DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    update_time  DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    KEY idx_outbox_status (status, next_retry_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='事件发件箱';
