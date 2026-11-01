import { useRef, useState } from 'react';
import { ArrowUpTrayIcon } from '@heroicons/react/24/outline';
import { MAX_IMAGE_MB, uploadImage } from '../../api/uploads';

interface Props {
  /** Called with the addresses of the pictures that were uploaded successfully. */
  onUploaded: (urls: string[]) => void;
  label?: string;
}

const ACCEPTED = 'image/jpeg,image/png,image/webp,image/gif';

/** "Upload picture" button: pick one or more files from your computer. */
export default function ImageUploadButton({ onUploaded, label = 'Upload picture' }: Props) {
  const input = useRef<HTMLInputElement>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');

  const handle = async (files: FileList | null) => {
    if (!files || files.length === 0) return;
    setError('');
    setBusy(true);
    const urls: string[] = [];
    const problems: string[] = [];
    for (const file of Array.from(files)) {
      if (!ACCEPTED.split(',').includes(file.type)) {
        problems.push(`${file.name}: only JPG, PNG, WebP and GIF pictures are allowed`);
        continue;
      }
      if (file.size > MAX_IMAGE_MB * 1024 * 1024) {
        problems.push(`${file.name}: larger than ${MAX_IMAGE_MB} MB`);
        continue;
      }
      try {
        urls.push((await uploadImage(file)).url);
      } catch (e) {
        problems.push(`${file.name}: ${e instanceof Error ? e.message : 'upload failed'}`);
      }
    }
    setBusy(false);
    if (input.current) input.current.value = ''; // allow choosing the same file again
    if (urls.length > 0) onUploaded(urls);
    if (problems.length > 0) setError(problems.join(' · '));
  };

  return (
    <span className="inline-flex flex-col items-start">
      <button
        type="button"
        disabled={busy}
        onClick={() => input.current?.click()}
        className="inline-flex items-center gap-1 rounded px-2 py-1 text-xs font-medium text-primary-600 hover:bg-primary-50 disabled:opacity-60"
      >
        <ArrowUpTrayIcon className="h-3 w-3" /> {busy ? 'Uploading…' : label}
      </button>
      <input ref={input} type="file" accept={ACCEPTED} multiple className="hidden" onChange={(e) => handle(e.target.files)} />
      {error && <span className="mt-1 max-w-xs text-[11px] text-red-600">{error}</span>}
    </span>
  );
}
