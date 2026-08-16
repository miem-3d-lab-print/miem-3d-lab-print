import type { AxiosProgressEvent } from 'axios';
import { apiClient } from './client';
import type {
  AdminApplicationDetails, AdminApplicationNotificationsResponse, AdminApplicationSummary, AdminMaterial, AdminStats,
  AdminStatusPatch, AdminStatusPatchResponse, AdminUser, AdminUserRoleResponse, CancelApplicationResponse,
  ConsentResponse, CreatedApplication, FileMeta, Material, PaginatedResponse, Profile, ProfilePatch,
  RefreshResponse, RequestOtpResponse, Role, UserApplicationDetails, UserApplicationSummary, VerifyOtpResponse,
} from '../types/api';
import { FILE_TRANSFER_TIMEOUT } from '../utils/constants';


export const authApi = {
  requestOtp: async (email: string) => (await apiClient.post<RequestOtpResponse>('/auth/request-otp', { email })).data,
  verifyOtp: async (email: string, code: string) => (await apiClient.post<VerifyOtpResponse>('/auth/verify-otp', { email, code })).data,
  refresh: async (refreshToken: string) => (await apiClient.post<RefreshResponse>('/auth/refresh', { refresh_token: refreshToken })).data,
  logout: async (refreshToken: string) => apiClient.post('/auth/logout', { refresh_token: refreshToken }),
};

export const profileApi = {
  get: async () => (await apiClient.get<Profile>('/profile')).data,
  update: async (payload: ProfilePatch) => (await apiClient.patch<Profile>('/profile', payload)).data,
  consent: async () => (await apiClient.post<ConsentResponse>('/profile/consent')).data,
};

export const materialsApi = {
  list: async () => (await apiClient.get<{ items: Material[] }>('/materials')).data.items,
};

export interface UserApplicationListParams {
  status?: string;
  page?: number;
  per_page?: number;
}

export interface CreateApplicationPayload {
  title: string;
  position: string;
  purpose: string;
  material_id: string;
  color_matters: boolean;
  color_id?: string;
  desired_date: string;
  comment?: string;
  file_url?: string;
  pending_file_ids: string[];
}

export type UploadProgressCallback = (progress: number) => void;

function reportUploadProgress(callback?: UploadProgressCallback) {
  return (event: AxiosProgressEvent) => {
    if (!callback || !event.total) return;
    callback(Math.min(100, Math.round((event.loaded / event.total) * 100)));
  };
}

export const applicationsApi = {
  list: async (params: UserApplicationListParams) =>
    (await apiClient.get<PaginatedResponse<UserApplicationSummary>>('/applications', { params })).data,
  get: async (id: string) => (await apiClient.get<UserApplicationDetails>(`/applications/${id}`)).data,
  create: async (payload: CreateApplicationPayload, onProgress?: UploadProgressCallback) => {
    const body = new FormData();
    body.append('title', payload.title);
    body.append('position', payload.position);
    body.append('purpose', payload.purpose);
    body.append('material_id', payload.material_id);
    body.append('color_matters', String(payload.color_matters));
    if (payload.color_matters && payload.color_id) body.append('color_id', payload.color_id);
    body.append('desired_date', payload.desired_date);
    if (payload.comment) body.append('comment', payload.comment);
    if (payload.file_url) body.append('file_url', payload.file_url);
    payload.pending_file_ids.forEach((id) => body.append('pending_file_ids[]', id));
    return (await apiClient.post<CreatedApplication>('/applications', body, {
      timeout: FILE_TRANSFER_TIMEOUT,
      onUploadProgress: reportUploadProgress(onProgress),
    })).data;
  },
  uploadPendingFile: async (file: File, onProgress?: UploadProgressCallback) => {
    const body = new FormData();
    body.append('file', file);
    return (await apiClient.post<FileMeta>('/applications/pending-files', body, {
      timeout: FILE_TRANSFER_TIMEOUT,
      onUploadProgress: reportUploadProgress(onProgress),
    })).data;
  },
  deletePendingFile: async (id: string) => apiClient.delete(`/applications/pending-files/${id}`),
  cancel: async (id: string) => (await apiClient.patch<CancelApplicationResponse>(`/applications/${id}/cancel`)).data,
  uploadFile: async (id: string, file: File, onProgress?: UploadProgressCallback) => {
    const body = new FormData();
    body.append('file', file);
    return (await apiClient.post<FileMeta>(`/applications/${id}/files`, body, {
      timeout: FILE_TRANSFER_TIMEOUT,
      onUploadProgress: reportUploadProgress(onProgress),
    })).data;
  },
  downloadFile: async (applicationId: string, fileId: string) =>
    apiClient.get<Blob>(`/applications/${applicationId}/files/${fileId}`, { responseType: 'blob', timeout: FILE_TRANSFER_TIMEOUT }),
};

