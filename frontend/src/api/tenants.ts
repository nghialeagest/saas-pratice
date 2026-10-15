import { apiClient } from "./client";
import {
  CreateTenantRequest,
  ListTenantsResponseRaw,
  RawTenant,
  TenantModel,
  transformTenant,
  UpdateTenantRequest,
} from "./types";

export interface ListTenantsParams {
  status?: string;
  page?: number;
  page_size?: number;
  pageSize?: number;
}

export interface ListTenantsResponse {
  items: TenantModel[];
  tenants: TenantModel[];
  total: number;
  totalCount: number;
}

// Notice: Tenants API runs in host scope, so we pass X-Tenant-ID: "" explicitly
export const listTenantsApi = async (params?: ListTenantsParams): Promise<ListTenantsResponse> => {
  const queryParams = params
    ? {
        status: params.status,
        page: params.page,
        page_size: params.page_size ?? params.pageSize,
      }
    : undefined;

  const resp = await apiClient.get<ListTenantsResponseRaw>("/v1/tenants", {
    params: queryParams,
    headers: { "X-Tenant-ID": "" },
  });
  const items = (resp.data?.items || []).map(transformTenant);
  const total = resp.data?.total || 0;

  return {
    items,
    tenants: items,
    total,
    totalCount: total,
  };
};

export const getTenantApi = async (id: string): Promise<TenantModel> => {
  const resp = await apiClient.get<RawTenant>(`/v1/tenants/${id}`, {
    headers: { "X-Tenant-ID": "" },
  });
  return transformTenant(resp.data);
};

export const createTenantApi = async (data: CreateTenantRequest): Promise<TenantModel> => {
  const resp = await apiClient.post<RawTenant>("/v1/tenants", data, {
    headers: { "X-Tenant-ID": "" },
  });
  return transformTenant(resp.data);
};

export async function updateTenantApi(
  idOrData: string | UpdateTenantRequest,
  maybeData?: Partial<UpdateTenantRequest>
): Promise<TenantModel> {
  let id: string;
  let payload: Partial<UpdateTenantRequest>;

  if (typeof idOrData === "string") {
    id = idOrData;
    payload = { id, ...maybeData };
  } else {
    id = idOrData.id;
    payload = idOrData;
  }

  const resp = await apiClient.put<RawTenant>(`/v1/tenants/${id}`, payload, {
    headers: { "X-Tenant-ID": "" },
  });
  return transformTenant(resp.data);
}
