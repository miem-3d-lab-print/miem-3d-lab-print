import { NavLink, Outlet } from 'react-router-dom';

export function AdminLayoutPage() {
  return <section className="page"><div className="page-heading"><div><span className="eyebrow">УПРАВЛЕНИЕ</span><h1>Админ-панель</h1><p>Заявки, каталог материалов, статистика и роли пользователей.</p></div></div>
    <nav className="admin-tabs"><NavLink to="/admin/applications">Заявки</NavLink><NavLink to="/admin/materials">Материалы</NavLink><NavLink to="/admin/stats">Статистика</NavLink><NavLink to="/admin/users">Администраторы</NavLink></nav>
    <Outlet />
  </section>;
}
