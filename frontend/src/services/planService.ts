import type { ApiResponse, Plan } from "../types";
import api from "./api";

export const planService = {
  async list(): Promise<ApiResponse<Plan[]>> {
    const { data } = await api.get<ApiResponse<Plan[]>>("/plans");
    return data;
  },
};
