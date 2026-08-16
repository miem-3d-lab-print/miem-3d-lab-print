import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { ArrowLeft, ArrowRight, Check, FileUp, Trash2 } from 'lucide-react';
import { useMemo, useRef, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { applicationsApi, materialsApi } from '../../api/endpoints';
import { ErrorState } from '../../components/ErrorState';
import { LoadingState } from '../../components/LoadingState';
import { Alert } from '../../components/ui/Alert';
import { Button } from '../../components/ui/Button';
import { Card } from '../../components/ui/Card';
import type { ApplicationPosition } from '../../types/api';
import { ALLOWED_FILE_EXTENSIONS, MAX_FILE_SIZE, MAX_FILES_PER_APPLICATION, POSITION_LABELS } from '../../utils/constants';
import { normalizeApiError } from '../../utils/errors';
import { formatBytes, getMaterialColor, toDateInputValue } from '../../utils/format';

interface Draft {
  title: string;
  position: ApplicationPosition | '';
  purpose: string;
  materialId: string;
  colorMatters: boolean;
  colorId: string;
  desiredDate: string;
  comment: string;
  fileUrl: string;
  files: DraftFile[];
}

interface DraftFile {
  localId: string;
  file: File;
  pendingId?: string;
  progress: number;
  status: 'uploading' | 'uploaded' | 'error';
  error?: string;
}

function isValidFileURL(value: string): boolean {
  if (!value.trim()) return true;
  try {
    const parsed = new URL(value);
    return (parsed.protocol === 'http:' || parsed.protocol === 'https:') && Boolean(parsed.host);
  } catch {
    return false;
  }
}

function validateFiles(files: File[]): string | null {
  if (files.length > MAX_FILES_PER_APPLICATION) return `На одну заявку можно прикрепить не более ${MAX_FILES_PER_APPLICATION} файлов.`;
  for (const file of files) {
    if (file.size <= 0) return `${file.name}: файл пустой.`;
    const extension = file.name.split('.').pop()?.toLowerCase() || '';
    if (!ALLOWED_FILE_EXTENSIONS.includes(extension)) return `${file.name}: разрешены только STL, STEP, STP, 3MF и ZIP.`;
    if (file.size > MAX_FILE_SIZE) return `${file.name}: размер превышает 20 МБ.`;
  }
  return null;
}

export function NewApplicationPage() {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const fileInput = useRef<HTMLInputElement>(null);
  const [step, setStep] = useState(1);
  const [error, setError] = useState('');
  const [draft, setDraft] = useState<Draft>({ title: '', position: '', purpose: '', materialId: '', colorMatters: true, colorId: '', desiredDate: '', comment: '', fileUrl: '', files: [] });
  const materials = useQuery({ queryKey: ['materials'], queryFn: materialsApi.list });
  const selectedMaterial = materials.data?.find((material) => material.id === draft.materialId);
  const minDate = useMemo(() => toDateInputValue(new Date()), []);
  const create = useMutation({
    mutationFn: () => applicationsApi.create({ title: draft.title.trim(), position: draft.position, purpose: draft.purpose.trim(), material_id: draft.materialId, color_matters: draft.colorMatters, color_id: draft.colorId || undefined, desired_date: draft.desiredDate, comment: draft.comment.trim() || undefined, file_url: draft.fileUrl.trim() || undefined, pending_file_ids: draft.files.flatMap((item) => item.pendingId ? [item.pendingId] : []) }),
    onSuccess: async (created) => { await queryClient.invalidateQueries({ queryKey: ['applications'] }); navigate(`/app/applications/${created.id}`, { replace: true }); },
    onError: (rawError) => {
      const apiError = normalizeApiError(rawError);
      if (apiError.code === 'PROFILE_INCOMPLETE') navigate('/profile', { state: { profileRequired: true, afterAuth: '/app/applications/new' } });
    },
  });
  const filesUploaded = draft.files.length > 0 && draft.files.every((item) => item.status === 'uploaded' && item.pendingId);
  const stepValid = step === 1 ? Boolean(draft.title.trim() && draft.position && draft.purpose.trim()) : step === 2 ? Boolean(draft.materialId && (!draft.colorMatters || draft.colorId)) : Boolean(draft.desiredDate && (filesUploaded || draft.fileUrl.trim()) && isValidFileURL(draft.fileUrl));
  const uploadDraftFile = async (entry: DraftFile) => {
    try {
      const uploaded = await applicationsApi.uploadPendingFile(entry.file, (progress) => {
        setDraft((current) => ({ ...current, files: current.files.map((item) => item.localId === entry.localId ? { ...item, progress } : item) }));
      });
      setDraft((current) => ({ ...current, files: current.files.map((item) => item.localId === entry.localId ? { ...item, pendingId: uploaded.id, progress: 100, status: 'uploaded', error: undefined } : item) }));
    } catch (rawError) {
      const message = normalizeApiError(rawError).message;
      setDraft((current) => ({ ...current, files: current.files.map((item) => item.localId === entry.localId ? { ...item, status: 'error', error: message } : item) }));
    }
  };
  const addFiles = (incoming: FileList | null) => {
    if (!incoming) return;
    const incomingFiles = Array.from(incoming);
    const validation = validateFiles([...draft.files.map((item) => item.file), ...incomingFiles]);
    if (validation) { setError(validation); return; }
    const entries = incomingFiles.map((file, index): DraftFile => ({ localId: `${Date.now()}-${index}-${file.name}`, file, progress: 0, status: 'uploading' }));
    setError('');
    setDraft((current) => ({ ...current, files: [...current.files, ...entries] }));
    entries.forEach((entry) => void uploadDraftFile(entry));
    if (fileInput.current) fileInput.current.value = '';
  };
  const removeFile = (entry: DraftFile) => {
    if (entry.status === 'uploading') return;
    setDraft((current) => ({ ...current, files: current.files.filter((item) => item.localId !== entry.localId) }));
    if (entry.pendingId) void applicationsApi.deletePendingFile(entry.pendingId).catch((rawError) => setError(normalizeApiError(rawError).message));
  };

  if (materials.isPending) return <LoadingState label="Загружаем каталог…" />;
  if (materials.isError) return <section className="page page--narrow"><ErrorState message="Не удалось получить актуальный каталог материалов." onRetry={() => void materials.refetch()} /></section>;
  const apiError = create.isError ? normalizeApiError(create.error) : null;

  return <section className="page page--wizard"><button className="back-link back-link--button" onClick={() => navigate(-1)}><ArrowLeft size={16} /> Назад</button>
    <div className="page-heading"><div><span className="eyebrow">НОВАЯ ЗАЯВКА</span><h1>Заявка на 3D-печать</h1><p>Заполните три шага. Каждый файл загружается сразу после выбора.</p></div></div>
    <div className="wizard-steps">{['Данные', 'Материал', 'Файлы и дата'].map((label, index) => { const number = index + 1; return <div className={number === step ? 'wizard-step wizard-step--active' : number < step ? 'wizard-step wizard-step--done' : 'wizard-step'} key={label}><span>{number < step ? <Check size={16} /> : number}</span><strong>{label}</strong></div>; })}</div>
    <Card className="wizard-card">
      {step === 1 ? <div className="form-stack"><h2>Что и для чего печатаем</h2><label className="field"><span>Название заявки</span><input maxLength={255} value={draft.title} onChange={(e) => setDraft({ ...draft, title: e.target.value })} placeholder="Например: корпус датчика" /></label><label className="field"><span>Категория заявителя</span><select value={draft.position} onChange={(e) => setDraft({ ...draft, position: e.target.value as ApplicationPosition })}><option value="">Выберите категорию</option>{Object.entries(POSITION_LABELS).map(([value, label]) => <option key={value} value={value}>{label}</option>)}</select></label><label className="field"><span>Цель печати</span><textarea rows={5} value={draft.purpose} onChange={(e) => setDraft({ ...draft, purpose: e.target.value })} placeholder="Например: корпус датчика для выпускной квалификационной работы" /></label></div> : null}
      {step === 2 ? <div className="form-stack"><h2>Материал и цвет</h2><div className="selection-grid">{materials.data.map((material) => <button type="button" className={draft.materialId === material.id ? 'selection-card selection-card--active' : 'selection-card'} key={material.id} onClick={() => setDraft({ ...draft, materialId: material.id, colorId: '' })}><strong>{material.name}</strong><span>{material.description || 'Без описания'}</span></button>)}</div>
        {selectedMaterial ? <><div className="toggle-row"><span>Важен ли цвет?</span><div className="segmented"><button className={draft.colorMatters ? 'active' : ''} onClick={() => setDraft({ ...draft, colorMatters: true })}>Да</button><button className={!draft.colorMatters ? 'active' : ''} onClick={() => setDraft({ ...draft, colorMatters: false, colorId: '' })}>Нет</button></div></div>{draft.colorMatters ? <div className="color-options">{selectedMaterial.colors.map((color) => <button type="button" className={draft.colorId === color.id ? 'color-option color-option--active' : 'color-option'} key={color.id} onClick={() => setDraft({ ...draft, colorId: color.id })}><span style={{ background: getMaterialColor(color.name) }} />{color.name}</button>)}</div> : <Alert tone="info">Лаборатория выберет любой доступный цвет.</Alert>}</> : null}
      </div> : null}
      {step === 3 ? <div className="form-stack"><h2>Файлы и желаемая дата</h2><label className="upload-zone"><FileUp size={30} /><strong>Добавьте модели</strong><span>STL, STEP, STP, 3MF или ZIP, до 20 МБ каждый · максимум 10 файлов. Вместо загрузки можно указать ссылку ниже.</span><input ref={fileInput} type="file" multiple accept=".stl,.step,.stp,.3mf,.zip" disabled={create.isPending} onChange={(event) => addFiles(event.target.files)} /><Button type="button" variant="secondary" disabled={create.isPending} onClick={() => fileInput.current?.click()}>Выбрать файлы</Button></label>
        {draft.files.length ? <div className="selected-files">{draft.files.map((entry) => <div key={entry.localId}><span><strong>{entry.file.name}</strong><small>{formatBytes(entry.file.size)} · {entry.status === 'uploading' ? `загрузка ${entry.progress}%` : entry.status === 'uploaded' ? 'загружен' : entry.error || 'ошибка загрузки'}</small></span>{entry.status === 'error' ? <button type="button" onClick={() => { const retry = { ...entry, status: 'uploading' as const, progress: 0, error: undefined }; setDraft((current) => ({ ...current, files: current.files.map((item) => item.localId === entry.localId ? retry : item) })); void uploadDraftFile(retry); }}>Повторить</button> : <button type="button" disabled={entry.status === 'uploading'} aria-label="Удалить файл" onClick={() => removeFile(entry)}><Trash2 size={17} /></button>}</div>)}</div> : null}
        <label className="field"><span>Ссылка на файл</span><input type="url" maxLength={2048} value={draft.fileUrl} onChange={(e) => setDraft({ ...draft, fileUrl: e.target.value })} placeholder="https://disk.example.ru/model.zip" /><small>{draft.fileUrl && !isValidFileURL(draft.fileUrl) ? 'Укажите корректную HTTP(S)-ссылку.' : 'Необязательно, если модели загружены выше.'}</small></label>
        <div className="two-columns"><label className="field"><span>Желаемая дата</span><input type="date" min={minDate} value={draft.desiredDate} onChange={(e) => setDraft({ ...draft, desiredDate: e.target.value })} /></label><label className="field"><span>Комментарий</span><input value={draft.comment} onChange={(e) => setDraft({ ...draft, comment: e.target.value })} placeholder="Необязательно" /></label></div>
      </div> : null}
      {error ? <Alert>{error}</Alert> : null}{apiError ? <Alert>{apiError.message}</Alert> : null}
      <div className="wizard-actions">{step > 1 ? <Button variant="secondary" onClick={() => setStep(step - 1)}><ArrowLeft size={17} /> Назад</Button> : <span />}{step < 3 ? <Button disabled={!stepValid} onClick={() => setStep(step + 1)}>Продолжить <ArrowRight size={17} /></Button> : <Button disabled={!stepValid} loading={create.isPending} onClick={() => create.mutate()}>Отправить заявку</Button>}</div>
    </Card>
  </section>;
}
