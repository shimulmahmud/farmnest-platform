import { useCallback, useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Dialog, DialogPanel, DialogTitle } from '@headlessui/react';
import { adminListOrders, adminShipOrder, adminUpdateOrderStatus } from '../../../api/orders';
import { formatCurrency, formatDateTime, cn } from '../../../lib/utils';
import DataTable from '../../../components/common/DataTable';
import LoadingSpinner from '../../../components/common/LoadingSpinner';
import ErrorMessage from '../../../components/common/ErrorMessage';
import EmptyState from '../../../components/common/EmptyState';
import StatusBadge from '../../../components/common/StatusBadge';
import DeliveryChargesPanel from './DeliveryChargesPanel';
import type { OrderResponse } from '../../../types';

const CARRIERS = ['Steadfast', 'Pathao', 'RedX', 'eCourier', 'Paperfly', 'Sundarban', 'SA Paribahan', 'Other'];

type Tab = 'shipments' | 'charges';
type Filter = 'TO_SHIP' | 'SHIPPED' | 'DELIVERED';

const FILTERS: { id: Filter; label: string; hint: string }[] = [
  { id: 'TO_SHIP', label: 'To ship', hint: 'Paid or confirmed orders waiting to be handed to a courier' },
  { id: 'SHIPPED', label: 'In transit', hint: 'With the courier' },
  { id: 'DELIVERED', label: 'Delivered', hint: 'Completed' },
];

const AREA_LABEL: Record<string, string> = { inside_dhaka: 'Inside Dhaka', outside_dhaka: 'Outside Dhaka' };

/** Orders waiting to ship come from three statuses, so load them together (oldest first). */
async function loadOrders(filter: Filter): Promise<OrderResponse[]> {
  if (filter === 'TO_SHIP') {
    const parts = await Promise.all(
      ['CONFIRMED', 'PAID', 'PROCESSING'].map((s) => adminListOrders(1, 50, s)),
    );
    return parts
      .flatMap((p) => p.data)
      .sort((a, b) => a.created_at.localeCompare(b.created_at));
  }
  return (await adminListOrders(1, 50, filter)).data;
}

