import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Mail, Search, ShieldCheck, UserRound } from 'lucide-react';
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

function UserRow({ user }: { user: AdminUser }) {
  const client = useQueryClient();
  const refreshUsers = async () => {
    await Promise.all([
      client.invalidateQueries({ queryKey: ['admins'] }),
      client.invalidateQueries({ queryKey: ['admin-users'] }),
    ]);
  };
  const roleMutation = useMutation({
    mutationFn: (role: Role) => adminUsersApi.setRole(user.id, role),
    onSuccess: refreshUsers,
  });
  const notificationsMutation = useMutation({
    mutationFn: (enabled: boolean) => adminUsersApi.setApplicationNotifications(user.id, enabled),
    onSuccess: refreshUsers,
  });
  const error = roleMutation.error ?? notificationsMutation.error;

  return (
    <div className="user-role-row">
      <div className={user.role === 'admin' ? 'role-icon role-icon--admin' : 'role-icon'}>
        {user.role === 'admin' ? <ShieldCheck /> : <UserRound />}
      </div>
      <div className="user-role-row__identity">
        <strong>{user.full_name || 'Профиль не заполнен'}</strong>
        <span>{user.email} · зарегистрирован {formatDate(user.created_at)}</span>
      </div>
      <span className={user.role === 'admin' ? 'role-label role-label--admin' : 'role-label'}>
        {user.role === 'admin' ? 'Администратор' : 'Пользователь'}
      </span>
      {user.role === 'admin' ? (
        <Button
          className={user.application_notifications ? 'notification-button notification-button--active' : 'notification-button'}
          size="sm"
          variant="secondary"
          loading={notificationsMutation.isPending}
          disabled={roleMutation.isPending}
          aria-pressed={user.application_notifications}
          title="Email-уведомления о новых заявках"
          onClick={() => notificationsMutation.mutate(!user.application_notifications)}
        >
          <Mail size={15} />
          {user.application_notifications ? 'Уведомления включены' : 'Уведомления выключены'}
        </Button>
      ) : null}
      <Button
        size="sm"
        variant={user.role === 'admin' ? 'danger' : 'secondary'}
        loading={roleMutation.isPending}
        disabled={notificationsMutation.isPending}
        onClick={() => roleMutation.mutate(user.role === 'admin' ? 'user' : 'admin')}
      >
        {user.role === 'admin' ? 'Снять роль' : 'Назначить admin'}
      </Button>
      {error ? <Alert>{getErrorMessage(error)}</Alert> : null}
    </div>
  );
}

export function AdminUsersPage() {
  const [draft, setDraft] = useState('');
  const [search, setSearch] = useState('');
  const admins = useQuery({ queryKey: ['admins'], queryFn: adminUsersApi.listAdmins });
  const users = useQuery({
    queryKey: ['admin-users', search],
    queryFn: () => adminUsersApi.search(search),
    enabled: search.length >= 3,
  });
  const submit = (event: React.FormEvent) => {
    event.preventDefault();
    setSearch(draft.trim());
  };

  return (
    <div className="admin-section admin-users-section">
      <section>
        <div className="admin-subsection-heading">
          <div>
            <h2>Текущие администраторы</h2>
            <p className="muted">Включите почтовые уведомления тем, кто должен узнавать о каждой новой заявке.</p>
          </div>
          {admins.data ? <span className="admin-count">{admins.data.length}</span> : null}
        </div>
        {admins.isPending ? <LoadingState /> : admins.isError ? (
          <ErrorState message={getErrorMessage(admins.error)} onRetry={() => void admins.refetch()} />
        ) : admins.data.length === 0 ? (
          <EmptyState title="Администраторов нет" />
        ) : (
          <div className="user-role-list">{admins.data.map((user) => <UserRow user={user} key={user.id} />)}</div>
        )}
      </section>

      <section className="admin-users-search-section">
        <div className="admin-subsection-heading">
          <div>
            <h2>Назначение ролей</h2>
            <p className="muted">Назначить роль можно только пользователю, который уже хотя бы один раз входил в систему.</p>
          </div>
        </div>
        <form className="user-search" onSubmit={submit}>
          <label className="search-field">
            <Search size={18} />
            <input value={draft} onChange={(event) => setDraft(event.target.value)} placeholder="Введите минимум 3 символа email" />
          </label>
          <Button type="submit" disabled={draft.trim().length < 3}>Найти пользователя</Button>
        </form>
        {!search ? null : users.isPending ? <LoadingState /> : users.isError ? (
          <ErrorState message={getErrorMessage(users.error)} onRetry={() => void users.refetch()} />
        ) : users.data.length === 0 ? (
          <EmptyState title="Пользователь не найден" />
        ) : (
          <div className="user-role-list">{users.data.map((user) => <UserRow user={user} key={user.id} />)}</div>
        )}
      </section>
    </div>
  );
}
