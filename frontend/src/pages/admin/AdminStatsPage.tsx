import { useQuery } from '@tanstack/react-query';
import { BarChart3, CheckCircle2, Clock3, Files } from 'lucide-react';
import { useMemo, useState } from 'react';
import { Bar, BarChart, CartesianGrid, ResponsiveContainer, Tooltip, XAxis, YAxis } from 'recharts';
import { adminStatsApi } from '../../api/endpoints';
import { ErrorState } from '../../components/ErrorState';
import { LoadingState } from '../../components/LoadingState';
import { Button } from '../../components/ui/Button';
import { Card } from '../../components/ui/Card';
import { STATUS_LABELS } from '../../utils/constants';
import { getErrorMessage } from '../../utils/errors';
import { hoursToHuman, toDateInputValue } from '../../utils/format';

function firstDayOfMonth(): string { const now = new Date(); return toDateInputValue(new Date(now.getFullYear(), now.getMonth(), 1)); }
function lastDayOfMonth(): string { const now = new Date(); return toDateInputValue(new Date(now.getFullYear(), now.getMonth() + 1, 0)); }

export function AdminStatsPage() {
  const [draftFrom, setDraftFrom] = useState(firstDayOfMonth());
  const [draftTo, setDraftTo] = useState(lastDayOfMonth());
  const [range, setRange] = useState({ from: draftFrom, to: draftTo });
  const query = useQuery({ queryKey: ['admin-stats', range], queryFn: () => adminStatsApi.get(range.from, range.to) });
  const statusRows = useMemo(() => query.data ? Object.entries(query.data.by_status_current).map(([status, count]) => ({ name: STATUS_LABELS[status as keyof typeof STATUS_LABELS], count })) : [], [query.data]);
  if (query.isPending) return <LoadingState />;
  if (query.isError) return <ErrorState message={getErrorMessage(query.error)} onRetry={() => void query.refetch()} />;
  const stats = query.data;
  return <div className="admin-section"><div className="stats-filter"><label>С <input type="date" value={draftFrom} onChange={(event) => setDraftFrom(event.target.value)} /></label><label>По <input type="date" value={draftTo} onChange={(event) => setDraftTo(event.target.value)} /></label><Button variant="secondary" onClick={() => setRange({ from: draftFrom, to: draftTo })}>Применить</Button></div>
    <div className="stats-cards"><Card><Files /><span>Заявок за период</span><strong>{stats.total_applications}</strong></Card><Card><CheckCircle2 /><span>Завершено</span><strong>{stats.completed_count}</strong></Card><Card><Clock3 /><span>Средний срок</span><strong>{hoursToHuman(stats.avg_completion_hours)}</strong></Card><Card><BarChart3 /><span>Материалов в топе</span><strong>{stats.by_material.length}</strong></Card></div>
    <div className="chart-grid"><Card className="chart-card"><h2>Заявки по материалам</h2><ResponsiveContainer width="100%" height={320}><BarChart data={stats.by_material}><CartesianGrid strokeDasharray="3 3" /><XAxis dataKey="material_name" /><YAxis allowDecimals={false} /><Tooltip /><Bar dataKey="count" fill="var(--accent)" radius={[4,4,0,0]} /></BarChart></ResponsiveContainer></Card><Card className="chart-card"><h2>Текущие статусы</h2><ResponsiveContainer width="100%" height={320}><BarChart data={statusRows} layout="vertical"><CartesianGrid strokeDasharray="3 3" /><XAxis type="number" allowDecimals={false} /><YAxis type="category" dataKey="name" width={110} /><Tooltip /><Bar dataKey="count" fill="var(--ink)" radius={[0,4,4,0]} /></BarChart></ResponsiveContainer></Card></div>
  </div>;
}
