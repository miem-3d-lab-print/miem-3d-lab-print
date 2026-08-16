import type { ApplicationStatus } from '../types/api';
import { STATUS_LABELS } from '../utils/constants';

export function StatusBadge({ status }: { status: ApplicationStatus }) {
  return <span className={`status-badge status-badge--${status}`}>{STATUS_LABELS[status]}</span>;
}