export interface AdminApplicationListParams {
  status?: string[];
  material_id?: string;
  created_from?: string;
  created_to?: string;
  desired_from?: string;
  desired_to?: string;
  search?: string;
  page?: number;
  per_page?: number;
}

function adminApplicationParams(params: AdminApplicationListParams): URLSearchParams {
  const result = new URLSearchParams();
  params.status?.forEach((status) => result.append('status', status));
  for (const key of ['material_id', 'created_from', 'created_to', 'desired_from', 'desired_to', 'search', 'page', 'per_page'] as const) {
    const value = params[key];
    if (value !== undefined && value !== '') result.set(key, String(value));
  }
  return result;
}

export const adminApplicationsApi = {
  list: async (params: AdminApplicationListParams) =>
    (await apiClient.get<PaginatedResponse<AdminApplicationSummary>>('/admin/applications', { params: adminApplicationParams(params) })).data,
  get: async (id: string) => (await apiClient.get<AdminApplicationDetails>(`/admin/applications/${id}`)).data,
  updateStatus: async (id: string, payload: AdminStatusPatch) =>
    (await apiClient.patch<AdminStatusPatchResponse>(`/admin/applications/${id}/status`, payload)).data,
  delete: async (id: string) => apiClient.delete(`/admin/applications/${id}`),
  downloadFile: async (applicationId: string, fileId: string) =>
    apiClient.get<Blob>(`/admin/applications/${applicationId}/files/${fileId}`, { responseType: 'blob', timeout: FILE_TRANSFER_TIMEOUT }),
};

export const adminMaterialsApi = {
  list: async () => (await apiClient.get<{ items: AdminMaterial[] }>('/admin/materials')).data.items,
  create: async (payload: { name: string; description?: string; is_active?: boolean }) =>
    (await apiClient.post<AdminMaterial>('/admin/materials', payload)).data,
  update: async (id: string, payload: Partial<Pick<AdminMaterial, 'name' | 'description' | 'is_active'>>) =>
    (await apiClient.patch<AdminMaterial>(`/admin/materials/${id}`, payload)).data,
  addColor: async (id: string, payload: { name: string; is_active?: boolean }) =>
    (await apiClient.post(`/admin/materials/${id}/colors`, payload)).data,
  updateColor: async (materialId: string, colorId: string, payload: { name?: string; is_active?: boolean }) =>
    (await apiClient.patch(`/admin/materials/${materialId}/colors/${colorId}`, payload)).data,
};

export const adminUsersApi = {
  search: async (email: string) => (await apiClient.get<{ items: AdminUser[] }>('/admin/users', { params: { email } })).data.items,
  listAdmins: async () => (await apiClient.get<{ items: AdminUser[] }>('/admin/admins')).data.items,
  setRole: async (id: string, role: Role) =>
    (await apiClient.patch<AdminUserRoleResponse>(`/admin/users/${id}/role`, { role })).data,
  setApplicationNotifications: async (id: string, enabled: boolean) =>
    (await apiClient.patch<AdminApplicationNotificationsResponse>(`/admin/users/${id}/application-notifications`, { enabled })).data,
};

export const adminStatsApi = {
  get: async (date_from?: string, date_to?: string) =>
    (await apiClient.get<AdminStats>('/admin/stats', { params: { date_from, date_to } })).data,
};
