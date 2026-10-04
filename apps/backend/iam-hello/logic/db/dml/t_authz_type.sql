-- t_authz_type-dml.sql
-- 服务启动时写入 t_authz_type 表的种子数据。
--
-- 顺序按"通用度"递增：从经典 RBAC 起步，向更细粒度模型延伸。
-- INSERT OR IGNORE 保证幂等。
INSERT OR IGNORE INTO t_authz_type (code, name, description, sort) VALUES ('rbac',  'RBAC',  '基于角色的访问控制', 0);
INSERT OR IGNORE INTO t_authz_type (code, name, description, sort) VALUES ('abac',  'ABAC',  '基于属性的访问控制', 1);
INSERT OR IGNORE INTO t_authz_type (code, name, description, sort) VALUES ('acl',   'ACL',   '基于资源访问控制列表', 2);
INSERT OR IGNORE INTO t_authz_type (code, name, description, sort) VALUES ('rebac', 'ReBAC', '基于关系的访问控制', 3);
