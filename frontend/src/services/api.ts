import axios, { AxiosError, InternalAxiosRequestConfig } from 'axios';
import { store } from '../store';
import { refreshAccessToken, logout } from '../store/slices/authSlice';

// Create axios instance
export const api = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL || '/api/v1',
  timeout: 30000,
  headers: {
    'Content-Type': 'application/json',
  },
});

// Request interceptor - add auth token
api.interceptors.request.use(
  (config: InternalAxiosRequestConfig) => {
    const state = store.getState();
    const token = state.auth.token;

    if (token) {
      config.headers.Authorization = `Bearer ${token}`;
    }

    return config;
  },
  (error: AxiosError) => {
    return Promise.reject(error);
  }
);

// Response interceptor - handle token refresh
let isRefreshing = false;
let failedQueue: Array<{
  resolve: (value?: any) => void;
  reject: (reason?: any) => void;
}> = [];

const processQueue = (error: Error | null, token: string | null = null) => {
  failedQueue.forEach((prom) => {
    if (error) {
      prom.reject(error);
    } else {
      prom.resolve(token);
    }
  });

  failedQueue = [];
};

api.interceptors.response.use(
  (response) => {
    return response;
  },
  async (error: AxiosError) => {
    const originalRequest = error.config as InternalAxiosRequestConfig & {
      _retry?: boolean;
    };

    // If error is 401 and we haven't retried yet
    if (error.response?.status === 401 && !originalRequest._retry) {
      if (isRefreshing) {
        // If token is already being refreshed, queue this request
        return new Promise((resolve, reject) => {
          failedQueue.push({ resolve, reject });
        })
          .then((token) => {
            if (originalRequest.headers) {
              originalRequest.headers.Authorization = `Bearer ${token}`;
            }
            return api(originalRequest);
          })
          .catch((err) => {
            return Promise.reject(err);
          });
      }

      originalRequest._retry = true;
      isRefreshing = true;

      try {
        // Attempt to refresh token
        const result = await store.dispatch(refreshAccessToken());

        if (refreshAccessToken.fulfilled.match(result)) {
          const newToken = result.payload.token;
          processQueue(null, newToken);

          // Retry original request with new token
          if (originalRequest.headers) {
            originalRequest.headers.Authorization = `Bearer ${newToken}`;
          }
          return api(originalRequest);
        } else {
          // Refresh failed, logout user
          processQueue(new Error('Token refresh failed'), null);
          store.dispatch(logout());
          window.location.href = '/login';
          return Promise.reject(error);
        }
      } catch (refreshError) {
        processQueue(refreshError as Error, null);
        store.dispatch(logout());
        window.location.href = '/login';
        return Promise.reject(refreshError);
      } finally {
        isRefreshing = false;
      }
    }

    return Promise.reject(error);
  }
);

// Helper functions for common requests
export const apiHelpers = {
  // Auth
  login: (email: string, password: string) =>
    api.post('/auth/login', { email, password }),

  register: (email: string, password: string, full_name: string) =>
    api.post('/auth/register', { email, password, full_name }),

  logout: () => api.post('/auth/logout'),

  // Conversations
  getConversations: () => api.get('/conversations'),

  createConversation: (title?: string, initial_message?: string) =>
    api.post('/conversations', { title, initial_message }),

  getConversation: (id: string) => api.get(`/conversations/${id}`),

  updateConversation: (id: string, data: { title?: string; status?: string }) =>
    api.put(`/conversations/${id}`, data),

  deleteConversation: (id: string) => api.delete(`/conversations/${id}`),

  // Messages
  getMessages: (conversationId: string) =>
    api.get(`/conversations/${conversationId}/messages`),

  // Upload
  uploadFile: (file: File, conversationId?: string) => {
    const formData = new FormData();
    formData.append('file', file);
    if (conversationId) {
      formData.append('conversation_id', conversationId);
    }
    return api.post('/upload', formData, {
      headers: { 'Content-Type': 'multipart/form-data' },
    });
  },

  // Videos
  generateVideo: (data: {
    conversation_id: string;
    product_name: string;
    product_desc: string;
    product_image_url: string;
    character_type?: string;
    duration?: number;
  }) => api.post('/videos/generate', data),

  getVideo: (id: string) => api.get(`/videos/${id}`),

  getVideoStatus: (id: string) => api.get(`/videos/${id}/status`),

  getUserVideos: (conversationId?: string, status?: string) => {
    const params = new URLSearchParams();
    if (conversationId) params.append('conversation_id', conversationId);
    if (status) params.append('status', status);
    return api.get(`/videos?${params.toString()}`);
  },

  deleteVideo: (id: string) => api.delete(`/videos/${id}`),

  retryVideo: (id: string) => api.post(`/videos/${id}/retry`),
};

export default api;
