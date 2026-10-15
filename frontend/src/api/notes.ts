import { apiClient } from "./client";
import {
  CountResponse,
  ListNotesResponse,
  ListNotesResponseRaw,
  NoteModel,
  RawNote,
  transformNote,
} from "./types";

export interface ListNotesParams {
  status?: string;
  page?: number;
  page_size?: number;
  pageSize?: number;
}

export const listNotesApi = async (params?: ListNotesParams): Promise<ListNotesResponse> => {
  const queryParams = params
    ? {
        status: params.status,
        page: params.page,
        page_size: params.page_size ?? params.pageSize,
      }
    : undefined;

  const resp = await apiClient.get<ListNotesResponseRaw>("/v1/notes", { params: queryParams });
  const rawItems = resp.data?.items || [];
  const transformed = rawItems.map(transformNote);
  const total = resp.data?.total || 0;

  return {
    items: transformed,
    notes: transformed,
    total,
    totalCount: total,
  };
};

export const getNoteApi = async (id: string): Promise<NoteModel> => {
  const resp = await apiClient.get<RawNote>(`/v1/notes/${id}`);
  return transformNote(resp.data);
};

export const countNotesApi = async (status?: string): Promise<CountResponse> => {
  const resp = await apiClient.get<CountResponse>("/v1/notes/count", {
    params: status ? { status } : undefined,
  });
  return {
    count: resp.data?.count || 0,
  };
};

export const createNoteApi = async (data: { title: string; body?: string }): Promise<NoteModel> => {
  const resp = await apiClient.post<RawNote>("/v1/notes", data);
  return transformNote(resp.data);
};

export const publishNoteApi = async (id: string): Promise<NoteModel> => {
  const resp = await apiClient.post<RawNote>(`/v1/notes/${id}/publish`);
  return transformNote(resp.data);
};
