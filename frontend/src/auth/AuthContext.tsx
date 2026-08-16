/* eslint-disable react-refresh/only-export-components */
import axios from 'axios';
import { createContext, useCallback, useContext, useEffect, useMemo, useState, type ReactNode } from 'react';
import { authApi, profileApi } from '../api/endpoints';
import type { Profile, VerifyOtpResponse } from '../types/api';
import { tokenStorage } from './tokenStorage';

interface AuthContextValue {
  profile: Profile | null;
  loading: boolean;
  isAuthenticated: boolean;
  establishSession: (response: VerifyOtpResponse) => Promise<Profile>;
  refreshProfile: () => Promise<Profile>;
  logout: () => Promise<void>;
}

const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [profile, setProfile] = useState<Profile | null>(null);
  const [loading, setLoading] = useState(true);

  const refreshProfile = useCallback(async () => {
    const nextProfile = await profileApi.get();
    setProfile(nextProfile);
    return nextProfile;
  }, []);

  const establishSession = useCallback(async (response: VerifyOtpResponse) => {
    tokenStorage.set(response.access_token, response.refresh_token);
    try {
      return await refreshProfile();
    } catch (error) {
      // Не оставляем частично созданную сессию, если профиль не удалось загрузить.
      tokenStorage.clear();
      setProfile(null);
      throw error;
    }
  }, [refreshProfile]);

  const logout = useCallback(async () => {
    const refreshToken = tokenStorage.getRefresh();
    try {
      if (refreshToken) await authApi.logout(refreshToken);
    } catch {
      // Локальный выход должен сработать, даже если API временно недоступен.
    } finally {
      tokenStorage.clear();
      setProfile(null);
    }
  }, []);

  useEffect(() => {
    let active = true;
    const init = async () => {
      if (!tokenStorage.getAccess() && !tokenStorage.getRefresh()) {
        if (active) setLoading(false);
        return;
      }
      try {
        const current = await profileApi.get();
        if (active) setProfile(current);
      } catch (error) {
        // Временная ошибка сети или backend не должна разлогинивать пользователя.
        // Токены удаляем только при окончательном отказе в авторизации.
        if (axios.isAxiosError(error) && error.response?.status === 401) {
          tokenStorage.clear();
        }
        if (active) setProfile(null);
      } finally {
        if (active) setLoading(false);
      }
    };
    void init();
    const onExpired = () => {
      tokenStorage.clear();
      setProfile(null);
    };
    const onTokenRefreshed = () => { void refreshProfile().catch(() => undefined); };
    window.addEventListener('auth:expired', onExpired);
    window.addEventListener('auth:token-refreshed', onTokenRefreshed);
    return () => {
      active = false;
      window.removeEventListener('auth:expired', onExpired);
      window.removeEventListener('auth:token-refreshed', onTokenRefreshed);
    };
  }, [refreshProfile]);

  const value = useMemo<AuthContextValue>(() => ({
    profile,
    loading,
    isAuthenticated: Boolean(profile || tokenStorage.getAccess()),
    establishSession,
    refreshProfile,
    logout,
  }), [profile, loading, establishSession, refreshProfile, logout]);

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth(): AuthContextValue {
  const value = useContext(AuthContext);
  if (!value) throw new Error('useAuth должен использоваться внутри AuthProvider');
  return value;
}
