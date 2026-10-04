import { createBrowserRouter, Navigate, Outlet } from 'react-router-dom';
import { useAuthStore } from '@/stores/authStore';
import LoginPage from '@/pages/LoginPage';
import RegisterPersonalPage from '@/pages/RegisterPersonalPage';
import RegisterOrganizationPage from '@/pages/RegisterOrganizationPage';
import InvitationAcceptPage from '@/pages/InvitationAcceptPage';
import DashboardPage from '@/pages/DashboardPage';
import MembersPage from '@/pages/MembersPage';
import AppLayout from '@/layouts/AppLayout';

// 登录守卫
function RequireAuth() {
  const uid = useAuthStore((s) => s.userId);
  if (!uid) return <Navigate to="/login" replace />;
  return <Outlet />;
}

export const router = createBrowserRouter([
  { path: '/', element: <Navigate to="/dashboard" replace /> },
  { path: '/login', element: <LoginPage /> },
  { path: '/register', element: <RegisterPersonalPage /> },
  { path: '/invitations/:token', element: <InvitationAcceptPage /> },

  {
    element: <RequireAuth />,
    children: [
      {
        element: <AppLayout />,
        children: [
          { path: '/dashboard', element: <DashboardPage /> },
          { path: '/register-org', element: <RegisterOrganizationPage /> },
          { path: '/accounts/:aid/members', element: <MembersPage /> },
        ],
      },
    ],
  },
]);
