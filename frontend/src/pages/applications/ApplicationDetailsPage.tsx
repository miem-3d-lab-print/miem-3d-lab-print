import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { ArrowLeft, CalendarDays, CircleUserRound, ExternalLink, MessageSquareText, Upload } from 'lucide-react';
import { useRef, useState } from 'react';
import { Link, useParams } from 'react-router-dom';
import { applicationsApi } from '../../api/endpoints';
import { ErrorState } from '../../components/ErrorState';
import { FileList } from '../../components/FileList';
import { LoadingState } from '../../components/LoadingState';
import { StatusBadge } from '../../components/StatusBadge';
import { UploadProgress } from '../../components/UploadProgress';
import { Alert } from '../../components/ui/Alert';
import { Button } from '../../components/ui/Button';
import { Card } from '../../components/ui/Card';
import type { FileMeta } from '../../types/api';
import { ALLOWED_FILE_EXTENSIONS, MAX_FILE_SIZE, MAX_FILES_PER_APPLICATION, POSITION_LABELS, STATUS_LABELS } from '../../utils/constants';
import { downloadBlob } from '../../utils/download';
import { getErrorMessage } from '../../utils/errors';
import { formatDate, formatDateTime } from '../../utils/format';

function validateFile(file: File): string | null {
  const extension = file.name.split('.').pop()?.toLowerCase() || '';
  if (!ALLOWED_FILE_EXTENSIONS.includes(extension)) return 'Разрешены только STL, STEP, STP, 3MF и ZIP.';
  if (file.size <= 0) return 'Файл пустой.';
  if (file.size > MAX_FILE_SIZE) return 'Размер файла не должен превышать 20 МБ.';
  return null;
}

