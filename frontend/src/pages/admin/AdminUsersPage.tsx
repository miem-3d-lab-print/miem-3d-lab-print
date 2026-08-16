import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Search, ShieldCheck, UserRound } from 'lucide-react';
import { useState } from 'react';
import { adminUsersApi } from '../../api/endpoints';
import { EmptyState } from '../../components/EmptyState';
import { ErrorState } from '../../components/ErrorState';
import { LoadingState } from '../../components/LoadingState';
import { Alert } from '../../components/ui/Alert';
import { Button } from '../../components/ui/Button';
import type { AdminUser, Role } from '../../types/api';
import { getErrorMessage } from '../../utils/errors';
import { formatDate } from '../../utils/format';

function UserRow({ user, queryKey }: { user: AdminUser; queryKey: string[] }) {
  const client = useQueryClient();
  const mutation = useMutation({ mutationFn: (role: Role) => adminUsersApi.setRole(user.id, role), onSuccess: () => client.invalidateQueries({ queryKey }) });
  return <div className="user-role-row"><div className={user.role === 'admin' ? 'role-icon role-icon--admin' : 'role-icon'}>{user.role === 'admin' ? <ShieldCheck /> : <UserRound />}</div><div><strong>{user.full_name || 'Профиль не заполнен'}</strong><span>{user.email} · зарегистрирован {formatDate(user.created_at)}</span></div><span className={user.role === 'admin' ? 'role-label role-label--admin' : 'role-label'}>{user.role === 'admin' ? 'Администратор' : 'Пользователь'}</span><Button size="sm" variant={user.role === 'admin' ? 'danger' : 'secondary'} loading={mutation.isPending} onClick={() => mutation.mutate(user.role === 'admin' ? 'user' : 'admin')}>{user.role === 'admin' ? 'Снять роль' : 'Назначить admin'}</Button>{mutation.isError ? <Alert>{getErrorMessage(mutation.error)}</Alert> : null}</div>;
}

export function AdminUsersPage() {
  const [draft, setDraft] = useState('');
  const [search, setSearch] = useState('');
  const queryKey = ['admin-users', search];
  const query = useQuery({ queryKey, queryFn: () => adminUsersApi.search(search), enabled: search.length >= 3 });
  const submit = (event: React.FormEvent) => { event.preventDefault(); setSearch(draft.trim()); };
  return <div className="admin-section"><form className="user-search" onSubmit={submit}><label className="search-field"><Search size={18} /><input value={draft} onChange={(event) => setDraft(event.target.value)} placeholder="Введите минимум 3 символа email" /></label><Button type="submit" disabled={draft.trim().length < 3}>Найти пользователя</Button></form><p className="muted">Назначить роль можно только пользователю, который уже хотя бы один раз входил в систему.</p>
    {!search ? <EmptyState title="Начните поиск" description="Введите часть корпоративного email." /> : query.isPending ? <LoadingState /> : query.isError ? <ErrorState message={getErrorMessage(query.error)} onRetry={() => void query.refetch()} /> : query.data.length === 0 ? <EmptyState title="Пользователь не найден" /> : <div className="user-role-list">{query.data.map((user) => <UserRow user={user} queryKey={queryKey} key={user.id} />)}</div>}
  </div>;
}
