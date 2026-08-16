export function formatDate(value?: string | null): string {
  if (!value) return '—';
  const dateOnly = /^(\d{4})-(\d{2})-(\d{2})$/.exec(value);
  const date = dateOnly
    ? new Date(Number(dateOnly[1]), Number(dateOnly[2]) - 1, Number(dateOnly[3]))
    : new Date(value);
  if (Number.isNaN(date.getTime())) return '—';
  return new Intl.DateTimeFormat('ru-RU', { day: '2-digit', month: 'long', year: 'numeric' }).format(date);
}

export function formatDateTime(value?: string | null): string {
  if (!value) return '—';
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return '—';
  return new Intl.DateTimeFormat('ru-RU', {
    day: '2-digit', month: 'long', year: 'numeric', hour: '2-digit', minute: '2-digit',
  }).format(date);
}

export function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} Б`;
  if (bytes < 1024 ** 2) return `${(bytes / 1024).toFixed(1)} КБ`;
  return `${(bytes / 1024 ** 2).toFixed(1)} МБ`;
}

export function toDateInputValue(date: Date): string {
  const offset = date.getTimezoneOffset();
  return new Date(date.getTime() - offset * 60_000).toISOString().slice(0, 10);
}

export function hoursToHuman(hours: number | null): string {
  if (hours === null) return '—';
  if (hours < 48) return `${Math.round(hours)} ч`;
  return `${(hours / 24).toFixed(1)} дн.`;
}

export function getMaterialColor(name: string): string {
  const known: Record<string, string> = {
    'красный': '#d94b3d', 'белый': '#ffffff', 'чёрный': '#232323', 'черный': '#232323',
    'синий': '#376ed8', 'серый': '#888888', 'зелёный': '#4d9b62', 'зеленый': '#4d9b62',
    'жёлтый': '#e5b82a', 'желтый': '#e5b82a', 'прозрачный': '#e7eef2',
  };
  return known[name.toLowerCase()] ?? '#c9c2b8';
}
