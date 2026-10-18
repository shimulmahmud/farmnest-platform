import { api } from './client';
import type { DeliverySettings } from '../types';

export async function getDeliverySettings(): Promise<DeliverySettings> {
  const res = await api.get<DeliverySettings>('/admin/settings/delivery');
  return res.data;
}

export async function updateDeliverySettings(data: Pick<DeliverySettings, 'methods' | 'free_shipping_threshold'>): Promise<DeliverySettings> {
  const res = await api.put<DeliverySettings>('/admin/settings/delivery', data);
  return res.data;
}
