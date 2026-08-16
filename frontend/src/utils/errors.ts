import axios from 'axios';
import type { ApiErrorBody } from '../types/api';

export interface NormalizedApiError {
  status?: number;
  code: string;
  message: string;
  details?: Record<string, unknown>;
}

export function normalizeApiError(error: unknown): NormalizedApiError {
  if (axios.isAxiosError<ApiErrorBody>(error)) {
    const data = error.response?.data;
    return {
      status: error.response?.status,
      code: data?.error?.code ?? 'NETWORK_ERROR',
      message: data?.error?.message ?? (error.message || 'Не удалось выполнить запрос'),
      details: data?.error?.details,
    };
  }
  if (error instanceof Error) return { code: 'UNKNOWN_ERROR', message: error.message };
  return { code: 'UNKNOWN_ERROR', message: 'Произошла неизвестная ошибка' };
}

export function getErrorMessage(error: unknown): string {
  return normalizeApiError(error).message;
}
