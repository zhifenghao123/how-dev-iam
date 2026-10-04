-- t_authz_type.sql（DDL）
-- ============================================================
-- 表名：t_authz_type
-- 表描述：受支持的授权方式（authz types）字典表。每一行代表服务支持的一种
--         访问控制模型（如 RBAC / ABAC），供 controller 层在 SupportTypes
--         接口枚举返回，并供策略评估子系统做模型选择 / 兼容性判断。
-- 数据稳定性：低频变更（产品迭代级），由初始化 dml 灌入；运行期可被运维直接维护。
-- ============================================================
--
-- 字段语义与 t_authn_type 一致：
--   id          自增主键，无业务含义；
--   code        机器枚举值，业务唯一键（如 "rbac"）；
--   name        面向人类的展示名；
--   description 补充说明；
--   sort        稳定排序键。
CREATE TABLE IF NOT EXISTS t_authz_type (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,                    -- 自增主键
    code        TEXT    NOT NULL UNIQUE,                              -- 业务唯一键
    name        TEXT    NOT NULL,                                     -- 展示名
    description TEXT    NOT NULL DEFAULT '',                          -- 补充说明
    sort        INTEGER NOT NULL DEFAULT 0                            -- 排序键
);
