import { apiClient } from "./client";
import {
  CreateUserRequest,
  ListUsersResponseRaw,
  RawUser,
  transformUser,
  UserModel,
} from "./types";

export interface ListUsersParams {
  page?: number;
  pageSize?: number;
  search?: string;
}

export interface ListUsersResult {
  items: UserModel[];
  total: number;
}

export const listUsersApi = async (params?: ListUsersParams): Promise<ListUsersResult> => {
  const resp = await apiClient.get<ListUsersResponseRaw>("/v1/users", {
    params: {
      page: params?.page,
      page_size: params?.pageSize,
      search: params?.search || undefined,
    },
  });
  return {
    items: (resp.data?.items || []).map(transformUser),
    total: resp.data?.total ?? 0,
  };
};

export const getUserApi = async (id: string): Promise<UserModel> => {
  const resp = await apiClient.get<RawUser>(`/v1/users/${id}`);
  return transformUser(resp.data);
};

export const createUserApi = async (data: CreateUserRequest): Promise<UserModel> => {
  const resp = await apiClient.post<RawUser>("/v1/users", data);
  return transformUser(resp.data);
};

export const updateUserApi = async (id: string, data: { name?: string }): Promise<UserModel> => {
  const resp = await apiClient.put<RawUser>(`/v1/users/${id}`, data);
  return transformUser(resp.data);
};

export const deleteUserApi = async (id: string): Promise<void> => {
  await apiClient.delete(`/v1/users/${id}`);
};

export const assignRolesApi = async (id: string, roleIds: string[]): Promise<UserModel> => {
  const resp = await apiClient.put<RawUser>(`/v1/users/${id}/roles`, { role_ids: roleIds });
  return transformUser(resp.data);
};
