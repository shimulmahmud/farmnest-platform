import { api } from './client';

export interface UploadedImage {
  url: string;
  width?: number;
  height?: number;
  size?: number;
  content_type?: string;
}

/** Biggest picture the server accepts (checked here first so people get a quick, clear message). */
export const MAX_IMAGE_MB = 8;

/** Uploads one picture to the shop and returns its address (for example /uploads/3f2a....jpg). */
export async function uploadImage(file: File): Promise<UploadedImage> {
  const form = new FormData();
  form.append('file', file);
  const res = await api.post<UploadedImage>('/admin/uploads/images', form, {
    headers: { 'Content-Type': 'multipart/form-data' },
  });
  return res.data;
}
