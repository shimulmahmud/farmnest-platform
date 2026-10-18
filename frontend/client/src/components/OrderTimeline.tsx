import { CheckCircle2, Circle, XCircle } from 'lucide-react';

export const STATUS_LABEL: Record<string, string> = {
  PENDING: 'Order received',
  PAYMENT_PENDING: 'Waiting for payment confirmation',
  CONFIRMED: 'Order confirmed',
  PAID: 'Payment confirmed',
  PROCESSING: 'Packing your order',
  SHIPPED: 'Shipped',
  DELIVERED: 'Delivered',
  CANCELLED: 'Cancelled',
  REFUND_PENDING: 'Refund in progress',
  REFUNDED: 'Refunded',
};

export const statusLabel = (s: string) => STATUS_LABEL[s] ?? s.replace(/_/g, ' ').toLowerCase();

export const statusTone = (s: string): string => {
  if (s === 'DELIVERED') return 'bg-emerald-50 text-emerald-700 border-emerald-200';
  if (s === 'CANCELLED' || s === 'REFUNDED' || s === 'REFUND_PENDING') return 'bg-red-50 text-red-700 border-red-200';
  if (s === 'SHIPPED') return 'bg-sky-50 text-sky-700 border-sky-200';
  return 'bg-amber-50 text-amber-700 border-amber-200';
};

interface Entry {
  status: string;
  changed_at: string;
  reason?: string;
}

/** Vertical timeline of the order's status history, newest last. */
export default function OrderTimeline({ history }: { history: Entry[] }) {
  if (history.length === 0) return null;
  const last = history.length - 1;
  return (
    <ol className="space-y-4">
      {history.map((h, i) => {
        const bad = h.status === 'CANCELLED' || h.status === 'REFUNDED' || h.status === 'REFUND_PENDING';
        const Icon = bad ? XCircle : i === last && h.status !== 'DELIVERED' ? Circle : CheckCircle2;
        return (
          <li key={`${h.status}-${i}`} className="flex gap-3">
            <Icon className={`mt-0.5 h-5 w-5 flex-shrink-0 ${bad ? 'text-red-500' : i === last && h.status !== 'DELIVERED' ? 'text-primary' : 'text-emerald-600'}`} />
            <div>
              <p className="text-sm font-semibold text-gray-900">{statusLabel(h.status)}</p>
              <p className="text-xs text-gray-500">
                {new Date(h.changed_at).toLocaleString('en-GB', { dateStyle: 'medium', timeStyle: 'short' })}
                {h.reason ? ` · ${h.reason}` : ''}
              </p>
            </div>
          </li>
        );
      })}
    </ol>
  );
}