export default function ShipmentsPage() {
  const queryClient = useQueryClient();
  const [tab, setTab] = useState<Tab>('shipments');
  const [filter, setFilter] = useState<Filter>('TO_SHIP');

  const [shipping, setShipping] = useState<OrderResponse | null>(null);
  const [carrier, setCarrier] = useState('Steadfast');
  const [tracking, setTracking] = useState('');
  const [courierCost, setCourierCost] = useState('');
  const [error, setError] = useState('');

  const ordersQuery = useQuery({
    queryKey: ['admin', 'shipping-orders', filter],
    queryFn: () => loadOrders(filter),
    enabled: tab === 'shipments',
  });

  const done = useCallback(() => {
    queryClient.invalidateQueries({ queryKey: ['admin'] });
    setShipping(null);
    setTracking('');
    setCourierCost('');
    setError('');
  }, [queryClient]);

  const shipMutation = useMutation({
    mutationFn: () =>
      adminShipOrder(shipping!.id, {
        carrier,
        tracking_number: tracking.trim(),
        courier_cost: courierCost ? parseInt(courierCost, 10) : undefined,
      }),
    onSuccess: done,
    onError: (e: unknown) => setError(e instanceof Error ? e.message : 'Could not ship the order'),
  });
  const deliveredMutation = useMutation({
    mutationFn: (id: string) => adminUpdateOrderStatus(id, { status: 'DELIVERED', reason: 'Delivered' }),
    onSuccess: done,
    onError: (e: unknown) => setError(e instanceof Error ? e.message : 'Could not update the order'),
  });

  const orders = ordersQuery.data ?? [];
  const charged = orders.reduce((n, o) => n + o.pricing.shipping, 0);
  const paidToCourier = orders.reduce((n, o) => n + (o.courier_cost ?? 0), 0);

  const columns = [
    {
      key: 'order_number',
      header: 'Order',
      render: (o: OrderResponse) => (
        <div>
          <p className="font-medium text-primary-600">{o.order_number}</p>
          <p className="text-xs text-gray-500">{formatDateTime(o.created_at)}</p>
        </div>
      ),
    },
    {
      key: 'customer',
      header: 'Deliver to',
      render: (o: OrderResponse) => (
        <div className="text-sm">
          <p className="text-gray-900">{o.shipping_address?.name}</p>
          <p className="text-xs text-gray-500">{o.shipping_address?.phone}</p>
          <p className="text-xs text-gray-500">
            {[o.shipping_address?.line1, o.shipping_address?.city].filter(Boolean).join(', ')}
            {o.delivery_area ? ` · ${AREA_LABEL[o.delivery_area] ?? o.delivery_area}` : ''}
          </p>
        </div>
      ),
    },
    {
      key: 'courier',
      header: 'Courier / tracking',
      render: (o: OrderResponse) =>
        o.tracking_number ? (
          <div className="text-sm">
            <p className="text-gray-900">{o.carrier}</p>
            <p className="font-mono text-xs text-gray-500">{o.tracking_number}</p>
          </div>
        ) : (
          <span className="text-gray-400">—</span>
        ),
    },
    {
      key: 'payment',
      header: 'Collect',
      render: (o: OrderResponse) =>
        o.payment_method === 'COD' && o.payment_status !== 'PAID' ? (
          <span className="badge-yellow" title="Cash to collect from the customer on delivery">COD {formatCurrency(o.pricing.total)}</span>
        ) : (
          <span className="badge-green">Paid</span>
        ),
    },
    {
      key: 'fee',
      header: 'Delivery fee',
      render: (o: OrderResponse) => (
        <div className="text-sm">
          <p className="text-gray-900" title="What the customer paid for delivery">{o.pricing.shipping === 0 ? 'Free' : formatCurrency(o.pricing.shipping)}</p>
          {o.courier_cost ? (
            <p className={cn('text-xs', o.pricing.shipping - o.courier_cost >= 0 ? 'text-emerald-600' : 'text-red-600')} title="Courier charged you">
              courier {formatCurrency(o.courier_cost)} ({o.pricing.shipping - o.courier_cost >= 0 ? '+' : ''}{formatCurrency(o.pricing.shipping - o.courier_cost)})
            </p>
          ) : null}
        </div>
      ),
    },
    {
      key: 'status',
      header: 'Status',
      render: (o: OrderResponse) => <StatusBadge status={o.status} type="order" />,
    },
    {
      key: 'actions',
      header: 'Actions',
      render: (o: OrderResponse) => (
        <div className="flex gap-1.5">
          {['PAID', 'CONFIRMED', 'PROCESSING'].includes(o.status) && (
            <button onClick={() => { setShipping(o); setError(''); }} className="rounded-lg bg-sky-50 px-2.5 py-1 text-xs font-medium text-sky-700 hover:bg-sky-100">Ship</button>
          )}
          {o.status === 'SHIPPED' && (
            <button onClick={() => deliveredMutation.mutate(o.id)} className="rounded-lg bg-emerald-50 px-2.5 py-1 text-xs font-medium text-emerald-700 hover:bg-emerald-100">Delivered</button>
          )}
        </div>
      ),
    },
  ];

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-gray-900">Shipping</h1>
        <p className="mt-1 text-sm text-gray-500">Hand orders to couriers, follow them, and set what customers pay for delivery.</p>
      </div>

      <div className="flex gap-2 border-b border-gray-200">
        {([['shipments', 'Shipments'], ['charges', 'Delivery charges']] as [Tab, string][]).map(([id, label]) => (
          <button
            key={id}
            onClick={() => setTab(id)}
            className={cn('-mb-px border-b-2 px-4 py-2 text-sm font-medium', tab === id ? 'border-primary-600 text-primary-700' : 'border-transparent text-gray-500 hover:text-gray-700')}
          >
            {label}
          </button>
        ))}
      </div>

      {tab === 'charges' ? (
        <DeliveryChargesPanel />
      ) : (
        <>
          <div className="flex flex-wrap items-center gap-2">
            {FILTERS.map((f) => (
              <button
                key={f.id}
                title={f.hint}
                onClick={() => setFilter(f.id)}
                className={cn('rounded-lg px-3 py-1.5 text-xs font-medium', filter === f.id ? 'bg-primary-600 text-white' : 'border border-gray-300 bg-white text-gray-600 hover:bg-gray-50')}
              >
                {f.label}
              </button>
            ))}
            {orders.length > 0 && (
              <p className="ml-auto text-xs text-gray-500">
                {orders.length} order{orders.length === 1 ? '' : 's'} · delivery fees charged {formatCurrency(charged)}
                {paidToCourier > 0 && <> · paid to couriers {formatCurrency(paidToCourier)}</>}
              </p>
            )}
          </div>

          {error && !shipping && <div className="rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-700">{error}</div>}

          {ordersQuery.isLoading ? (
            <LoadingSpinner />
          ) : ordersQuery.isError ? (
            <ErrorMessage message={ordersQuery.error?.message || 'Failed to load orders'} onRetry={() => ordersQuery.refetch()} />
          ) : orders.length === 0 ? (
            <EmptyState title="Nothing here" description={filter === 'TO_SHIP' ? 'No orders are waiting to be shipped.' : 'No orders in this list yet.'} />
          ) : (
            <DataTable columns={columns} data={orders as unknown as Record<string, unknown>[]} />
          )}
        </>
      )}

      <Dialog open={shipping !== null} onClose={() => setShipping(null)} className="relative z-50">
        <div className="fixed inset-0 bg-black/30" aria-hidden="true" />
        <div className="fixed inset-0 flex items-center justify-center p-4">
          <DialogPanel className="w-full max-w-md rounded-2xl bg-white p-6 shadow-xl">
            <DialogTitle className="text-lg font-semibold text-gray-900">Ship order {shipping?.order_number}</DialogTitle>
            {shipping && (
              <p className="mt-1 text-sm text-gray-500">
                The customer paid <strong>{shipping.pricing.shipping === 0 ? 'no delivery fee' : `${formatCurrency(shipping.pricing.shipping)} for delivery`}</strong>.
                {shipping.payment_method === 'COD' && shipping.payment_status !== 'PAID' && <> Collect <strong>{formatCurrency(shipping.pricing.total)}</strong> cash on delivery.</>}
              </p>
            )}
            {error && <div className="mt-3 rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-700">{error}</div>}
            <form className="mt-4 space-y-4" onSubmit={(e) => { e.preventDefault(); shipMutation.mutate(); }}>
              <div>
                <label className="label-field">Courier</label>
                <select value={carrier} onChange={(e) => setCarrier(e.target.value)} className="input-field">
                  {CARRIERS.map((c) => <option key={c}>{c}</option>)}
                </select>
              </div>
              <div>
                <label className="label-field">Tracking number <span className="text-red-500">*</span></label>
                <input required value={tracking} onChange={(e) => setTracking(e.target.value)} className="input-field" placeholder="e.g. SF123456" />
              </div>
              <div>
                <label className="label-field">What the courier charged you (৳)</label>
                <input type="number" min="0" step="1" value={courierCost} onChange={(e) => setCourierCost(e.target.value)} className="input-field" placeholder="Optional" />
                <p className="mt-1 text-xs text-gray-500">Used in the profit report, to compare with the delivery fee the customer paid.</p>
              </div>
              <div className="flex justify-end gap-3 pt-2">
                <button type="button" onClick={() => setShipping(null)} className="btn-secondary">Cancel</button>
                <button type="submit" disabled={shipMutation.isPending || !tracking.trim()} className="btn-primary">
                  {shipMutation.isPending ? 'Shipping…' : 'Mark as shipped'}
                </button>
              </div>
            </form>
          </DialogPanel>
        </div>
      </Dialog>
    </div>
  );
}
