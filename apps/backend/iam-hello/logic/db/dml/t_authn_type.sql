-- t_authn_type-dml.sql
-- 服务启动时写入 t_authn_type 表的种子数据。
--
-- 顺序按"常用度"排列，sort 字段控制 ListAll 的返回顺序。
-- INSERT OR IGNORE 保证幂等：多次启动不会重复插入，也不会覆盖运行期人为修改。
INSERT OR IGNORE INTO t_authn_type (code, name, description, sort) VALUES ('password', '用户名密码', '经典账号 + 密码登录', 0);
INSERT OR IGNORE INTO t_authn_type (code, name, description, sort) VALUES ('sms',      '短信验证码', '手机号 + 一次性验证码', 1);
INSERT OR IGNORE INTO t_authn_type (code, name, description, sort) VALUES ('oauth2',   'OAuth2',     '通过第三方 OAuth2 提供方授权登录', 2);
INSERT OR IGNORE INTO t_authn_type (code, name, description, sort) VALUES ('oidc',     'OIDC',       '基于 OpenID Connect 的单点登录', 3);
INSERT OR IGNORE INTO t_authn_type (code, name, description, sort) VALUES ('ldap',     'LDAP',       '对接企业目录服务', 4);
