import React, { createContext, useContext, useState, useEffect, useCallback } from "react";
import {
  getAccessToken,
  setAccessToken,
  getRefreshToken,
  setRefreshToken,
  clearAuthSession,
  setCurrentTenantId,
} from "@/api/client";
import { loginApi, refreshApi, logoutApi } from "@/api/auth";
import { LoginRequest } from "@/api/types";

export interface UserSession {
  id: string;
  tenantId: string;
  sessionId: string;
  username: string;
  permissions: string[];
}

interface AuthContextType {
  user: UserSession | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  login: (data: LoginRequest) => Promise<void>;
  logout: () => Promise<void>;
  can: (resource: string, action: string) => boolean;
}

function decodeJwtPayload(token: string): { uid?: string; tid?: string; sid?: string; sub?: string, username?: string } | null {
  try {
    const parts = token.split(".");
    if (parts.length < 2) return null;
    const base64Url = parts[1];
    const base64 = base64Url.replace(/-/g, "+").replace(/_/g, "/");
    const jsonPayload = decodeURIComponent(
      atob(base64)
        .split("")
        .map((c) => "%" + ("00" + c.charCodeAt(0).toString(16)).slice(-2))
        .join("")
    );
    return JSON.parse(jsonPayload);
  } catch {
    return null;
  }
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

// Default permissions granted to authenticated users in SaaS client
const DEFAULT_PERMISSIONS = [
  "note.note:read",
  "note.note:write",
  "task.task:read",
  "task.task:write",
  "tenant.tenant:read",
  "tenant.tenant:write",
  "user.user:read",
  "user.user:write",
  "user.role:read",
  "user.role:write",
];

export const AuthProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const [user, setUser] = useState<UserSession | null>(null);
  const [isLoading, setIsLoading] = useState<boolean>(true);

  const initUserFromToken = useCallback((accessToken: string) => {
    const decoded = decodeJwtPayload(accessToken);
    if (decoded && (decoded.uid || decoded.sub)) {
      const uid = decoded.uid || decoded.sub || "user";
      const tid = decoded.tid || "";
      const sid = decoded.sid || "";
      const username = decoded.username || "";
      setUser({
        id: uid,
        tenantId: tid,
        sessionId: sid,
        username: username,
        permissions: DEFAULT_PERMISSIONS,
      });
      if (tid) {
        setCurrentTenantId(tid);
      }
    }
  }, []);

  // Silent refresh on app mount
  useEffect(() => {
    const initializeAuth = async () => {
      const rf = getRefreshToken();
      if (!rf) {
        setIsLoading(false);
        return;
      }

      try {
        const tokens = await refreshApi(rf);
        setAccessToken(tokens.access_token);
        setRefreshToken(tokens.refresh_token);
        initUserFromToken(tokens.access_token);
      } catch {
        clearAuthSession();
        setUser(null);
      } finally {
        setIsLoading(false);
      }
    };

    initializeAuth();

    // Listen to unauthorized event dispatched from axios interceptor
    const handleUnauthorized = () => {
      setUser(null);
      clearAuthSession();
    };

    window.addEventListener("auth:unauthorized", handleUnauthorized);
    return () => {
      window.removeEventListener("auth:unauthorized", handleUnauthorized);
    };
  }, [initUserFromToken]);

  const login = useCallback(
    async (credentials: LoginRequest) => {
      const tokens = await loginApi(credentials);
      // Access token stored in memory only
      setAccessToken(tokens.access_token);
      // Refresh token stored in localStorage (documented tech debt)
      setRefreshToken(tokens.refresh_token);
      if (credentials.tenant_id) {
        setCurrentTenantId(credentials.tenant_id);
      }
      initUserFromToken(tokens.access_token);
    },
    [initUserFromToken]
  );

  const logout = useCallback(async () => {
    try {
      if (user?.sessionId) {
        await logoutApi(user.sessionId);
      } else {
        await logoutApi();
      }
    } catch {
      // Ignore logout errors
    } finally {
      clearAuthSession();
      setUser(null);
    }
  }, [user]);

  const can = useCallback(
    (resource: string, action: string): boolean => {
      if (!user) return false;
      const permKey = `${resource}:${action}`;
      return user.permissions.includes(permKey) || user.permissions.includes("*");
    },
    [user]
  );

  return (
    <AuthContext.Provider
      value={{
        user,
        isAuthenticated: !!user && !!getAccessToken(),
        isLoading,
        login,
        logout,
        can,
      }}
    >
      {children}
    </AuthContext.Provider>
  );
};

export const useAuth = (): AuthContextType => {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error("useAuth must be used within an AuthProvider");
  }
  return context;
};
