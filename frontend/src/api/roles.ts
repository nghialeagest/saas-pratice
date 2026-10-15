import { apiClient } from "./client";
import { ListRolesResponseRaw, Permission, PermissionListRaw, Role } from "./types";

export const listRolesApi = async (): Promise<Role[]> => {
  const resp = await apiClient.get<ListRolesResponseRaw>("/v1/roles");
  return resp.data?.items || [];
};

export const createRoleApi = async (name: string): Promise<Role> => {
  const resp = await apiClient.post<Role>("/v1/roles", { name });
  return resp.data;
};

export const deleteRoleApi = async (id: string): Promise<void> => {
  await apiClient.delete(`/v1/roles/${id}`);
};

export const getRolePermissionsApi = async (roleId: string): Promise<Permission[]> => {
  const resp = await apiClient.get<PermissionListRaw>(`/v1/roles/${roleId}/permissions`);
  return resp.data?.items || [];
};

export const setRolePermissionsApi = async (
  roleId: string,
  permissions: Permission[]
): Promise<void> => {
  await apiClient.put(`/v1/roles/${roleId}/permissions`, {
    permissions: permissions.map((p) => ({ namespace: p.namespace, action: p.action })),
  });
};
