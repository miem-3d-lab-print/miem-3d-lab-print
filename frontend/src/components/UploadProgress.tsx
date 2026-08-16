interface UploadProgressProps {
  value: number;
  label?: string;
}

export function UploadProgress({ value, label = 'Загрузка файлов' }: UploadProgressProps) {
  const progress = Math.max(0, Math.min(100, Math.round(value)));
  return (
    <div className="upload-progress" role="progressbar" aria-label={label} aria-valuemin={0} aria-valuemax={100} aria-valuenow={progress}>
      <div className="upload-progress__label">
        <span>{progress === 100 ? `${label}: обработка на сервере…` : label}</span>
        <strong>{progress}%</strong>
      </div>
      <div className="upload-progress__track"><span style={{ width: `${progress}%` }} /></div>
    </div>
  );
}
