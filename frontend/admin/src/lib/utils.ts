import { format, parseISO } from 'date-fns';

/** Money everywhere is whole BDT taka (no paisa), shown as ৳1,299. */
export function formatCurrency(amount: number): string {
  const n = Math.round(Number.isFinite(amount) ? amount : 0);
  return `${n < 0 ? '-' : ''}৳${Math.abs(n).toLocaleString('en-BD')}`;
}

export function formatDate(iso: string): string {
  try {
    return format(parseISO(iso), 'MMM dd, yyyy');
  } catch {
    return iso;
  }
}

export function formatDateTime(iso: string): string {
  try {
    return format(parseISO(iso), 'MMM dd, yyyy HH:mm');
  } catch {
    return iso;
  }
}

export function getOrderStatusColor(status: string): string {
  const map: Record<string, string> = {
    PENDING: 'badge-yellow',
    PAYMENT_PENDING: 'badge-yellow',
    CONFIRMED: 'badge-blue',
    PAID: 'badge-blue',
    REFUND_PENDING: 'badge-red',
    PROCESSING: 'badge-blue',
    SHIPPED: 'badge-blue',
    DELIVERED: 'badge-green',
    CANCELLED: 'badge-red',
    REFUNDED: 'badge-red',
  };
  return map[status] || 'badge-gray';
}

export function getPaymentStatusColor(status: string): string {
  const map: Record<string, string> = {
    PENDING: 'badge-yellow',
    AUTHORIZED: 'badge-blue',
    PAID: 'badge-green',
    FAILED: 'badge-red',
    REFUNDED: 'badge-red',
  };
  return map[status] || 'badge-gray';
}

export function getShipmentStatusColor(status: string): string {
  const map: Record<string, string> = {
    PENDING: 'badge-yellow',
    PROCESSING: 'badge-blue',
    SHIPPED: 'badge-blue',
    DELIVERED: 'badge-green',
    CANCELLED: 'badge-red',
  };
  return map[status] || 'badge-gray';
}

export function truncate(str: string, len: number): string {
  if (str.length <= len) return str;
  return str.slice(0, len) + '...';
}

export function cn(...classes: (string | undefined | null | false)[]): string {
  return classes.filter(Boolean).join(' ');
}
