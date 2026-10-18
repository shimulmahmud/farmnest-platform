import { api } from './client';
import type {
  OrderResponse,
  CreateOrderRequest,
  UpdateOrderStatusRequest,
  ShipOrderRequest,
  PaginatedResponse,
} from '../types';

export async function createOrder(data: CreateOrderRequest): Promise<OrderResponse> {
  const res = await api.post<OrderResponse>('/orders', data);
  return res.data;
}

export async function listOrders(
  page = 1,
  perPage = 20,
): Promise<PaginatedResponse<OrderResponse>> {
  const res = await api.get<PaginatedResponse<OrderResponse>>('/orders', {
    params: { page, per_page: perPage },
  });
  return res.data;
}

export async function getOrder(id: string): Promise<OrderResponse> {
  const res = await api.get<OrderResponse>(`/orders/${id}`);
  return res.data;
}

export async function adminListOrders(
  page = 1,
  perPage = 20,
  status?: string,
): Promise<PaginatedResponse<OrderResponse>> {
  const params: Record<string, string | number | undefined> = {
    page,
    per_page: perPage,
  };
  if (status !== undefined) {
    params.status = status;
  }
  const res = await api.get<PaginatedResponse<OrderResponse>>(
    '/admin/orders',
    { params },
  );
  return res.data;
}

export async function adminUpdateOrderStatus(
  id: string,
  data: UpdateOrderStatusRequest,
): Promise<OrderResponse> {
  const res = await api.patch<OrderResponse>(
    `/admin/orders/${id}/status`,
    data,
  );
  return res.data;
}

/** PAID / CONFIRMED / PROCESSING -> SHIPPED, saving courier and tracking. */
export async function adminShipOrder(
  id: string,
  data: ShipOrderRequest,
): Promise<OrderResponse> {
  const res = await api.post<OrderResponse>(`/admin/orders/${id}/ship`, data);
  return res.data;
}

/** Cash-on-delivery: record that the cash was collected. */
export async function adminMarkOrderPaid(id: string): Promise<OrderResponse> {
  const res = await api.post<OrderResponse>(`/admin/orders/${id}/mark-paid`, {});
  return res.data;
}
