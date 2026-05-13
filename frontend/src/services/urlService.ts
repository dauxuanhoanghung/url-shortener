import type { ApiResponse, ListURLResponse, ShortURL } from "../types";
import api from "./api";

export const urlService = {
  async create(originalUrl: string): Promise<ApiResponse<ShortURL>> {
    const { data } = await api.post<ApiResponse<ShortURL>>("/urls", {
      original_url: originalUrl,
    });
    return data;
  },

  async list(): Promise<ApiResponse<ListURLResponse>> {
    const { data } = await api.get<ApiResponse<ListURLResponse>>("/urls");
    return data;
  },

  async remove(id: string): Promise<ApiResponse<null>> {
    const { data } = await api.delete<ApiResponse<null>>(`/urls/${id}`);
    return data;
  },
};