export function ApplicationDetailsPage() {
  const { id = '' } = useParams();
  const queryClient = useQueryClient();
  const inputRef = useRef<HTMLInputElement>(null);
  const [fileError, setFileError] = useState('');
  const [uploadProgress, setUploadProgress] = useState(0);
  const [downloadingId, setDownloadingId] = useState<string | null>(null);
  const query = useQuery({ queryKey: ['application', id], queryFn: () => applicationsApi.get(id), enabled: Boolean(id) });
  const cancel = useMutation({ mutationFn: () => applicationsApi.cancel(id), onSuccess: async () => { await queryClient.invalidateQueries({ queryKey: ['application', id] }); await queryClient.invalidateQueries({ queryKey: ['applications'] }); } });
  const upload = useMutation({ mutationFn: (file: File) => applicationsApi.uploadFile(id, file, setUploadProgress), onMutate: () => setUploadProgress(0), onSuccess: async () => { setFileError(''); if (inputRef.current) inputRef.current.value = ''; await queryClient.invalidateQueries({ queryKey: ['application', id] }); } });

  const handleFile = (file?: File) => {
    if (!file) return;
    if ((query.data?.files.length ?? 0) >= MAX_FILES_PER_APPLICATION) {
      setFileError(`Достигнут лимит: не более ${MAX_FILES_PER_APPLICATION} файлов на заявку.`);
      if (inputRef.current) inputRef.current.value = '';
      return;
    }
    const validation = validateFile(file);
    if (validation) { setFileError(validation); return; }
    upload.mutate(file);
  };
  const download = async (file: FileMeta) => {
    setDownloadingId(file.id);
    try {
      const response = await applicationsApi.downloadFile(id, file.id);
      downloadBlob(response.data, file.filename);
    } catch (error) { setFileError(getErrorMessage(error)); } finally { setDownloadingId(null); }
  };

  if (query.isPending) return <LoadingState />;
  if (query.isError) return <section className="page page--narrow"><ErrorState message={getErrorMessage(query.error)} onRetry={() => void query.refetch()} /></section>;
  const app = query.data;
  return <section className="page page--detail"><Link className="back-link" to="/app/applications"><ArrowLeft size={16} /> Мои заявки</Link>
    <div className="page-heading page-heading--actions"><div><span className="eyebrow">ЗАЯВКА № {app.number}</span><h1>{app.title}</h1><p>Подана {formatDateTime(app.created_at)}</p></div><StatusBadge status={app.status} /></div>
    {app.rejection_reason ? <Alert><strong>Причина отклонения:</strong> {app.rejection_reason}</Alert> : null}
    <div className="details-grid"><Card className="detail-card detail-card--wide"><h2>Параметры печати</h2><dl className="detail-list">
      <div><dt>Цель</dt><dd>{app.purpose}</dd></div><div><dt>Категория</dt><dd>{POSITION_LABELS[app.position]}</dd></div>
      <div><dt>Материал</dt><dd>{app.material.snapshot_name}</dd></div><div><dt>Цвет</dt><dd>{app.color_matters ? app.color?.snapshot_name || '—' : 'Не важен'}</dd></div>
      <div><dt>Желаемая дата</dt><dd>{formatDate(app.desired_date)}</dd></div><div><dt>Хранение файлов</dt><dd>{app.files_delete_after ? `до ${formatDateTime(app.files_delete_after)}` : 'Без ограничения для активной заявки'}</dd></div>
    </dl>{app.comment ? <div className="note-box"><MessageSquareText size={18} /><div><strong>Комментарий к заявке</strong><p>{app.comment}</p></div></div> : null}</Card>
    <Card className="detail-card"><h2>Файлы</h2><FileList files={app.files} onDownload={download} downloadingId={downloadingId} />{app.file_url ? <a className="external-file-link" href={app.file_url} target="_blank" rel="noreferrer"><ExternalLink size={17} /> Открыть файл по ссылке</a> : null}
      {app.status === 'new' ? <div className="upload-inline"><input ref={inputRef} type="file" accept=".stl,.step,.stp,.3mf,.zip" disabled={upload.isPending} onChange={(event) => handleFile(event.target.files?.[0])} /><Button variant="secondary" loading={upload.isPending} disabled={app.files.length >= MAX_FILES_PER_APPLICATION} onClick={() => inputRef.current?.click()}><Upload size={17} /> Добавить файл</Button><small>{app.files.length >= MAX_FILES_PER_APPLICATION ? `Достигнут лимит ${MAX_FILES_PER_APPLICATION} файлов.` : `STL, STEP, STP, 3MF или ZIP до 20 МБ. Файлов: ${app.files.length}/${MAX_FILES_PER_APPLICATION}. После перехода в рассмотрение файлы блокируются.`}</small>{upload.isPending ? <UploadProgress value={uploadProgress} label="Загрузка файла" /> : null}</div> : null}
      {fileError ? <Alert>{fileError}</Alert> : null}{upload.isError ? <Alert>{getErrorMessage(upload.error)}</Alert> : null}
    </Card></div>
    <Card className="detail-card"><h2>История статусов</h2><div className="timeline">{app.status_history.map((entry, index) => <div className="timeline-item" key={`${entry.created_at}-${index}`}><span className="timeline-dot" /><div><strong>{STATUS_LABELS[entry.status]}</strong><span className="timeline-meta"><CalendarDays size={14} /> {formatDateTime(entry.created_at)} · <CircleUserRound size={14} /> {entry.changed_by_role === 'admin' ? 'Администратор' : 'Пользователь'}</span>{entry.comment ? <p>{entry.comment}</p> : null}</div></div>)}</div></Card>
    {app.status === 'new' ? <div className="danger-zone"><div><strong>Отмена заявки</strong><p>Отменить заявку можно только до начала рассмотрения.</p></div><Button variant="danger" loading={cancel.isPending} onClick={() => { if (window.confirm('Отменить заявку? Это действие нельзя отменить.')) cancel.mutate(); }}>Отменить заявку</Button></div> : null}
    {cancel.isError ? <Alert>{getErrorMessage(cancel.error)}</Alert> : null}
  </section>;
}
