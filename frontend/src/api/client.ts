import axios, { AxiosError, type InternalAxiosRequestConfig } from 'axios';
import { tokenStorage } from '../auth/tokenStorage';
import type { ApiErrorBody, RefreshResponse } from '../types/api';

const baseURL = import.meta.env.VITE_API_URL || 'http://localhost:8080/api';

export const apiClient = axios.create({ baseURL, timeout: 30_000 });
const refreshClient = axios.create({ baseURL, timeout: 30_000 });

apiClient.interceptors.request.use((config) => {
  const token = tokenStorage.getAccess();
  if (token) config.headers.Authorization = `Bearer ${token}`;
  return config;
});

interface RetriableConfig extends InternalAxiosRequestConfig {
  _retry?: boolean;
}

let refreshPromise: Promise<string> | null = null;

async function refreshAccessToken(): Promise<string> {
  if (!refreshPromise) {
    refreshPromise = (async () => {
      const refreshToken = tokenStorage.getRefresh();
      if (!refreshToken) throw new Error('Refresh token отсутствует');
      const { data } = await refreshClient.post<RefreshResponse>('/auth/refresh', { refresh_token: refreshToken });
      tokenStorage.set(data.access_token, data.refresh_token);
      window.dispatchEvent(new CustomEvent('auth:token-refreshed'));
      return data.access_token;
    })().finally(() => {
      refreshPromise = null;
    });
  }
  return refreshPromise;
}

apiClient.interceptors.response.use(
  (response) => response,
  async (error: AxiosError<ApiErrorBody>) => {
    const config = error.config as RetriableConfig | undefined;
    const code = error.response?.data?.error?.code;
    if (error.response?.status === 403 && code === 'CONSENT_REQUIRED' && window.location.pathname !== '/consent') {
      window.location.assign('/consent');
    }

    const canRefresh = config && !config._retry && !config.url?.includes('/auth/refresh') && tokenStorage.getRefresh();
    const needsRefresh = error.response?.status === 401 && (code === 'TOKEN_EXPIRED' || code === 'UNAUTHORIZED');

    if (canRefresh && needsRefresh) {
      config._retry = true;
      try {
        const token = await refreshAccessToken();
        config.headers.Authorization = `Bearer ${token}`;
        return apiClient(config);
      } catch (refreshError) {
        tokenStorage.clear();
        window.dispatchEvent(new CustomEvent('auth:expired'));
        return Promise.reject(refreshError);
      }
    }
    return Promise.reject(error);
  },
);
