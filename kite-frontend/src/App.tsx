import type { ReactElement } from 'react';
import { BrowserRouter, Routes, Route, Navigate, useLocation } from 'react-router-dom';
import { App as AntdApp, ConfigProvider } from 'antd';
import { kiteTheme } from './theme';
import { useAuthStore } from './store/useAuthStore';
import { LandingPage } from './pages/LandingPage';
import { SignupPage } from './pages/SignupPage';
import { UserDashboard } from './pages/UserDashboard';
import { VmDetail } from './pages/VmDetail';
import { VmConsolePage } from './pages/VmConsolePage';
import { AdminSettings } from './pages/AdminSettings';
import { AdminDashboard } from './pages/AdminDashboard';
import { DEBUG_DIRECT_ROUTES_ENABLED } from './config/debug';

// 인증 가드 (로그인 안된 유저 튕겨내기)
const RequireAuth = ({ children }: { children: ReactElement }) => {
  const token = useAuthStore((state) => state.token);
  const location = useLocation();
  if (!token && !DEBUG_DIRECT_ROUTES_ENABLED) return <Navigate to="/" state={{ requireLogin: true, path: location.pathname }} replace />;
  return children;
};

// 관리자 가드 (레벨 2 이상만 접근)
const RequireAdmin = ({ children }: { children: ReactElement }) => {
  const token = useAuthStore((state) => state.token);
  const accessLevel = useAuthStore((state) => state.accessLevel);
  const location = useLocation();
  
  if (!token && !DEBUG_DIRECT_ROUTES_ENABLED) return <Navigate to="/" state={{ requireLogin: true, path: location.pathname }} replace />;
  if (!DEBUG_DIRECT_ROUTES_ENABLED && (accessLevel === null || accessLevel < 2)) return <Navigate to="/dashboard" replace />;
  return children;
};

function App() {
  return (
    <ConfigProvider theme={kiteTheme}>
      <AntdApp>
        <BrowserRouter>
          <Routes>
            {/* 랜딩 겸 로그인 페이지 */}
            <Route path="/" element={<LandingPage />} />
            <Route path="/signup" element={<SignupPage />} />
            <Route path="/login" element={<Navigate to="/" replace />} />

            {/* User Routes */}
            <Route path="/dashboard" element={
              <RequireAuth>
                <UserDashboard />
              </RequireAuth>
            } />
            <Route path="/dashboard/kite-machine/:vmName" element={
              <RequireAuth>
                <VmDetail />
              </RequireAuth>
            } />
            <Route path="/dashboard/kite-machine/:vmName/console" element={
              <RequireAuth>
                <VmConsolePage />
              </RequireAuth>
            } />

            {/* Admin Routes */}
            <Route path="/admin/settings" element={
              <RequireAdmin>
                <AdminSettings />
              </RequireAdmin>
            } />
            <Route path="/admin/*" element={
              <RequireAdmin>
                <AdminDashboard />
              </RequireAdmin>
            } />
          </Routes>
        </BrowserRouter>
      </AntdApp>
    </ConfigProvider>
  );
}

export default App;
