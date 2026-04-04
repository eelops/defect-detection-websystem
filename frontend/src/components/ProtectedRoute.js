import React from 'react';
import { Navigate, Outlet, useLocation } from 'react-router-dom';
import { Skeleton } from 'antd';
import { useAuth } from '../context/AuthContext';

export default function ProtectedRoute({ roles }) {
  const location = useLocation();
  const { initializing, isAuthenticated, hasRole } = useAuth();

  if (initializing) {
    return <Skeleton active paragraph={{ rows: 10 }} style={{ padding: 24 }} />;
  }

  if (!isAuthenticated) {
    return <Navigate to="/login" replace state={{ from: location }} />;
  }

  if (roles?.length && !hasRole(...roles)) {
    return <Navigate to="/dashboard" replace />;
  }

  return <Outlet />;
}
