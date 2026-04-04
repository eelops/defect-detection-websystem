import React, {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
} from 'react';
import {
  authEventTarget,
  getCurrentUser,
  getStoredToken,
  login as loginRequest,
  logout as logoutRequest,
  setStoredToken,
} from '../api';

const AuthContext = createContext(null);

export function AuthProvider({ children }) {
  const [token, setToken] = useState(() => getStoredToken());
  const [currentUser, setCurrentUser] = useState(null);
  const [initializing, setInitializing] = useState(true);

  const clearAuth = useCallback(() => {
    setStoredToken('');
    setToken('');
    setCurrentUser(null);
  }, []);

  useEffect(() => {
    const handleTokenChange = (event) => {
      const nextToken = event.detail || '';
      setToken(nextToken);
      if (!nextToken) {
        setCurrentUser(null);
      }
    };
    authEventTarget.addEventListener('token-change', handleTokenChange);
    return () => authEventTarget.removeEventListener('token-change', handleTokenChange);
  }, []);

  const loadCurrentUser = useCallback(async () => {
    const storedToken = getStoredToken();
    if (!storedToken) {
      setCurrentUser(null);
      setInitializing(false);
      return null;
    }

    try {
      const user = await getCurrentUser();
      setCurrentUser(user);
      return user;
    } catch (error) {
      clearAuth();
      return null;
    } finally {
      setInitializing(false);
    }
  }, [clearAuth]);

  useEffect(() => {
    loadCurrentUser();
  }, [loadCurrentUser]);

  const login = useCallback(async (payload) => {
    const result = await loginRequest(payload);
    setStoredToken(result.token || '');
    setToken(result.token || '');
    setCurrentUser(result.user || null);
    return result.user;
  }, []);

  const logout = useCallback(async () => {
    try {
      if (getStoredToken()) {
        await logoutRequest();
      }
    } finally {
      clearAuth();
    }
  }, [clearAuth]);

  const hasRole = useCallback((...roles) => {
    if (!currentUser) return false;
    return roles.includes(currentUser.role);
  }, [currentUser]);

  const contextValue = useMemo(() => ({
    token,
    currentUser,
    initializing,
    isAuthenticated: !!token,
    login,
    logout,
    reloadCurrentUser: loadCurrentUser,
    hasRole,
  }), [token, currentUser, initializing, login, logout, loadCurrentUser, hasRole]);

  return <AuthContext.Provider value={contextValue}>{children}</AuthContext.Provider>;
}

export function useAuth() {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error('useAuth 必须在 AuthProvider 内使用');
  }
  return context;
}
