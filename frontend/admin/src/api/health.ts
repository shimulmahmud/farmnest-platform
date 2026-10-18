import { rootApi } from './client';
import type { HealthResponse, ReadyResponse } from '../types';

export async function checkHealth(): Promise<HealthResponse> {
  const res = await rootApi.get<HealthResponse>('/health');
  return res.data;
}

export async function checkReady(): Promise<ReadyResponse> {
  const res = await rootApi.get<ReadyResponse>('/health/ready');
  return res.data;
}
