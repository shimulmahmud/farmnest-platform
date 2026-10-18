import { api } from './client';
import type { ProfitReport, ProductProfit, OrderProfit, PaginatedResponse } from '../types';

export interface ReportRange {
  from?: string; // YYYY-MM-DD (Bangladesh time)
  to?: string;
  statuses?: string; // comma separated, default SHIPPED,DELIVERED
}

export async function getProfitReport(
  range: ReportRange,
  groupBy: 'day' | 'month' = 'day',
): Promise<ProfitReport> {
  const res = await api.get<ProfitReport>('/admin/reports/profit', {
    params: { ...range, group_by: groupBy },
  });
  return res.data;
}

export async function getProductProfit(
  range: ReportRange,
  sort: 'best' | 'worst' = 'best',
  limit = 20,
): Promise<ProductProfit[]> {
  const res = await api.get<ProductProfit[]>('/admin/reports/profit/products', {
    params: { ...range, sort, limit },
  });
  return res.data;
}

export async function getOrderProfit(
  range: ReportRange,
  page = 1,
  perPage = 20,
): Promise<PaginatedResponse<OrderProfit>> {
  const res = await api.get<PaginatedResponse<OrderProfit>>('/admin/reports/profit/orders', {
    params: { ...range, page, per_page: perPage },
  });
  return res.data;
}
