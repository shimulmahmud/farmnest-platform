import { api } from './client';
import type {
  OverviewResponse,
  TopProductResponse,
  DailyCountResponse,
} from '../types';

export async function getOverview(
  startAt?: string,
  endAt?: string,
): Promise<OverviewResponse> {
  const params: Record<string, string | undefined> = {};
  if (startAt !== undefined) params.start_at = startAt;
  if (endAt !== undefined) params.end_at = endAt;

  const res = await api.get<OverviewResponse>('/analytics/overview', {
    params,
  });
  return res.data;
}

export async function getTopProducts(
  limit = 10,
  startAt?: string,
  endAt?: string,
): Promise<TopProductResponse[]> {
  const params: Record<string, string | number | undefined> = {
    limit,
  };
  if (startAt !== undefined) params.start_at = startAt;
  if (endAt !== undefined) params.end_at = endAt;

  const res = await api.get<TopProductResponse[]>('/analytics/top-products', {
    params,
  });
  return res.data;
}

export async function getDailyTrend(
  startAt: string,
  endAt: string,
  eventType = 'order_created',
): Promise<DailyCountResponse[]> {
  const params: Record<string, string | undefined> = {
    start_at: startAt,
    end_at: endAt,
  };

  const res = await api.get<DailyCountResponse[]>(
    `/analytics/trends/${eventType}`,
    { params },
  );
  return res.data;
}
