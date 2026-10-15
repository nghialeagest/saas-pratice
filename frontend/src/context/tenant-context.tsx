import React, { createContext, useContext, useState, useEffect, useCallback } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { getCurrentTenantId, setCurrentTenantId } from "@/api/client";
import { Tenant } from "@/api/types";
import { listTenantsApi } from "@/api/tenants";
import { useAuth } from "@/hooks/use-auth";

interface TenantContextType {
  tenantId: string;
  setTenantId: (id: string) => void;
  currentTenant: Tenant | null;
  tenants: Tenant[];
  isLoadingTenants: boolean;
  refreshTenants: () => Promise<void>;
}

const TenantContext = createContext<TenantContextType | undefined>(undefined);

export const TenantProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const queryClient = useQueryClient();
  const { isAuthenticated } = useAuth();
  const [tenantId, setTenantIdState] = useState<string>(getCurrentTenantId());
  const [tenants, setTenants] = useState<Tenant[]>([]);
  const [isLoadingTenants, setIsLoadingTenants] = useState<boolean>(false);

  const refreshTenants = useCallback(async () => {
    if (!isAuthenticated) {
      setTenants([]);
      return;
    }
    try {
      setIsLoadingTenants(true);
      // Fetch tenants using host scope
      const res = await listTenantsApi({ page: 1, pageSize: 50 });
      setTenants(res.tenants);
      // If current tenant is not set and we have tenants, default to first tenant
      const stored = getCurrentTenantId();
      if (!stored && res.tenants.length > 0) {
        const first = res.tenants[0].id;
        setCurrentTenantId(first);
        setTenantIdState(first);
      }
    } catch (err) {
      setTenants([]);
      console.error("failed to load tenants", err);
    } finally {
      setIsLoadingTenants(false);
    }
  }, [isAuthenticated]);

  useEffect(() => {
    refreshTenants();
  }, [refreshTenants]);

  const setTenantId = useCallback(
    (newId: string) => {
      setCurrentTenantId(newId);
      setTenantIdState(newId);
      // Mandatory: switching tenant MUST purge/invalidate all TanStack Query cache
      queryClient.clear();
    },
    [queryClient]
  );

  const currentTenant = tenants.find((t) => t.id === tenantId) || null;

  return (
    <TenantContext.Provider
      value={{
        tenantId,
        setTenantId,
        currentTenant,
        tenants,
        isLoadingTenants,
        refreshTenants,
      }}
    >
      {children}
    </TenantContext.Provider>
  );
};

export const useTenant = (): TenantContextType => {
  const context = useContext(TenantContext);
  if (!context) {
    throw new Error("useTenant must be used within a TenantProvider");
  }
  return context;
};
