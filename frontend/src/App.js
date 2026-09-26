import React from 'react';
import { ConfigProvider } from 'antd';
import zhCN from 'antd/locale/zh_CN';
import { Navigate, Route, Routes } from 'react-router-dom';
import ProtectedRoute from './components/ProtectedRoute';
import { AlertProvider } from './context/AlertContext';
import { AuthProvider } from './context/AuthContext';
import AppLayout from './layouts/AppLayout';
import DashboardPage from './pages/DashboardPage';
import DefectListPage from './pages/DefectListPage';
import DefectCreatePage from './pages/DefectCreatePage';
import DefectDetailPage from './pages/DefectDetailPage';
import LoginPage from './pages/LoginPage';
import OperationLogsPage from './pages/OperationLogsPage';
import UserManagementPage from './pages/UserManagementPage';
import ProductionLineManagePage from './pages/ProductionLineManagePage';
import EquipmentManagePage from './pages/EquipmentManagePage';
import DefectTypeManagePage from './pages/DefectTypeManagePage';

const appTheme = {
  token: {
    colorPrimary: '#2563eb',
    colorSuccess: '#16a34a',
    colorWarning: '#d97706',
    colorError: '#dc2626',
    colorInfo: '#0891b2',
    colorText: '#1f2937',
    colorTextSecondary: '#667085',
    colorBorder: '#d9e2ec',
    colorBgLayout: '#f4f7fb',
    colorBgContainer: '#ffffff',
    borderRadius: 8,
    fontFamily: '"Microsoft YaHei", "PingFang SC", "Noto Sans SC", Arial, sans-serif',
  },
  components: {
    Layout: {
      bodyBg: '#f4f7fb',
      headerBg: '#ffffff',
      siderBg: '#263445',
    },
    Menu: {
      darkItemBg: 'transparent',
      darkSubMenuItemBg: 'transparent',
      darkItemSelectedBg: '#2563eb',
      darkItemColor: 'rgba(255,255,255,0.78)',
      darkItemHoverBg: 'rgba(255,255,255,0.10)',
    },
    Card: {
      headerBg: '#ffffff',
      colorBorderSecondary: '#e8edf3',
    },
    Table: {
      headerBg: '#f7f9fc',
      headerColor: '#344054',
      rowHoverBg: '#f6f9ff',
    },
    Button: {
      controlHeight: 34,
    },
  },
};

function App() {
  return (
    <ConfigProvider locale={zhCN} theme={appTheme}>
      <AuthProvider>
        <AlertProvider>
          <Routes>
            <Route path="/login" element={<LoginPage />} />
            <Route element={<ProtectedRoute />}>
              <Route path="/" element={<AppLayout />}>
                <Route index element={<Navigate to="/dashboard" replace />} />
                <Route path="dashboard" element={<DashboardPage />} />
                <Route path="defects" element={<DefectListPage />} />
                <Route element={<ProtectedRoute roles={['admin', 'inspector']} />}>
                  <Route path="defects/new" element={<DefectCreatePage />} />
                </Route>
                <Route path="defects/:id" element={<DefectDetailPage />} />
                <Route path="production-lines" element={<ProductionLineManagePage />} />
                <Route path="equipment" element={<EquipmentManagePage />} />
                <Route path="defect-types" element={<DefectTypeManagePage />} />
                <Route element={<ProtectedRoute roles={['admin']} />}>
                  <Route path="users" element={<UserManagementPage />} />
                  <Route path="operation-logs" element={<OperationLogsPage />} />
                </Route>
              </Route>
            </Route>
            <Route path="*" element={<Navigate to="/" replace />} />
          </Routes>
        </AlertProvider>
      </AuthProvider>
    </ConfigProvider>
  );
}

export default App;
