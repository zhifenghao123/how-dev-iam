-- t_authn_type.sql（DDL）
-- ============================================================
-- 表名：t_authn_type
-- 表描述：受支持的认证方式（authn types）字典表。每一行代表服务支持的一种
--         登录 / 身份核验方式，供 controller 层在 SupportTypes 接口枚举返回，
--         也供后续具体登录流程做合法性校验。
-- 数据稳定性：低频变更（产品迭代级），由初始化 dml 灌入；运行期可被运维直接维护。
-- ============================================================
--
-- 字段语义：
--   id          自增主键，无业务含义，仅做物理唯一定位（便于关联表外键、变更日志等）；
--   code        机器枚举值，业务唯一键，上下游用其做路由 / 鉴权决策；
--   name        面向人类的展示名（前端下拉用）；
--   description 补充说明，便于接入方理解该认证方式语义；
--   sort        稳定排序键，避免依赖隐式 rowid。
--
-- 兼容 SQLite；移植到 MySQL/PG 时：
--   - INTEGER PRIMARY KEY AUTOINCREMENT 改为 BIGINT AUTO_INCREMENT / BIGSERIAL；
--   - 行尾 "-- 描述" 可改为 COMMENT '描述' 子句保留列注释。
CREATE TABLE IF NOT EXISTS t_authn_type (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,                    -- 自增主键
    code        TEXT    NOT NULL UNIQUE,                              -- 业务唯一键
    name        TEXT    NOT NULL,                                     -- 展示名
    description TEXT    NOT NULL DEFAULT '',                          -- 补充说明
    sort        INTEGER NOT NULL DEFAULT 0                            -- 排序键
);
