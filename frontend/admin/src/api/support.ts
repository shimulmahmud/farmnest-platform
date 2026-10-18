import { api } from './client';
import type { PaginatedResponse } from '../types';

export type TicketStatus = 'OPEN' | 'ANSWERED' | 'CLOSED';

export interface SupportMessage {
  id: string;
  sender_role: 'CUSTOMER' | 'STAFF';
  sender_name: string;
  body: string;
  created_at: string;
}

export interface SupportTicketSummary {
  id: string;
  number: string;
  subject: string;
  category: string;
  order_number?: string;
  status: TicketStatus;
  customer_name?: string;
  customer_email?: string;
  staff_unread: boolean;
  last_message?: { sender_role: string; body: string };
  last_message_at: string;
  created_at: string;
}

export interface SupportTicket extends SupportTicketSummary {
  customer_id: string;
  messages: SupportMessage[];
}

export interface SupportSummary {
  open: number;
  answered: number;
  closed: number;
  unread: number;
}

export async function listSupportTickets(
  page = 1,
  filters: { status?: string; q?: string } = {},
): Promise<PaginatedResponse<SupportTicketSummary>> {
  const res = await api.get<PaginatedResponse<SupportTicketSummary>>('/admin/support/tickets', {
    params: { page, per_page: 30, ...filters },
  });
  return res.data;
}

export async function getSupportTicket(id: string): Promise<SupportTicket> {
  const res = await api.get<SupportTicket>(`/admin/support/tickets/${id}`);
  return res.data;
}

export async function replySupportTicket(id: string, message: string): Promise<SupportTicket> {
  const res = await api.post<SupportTicket>(`/admin/support/tickets/${id}/messages`, { message });
  return res.data;
}

export async function setSupportStatus(id: string, status: TicketStatus): Promise<SupportTicket> {
  const res = await api.patch<SupportTicket>(`/admin/support/tickets/${id}/status`, { status });
  return res.data;
}

export async function getSupportSummary(): Promise<SupportSummary> {
  const res = await api.get<SupportSummary>('/admin/support/summary');
  return res.data;
}
