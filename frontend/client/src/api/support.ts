import { get, getPage, post } from './http';

export type TicketStatus = 'OPEN' | 'ANSWERED' | 'CLOSED';

export interface TicketMessage {
  id: string;
  sender_role: 'CUSTOMER' | 'STAFF';
  sender_name: string;
  body: string;
  created_at: string;
}

export interface TicketSummary {
  id: string;
  number: string;
  subject: string;
  category: string;
  order_number?: string;
  status: TicketStatus;
  customer_unread: boolean;
  last_message?: { sender_role: string; body: string };
  last_message_at: string;
  created_at: string;
}

export interface Ticket extends TicketSummary {
  messages: TicketMessage[];
}

export const CATEGORIES: { id: string; label: string }[] = [
  { id: 'ORDER', label: 'My order' },
  { id: 'DELIVERY', label: 'Delivery' },
  { id: 'PAYMENT', label: 'Payment' },
  { id: 'PRODUCT', label: 'A product' },
  { id: 'ACCOUNT', label: 'My account' },
  { id: 'OTHER', label: 'Something else' },
];

export const createTicket = (input: { subject: string; category: string; order_number?: string; message: string }) =>
  post<Ticket>('/support/tickets', input);

export const listMyTickets = () => getPage<TicketSummary>('/support/tickets', { page: 1, per_page: 50 }, true);

export const getTicket = (id: string) => get<Ticket>(`/support/tickets/${id}`, undefined, true);

export const replyTicket = (id: string, message: string) => post<Ticket>(`/support/tickets/${id}/messages`, { message });

export const closeTicket = (id: string) => post<Ticket>(`/support/tickets/${id}/close`, {});

export const unreadCount = async (): Promise<number> => (await get<{ count: number }>('/support/unread', undefined, true)).count;
