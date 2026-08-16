import { useEffect, useState } from 'react';
import { Link, NavLink, useLocation, useNavigate } from 'react-router-dom';
import { LogOut, Menu, Plus, UserRound, X } from 'lucide-react';
import { useAuth } from '../../auth/AuthContext';
import { Button } from '../ui/Button';
import logo from '../../assets/miem-logo.webp';

export function AppHeader() {
  const { profile, isAuthenticated, logout } = useAuth();
  const navigate = useNavigate();
  const location = useLocation();
  const [menuOpen, setMenuOpen] = useState(false);

  useEffect(() => {
    setMenuOpen(false);
  }, [location.pathname, location.search, location.hash]);

  const handleLogout = async () => {
    await logout();
    setMenuOpen(false);
    navigate('/');
  };

  return (
    <header className="site-header">
      <Link className="brand" to="/" onClick={() => setMenuOpen(false)}>
        <img src={logo} alt="МИЭМ" />
        <span><strong>3D-печать МИЭМ</strong><small>Лаборатория 3D-визуализации и компьютерной графики</small></span>
      </Link>
      <button className="mobile-menu-button" aria-label="Открыть меню" onClick={() => setMenuOpen((v) => !v)}>
        {menuOpen ? <X /> : <Menu />}
      </button>
      <div className={`header-actions ${menuOpen ? 'header-actions--open' : ''}`}>
        <nav className="main-nav">
          <NavLink to="/" onClick={() => setMenuOpen(false)}>Главная</NavLink>
          {isAuthenticated ? <NavLink to="/app/applications" onClick={() => setMenuOpen(false)}>Мои заявки</NavLink> : null}
          {profile?.role === 'admin' ? <NavLink to="/admin/applications" onClick={() => setMenuOpen(false)}>Админ-панель</NavLink> : null}
        </nav>
        {isAuthenticated ? (
          <div className="header-user-actions">
            <Button size="sm" onClick={() => { setMenuOpen(false); navigate('/app/applications/new'); }}><Plus size={17} /> Оформить заявку</Button>
            <Link className="avatar-button" to="/profile" title="Мой профиль" onClick={() => setMenuOpen(false)}>
              {(profile?.full_name || profile?.email || 'U').trim().charAt(0).toUpperCase()}
            </Link>
            <Button size="sm" variant="ghost" onClick={handleLogout}><LogOut size={17} /> Выйти</Button>
          </div>
        ) : (
          <Button variant="dark" size="sm" onClick={() => { setMenuOpen(false); navigate('/auth/login'); }}><UserRound size={17} /> Войти</Button>
        )}
      </div>
    </header>
  );
}
