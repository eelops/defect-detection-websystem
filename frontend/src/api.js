import axios from 'axios';

const API = axios.create({
  baseURL: 'http://localhost:8000',
});

const AUTH_TOKEN_KEY = 'defect-system-token';

export const authEventTarget = new EventTarget();

export const getStoredToken = () => window.localStorage.getItem(AUTH_TOKEN_KEY) || '';

export const setStoredToken = (token) => {
  if (token) {
    window.localStorage.setItem(AUTH_TOKEN_KEY, token);
  } else {
    window.localStorage.removeItem(AUTH_TOKEN_KEY);
  }
  authEventTarget.dispatchEvent(new CustomEvent('token-change', { detail: token || '' }));
};

API.interceptors.request.use((config) => {
  const token = getStoredToken();
  if (token) {
    config.headers = config.headers || {};
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

API.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error?.response?.status === 401) {
      setStoredToken('');
    }
    return Promise.reject(error);
  }
);

const normalizeListResponse = (payload) => {
  if (Array.isArray(payload)) {
    return { items: payload, total: payload.length };
  }

  if (Array.isArray(payload?.items)) {
    return {
      items: payload.items,
      total: payload.total ?? payload.items.length,
    };
  }

  if (Array.isArray(payload?.results)) {
    return {
      items: payload.results,
      total: payload.count ?? payload.total ?? payload.results.length,
    };
  }

  if (Array.isArray(payload?.data)) {
    return {
      items: payload.data,
      total: payload.total ?? payload.data.length,
    };
  }

  return { items: [], total: 0 };
};

export const getImageUrl = (path) => {
  if (!path) return '';
  if (/^https?:\/\//.test(path)) return path;
  return `${API.defaults.baseURL}${path}`;
};

export const login = async (payload) => {
  const response = await API.post('/api/auth/login', payload);
  return response.data;
};

export const logout = async () => {
  const response = await API.post('/api/auth/logout');
  return response.data;
};

export const getCurrentUser = async () => {
  const response = await API.get('/api/auth/me');
  return response.data;
};

export const changePassword = async (payload) => {
  const response = await API.put('/api/auth/password', payload);
  return response.data;
};

export const fetchUsers = async () => {
  const response = await API.get('/api/users');
  return normalizeListResponse(response.data).items;
};

export const createUser = async (payload) => {
  const response = await API.post('/api/users', payload);
  return response.data;
};

export const updateUser = async (id, payload) => {
  const response = await API.put(`/api/users/${id}`, payload);
  return response.data;
};

export const fetchDefects = async (params) => {
  const response = await API.get('/api/defects', { params });
  return normalizeListResponse(response.data);
};

export const fetchAlerts = async () => {
  const response = await API.get('/api/alerts');
  return normalizeListResponse(response.data).items;
};

export const fetchDefect = async (id) => {
  const response = await API.get(`/api/defects/${id}`);
  return response.data;
};

export const fetchDefectFlowLogs = async (id) => {
  const response = await API.get(`/api/defects/${id}/flow-logs`);
  return normalizeListResponse(response.data).items;
};

export const createDefect = async (formData) => {
  const response = await API.post('/api/defects', formData, {
    headers: { 'Content-Type': 'multipart/form-data' },
  });
  return response.data;
};

export const updateDefect = async (id, data) => {
  const response = await API.put(`/api/defects/${id}`, data);
  return response.data;
};

export const deleteDefect = async (id) => {
  const response = await API.delete(`/api/defects/${id}`);
  return response.data;
};

export const executeDefectAction = async (id, payload) => {
  const response = await API.post(`/api/defects/${id}/actions`, payload);
  return response.data;
};

export const downloadFile = (blob, filename) => {
  const url = window.URL.createObjectURL(blob);
  const link = document.createElement('a');
  link.href = url;
  link.download = filename;
  document.body.appendChild(link);
  link.click();
  link.remove();
  window.URL.revokeObjectURL(url);
};

export const exportDefectsExcel = async (params) => {
  const response = await API.get('/api/defects/export.xlsx', {
    params,
    responseType: 'blob',
  });
  return response.data;
};

export const exportDefectDetailPdf = async (id) => {
  const response = await API.get(`/api/defects/${id}/export.pdf`, {
    responseType: 'blob',
  });
  return response.data;
};

export const fetchOperationLogs = async (params) => {
  const response = await API.get('/api/operation-logs', { params });
  return normalizeListResponse(response.data);
};

export const fetchProductionLines = async (params) => {
  const response = await API.get('/api/production-lines', { params });
  return normalizeListResponse(response.data);
};

export const createProductionLine = async (payload) => {
  const response = await API.post('/api/production-lines', payload);
  return response.data;
};

export const updateProductionLine = async (id, payload) => {
  const response = await API.put(`/api/production-lines/${id}`, payload);
  return response.data;
};

export const fetchEquipment = async (params) => {
  const response = await API.get('/api/equipment', { params });
  return normalizeListResponse(response.data);
};

export const createEquipment = async (payload) => {
  const response = await API.post('/api/equipment', payload);
  return response.data;
};

export const updateEquipment = async (id, payload) => {
  const response = await API.put(`/api/equipment/${id}`, payload);
  return response.data;
};

export const fetchDefectTypes = async (params) => {
  const response = await API.get('/api/defect-types', { params });
  return normalizeListResponse(response.data);
};

export const createDefectType = async (payload) => {
  const response = await API.post('/api/defect-types', payload);
  return response.data;
};

export const updateDefectType = async (id, payload) => {
  const response = await API.put(`/api/defect-types/${id}`, payload);
  return response.data;
};

