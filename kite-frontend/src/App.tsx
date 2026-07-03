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
import { KiteDocsPage } from './pages/KiteDocsPage';
import { DEBUG_DIRECT_ROUTES_ENABLED } from './config/debug';
import { GlobalFooter } from './components/GlobalFooter';

const RequireAuth = ({ children }: { children: ReactElement }) => {
  const authenticated = useAuthStore((state) => state.authenticated);
  const location = useLocation();
  if (!authenticated && !DEBUG_DIRECT_ROUTES_ENABLED) return <Navigate to="/" state={{ requireLogin: true, path: location.pathname }} replace />;
  return children;
};

const RequireAdmin = ({ children, minimumAccessLevel = 2 }: { children: ReactElement; minimumAccessLevel?: number }) => {
  const authenticated = useAuthStore((state) => state.authenticated);
  const accessLevel = useAuthStore((state) => state.accessLevel);
  const location = useLocation();
  
  if (!authenticated && !DEBUG_DIRECT_ROUTES_ENABLED) return <Navigate to="/" state={{ requireLogin: true, path: location.pathname }} replace />;
  if (!DEBUG_DIRECT_ROUTES_ENABLED && (accessLevel === null || accessLevel < minimumAccessLevel)) return <Navigate to="/dashboard" replace />;
  return children;
};

function App() {
  return (
    <ConfigProvider theme={kiteTheme}>
      <AntdApp>
        <BrowserRouter>
          <Routes>
            <Route path="/" element={<LandingPage />} />
            <Route path="/kite-docs" element={<KiteDocsPage />} />
            <Route path="/signup" element={<SignupPage />} />
            <Route path="/login" element={<Navigate to="/" replace />} />

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

            <Route path="/admin/settings" element={
              <RequireAdmin minimumAccessLevel={3}>
                <AdminSettings />
              </RequireAdmin>
            } />
            <Route path="/admin/*" element={
              <RequireAdmin>
                <AdminDashboard />
              </RequireAdmin>
            } />
          </Routes>
          <GlobalFooter />
        </BrowserRouter>
      </AntdApp>
    </ConfigProvider>
  );
}

export default App;
