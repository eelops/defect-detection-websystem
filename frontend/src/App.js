import React from 'react';
import { Navigate, Route, Routes } from 'react-router-dom';
import { AlertProvider } from './context/AlertContext';
import AppLayout from './layouts/AppLayout';
import DashboardPage from './pages/DashboardPage';
import DefectListPage from './pages/DefectListPage';
import DefectCreatePage from './pages/DefectCreatePage';
import DefectDetailPage from './pages/DefectDetailPage';

function App() {
  return (
    <AlertProvider>
      <Routes>
        <Route path="/" element={<AppLayout />}>
          <Route index element={<Navigate to="/dashboard" replace />} />
          <Route path="dashboard" element={<DashboardPage />} />
          <Route path="defects" element={<DefectListPage />} />
          <Route path="defects/new" element={<DefectCreatePage />} />
          <Route path="defects/:id" element={<DefectDetailPage />} />
        </Route>
      </Routes>
    </AlertProvider>
  );
}

export default App;
