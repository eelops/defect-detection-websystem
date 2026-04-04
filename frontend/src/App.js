import React from 'react';
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

function App() {
  return (
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
  );
}

export default App;
