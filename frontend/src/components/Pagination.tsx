import { Button } from './ui/Button';
import type { PaginationMeta } from '../types/api';

export function Pagination({ meta, onPageChange }: { meta: PaginationMeta; onPageChange: (page: number) => void }) {
  if (meta.total_pages <= 1) return null;
  return (
    <nav className="pagination" aria-label="Пагинация">
      <Button variant="secondary" size="sm" disabled={meta.page <= 1} onClick={() => onPageChange(meta.page - 1)}>Назад</Button>
      <span>Страница {meta.page} из {meta.total_pages}</span>
      <Button variant="secondary" size="sm" disabled={meta.page >= meta.total_pages} onClick={() => onPageChange(meta.page + 1)}>Вперёд</Button>
    </nav>
  );
}
