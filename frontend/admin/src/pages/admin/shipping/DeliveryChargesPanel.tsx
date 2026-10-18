import { useEffect, useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { getDeliverySettings, updateDeliverySettings } from '../../../api/settings';
import { formatCurrency } from '../../../lib/utils';
import LoadingSpinner from '../../../components/common/LoadingSpinner';
import ErrorMessage from '../../../components/common/ErrorMessage';
import type { DeliveryMethod } from '../../../types';

// Fees are typed as text so an empty box can be cleared while typing; they are whole taka when saved.
interface Row extends Omit<DeliveryMethod, 'fee_inside_dhaka' | 'fee_outside_dhaka'> {
  fee_inside_dhaka: string;
  fee_outside_dhaka: string;
}

const toInt = (v: string) => Math.max(0, parseInt(v, 10) || 0);

/** Where the shop sets what customers pay for delivery. It applies to NEW orders only. */
export default function DeliveryChargesPanel() {
  const queryClient = useQueryClient();
  const settingsQuery = useQuery({ queryKey: ['admin', 'delivery-settings'], queryFn: getDeliverySettings });

  const [rows, setRows] = useState<Row[]>([]);
  const [threshold, setThreshold] = useState('0');
  const [saved, setSaved] = useState(false);
  const [error, setError] = useState('');

  useEffect(() => {
    const d = settingsQuery.data;
    if (!d) return;
    setRows(d.methods.map((m) => ({ ...m, fee_inside_dhaka: String(m.fee_inside_dhaka), fee_outside_dhaka: String(m.fee_outside_dhaka) })));
    setThreshold(String(d.free_shipping_threshold));
  }, [settingsQuery.data]);

  const mutation = useMutation({
    mutationFn: () =>
      updateDeliverySettings({
        free_shipping_threshold: toInt(threshold),
        methods: rows.map((r) => ({
          id: r.id, label: r.label.trim(), detail: r.detail.trim(), enabled: r.enabled,
          fee_inside_dhaka: toInt(r.fee_inside_dhaka), fee_outside_dhaka: toInt(r.fee_outside_dhaka),
        })),
      }),
    onSuccess: () => {
      setSaved(true);
      setError('');
      queryClient.invalidateQueries({ queryKey: ['admin', 'delivery-settings'] });
    },
    onError: (e: unknown) => {
      setSaved(false);
      setError(e instanceof Error ? e.message : 'Could not save the delivery charges');
    },
  });

  const setRow = (i: number, patch: Partial<Row>) => {
    setSaved(false);
    setRows((prev) => prev.map((r, j) => (j === i ? { ...r, ...patch } : r)));
  };

  if (settingsQuery.isLoading) return <LoadingSpinner />;
  if (settingsQuery.isError) return <ErrorMessage message={settingsQuery.error?.message || 'Could not load delivery charges'} onRetry={() => settingsQuery.refetch()} />;

  const standard = rows.find((r) => r.id === 'standard' && r.enabled);
  const free = toInt(threshold);

  return (
    <form
      className="space-y-6"
      onSubmit={(e) => { e.preventDefault(); mutation.mutate(); }}
    >
      <div className="card space-y-1">
        <h2 className="text-lg font-semibold text-gray-900">Delivery charges</h2>
        <p className="text-sm text-gray-500">
          This is what customers pay for delivery. It is added to the order at checkout. Changes apply to <strong>new orders only</strong>;
          orders already placed keep the charge they were placed with. (What the courier charges <em>you</em> is entered when you ship an order.)
        </p>
      </div>

      {error && <div className="rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-700">{error}</div>}
      {saved && <div className="rounded-lg border border-emerald-200 bg-emerald-50 p-3 text-sm text-emerald-800">Saved. New orders will use these charges.</div>}

      <div className="card space-y-5">
        {rows.map((r, i) => (
          <div key={r.id} className={`rounded-xl border p-4 ${r.enabled ? 'border-gray-200' : 'border-gray-100 bg-gray-50'}`}>
            <div className="mb-3 flex items-center justify-between gap-3">
              <h3 className="text-sm font-semibold uppercase tracking-wide text-gray-500">{r.id === 'standard' ? 'Standard delivery' : 'Express delivery'}</h3>
              <label className="flex cursor-pointer items-center gap-2 text-sm">
                <input type="checkbox" checked={r.enabled} onChange={(e) => setRow(i, { enabled: e.target.checked })} className="h-4 w-4 accent-primary-600" />
                {r.enabled ? 'Offered to customers' : 'Switched off'}
              </label>
            </div>
            <div className="grid gap-4 sm:grid-cols-2">
              <div>
                <label className="label-field">Name shown at checkout</label>
                <input value={r.label} onChange={(e) => setRow(i, { label: e.target.value })} maxLength={40} className="input-field" />
              </div>
              <div>
                <label className="label-field">Short description</label>
                <input value={r.detail} onChange={(e) => setRow(i, { detail: e.target.value })} maxLength={80} placeholder="e.g. 2-4 working days" className="input-field" />
              </div>
              <div>
                <label className="label-field">Charge inside Dhaka city (৳)</label>
                <input type="number" min="0" step="1" value={r.fee_inside_dhaka} onChange={(e) => setRow(i, { fee_inside_dhaka: e.target.value })} className="input-field" />
              </div>
              <div>
                <label className="label-field">Charge outside Dhaka (৳)</label>
                <input type="number" min="0" step="1" value={r.fee_outside_dhaka} onChange={(e) => setRow(i, { fee_outside_dhaka: e.target.value })} className="input-field" />
              </div>
            </div>
          </div>
        ))}

        <div className="rounded-xl border border-gray-200 p-4">
          <label className="label-field">Free standard delivery on orders of (৳)</label>
          <input type="number" min="0" step="1" value={threshold} onChange={(e) => { setSaved(false); setThreshold(e.target.value); }} className="input-field max-w-xs" />
          <p className="mt-1 text-xs text-gray-500">
            {free > 0
              ? `Customers whose items total ${formatCurrency(free)} or more pay nothing for standard delivery (express is always charged).`
              : 'Set to 0 if you never want to offer free delivery.'}
          </p>
        </div>
      </div>

      {standard && (
        <div className="card text-sm text-gray-600">
          <p className="mb-1 font-semibold text-gray-800">How a customer sees it</p>
          <p>
            Order of {formatCurrency(500)} → {standard.label}: inside Dhaka <strong>{formatCurrency(toInt(standard.fee_inside_dhaka))}</strong>,
            outside Dhaka <strong>{formatCurrency(toInt(standard.fee_outside_dhaka))}</strong>.
            {free > 0 && <> Order of {formatCurrency(free)} → <strong>free</strong>.</>}
          </p>
        </div>
      )}

      <div className="flex justify-end">
        <button type="submit" disabled={mutation.isPending} className="btn-primary">
          {mutation.isPending ? 'Saving…' : 'Save delivery charges'}
        </button>
      </div>
    </form>
  );
}
