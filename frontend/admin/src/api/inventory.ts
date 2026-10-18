import { api } from './client';
import type {
  StockResponse,
  CreateStockRequest,
  AdjustStockRequest,
  ReserveRequest,
  ReserveResponse,
  ReservationResponse,
  MovementResponse,
  WarehouseResponse,
  CreateWarehouseRequest,
  PaginatedResponse,
} from '../types';

// ---------------------------------------------------------------------------
// Stock
// ---------------------------------------------------------------------------

export async function createStock(
  data: CreateStockRequest,
): Promise<StockResponse> {
  const res = await api.post<StockResponse>('/admin/inventory/stock', data);
  return res.data;
}

export async function getStock(
  sku: string,
  warehouseId: string,
): Promise<StockResponse> {
  const res = await api.get<StockResponse>(
    `/admin/inventory/stock/${sku}/${warehouseId}`,
  );
  return res.data;
}

export async function adjustStock(
  data: AdjustStockRequest,
): Promise<StockResponse> {
  const res = await api.post<StockResponse>(
    '/admin/inventory/stock/adjust',
    data,
  );
  return res.data;
}

// ---------------------------------------------------------------------------
// Reservations
// ---------------------------------------------------------------------------

export async function reserve(data: ReserveRequest): Promise<ReserveResponse> {
  const res = await api.post<ReserveResponse>(
    '/admin/inventory/reserve',
    data,
  );
  return res.data;
}

export async function release(
  reservationId: string,
): Promise<ReservationResponse> {
  const res = await api.post<ReservationResponse>(
    `/admin/inventory/release/${reservationId}`,
  );
  return res.data;
}

export async function consume(
  reservationId: string,
): Promise<ReservationResponse> {
  const res = await api.post<ReservationResponse>(
    `/admin/inventory/consume/${reservationId}`,
  );
  return res.data;
}

export async function getReservation(
  id: string,
): Promise<ReservationResponse> {
  const res = await api.get<ReservationResponse>(
    `/admin/inventory/reservations/${id}`,
  );
  return res.data;
}

export async function listReservations(
  sku?: string,
  warehouseId?: string,
  status?: string,
  page = 1,
  perPage = 20,
): Promise<PaginatedResponse<ReservationResponse>> {
  const params: Record<string, string | number | undefined> = {
    page,
    per_page: perPage,
  };
  if (sku !== undefined) params.sku = sku;
  if (warehouseId !== undefined) params.warehouse_id = warehouseId;
  if (status !== undefined) params.status = status;

  const res = await api.get<PaginatedResponse<ReservationResponse>>(
    '/admin/inventory/reservations',
    { params },
  );
  return res.data;
}

// ---------------------------------------------------------------------------
// Movements
// ---------------------------------------------------------------------------

export async function listMovements(
  sku?: string,
  warehouseId?: string,
  page = 1,
  perPage = 20,
): Promise<PaginatedResponse<MovementResponse>> {
  const params: Record<string, string | number | undefined> = {
    page,
    per_page: perPage,
  };
  if (sku !== undefined) params.sku = sku;
  if (warehouseId !== undefined) params.warehouse_id = warehouseId;

  const res = await api.get<PaginatedResponse<MovementResponse>>(
    '/admin/inventory/movements',
    { params },
  );
  return res.data;
}

// ---------------------------------------------------------------------------
// Low-stock alerts
// ---------------------------------------------------------------------------

export async function listLowStock(
  page = 1,
  perPage = 20,
): Promise<PaginatedResponse<StockResponse>> {
  const res = await api.get<PaginatedResponse<StockResponse>>(
    '/admin/inventory/low-stock',
    { params: { page, per_page: perPage } },
  );
  return res.data;
}

// ---------------------------------------------------------------------------
// Warehouses
// ---------------------------------------------------------------------------

export async function createWarehouse(
  data: CreateWarehouseRequest,
): Promise<WarehouseResponse> {
  const res = await api.post<WarehouseResponse>(
    '/admin/inventory/warehouses',
    data,
  );
  return res.data;
}

export async function listWarehouses(): Promise<
  PaginatedResponse<WarehouseResponse>
> {
  const res = await api.get<PaginatedResponse<WarehouseResponse>>(
    '/admin/inventory/warehouses',
  );
  return res.data;
}
