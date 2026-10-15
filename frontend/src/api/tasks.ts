import { apiClient } from "./client";
import { CountResponse, RawTask, TaskModel, transformTask } from "./types";

export interface CreateTaskPayload {
  title: string;
  note_id?: string;
}

export const createTaskApi = async (data: CreateTaskPayload): Promise<TaskModel> => {
  const resp = await apiClient.post<RawTask>("/v1/tasks", data);
  return transformTask(resp.data);
};

export const countTasksApi = async (): Promise<CountResponse> => {
  const resp = await apiClient.get<CountResponse>("/v1/tasks/count");
  return {
    count: resp.data?.count || 0,
  };
};

export const getTaskApi = async (id: string): Promise<TaskModel> => {
  const resp = await apiClient.get<RawTask>(`/v1/tasks/${id}`);
  return transformTask(resp.data);
};
