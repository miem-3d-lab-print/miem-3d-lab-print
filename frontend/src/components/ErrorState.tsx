import { Button } from './ui/Button';

export function ErrorState({ message, onRetry }: { message: string; onRetry?: () => void }) {
  return (
    <div className="state-box state-box--error">
      <strong>Не удалось загрузить данные</strong>
      <span>{message}</span>
      {onRetry ? <Button variant="secondary" size="sm" onClick={onRetry}>Повторить</Button> : null}
    </div>
  );
}
