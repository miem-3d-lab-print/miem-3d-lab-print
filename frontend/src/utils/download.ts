export function downloadBlob(blob: Blob, filename: string) {
  const url = URL.createObjectURL(blob);
  const anchor = document.createElement('a');
  anchor.href = url;
  anchor.download = filename;
  anchor.style.display = 'none';
  document.body.appendChild(anchor);
  anchor.click();
  anchor.remove();

  // Safari can start reading the Blob after click() returns, so revoking the
  // URL synchronously intermittently cancels the download.
  window.setTimeout(() => URL.revokeObjectURL(url), 1_000);
}
