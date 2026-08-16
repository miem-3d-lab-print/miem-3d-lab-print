import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { ArrowLeft, CalendarDays, CircleUserRound, ExternalLink, Mail, MessageCircle } from 'lucide-react';
import { useState } from 'react';
import { Link, useParams } from 'react-router-dom';
import { adminApplicationsApi } from '../../api/endpoints';
import { ErrorState } from '../../components/ErrorState';
import { FileList } from '../../components/FileList';
import { LoadingState } from '../../components/LoadingState';
import { StatusBadge } from '../../components/StatusBadge';
import { Alert } from '../../components/ui/Alert';
import { Button } from '../../components/ui/Button';
import { Card } from '../../components/ui/Card';
import type { ApplicationStatus, FileMeta } from '../../types/api';
import { ADMIN_SETTABLE_STATUSES, POSITION_LABELS, STATUS_LABELS } from '../../utils/constants';
import { downloadBlob } from '../../utils/download';
import { getErrorMessage } from '../../utils/errors';
import { formatDate, formatDateTime } from '../../utils/format';

function contactURL(value: string, service: 'telegram' | 'max'): string {
  const trimmed = value.trim();
  if (/^https?:\/\//i.test(trimmed)) return trimmed;
  const username = trimmed.replace(/^@/, '');
  return service === 'telegram'
    ? `https://t.me/${encodeURIComponent(username)}`
    : `https://max.ru/${encodeURIComponent(username)}`;
}

export function AdminApplicationDetailsPage() {
  const { id = '' } = useParams();
  const queryClient = useQueryClient();
  const [status, setStatus] = useState<ApplicationStatus | ''>('');
  const [comment, setComment] = useState('');
  const [rejectionReason, setRejectionReason] = useState('');
  const [downloadingId, setDownloadingId] = useState<string | null>(null);
  const [downloadError, setDownloadError] = useState('');
  const query = useQuery({ queryKey: ['admin-application', id], queryFn: () => adminApplicationsApi.get(id), enabled: Boolean(id) });
  const update = useMutation({
    mutationFn: () => {
      const targetStatus = (status || query.data?.status) as Exclude<ApplicationStatus, 'new' | 'cancelled'>;
      const fallbackReason = targetStatus === query.data?.status ? query.data?.rejection_reason?.trim() : '';
      return adminApplicationsApi.updateStatus(id, {
        status: targetStatus,
        comment: comment.trim() || undefined,
        rejection_reason: targetStatus === 'rejected' ? (rejectionReason.trim() || fallbackReason || undefined) : undefined,
      });
    },
    onSuccess: async () => { setComment(''); setRejectionReason(''); await queryClient.invalidateQueries({ queryKey: ['admin-application', id] }); await queryClient.invalidateQueries({ queryKey: ['admin-applications'] }); },
  });
  const download = async (file: FileMeta) => {
    setDownloadingId(file.id);
    setDownloadError('');
    try { const response = await adminApplicationsApi.downloadFile(id, file.id); downloadBlob(response.data, file.filename); } catch (error) { setDownloadError(getErrorMessage(error)); } finally { setDownloadingId(null); }
  };
  if (query.isPending) return <LoadingState />;
  if (query.isError) return <ErrorState message={getErrorMessage(query.error)} onRetry={() => void query.refetch()} />;
  const app = query.data;
  const canRepeatCurrentStatus = ADMIN_SETTABLE_STATUSES.includes(app.status as (typeof ADMIN_SETTABLE_STATUSES)[number]);
  const selectedStatus = status || (canRepeatCurrentStatus ? app.status : '');
  const sameStatus = Boolean(selectedStatus && selectedStatus === app.status);
  const invalid = app.status === 'cancelled' || !selectedStatus || (sameStatus && !comment.trim()) || (selectedStatus === 'rejected' && !(rejectionReason.trim() || (selectedStatus === app.status ? app.rejection_reason?.trim() : '')));

  return <div className="admin-section admin-detail"><Link className="back-link" to="/admin/applications"><ArrowLeft size={16} /> Все заявки</Link><div className="page-heading page-heading--actions"><div><span className="eyebrow">ЗАЯВКА № {app.number}</span><h1>{app.title}</h1><p>Подана {formatDateTime(app.created_at)}</p></div><StatusBadge status={app.status} /></div>
    <div className="details-grid"><Card className="detail-card"><h2>Заявитель</h2><div className="contact-list"><span><CircleUserRound size={18} /><strong>{app.applicant.snapshot_full_name}</strong></span><span><Mail size={18} />{app.applicant.snapshot_email}</span>{app.applicant.telegram ? <span><MessageCircle size={18} /><a href={contactURL(app.applicant.telegram, 'telegram')} target="_blank" rel="noreferrer">{app.applicant.telegram}</a></span> : null}{app.applicant.max ? <span><MessageCircle size={18} /><a href={contactURL(app.applicant.max, 'max')} target="_blank" rel="noreferrer">MAX: {app.applicant.max}</a></span> : null}</div></Card>
      <Card className="detail-card detail-card--wide"><h2>Параметры</h2><dl className="detail-list"><div><dt>Цель</dt><dd>{app.purpose}</dd></div><div><dt>Категория</dt><dd>{POSITION_LABELS[app.position]}</dd></div><div><dt>Материал</dt><dd>{app.material.snapshot_name}{!app.material.is_active_now ? ' · сейчас недоступен' : ''}</dd></div><div><dt>Цвет</dt><dd>{app.color_matters ? app.color?.snapshot_name || '—' : 'Не важен'}</dd></div><div><dt>Желаемая дата</dt><dd>{formatDate(app.desired_date)}</dd></div><div><dt>Комментарий</dt><dd>{app.comment || '—'}</dd></div></dl></Card></div>
    {app.rejection_reason ? <Alert><strong>Причина отклонения:</strong> {app.rejection_reason}</Alert> : null}
    <div className="details-grid"><Card className="detail-card"><h2>Файлы</h2><FileList files={app.files} onDownload={download} downloadingId={downloadingId} />{app.file_url ? <a className="external-file-link" href={app.file_url} target="_blank" rel="noreferrer"><ExternalLink size={17} /> Открыть файл по ссылке</a> : null}{downloadError ? <Alert>{downloadError}</Alert> : null}</Card><Card className="detail-card detail-card--wide"><h2>История</h2><div className="timeline">{app.status_history.map((entry, index) => <div className="timeline-item" key={`${entry.created_at}-${index}`}><span className="timeline-dot" /><div><strong>{STATUS_LABELS[entry.status]}</strong><span className="timeline-meta"><CalendarDays size={14} /> {formatDateTime(entry.created_at)} · {entry.changed_by ? `${entry.changed_by.full_name || 'Без имени'} (${entry.changed_by.role === 'admin' ? 'администратор' : 'пользователь'})` : 'Системное изменение'}</span>{entry.comment ? <p>{entry.comment}</p> : null}</div></div>)}</div></Card></div>
    <Card className="status-editor"><h2>Изменить статус или добавить комментарий</h2>{app.status === 'cancelled' ? <Alert tone="warning">Отменённая заявка финальна и не может менять статус.</Alert> : <div className="form-stack"><label className="field"><span>Новый статус</span><select value={selectedStatus} onChange={(event) => setStatus(event.target.value as ApplicationStatus)}>{canRepeatCurrentStatus ? <option value={app.status}>{STATUS_LABELS[app.status]} — текущий</option> : <option value="">Выберите статус</option>}{ADMIN_SETTABLE_STATUSES.filter((item) => item !== app.status).map((item) => <option key={item} value={item}>{STATUS_LABELS[item]}</option>)}</select></label><label className="field"><span>Комментарий {sameStatus ? '(обязателен без смены статуса)' : '(необязательно)'}</span><textarea rows={3} value={comment} onChange={(event) => setComment(event.target.value)} /></label>{selectedStatus === 'rejected' ? <label className="field"><span>Причина отклонения</span><textarea rows={3} value={rejectionReason} onChange={(event) => setRejectionReason(event.target.value)} /></label> : null}{update.isError ? <Alert>{getErrorMessage(update.error)}</Alert> : null}<Button disabled={invalid} loading={update.isPending} onClick={() => update.mutate()}>Сохранить изменение</Button></div>}</Card>
  </div>;
}
