import { Link } from 'react-router-dom';
import { Button } from '../components/ui/Button';

export function NotFoundPage() {
  return <section className="page page--narrow"><div className="empty-state"><h1>404</h1><strong>Страница не найдена</strong><Link to="/"><Button>На главную</Button></Link></div></section>;
}
