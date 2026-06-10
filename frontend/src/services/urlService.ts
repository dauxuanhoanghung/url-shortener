import type { ApiResponse, ListURLResponse, ShortURL, URLTagsResponse } from "../types";
import api from "./api";

export interface URLListParams {
  limit?: number;
  offset?: number;
  search?: string;
  fetch_status?: "pending" | "ok" | "failed" | "";
  sort?: "newest" | "oldest" | "most_clicked";
  date_from?: string; // YYYY-MM-DD
  date_to?: string; // YYYY-MM-DD
}

export const urlService = {
  async create(originalUrl: string, tags: string[] = []): Promise<ApiResponse<ShortURL>> {
    const { data } = await api.post<ApiResponse<ShortURL>>("/urls", {
      original_url: originalUrl,
      tags,
    });
    return data;
  },

  async list(params: URLListParams = {}): Promise<ApiResponse<ListURLResponse>> {
    const query: Record<string, string> = {};
    if (params.limit) query.limit = String(params.limit);
    if (params.offset) query.offset = String(params.offset);
    if (params.search) query.search = params.search;
    if (params.fetch_status) query.fetch_status = params.fetch_status;
    if (params.sort) query.sort = params.sort;
    if (params.date_from) query.date_from = params.date_from;
    if (params.date_to) query.date_to = params.date_to;
    const { data } = await api.get<ApiResponse<ListURLResponse>>("/urls", { params: query });
    return data;
  },

  async remove(id: string): Promise<ApiResponse<null>> {
    const { data } = await api.delete<ApiResponse<null>>(`/urls/${id}`);
    return data;
  },

  async updateTags(id: string, tags: string[]): Promise<ApiResponse<URLTagsResponse>> {
    const { data } = await api.put<ApiResponse<URLTagsResponse>>(`/urls/${id}/tags`, { tags });
    return data;
  },
};
