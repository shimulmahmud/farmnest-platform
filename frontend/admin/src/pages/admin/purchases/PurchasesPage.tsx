import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { PlusIcon, TrashIcon } from '@heroicons/react/24/outline';
import { receiveStock, listPurchases } from '../../../api/purchases';
import { listWarehouses } from '../../../api/inventory';
import { adminListProducts } from '../../../api/products';
import { formatCurrency, formatDateTime } from '../../../lib/utils';
import LoadingSpinner from '../../../components/common/LoadingSpinner';
import ErrorMessage from '../../../components/common/ErrorMessage';
import type { PurchaseResponse } from '../../../types';

interface Line {
  sku: string;
  quantity: string;
  unit_cost: string;
}

const emptyLine = (): Line => ({ sku: '', quantity: '', unit_cost: '' });

export default function PurchasesPage() {
  const queryClient = useQueryClient();
  const [supplier, setSupplier] = useState('');
  const [reference, setReference] = useState('');
  const [warehouseId, setWarehouseId] = useState('');
  const [note, setNote] = useState('');
  const [lines, setLines] = useState<Line[]>([emptyLine()]);
  const [error, setError] = useState('');
  const [last, setLast] = useState<PurchaseResponse | null>(null);

  const warehousesQuery = useQuery({ queryKey: ['warehouses'], queryFn: listWarehouses });
  const warehouses = warehousesQuery.data?.data ?? [];

  // All variants (SKU picker). Fine for a few hundred products.
  const productsQuery = useQuery({
    queryKey: ['admin', 'products', 'all-for-purchase'],
    queryFn: () => adminListProducts(1, 100),
  });
  const skus = (productsQuery.data?.data ?? []).flatMap((p) =>
    p.variants
      .filter((v) => v.status === 'ACTIVE')
      .map((v) => ({
        sku: v.sku,
        label: `${p.name}${Object.values(v.options ?? {}).length ? ' — ' + Object.values(v.options).join(', ') : ''}`,
        cost: v.cost_price,
        available: v.available ?? 0,
      })),
  );

  const historyQuery = useQuery({
    queryKey: ['admin', 'purchases'],
    queryFn: () => listPurchases(1, 20),
  });

  const mutation = useMutation({
    mutationFn: receiveStock,
    onSuccess: (p) => {
      setLast(p);
      setLines([emptyLine()]);
      setReference('');
      setNote('');
      setError('');
      queryClient.invalidateQueries({ queryKey: ['admin'] });
    },
    onError: (e: unknown) => setError(e instanceof Error ? e.message : 'Could not save purchase'),
  });

  const setLine = (i: number, patch: Partial<Line>) =>
    setLines((prev) => prev.map((l, j) => (j === i ? { ...l, ...patch } : l)));

  const total = lines.reduce(
    (sum, l) => sum + (parseInt(l.quantity, 10) || 0) * (parseInt(l.unit_cost, 10) || 0),
    0,
  );

  const submit = (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setLast(null);
    mutation.mutate({
      supplier_name: supplier.trim(),
      reference: reference.trim() || undefined,
      warehouse_id: warehouseId,
      note: note.trim() || undefined,
      items: lines.map((l) => ({
        sku: l.sku.trim(),
        quantity: parseInt(l.quantity, 10) || 0,
        unit_cost: parseInt(l.unit_cost, 10) || 0,
      })),
    });
  };

  if (warehousesQuery.isLoading || productsQuery.isLoading) return <LoadingSpinner />;
  if (warehousesQuery.isError) {
    return <ErrorMessage message="Could not load warehouses" onRetry={() => warehousesQuery.refetch()} />;
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-gray-900">Purchases (receive stock)</h1>
        <p className="mt-1 text-sm text-gray-500">
          Record goods you bought. Stock goes up, and each product&apos;s cost becomes the average of what you paid.
        </p>
      </div>

      {warehouses.length === 0 && (
        <div className="rounded-lg bg-amber-50 border border-amber-200 p-3 text-sm text-amber-800">
          Create a warehouse first (Inventory page). Stock must be received into a warehouse.
        </div>
      )}

      {error && <div className="rounded-lg bg-red-50 border border-red-200 p-3 text-sm text-red-700">{error}</div>}

      {last && (
        <div className="rounded-lg bg-emerald-50 border border-emerald-200 p-4 text-sm text-emerald-800 space-y-2">
          <p className="font-semibold">Saved {last.number} — total {formatCurrency(last.total_cost)}</p>
          <ul className="space-y-0.5">
            {last.items.map((it) => (
              <li key={it.sku}>
                <span className="font-mono">{it.sku}</span>: stock {it.stock_before} → {it.stock_after}, cost{' '}
                {formatCurrency(it.cost_before)} → {formatCurrency(it.cost_after)}
              </li>
            ))}
          </ul>
        </div>
      )}

      <form onSubmit={submit} className="card space-y-5">
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
          <div>
            <label className="label-field">Supplier <span className="text-red-500">*</span></label>
            <input required value={supplier} onChange={(e) => setSupplier(e.target.value)} className="input-field" placeholder="e.g. Dhaka Traders" />
          </div>
          <div>
            <label className="label-field">Invoice / reference</label>
            <input value={reference} onChange={(e) => setReference(e.target.value)} className="input-field" placeholder="Supplier invoice no." />
          </div>
          <div>
            <label className="label-field">Receive into <span className="text-red-500">*</span></label>
            <select required value={warehouseId} onChange={(e) => setWarehouseId(e.target.value)} className="input-field">
              <option value="">Select warehouse</option>
              {warehouses.map((w) => (
                <option key={w.id} value={w.id}>{w.name} ({w.code})</option>
              ))}
            </select>
          </div>
        </div>

        <div className="space-y-2">
          <div className="grid grid-cols-12 gap-2 text-xs font-semibold uppercase text-gray-500">
            <span className="col-span-6">Product (SKU)</span>
            <span className="col-span-2">Quantity</span>
            <span className="col-span-3">Unit cost (৳)</span>
            <span className="col-span-1" />
          </div>
          {lines.map((l, i) => {
            const match = skus.find((s) => s.sku === l.sku);
            return (
              <div key={i} className="grid grid-cols-12 gap-2 items-start">
                <div className="col-span-6">
                  <select
                    required
                    value={l.sku}
                    onChange={(e) => {
                      const s = skus.find((x) => x.sku === e.target.value);
                      setLine(i, { sku: e.target.value, unit_cost: l.unit_cost || (s && s.cost ? String(s.cost) : '') });
                    }}
                    className="input-field"
                  >
                    <option value="">Select product / variant</option>
                    {skus.map((s) => (
                      <option key={s.sku} value={s.sku}>{s.sku} — {s.label}</option>
                    ))}
                  </select>
                  {match && <p className="mt-1 text-xs text-gray-500">In stock now: {match.available} · current cost {formatCurrency(match.cost)}</p>}
                </div>
                <input required type="number" min="1" step="1" value={l.quantity} onChange={(e) => setLine(i, { quantity: e.target.value })} className="input-field col-span-2" placeholder="Qty" />
                <input required type="number" min="1" step="1" value={l.unit_cost} onChange={(e) => setLine(i, { unit_cost: e.target.value })} className="input-field col-span-3" placeholder="Cost per unit" />
                <button
                  type="button"
                  onClick={() => setLines((p) => (p.length > 1 ? p.filter((_, j) => j !== i) : p))}
                  className="col-span-1 rounded p-2 text-gray-400 hover:text-red-500 hover:bg-red-50"
                >
                  <TrashIcon className="h-4 w-4" />
                </button>
              </div>
            );
          })}
          <button type="button" onClick={() => setLines((p) => [...p, emptyLine()])} className="inline-flex items-center gap-1 text-sm font-medium text-primary-600">
            <PlusIcon className="h-4 w-4" /> Add another product
          </button>
        </div>

        <div>
          <label className="label-field">Note</label>
          <input value={note} onChange={(e) => setNote(e.target.value)} className="input-field" placeholder="Optional" />
        </div>

        <div className="flex items-center justify-between border-t pt-4">
          <p className="text-sm text-gray-600">Purchase total: <span className="text-lg font-semibold text-gray-900">{formatCurrency(total)}</span></p>
          <button type="submit" disabled={mutation.isPending || warehouses.length === 0} className="btn-primary">
            {mutation.isPending ? 'Saving…' : 'Receive stock'}
          </button>
        </div>
      </form>

      <div className="card">
        <h2 className="text-lg font-semibold text-gray-900 mb-3">Recent purchases</h2>
        {historyQuery.isLoading ? (
          <LoadingSpinner />
        ) : (historyQuery.data?.data ?? []).length === 0 ? (
          <p className="text-sm text-gray-500">No purchases yet.</p>
        ) : (
          <table className="w-full text-sm">
            <thead>
              <tr className="text-left text-xs uppercase text-gray-500 border-b">
                <th className="py-2">Number</th><th className="py-2">Supplier</th><th className="py-2">Items</th><th className="py-2 text-right">Total</th><th className="py-2">Date</th>
              </tr>
            </thead>
            <tbody>
              {historyQuery.data!.data.map((p) => (
                <tr key={p.id} className="border-b border-gray-100">
                  <td className="py-2 font-mono text-xs">{p.number}</td>
                  <td className="py-2">{p.supplier_name}{p.reference ? <span className="text-xs text-gray-500"> · {p.reference}</span> : null}</td>
                  <td className="py-2 text-gray-600">{p.items.map((i) => `${i.sku} ×${i.quantity}`).join(', ')}</td>
                  <td className="py-2 text-right">{formatCurrency(p.total_cost)}</td>
                  <td className="py-2 text-gray-500">{formatDateTime(p.received_at)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </div>
  );
}
