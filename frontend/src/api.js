import axios from 'axios';

const API = axios.create({
  baseURL: 'http://localhost:8000',
});

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
