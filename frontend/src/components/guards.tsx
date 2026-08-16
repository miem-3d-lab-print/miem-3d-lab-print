import type { ReactNode } from 'react';
import { Navigate, useLocation } from 'react-router-dom';
import { useAuth } from '../auth/AuthContext';
import { LoadingState } from './LoadingState';

export function ProtectedRoute({ children }: { children: ReactNode }) {
  const { isAuthenticated, loading } = useAuth();
  const location = useLocation();
  if (loading) return <LoadingState label="Проверяем сессию…" />;
  if (!isAuthenticated) return <Navigate to="/auth/login" replace state={{ from: `${location.pathname}${location.search}${location.hash}` }} />;
  return children;
}

export function ConsentGuard({ children }: { children: ReactNode }) {
  const { profile, loading } = useAuth();
  if (loading) return <LoadingState />;
  if (profile && !profile.consent_given) return <Navigate to="/consent" replace />;
  return children;
}

export function ProfileCompleteGuard({ children }: { children: ReactNode }) {
  const { profile, loading } = useAuth();
  const location = useLocation();
  if (loading) return <LoadingState />;
  const complete = Boolean(profile?.full_name?.trim() && (profile.telegram?.trim() || profile.max?.trim()));
  if (!complete) return <Navigate to="/profile" replace state={{ profileRequired: true, afterAuth: `${location.pathname}${location.search}${location.hash}` }} />;
  return children;
}

export function AdminRoute({ children }: { children: ReactNode }) {
  const { profile, loading } = useAuth();
  if (loading) return <LoadingState />;
  if (profile?.role !== 'admin') return <Navigate to="/app/applications" replace />;
  return children;
}
