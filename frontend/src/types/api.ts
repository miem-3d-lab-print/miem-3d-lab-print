export type Role = 'user' | 'admin';
export type ApplicationStatus =
  | 'new'
  | 'in_review'
  | 'printing'
  | 'ready'
  | 'issued'
  | 'rejected'
  | 'cancelled';
export type ApplicationPosition = 'bachelor' | 'master' | 'postgraduate' | 'employee';
export type FileFormat = 'STL' | 'STEP' | '3MF';

export interface ApiErrorBody {
  error: {
    code: string;
    message: string;
    details?: Record<string, unknown>;
  };
}

export interface PaginationMeta {
  page: number;
  per_page: number;
  total: number;
  total_pages: number;
}

export interface PaginatedResponse<T> {
  items: T[];
  meta: PaginationMeta;
}

export interface RequestOtpResponse {
  message: string;
  expires_in: number;
}

export interface VerifyOtpResponse {
  access_token: string;
  refresh_token: string;
  token_type: 'Bearer';
  expires_in: number;
  role: Role;
  consent_required: boolean;
}

export interface RefreshResponse {
  access_token: string;
  refresh_token?: string;
  expires_in: number;
}

export interface Profile {
  id: string;
  email: string;
  full_name: string | null;
  telegram: string | null;
  max: string | null;
  role: Role;
  consent_given: boolean;
  consent_date: string | null;
  created_at: string;
}

export interface ProfilePatch {
  full_name?: string | null;
  telegram?: string | null;
  max?: string | null;
}

export interface ConsentResponse {
  consent_given: true;
  consent_date: string;
}

export interface MaterialColor {
  id: string;
  name: string;
}

export interface Material {
  id: string;
  name: string;
  description: string | null;
  colors: MaterialColor[];
}

export interface AdminMaterialColor extends MaterialColor {
  is_active: boolean;
}

export interface AdminMaterial extends Omit<Material, 'colors'> {
  is_active: boolean;
  created_at: string;
  colors: AdminMaterialColor[];
}

export interface UserApplicationSummary {
  id: string;
  number: string;
  status: ApplicationStatus;
  material_name: string;
  color_name: string | null;
  desired_date: string;
  created_at: string;
  files_count: number;
}

export interface FileMeta {
  id: string;
  filename: string;
  size: number;
  format: FileFormat;
  created_at?: string;
}

export interface UserStatusHistory {
  status: ApplicationStatus;
  comment: string | null;
  changed_by_role: Role;
  created_at: string;
}

export interface UserApplicationDetails {
  id: string;
  number: string;
  status: ApplicationStatus;
  rejection_reason: string | null;
  position: ApplicationPosition;
  purpose: string;
  material: { id: string; snapshot_name: string };
  color_matters: boolean;
  color: { id: string; snapshot_name: string } | null;
  desired_date: string;
  comment: string | null;
  files: FileMeta[];
  files_delete_after: string | null;
  status_history: UserStatusHistory[];
  created_at: string;
}

export interface CreatedApplication {
  id: string;
  number: string;
  status: 'new';
  created_at: string;
}

export interface CancelApplicationResponse {
  id: string;
  status: 'cancelled';
  files_delete_after: string;
}

export interface AdminApplicationSummary {
  id: string;
  number: string;
  full_name: string;
  created_at: string;
  desired_date: string;
  material_name: string;
  status: ApplicationStatus;
  deadline_soon: boolean;
}

export interface AdminHistoryActor {
  id: string;
  full_name: string;
  role: Role;
}

export interface AdminStatusHistory {
  status: ApplicationStatus;
  comment: string | null;
  changed_by: AdminHistoryActor | null;
  created_at: string;
}

export interface AdminApplicationDetails {
  id: string;
  number: string;
  applicant: {
    user_id: string;
    snapshot_full_name: string;
    snapshot_email: string;
    telegram: string | null;
    max: string | null;
  };
  position: ApplicationPosition;
  purpose: string;
  material: { id: string; snapshot_name: string; is_active_now: boolean };
  color_matters: boolean;
  color: { id: string; snapshot_name: string } | null;
  desired_date: string;
  comment: string | null;
  status: ApplicationStatus;
  rejection_reason: string | null;
  files: FileMeta[];
  status_history: AdminStatusHistory[];
  created_at: string;
}

export interface AdminStatusPatch {
  status: Exclude<ApplicationStatus, 'new' | 'cancelled'>;
  comment?: string;
  rejection_reason?: string;
}

export interface AdminStatusPatchResponse {
  id: string;
  status: ApplicationStatus;
  files_delete_after: string | null;
}

export interface AdminUser {
  id: string;
  email: string;
  full_name: string | null;
  role: Role;
  created_at: string;
}

export interface AdminUserRoleResponse {
  id: string;
  email: string;
  role: Role;
}

export interface AdminStats {
  period: { date_from: string; date_to: string };
  total_applications: number;
  avg_completion_hours: number | null;
  completed_count: number;
  by_material: Array<{ material_name: string; count: number }>;
  by_status_current: Record<ApplicationStatus, number>;
}
