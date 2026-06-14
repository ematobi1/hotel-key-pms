import axios from 'axios';

const BASE_URL = import.meta.env.VITE_API_BASE_URL ?? '/api';

export const apiClient = axios.create({
  baseURL: BASE_URL,
  timeout: 10_000,
  headers: { 'Content-Type': 'application/json' },
});

apiClient.interceptors.request.use((config) => {
  const token = sessionStorage.getItem('access_token');
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  const tenantId = sessionStorage.getItem('tenant_id');
  if (tenantId) {
    config.headers['X-Tenant-ID'] = tenantId;
  }
  return config;
});

apiClient.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401) {
      sessionStorage.removeItem('access_token');
      window.location.href = '/login';
    }
    return Promise.reject(error);
  }
);

export function setAuthTokens(accessToken: string, tenantId: string) {
  sessionStorage.setItem('access_token', accessToken);
  sessionStorage.setItem('tenant_id', tenantId);
}

export function clearAuthTokens() {
  sessionStorage.removeItem('access_token');
  sessionStorage.removeItem('tenant_id');
}

// Resource helpers
export const hotelsApi = {
  list: (params?: { limit?: number; offset?: number }) =>
    apiClient.get('/v1/hotels', { params }),
  get: (id: string) =>
    apiClient.get(`/v1/hotels/${id}`),
  create: (data: unknown) =>
    apiClient.post('/v1/hotels', data),
  update: (id: string, data: unknown) =>
    apiClient.put(`/v1/hotels/${id}`, data),
  delete: (id: string) =>
    apiClient.delete(`/v1/hotels/${id}`),
};

export const reservationsApi = {
  list: (params: { hotel_id: string; limit?: number; offset?: number }) =>
    apiClient.get('/v1/reservations', { params }),
  get: (id: string) =>
    apiClient.get(`/v1/reservations/${id}`),
  create: (data: unknown) =>
    apiClient.post('/v1/reservations', data),
  checkIn: (id: string, roomId: string) =>
    apiClient.post(`/v1/reservations/${id}/check-in`, { room_id: roomId }),
  checkOut: (id: string) =>
    apiClient.post(`/v1/reservations/${id}/check-out`),
  cancel: (id: string) =>
    apiClient.delete(`/v1/reservations/${id}`),
};

export const roomsApi = {
  list: (params: { hotel_id?: string }) =>
    apiClient.get('/v1/rooms', { params }),
  updateStatus: (id: string, status: string) =>
    apiClient.patch(`/v1/rooms/${id}/status`, { status }),
};

export const folioApi = {
  get: (reservationId: string) =>
    apiClient.get(`/v1/billing/folio/${reservationId}`),
  charge: (reservationId: string, data: { description: string; amount: number; currency?: string }) =>
    apiClient.post(`/v1/billing/folio/${reservationId}/charge`, data),
};
