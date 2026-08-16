import { lazy, Suspense } from 'react';
import { Navigate, Route, Routes } from 'react-router-dom';
import { AppLayout } from '../components/layout/AppLayout';
import { AdminRoute, ConsentGuard, ProfileCompleteGuard, ProtectedRoute } from '../components/guards';
import { HomePage } from '../pages/HomePage';
import { NotFoundPage } from '../pages/NotFoundPage';
import { ConsentPage } from '../pages/ConsentPage';
import { ProfilePage } from '../pages/ProfilePage';
import { LoginPage } from '../pages/auth/LoginPage';
import { VerifyOtpPage } from '../pages/auth/VerifyOtpPage';
import { ApplicationsPage } from '../pages/applications/ApplicationsPage';
import { ApplicationDetailsPage } from '../pages/applications/ApplicationDetailsPage';
import { NewApplicationPage } from '../pages/applications/NewApplicationPage';
import { AdminLayoutPage } from '../pages/admin/AdminLayoutPage';
import { AdminApplicationsPage } from '../pages/admin/AdminApplicationsPage';
import { AdminApplicationDetailsPage } from '../pages/admin/AdminApplicationDetailsPage';
import { AdminMaterialsPage } from '../pages/admin/AdminMaterialsPage';
import { AdminUsersPage } from '../pages/admin/AdminUsersPage';
import { LoadingState } from '../components/LoadingState';

const AdminStatsPage = lazy(() => import('../pages/admin/AdminStatsPage').then((module) => ({ default: module.AdminStatsPage })));

export function App() {
  return (
    <Routes>
      <Route element={<AppLayout />}>
        <Route index element={<HomePage />} />
        <Route path="auth/login" element={<LoginPage />} />
        <Route path="auth/verify" element={<VerifyOtpPage />} />
        <Route path="consent" element={<ProtectedRoute><ConsentPage /></ProtectedRoute>} />
        <Route path="profile" element={<ProtectedRoute><ProfilePage /></ProtectedRoute>} />

        <Route path="app/applications" element={<ProtectedRoute><ConsentGuard><ApplicationsPage /></ConsentGuard></ProtectedRoute>} />
        <Route path="app/applications/new" element={<ProtectedRoute><ConsentGuard><ProfileCompleteGuard><NewApplicationPage /></ProfileCompleteGuard></ConsentGuard></ProtectedRoute>} />
        <Route path="app/applications/:id" element={<ProtectedRoute><ConsentGuard><ApplicationDetailsPage /></ConsentGuard></ProtectedRoute>} />

        <Route path="admin" element={<ProtectedRoute><ConsentGuard><AdminRoute><AdminLayoutPage /></AdminRoute></ConsentGuard></ProtectedRoute>}>
          <Route index element={<Navigate to="applications" replace />} />
          <Route path="applications" element={<AdminApplicationsPage />} />
          <Route path="applications/:id" element={<AdminApplicationDetailsPage />} />
          <Route path="materials" element={<AdminMaterialsPage />} />
          <Route path="stats" element={<Suspense fallback={<LoadingState label="Загружаем статистику…" />}><AdminStatsPage /></Suspense>} />
          <Route path="users" element={<AdminUsersPage />} />
        </Route>
        <Route path="*" element={<NotFoundPage />} />
      </Route>
    </Routes>
  );
}
