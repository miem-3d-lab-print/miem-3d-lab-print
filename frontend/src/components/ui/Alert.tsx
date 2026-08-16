import type { ReactNode } from 'react';
import clsx from 'clsx';

export function Alert({ children, tone = 'error' }: { children: ReactNode; tone?: 'error' | 'success' | 'info' | 'warning' }) {
  return <div className={clsx('alert', `alert--${tone}`)} role={tone === 'error' ? 'alert' : 'status'}>{children}</div>;
}
