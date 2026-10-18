import { api } from './client';
import type {
  ProductResponse,
  CreateProductRequest,
  UpdateProductRequest,
  SearchProductsParams,
  PaginatedResponse,
  MessageResponse,
} from '../types';

export async function listProducts(
  page = 1,
  perPage = 20,
): Promise<PaginatedResponse<ProductResponse>> {
  const res = await api.get<PaginatedResponse<ProductResponse>>('/products', {
    params: { page, per_page: perPage },
  });
  return res.data;
}

export async function searchProducts(
  params: SearchProductsParams,
): Promise<PaginatedResponse<ProductResponse>> {
  const searchParams: Record<string, string | number | undefined> = {
    page: params.page,
    per_page: params.per_page,
  };

  if (params.q !== undefined) searchParams.q = params.q;
  if (params.category_id !== undefined) searchParams.category_id = params.category_id;
  if (params.brand !== undefined) searchParams.brand = params.brand;
  if (params.tags !== undefined) searchParams.tags = params.tags;
  if (params.min_price !== undefined) searchParams.min_price = params.min_price;
  if (params.max_price !== undefined) searchParams.max_price = params.max_price;
  if (params.status !== undefined) searchParams.status = params.status;
  if (params.sort_by !== undefined) searchParams.sort_by = params.sort_by;

  const res = await api.get<PaginatedResponse<ProductResponse>>(
    '/products/search',
    { params: searchParams },
  );
  return res.data;
}

export async function getProduct(slug: string): Promise<ProductResponse> {
  const res = await api.get<ProductResponse>(`/products/${slug}`);
  return res.data;
}

export async function createProduct(
  data: CreateProductRequest,
): Promise<ProductResponse> {
  const res = await api.post<ProductResponse>('/admin/products', data);
  return res.data;
}

export async function updateProduct(
  id: string,
  data: UpdateProductRequest,
): Promise<ProductResponse> {
  const res = await api.patch<ProductResponse>(`/admin/products/${id}`, data);
  return res.data;
}

export async function deleteProduct(id: string): Promise<MessageResponse> {
  const res = await api.delete<MessageResponse>(`/admin/products/${id}`);
  return res.data;
}

// ---------------------------------------------------------------------------
// Admin (full data: cost price, any status, stock)
// ---------------------------------------------------------------------------

export async function adminListProducts(
  page = 1,
  perPage = 20,
  filters: { status?: string; search?: string; category_id?: string } = {},
): Promise<PaginatedResponse<ProductResponse>> {
  const res = await api.get<PaginatedResponse<ProductResponse>>('/admin/products', {
    params: { page, per_page: perPage, ...filters },
  });
  return res.data;
}

export async function adminGetProduct(id: string): Promise<ProductResponse> {
  const res = await api.get<ProductResponse>(`/admin/products/${id}`);
  return res.data;
}
