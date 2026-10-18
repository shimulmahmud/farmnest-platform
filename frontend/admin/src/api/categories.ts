import { api } from './client';
import type {
  CategoryResponse,
  CreateCategoryRequest,
  UpdateCategoryRequest,
  PaginatedResponse,
  MessageResponse,
} from '../types';

export async function listCategories(): Promise<
  PaginatedResponse<CategoryResponse>
> {
  const res = await api.get<PaginatedResponse<CategoryResponse>>('/categories');
  return res.data;
}

export async function getCategory(slug: string): Promise<CategoryResponse> {
  const res = await api.get<CategoryResponse>(`/categories/${slug}`);
  return res.data;
}

export async function createCategory(
  data: CreateCategoryRequest,
): Promise<CategoryResponse> {
  const res = await api.post<CategoryResponse>('/admin/categories', data);
  return res.data;
}

export async function updateCategory(
  id: string,
  data: UpdateCategoryRequest,
): Promise<CategoryResponse> {
  const res = await api.patch<CategoryResponse>(
    `/admin/categories/${id}`,
    data,
  );
  return res.data;
}

export async function deleteCategory(id: string): Promise<MessageResponse> {
  const res = await api.delete<MessageResponse>(`/admin/categories/${id}`);
  return res.data;
}
