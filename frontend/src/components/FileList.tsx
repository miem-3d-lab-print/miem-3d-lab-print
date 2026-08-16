import { Download, FileBox } from 'lucide-react';
import type { FileMeta } from '../types/api';
import { formatBytes } from '../utils/format';
import { Button } from './ui/Button';

export function FileList({ files, onDownload, downloadingId }: {
  files: FileMeta[];
  onDownload: (file: FileMeta) => void;
  downloadingId?: string | null;
}) {
  return (
    <div className="file-list">
      {files.map((file) => (
        <div className="file-row" key={file.id}>
          <FileBox size={20} />
          <div className="file-row__body">
            <strong>{file.filename}</strong>
            <span>{file.format} · {formatBytes(file.size)}</span>
          </div>
          <Button variant="ghost" size="sm" loading={downloadingId === file.id} onClick={() => onDownload(file)}>
            <Download size={16} /> Скачать
          </Button>
        </div>
      ))}
    </div>
  );
}
