import type { AdminAuditList, AdminUser, AdminUserList, ApiResponse, Plan } from "../types";
import api from "./api";

export const adminService = {
  async listUsers(limit = 50, offset = 0): Promise<ApiResponse<AdminUserList>> {
    const { data } = await api.get<ApiResponse<AdminUserList>>("/admin/users", {
      params: { limit, offset },
    });
    return data;
  },

  async getUser(id: string): Promise<ApiResponse<AdminUser>> {
    const { data } = await api.get<ApiResponse<AdminUser>>(`/admin/users/${id}`);
    return data;
  },

  async disableUser(id: string): Promise<ApiResponse<AdminUser>> {
    const { data } = await api.post<ApiResponse<AdminUser>>(`/admin/users/${id}/disable`);
    return data;
  },

  async enableUser(id: string): Promise<ApiResponse<AdminUser>> {
    const { data } = await api.post<ApiResponse<AdminUser>>(`/admin/users/${id}/enable`);
    return data;
  },

  async updatePlanFeatures(
    code: string,
    features: Record<string, boolean>,
  ): Promise<ApiResponse<Plan>> {
    const { data } = await api.patch<ApiResponse<Plan>>(`/admin/plans/${code}/features`, {
      features,
    });
    return data;
  },

  async listAudit(limit = 50, offset = 0): Promise<ApiResponse<AdminAuditList>> {
    const { data } = await api.get<ApiResponse<AdminAuditList>>("/admin/audit", {
      params: { limit, offset },
    });
    return data;
  },
};
