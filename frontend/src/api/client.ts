import axios, { AxiosError, InternalAxiosRequestConfig } from "axios";
import { AuthTokens } from "./types";

const BASE_URL = import.meta.env.VITE_API_URL || "http://localhost:8080";

// In-memory access token storage (Security requirement: NOT in localStorage)
let inMemoryAccessToken: string | null = null;

// LocalStorage keys for refresh token and active tenant
const REFRESH_TOKEN_KEY = "saasbase_refresh_token";
const CURRENT_TENANT_KEY = "saasbase_current_tenant_id";

export const getAccessToken = (): string | null => inMemoryAccessToken;

export const setAccessToken = (token: string | null): void => {
  inMemoryAccessToken = token;
};

export const getRefreshToken = (): string | null => {
  return localStorage.getItem(REFRESH_TOKEN_KEY);
};

export const setRefreshToken = (token: string | null): void => {
  if (token) {
    localStorage.setItem(REFRESH_TOKEN_KEY, token);
  } else {
    localStorage.removeItem(REFRESH_TOKEN_KEY);
  }
};

export const getCurrentTenantId = (): string => {
  return localStorage.getItem(CURRENT_TENANT_KEY) || "";
};

export const setCurrentTenantId = (tenantId: string): void => {
  localStorage.setItem(CURRENT_TENANT_KEY, tenantId);
};

export const clearAuthSession = (): void => {
  inMemoryAccessToken = null;
  localStorage.removeItem(REFRESH_TOKEN_KEY);
};

// Create main axios instance
export const apiClient = axios.create({
  baseURL: BASE_URL,
  headers: {
    "Content-Type": "application/json",
  },
});

// Request interceptor: attach Authorization and X-Tenant-ID
apiClient.interceptors.request.use(
  (config: InternalAxiosRequestConfig) => {
    // 1. Attach Bearer token if present
    if (inMemoryAccessToken && !config.headers.Authorization) {
      config.headers.Authorization = `Bearer ${inMemoryAccessToken}`;
    }

    // 2. Attach X-Tenant-ID
    // If config explicitly sets X-Tenant-ID (e.g. "" for host scope), keep it.
    // Otherwise, attach current tenant from localStorage.
    if (config.headers["X-Tenant-ID"] === undefined) {
      const tenantId = getCurrentTenantId();
      if (tenantId) {
        config.headers["X-Tenant-ID"] = tenantId;
      }
    }

    return config;
  },
  (error) => Promise.reject(error)
);

// Concurrency queue for 401 token refresh (Mandatory requirement)
let isRefreshing = false;
let failedQueue: Array<{
  resolve: (value?: unknown) => void;
  reject: (reason?: unknown) => void;
}> = [];

const processQueue = (error: Error | null) => {
  failedQueue.forEach((prom) => {
    if (error) {
      prom.reject(error);
    } else {
      prom.resolve();
    }
  });
  failedQueue = [];
};

// Response interceptor: handle 401 with concurrency-safe token rotation
apiClient.interceptors.response.use(
  (response) => response,
  async (error: AxiosError) => {
    const originalRequest = error.config as InternalAxiosRequestConfig & {
      _retry?: boolean;
    };

    if (!originalRequest) {
      return Promise.reject(error);
    }

    // Do not attempt refresh on public auth endpoints
    const url = originalRequest.url || "";
    if (url.includes("/v1/auth/login") || url.includes("/v1/auth/refresh")) {
      return Promise.reject(error);
    }

    if (error.response?.status === 401 && !originalRequest._retry) {
      const refreshToken = getRefreshToken();
      if (!refreshToken) {
        clearAuthSession();
        window.dispatchEvent(new CustomEvent("auth:unauthorized"));
        return Promise.reject(error);
      }

      if (isRefreshing) {
        // Queue this request while refresh is in progress
        return new Promise((resolve, reject) => {
          failedQueue.push({ resolve, reject });
        })
          .then(() => {
            originalRequest.headers.Authorization = `Bearer ${inMemoryAccessToken}`;
            return apiClient(originalRequest);
          })
          .catch((err) => Promise.reject(err));
      }

      originalRequest._retry = true;
      isRefreshing = true;

      try {
        // Single refresh request for all concurrent 401s
        const refreshResponse = await axios.post<AuthTokens>(
          `${BASE_URL}/v1/auth/refresh`,
          { refresh_token: refreshToken },
          { headers: { "Content-Type": "application/json" } }
        );

        const { access_token, refresh_token: newRefreshToken } = refreshResponse.data;

        // Store rotated tokens
        setAccessToken(access_token);
        setRefreshToken(newRefreshToken);

        // Resume all queued requests
        processQueue(null);

        // Retry original request
        originalRequest.headers.Authorization = `Bearer ${access_token}`;
        return apiClient(originalRequest);
      } catch (refreshErr) {
        // Refresh failed (token expired, revoked, or reused)
        processQueue(refreshErr as Error);
        clearAuthSession();
        window.dispatchEvent(new CustomEvent("auth:unauthorized"));
        return Promise.reject(refreshErr);
      } finally {
        isRefreshing = false;
      }
    }

    return Promise.reject(error);
  }
);
