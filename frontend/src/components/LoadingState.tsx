export function LoadingState({ label = 'Загрузка…' }: { label?: string }) {
  return <div className="state-box"><span className="page-spinner" aria-hidden="true" /><span>{label}</span></div>;
}
