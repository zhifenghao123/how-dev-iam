import { post, get, del } from './client';
import type {
  SendVerifyCodeResp,
  RegisterResp,
  MembershipView,
  InvitationView,
} from './types';

// ========== 验证码 ==========
export function apiSendVerifyCode(params: {
  channel: 'email' | 'sms';
  target: string;
  scene: 'register' | 'reset_pwd' | 'bind' | 'invite' | 'mfa';
}) {
  return post<SendVerifyCodeResp>('/verify-code', params);
}

// ========== 注册 ==========
export function apiRegisterPersonal(params: {
  email?: string;
  phone?: string;
  code: string;
  password: string;
  nickname?: string;
}) {
  return post<RegisterResp>('/register/personal', params);
}

export function apiRegisterOrganization(params: {
  name: string;
  legal_name?: string;
  unified_credit_no?: string;
  industry?: string;
  scale?: string;
  province?: string;
  city?: string;
  address?: string;
  contact_email?: string;
  contact_phone?: string;
}) {
  return post<RegisterResp>('/register/organization', params);
}

// ========== 我的账户 ==========
export function apiListMyAccounts() {
  return get<MembershipView[]>('/me/accounts');
}

// ========== 邀请 ==========
export function apiCreateInvitation(
  accountId: number,
  params: { email?: string; phone?: string; role?: string },
) {
  return post<InvitationView>(`/accounts/${accountId}/invitations`, params);
}

export function apiAcceptInvitation(token: string) {
  return post<MembershipView>(`/invitations/${token}/accept`, {});
}

// ========== 成员 ==========
export function apiCreateMember(
  accountId: number,
  params: {
    email?: string;
    phone?: string;
    password: string;
    role?: string;
    nickname?: string;
    real_name?: string;
  },
) {
  return post<MembershipView>(`/accounts/${accountId}/members`, params);
}

export function apiListMembers(accountId: number, offset = 0, limit = 50) {
  return get<MembershipView[]>(`/accounts/${accountId}/members`, {
    offset,
    limit,
  });
}

export function apiRemoveMember(accountId: number, membershipId: number) {
  return del<Record<string, never>>(
    `/accounts/${accountId}/members/${membershipId}`,
  );
}
