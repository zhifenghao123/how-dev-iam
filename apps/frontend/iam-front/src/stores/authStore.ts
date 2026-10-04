import { create } from 'zustand';
import { persist } from 'zustand/middleware';

// AuthStore 保存前端"登录态"。
// 说明：iam-account 目前不发 JWT，登录态由前端本地维护，向后端调用时携带 X-User-Id。
// 后续接入 iam-authn 后，此 store 会改为存 access_token / refresh_token。
export interface AuthState {
  userId: number | null;
  nickname: string;
  activeAccountId: number | null;

  setUser: (userId: number, nickname: string) => void;
  setActiveAccount: (accountId: number) => void;
  logout: () => void;
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set) => ({
      userId: null,
      nickname: '',
      activeAccountId: null,
      setUser: (userId, nickname) => set({ userId, nickname }),
      setActiveAccount: (accountId) => set({ activeAccountId: accountId }),
      logout: () => set({ userId: null, nickname: '', activeAccountId: null }),
    }),
    { name: 'iam-front-auth' },
  ),
);
