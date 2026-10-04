// TypeScript 类型：与后端 service.*View / Resp 保持同步。
// 后端 JSON 字段为 snake_case，本文件保持一致，避免额外映射。

export interface UserView {
  id: number;
  username?: string;
  email?: string;
  email_verified: boolean;
  phone?: string;
  phone_verified: boolean;
  nickname?: string;
  real_name?: string;
  avatar_url?: string;
  gender: number;
  status: string;
  must_change_pwd: boolean;
  is_platform_admin: boolean;
  last_login_at?: string;
  create_time: string;
}

export interface AccountView {
  id: number;
  type: 'personal' | 'organization';
  code: string;
  name: string;
  owner_user_id: number;
  logo_url?: string;
  status: string;
  verified: boolean;
  plan: string;
  member_limit: number;
  create_time: string;
}

export interface MembershipView {
  id: number;
  account_id: number;
  user_id: number;
  role: 'owner' | 'admin' | 'member' | 'viewer';
  status: 'pending' | 'active' | 'disabled' | 'left';
  display_name?: string;
  join_source: string;
  join_time: string;
  leave_time?: string;
  invited_by?: number;

  // 可选展开
  user?: UserView;
  account?: AccountView;
}

export interface InvitationView {
  id: number;
  account_id: number;
  inviter_id: number;
  invitee_email?: string;
  invitee_phone?: string;
  role: string;
  token: string;
  status: string;
  expire_at: string;
  create_time: string;
}

export interface RegisterResp {
  user_id: number;
  account_id: number;
  account: AccountView;
}

export interface SendVerifyCodeResp {
  expire_in_sec: number;
  debug_code?: string;
}

// 统一响应包装
export interface ApiOK<T> {
  code: 'OK';
  data: T;
}

export interface ApiErr {
  code: string;
  message: string;
}
