import { useQuery } from '@tanstack/react-query';
import { AlertTriangle, Search } from 'lucide-react';
import { useState } from 'react';
import { Link } from 'react-router-dom';
import { adminApplicationsApi, adminMaterialsApi } from '../../api/endpoints';
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

export function AdminApplicationsPage() {
  const [page, setPage] = useState(1);
  const [searchDraft, setSearchDraft] = useState('');
  const [search, setSearch] = useState('');
  const [materialId, setMaterialId] = useState('');
  const [statuses, setStatuses] = useState<ApplicationStatus[]>([]);
  const [createdFrom, setCreatedFrom] = useState('');
  const [createdTo, setCreatedTo] = useState('');
  const materials = useQuery({ queryKey: ['admin-materials'], queryFn: adminMaterialsApi.list });
  const query = useQuery({
    queryKey: ['admin-applications', page, search, materialId, statuses, createdFrom, createdTo],
    queryFn: () => adminApplicationsApi.list({ page, per_page: 20, search: search || undefined, material_id: materialId || undefined, status: statuses.length ? statuses : undefined, created_from: createdFrom || undefined, created_to: createdTo || undefined }),
  });
  const toggleStatus = (status: ApplicationStatus) => { setStatuses((current) => current.includes(status) ? current.filter((item) => item !== status) : [...current, status]); setPage(1); };
  const submitSearch = (event: React.FormEvent) => { event.preventDefault(); setSearch(searchDraft.trim()); setPage(1); };

  return <div className="admin-section"><form className="admin-filters" onSubmit={submitSearch}><label className="search-field"><Search size={17} /><input value={searchDraft} onChange={(event) => setSearchDraft(event.target.value)} placeholder="ФИО или точный номер заявки" /></label><select value={materialId} onChange={(event) => { setMaterialId(event.target.value); setPage(1); }}><option value="">Все материалы</option>{materials.data?.map((material) => <option key={material.id} value={material.id}>{material.name}</option>)}</select><input type="date" value={createdFrom} onChange={(event) => { setCreatedFrom(event.target.value); setPage(1); }} aria-label="Дата подачи от" /><input type="date" value={createdTo} onChange={(event) => { setCreatedTo(event.target.value); setPage(1); }} aria-label="Дата подачи до" /><Button type="submit" variant="secondary">Найти</Button></form>
    <div className="filter-chips filter-chips--compact">{Object.entries(STATUS_LABELS).map(([status, label]) => <button key={status} className={statuses.includes(status as ApplicationStatus) ? 'chip chip--active' : 'chip'} onClick={() => toggleStatus(status as ApplicationStatus)}>{label}</button>)}</div>
    {query.isPending ? <LoadingState /> : query.isError ? <ErrorState message={getErrorMessage(query.error)} onRetry={() => void query.refetch()} /> : query.data.items.length === 0 ? <EmptyState title="Заявки не найдены" description="Измените фильтры или поисковый запрос." /> : <><div className="table-wrap"><table className="data-table"><thead><tr><th>Номер и название</th><th>Заявитель</th><th>Материал</th><th>Подана</th><th>Желаемая дата</th><th>Статус</th></tr></thead><tbody>{query.data.items.map((app) => <tr key={app.id}><td><Link className="table-link" to={`/admin/applications/${app.id}`}>№ {app.number} · {app.title}</Link>{app.deadline_soon ? <span className="deadline-flag"><AlertTriangle size={13} /> Горит дедлайн</span> : null}</td><td>{app.full_name}</td><td>{app.material_name}</td><td>{formatDate(app.created_at)}</td><td>{formatDate(app.desired_date)}</td><td><StatusBadge status={app.status} /></td></tr>)}</tbody></table></div><Pagination meta={query.data.meta} onPageChange={setPage} /></>}
  </div>;
}
