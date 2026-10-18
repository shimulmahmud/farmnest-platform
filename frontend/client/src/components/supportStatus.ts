import type { TicketStatus } from '@/api/support';

export const statusText = (s: TicketStatus): string =>
  s === 'OPEN' ? 'Waiting for us' : s === 'ANSWERED' ? 'We replied' : 'Closed';

export const statusStyle = (s: TicketStatus): string =>
  s === 'OPEN' ? 'bg-amber-50 text-amber-700 border-amber-200'
    : s === 'ANSWERED' ? 'bg-emerald-50 text-emerald-700 border-emerald-200'
    : 'bg-gray-100 text-gray-600 border-gray-200';
