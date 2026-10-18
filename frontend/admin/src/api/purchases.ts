import { api } from './client';
import type { PurchaseResponse, ReceiveStockRequest, PaginatedResponse } from '../types';

export async function receiveStock(data: ReceiveStockRequest): Promise<PurchaseResponse> {
  const res = await api.post<PurchaseResponse>('/admin/purchases', data);
  return res.data;
}

export async function listPurchases(
  page = 1,
  perPage = 20,
): Promise<PaginatedResponse<PurchaseResponse>> {
  const res = await api.get<PaginatedResponse<PurchaseResponse>>('/admin/purchases', {
    params: { page, per_page: perPage },
  });
  return res.data;
}
