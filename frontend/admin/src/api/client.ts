import axios, {
  type AxiosError,
  type InternalAxiosRequestConfig,
} from 'axios';

// ---------------------------------------------------------------------------
// Base URL — uses /api/v1 prefix to match backend routes
// ---------------------------------------------------------------------------
const API_BASE_URL =
  import.meta.env.VITE_API_URL || '/api/v1';

const api = axios.create({
  baseURL: API_BASE_URL,
  headers: {
    'Content-Type': 'application/json',
  },
});

// Separate instance for non-prefixed endpoints (health)
export const rootApi = axios.create({
  baseURL: import.meta.env.VITE_API_URL ? import.meta.env.VITE_API_URL.replace(/\/api\/v1\/?$/, '') : '',
  headers: {
    'Content-Type': 'application/json',
  },
});

// Track whether a token refresh is already in flight to prevent concurrent refreshes
let isRefreshing = false;
let failedQueue: {
  resolve: (token: string) => void;
  reject: (error: unknown) => void;
  config: InternalAxiosRequestConfig;
}[] = [];

function processQueue(error: unknown, token: string | null = null) {
  failedQueue.forEach((promise) => {
    if (error) {
      promise.reject(error);
    } else if (token) {
      promise.resolve(token);
    }
  });
  failedQueue = [];
}

// ---------------------------------------------------------------------------
// Response interceptor — unwrap backend envelope {success, data, meta}
// ---------------------------------------------------------------------------
api.interceptors.response.use(
  (response) => {
    const d = response.data;
    if (d && typeof d === 'object' && 'success' in d && 'data' in d) {
      if (
        Array.isArray(d.data) &&
        d.meta &&
        ('page' in d.meta || 'per_page' in d.meta || 'total' in d.meta || 'total_pages' in d.meta)
      ) {
        // Paginated response: transform meta → pagination.
        // The API leaves out zero values, so an empty list has no total / total_pages: default them.
        const { request_id: _, ...rest } = d.meta;
        const pagination = { page: 1, per_page: d.data.length, total: 0, total_pages: 0, ...rest };
        response.data = { data: d.data, pagination };
      } else {
        // Single item / object response
        response.data = d.data;
      }
    }
    return response;
  },
  // Error interceptor is chained below
);

// ---------------------------------------------------------------------------
// Request interceptor — attach Bearer token
// ---------------------------------------------------------------------------
api.interceptors.request.use(
  (config: InternalAxiosRequestConfig) => {
    if (typeof window === 'undefined') {
      return config;
    }
    const token = localStorage.getItem('ecommerce_access_token');
    if (token && config.headers) {
      config.headers.Authorization = `Bearer ${token}`;
    }
    return config;
  },
  (error: AxiosError) => {
    return Promise.reject(error);
  },
);

// ---------------------------------------------------------------------------
// Response interceptor — handle 401 with silent token refresh
// ---------------------------------------------------------------------------
api.interceptors.response.use(
  (response) => response,
  async (error: AxiosError) => {
    const originalRequest = error.config as InternalAxiosRequestConfig & {
      _retry?: boolean;
    };

    // Use the server's own explanation ("coupon already used", "sku exists"...) as the error text,
    // instead of axios' generic "Request failed with status code 422".
    const serverMessage = (error.response?.data as { error?: { message?: string } } | undefined)?.error?.message;
    if (serverMessage) {
      error.message = serverMessage;
    }

    // Only attempt refresh on 401 responses that haven't already been retried
    if (error.response?.status !== 401 || originalRequest._retry) {
      return Promise.reject(error);
    }

    // Don't try to refresh if the failing request IS the refresh endpoint itself
    if (originalRequest.url?.includes('/auth/refresh')) {
      clearTokensAndRedirect();
      return Promise.reject(error);
    }

    if (isRefreshing) {
      // Queue up while another refresh is in progress
      return new Promise((resolve, reject) => {
        failedQueue.push({
          resolve: (token: string) => {
            if (originalRequest.headers) {
              originalRequest.headers.Authorization = `Bearer ${token}`;
            }
            resolve(api(originalRequest));
          },
          reject,
          config: originalRequest,
        });
      });
    }

    originalRequest._retry = true;
    isRefreshing = true;

    const refreshToken = localStorage.getItem('ecommerce_refresh_token');

    if (!refreshToken) {
      clearTokensAndRedirect();
      processQueue(error, null);
      return Promise.reject(error);
    }

    try {
      // Use raw axios to avoid envelope unwrapping on the refresh response
      const { data } = await axios.post(
        `${API_BASE_URL}/auth/refresh`,
        { refresh_token: refreshToken },
      );

      // The refresh response is also wrapped in the envelope
      const tokenData = data?.data ?? data;
      const newAccessToken: string = tokenData?.access_token ?? tokenData?.tokens?.access_token;
      const newRefreshToken: string | undefined =
        tokenData?.refresh_token ?? tokenData?.tokens?.refresh_token;

      localStorage.setItem('ecommerce_access_token', newAccessToken);
      if (newRefreshToken) {
        localStorage.setItem('ecommerce_refresh_token', newRefreshToken);
      }

      if (originalRequest.headers) {
        originalRequest.headers.Authorization = `Bearer ${newAccessToken}`;
      }

      processQueue(null, newAccessToken);
      return api(originalRequest);
    } catch (refreshError) {
      processQueue(refreshError, null);
      clearTokensAndRedirect();
      return Promise.reject(refreshError);
    } finally {
      isRefreshing = false;
    }
  },
);

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function clearTokensAndRedirect() {
  if (typeof window === 'undefined') return;
  localStorage.removeItem('ecommerce_access_token');
  localStorage.removeItem('ecommerce_refresh_token');
  localStorage.removeItem('ecommerce_user');
  // Redirect to login page
  const currentPath = window.location.pathname;
  if (currentPath !== '/login') {
    window.location.href = '/login';
  }
}

export { api, API_BASE_URL };
export default api;
