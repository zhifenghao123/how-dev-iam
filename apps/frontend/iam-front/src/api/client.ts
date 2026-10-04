import axios, { AxiosError, AxiosInstance } from 'axios';
import { message } from 'antd';
import { useAuthStore } from '@/stores/authStore';
import type { ApiOK, ApiErr } from './types';

// axios 实例：默认 baseURL 为 /api，走 vite dev proxy 到 iam-account。
export const http: AxiosInstance = axios.create({
  baseURL: '/api',
  timeout: 15000,
});

// 请求拦截器：自动带 X-User-Id（未接入正式 authn 前，模拟登录态）。
http.interceptors.request.use((cfg) => {
  const uid = useAuthStore.getState().userId;
  if (uid) {
    cfg.headers = cfg.headers ?? {};
    (cfg.headers as any)['X-User-Id'] = String(uid);
  }
  return cfg;
});

// 响应拦截器：解出 data；失败抛出 ApiErr。
http.interceptors.response.use(
  (resp) => {
    const body = resp.data as ApiOK<any> | ApiErr;
    if ('code' in body && body.code !== 'OK') {
      return Promise.reject(body as ApiErr);
    }
    return resp;
  },
  (err: AxiosError) => {
    const body = err.response?.data as ApiErr | undefined;
    const msg = body?.message || err.message || 'network error';
    message.error(msg);
    return Promise.reject(body ?? { code: 'NETWORK', message: msg });
  },
);

// 便捷方法：直接返回 data 字段。
export async function post<T>(url: string, body?: any): Promise<T> {
  const r = await http.post<ApiOK<T>>(url, body);
  return r.data.data;
}

export async function get<T>(url: string, params?: any): Promise<T> {
  const r = await http.get<ApiOK<T>>(url, { params });
  return r.data.data;
}

export async function del<T>(url: string): Promise<T> {
  const r = await http.delete<ApiOK<T>>(url);
  return r.data.data;
}
