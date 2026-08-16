import { useQuery } from '@tanstack/react-query';
import { FilePlus2, Files } from 'lucide-react';
import { useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { applicationsApi } from '../../api/endpoints';
import { EmptyState } from '../../components/EmptyState';
import { ErrorState } from '../../components/ErrorState';
import { LoadingState } from '../../components/LoadingState';
import { Pagination } from '../../components/Pagination';
import { StatusBadge } from '../../components/StatusBadge';
import { Button } from '../../components/ui/Button';
import type { ApplicationStatus } from '../../types/api';
import { STATUS_LABELS } from '../../utils/constants';
import { getErrorMessage } from '../../utils/errors';
import { formatDate } from '../../utils/format';

const filters: Array<{ value: '' | ApplicationStatus; label: string }> = [
  { value: '', label: 'Все' },
  ...Object.entries(STATUS_LABELS).map(([value, label]) => ({ value: value as ApplicationStatus, label })),
];

export function ApplicationsPage() {
  const navigate = useNavigate();
  const [status, setStatus] = useState<'' | ApplicationStatus>('');
  const [page, setPage] = useState(1);
  const query = useQuery({
    queryKey: ['applications', status, page],
    queryFn: () => applicationsApi.list({ status: status || undefined, page, per_page: 20 }),
  });
  const chooseStatus = (value: '' | ApplicationStatus) => { setStatus(value); setPage(1); };

  return <section className="page"><div className="page-heading page-heading--actions"><div><span className="eyebrow">ЛИЧНЫЙ КАБИНЕТ</span><h1>Мои заявки</h1><p>Все заявки, файлы и история обработки в одном месте.</p></div><Button onClick={() => navigate('/app/applications/new')}><FilePlus2 size={18} /> Новая заявка</Button></div>
    <div className="filter-chips">{filters.map((item) => <button key={item.value || 'all'} className={status === item.value ? 'chip chip--active' : 'chip'} onClick={() => chooseStatus(item.value)}>{item.label}</button>)}</div>
    {query.isPending ? <LoadingState /> : query.isError ? <ErrorState message={getErrorMessage(query.error)} onRetry={() => void query.refetch()} /> : query.data.items.length === 0 ? <EmptyState title="Заявок нет" description={status ? 'В выбранном статусе заявок пока нет.' : 'Создайте первую заявку на 3D-печать.'} action={!status ? <Button onClick={() => navigate('/app/applications/new')}>Оформить заявку</Button> : undefined} /> : <>
      <div className="application-list">{query.data.items.map((app) => <Link to={`/app/applications/${app.id}`} className="application-card" key={app.id}>
        <div className="application-card__number"><Files size={18} /><strong>№ {app.number}</strong><span>{formatDate(app.created_at)}</span></div>
        <div className="application-card__body"><strong>{app.material_name}{app.color_name ? ` · ${app.color_name}` : ' · цвет не важен'}</strong><span>Желаемая дата: {formatDate(app.desired_date)} · файлов: {app.files_count}</span></div>
        <StatusBadge status={app.status} />
      </Link>)}</div>
      <Pagination meta={query.data.meta} onPageChange={setPage} />
    </>}
  </section>;
}
