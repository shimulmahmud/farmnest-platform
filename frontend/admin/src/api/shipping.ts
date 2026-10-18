import { api } from './client';
import type {
  ShippingRateResponse,
  ShipmentResponse,
  CreateShipmentRequest,
  PaginatedResponse,
  MessageResponse,
  Address,
} from '../types';

// ---------------------------------------------------------------------------
// Customer-facing
// ---------------------------------------------------------------------------

export async function calculateRate(
  destination: Address,
  items: { sku: string; quantity: number }[],
): Promise<ShippingRateResponse[]> {
  const res = await api.post<ShippingRateResponse[]>(
    '/shipping/rates',
    { destination, items },
  );
  return res.data;
}

export async function getShipment(id: string): Promise<ShipmentResponse> {
  const res = await api.get<ShipmentResponse>(`/shipments/${id}`);
  return res.data;
}

export async function getShipmentByOrder(
  orderId: string,
): Promise<ShipmentResponse> {
  const res = await api.get<ShipmentResponse>(
    `/shipping/order/${orderId}`,
  );
  return res.data;
}

export async function listShipments(
  page = 1,
  perPage = 20,
): Promise<PaginatedResponse<ShipmentResponse>> {
  const res = await api.get<PaginatedResponse<ShipmentResponse>>(
    '/shipments',
    { params: { page, per_page: perPage } },
  );
  return res.data;
}

export async function trackShipment(
  id: string,
): Promise<ShipmentResponse> {
  const res = await api.post<ShipmentResponse>(`/shipping/${id}/track`);
  return res.data;
}

export async function cancelShipment(
  id: string,
): Promise<MessageResponse> {
  const res = await api.post<MessageResponse>(
    `/shipments/${id}/cancel`,
  );
  return res.data;
}

// ---------------------------------------------------------------------------
// Admin-facing
// ---------------------------------------------------------------------------

export async function adminCreateShipment(
  data: CreateShipmentRequest,
): Promise<ShipmentResponse> {
  const res = await api.post<ShipmentResponse>(
    '/admin/shipments',
    data,
  );
  return res.data;
}

export async function adminListShipments(
  page = 1,
  perPage = 20,
  status?: string,
): Promise<PaginatedResponse<ShipmentResponse>> {
  const params: Record<string, string | number | undefined> = {
    page,
    per_page: perPage,
  };
  if (status !== undefined) {
    params.status = status;
  }
  const res = await api.get<PaginatedResponse<ShipmentResponse>>(
    '/admin/shipments',
    { params },
  );
  return res.data;
}

export async function adminUpdateShipmentStatus(
  id: string,
  data: { status: string },
): Promise<ShipmentResponse> {
  const res = await api.patch<ShipmentResponse>(
    `/admin/shipments/${id}/status`,
    data,
  );
  return res.data;
}
