import type { ApplicationPosition, ApplicationStatus } from '../types/api';

export const STATUS_LABELS: Record<ApplicationStatus, string> = {
  new: 'Новая',
  in_review: 'На рассмотрении',
  printing: 'В печати',
  ready: 'Готова',
  issued: 'Выдана',
  rejected: 'Отклонена',
  cancelled: 'Отменена',
};

export const POSITION_LABELS: Record<ApplicationPosition, string> = {
  bachelor: 'Студент бакалавриата',
  master: 'Студент магистратуры',
  postgraduate: 'Аспирант',
  employee: 'Сотрудник',
};

export const ACTIVE_STATUSES: ApplicationStatus[] = ['new', 'in_review', 'printing'];
export const ADMIN_SETTABLE_STATUSES = ['in_review', 'printing', 'ready', 'issued', 'rejected'] as const;
export const ALLOWED_FILE_EXTENSIONS = ['stl', 'step', 'stp', '3mf', 'zip'];
export const MAX_FILE_SIZE = 20 * 1024 * 1024;
export const MAX_FILES_PER_APPLICATION = 10;
export const FILE_FIELD_NAME = 'files[]';

export const FILE_TRANSFER_TIMEOUT = 5 * 60 * 1000;
