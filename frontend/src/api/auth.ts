import { apiClient, setAccessToken, setRefreshToken, clearAuthSession } from "./client";
import { AuthTokens, LoginRequest } from "./types";

export const loginApi = async (req: LoginRequest): Promise<AuthTokens> => {
  const resp = await apiClient.post<AuthTokens>("/v1/auth/login", req);
  setAccessToken(resp.data.access_token);
  setRefreshToken(resp.data.refresh_token);
  return resp.data;
};

export const refreshApi = async (refreshToken: string): Promise<AuthTokens> => {
  const resp = await apiClient.post<AuthTokens>("/v1/auth/refresh", {
    refresh_token: refreshToken,
  });
  setAccessToken(resp.data.access_token);
  setRefreshToken(resp.data.refresh_token);
  return resp.data;
};

export const logoutApi = async (sessionId?: string): Promise<void> => {
  try {
    await apiClient.post("/v1/auth/logout", {
      session_id: sessionId,
    });
  } finally {
    clearAuthSession();
  }
};
