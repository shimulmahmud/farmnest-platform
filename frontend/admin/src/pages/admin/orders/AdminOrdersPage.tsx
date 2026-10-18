import { useState, useCallback } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useSearchParams } from 'react-router-dom';
import { EyeIcon } from '@heroicons/react/24/outline';
import { Dialog, DialogPanel, DialogTitle } from '@headlessui/react';
import {
  adminListOrders,
  adminUpdateOrderStatus,
  adminShipOrder,
  adminMarkOrderPaid,
} from '../../../api/orders';
import { formatCurrency, formatDateTime, cn } from '../../../lib/utils';
import DataTable from '../../../components/common/DataTable';
import LoadingSpinner from '../../../components/common/LoadingSpinner';
import ErrorMessage from '../../../components/common/ErrorMessage';
import EmptyState from '../../../components/common/EmptyState';
import StatusBadge from '../../../components/common/StatusBadge';
import Pagination from '../../../components/common/Pagination';
import type { OrderResponse } from '../../../types';

const ORDER_STATUSES = [
  'ALL',
  'PAYMENT_PENDING',
  'CONFIRMED',
  'PAID',
  'PROCESSING',
  'SHIPPED',
  'DELIVERED',
  'CANCELLED',
  'REFUND_PENDING',
  'REFUNDED',
] as const;

type OrderStatus = (typeof ORDER_STATUSES)[number];

const CARRIERS = ['Steadfast', 'Pathao', 'RedX', 'eCourier', 'Paperfly', 'Sundarban', 'SA Paribahan', 'Other'];

const label = (s: string) =>
  s.replace(/_/g, ' ').toLowerCase().replace(/\b\w/g, (c) => c.toUpperCase());

type Dialogs = 'none' | 'detail' | 'ship' | 'cancel' | 'refund';

export default function AdminOrdersPage() {
  const queryClient = useQueryClient();
  const [searchParams, setSearchParams] = useSearchParams();

  const page = Number(searchParams.get('page')) || 1;
  const statusFilter = (searchParams.get('status') as OrderStatus) || 'ALL';

  const [dialog, setDialog] = useState<Dialogs>('none');
  const [target, setTarget] = useState<OrderResponse | null>(null);
  const [carrier, setCarrier] = useState('Steadfast');
  const [tracking, setTracking] = useState('');
  const [courierCost, setCourierCost] = useState('');
  const [reason, setReason] = useState('');
  const [actionError, setActionError] = useState('');

  const ordersQuery = useQuery({
    queryKey: ['admin', 'orders', page, statusFilter],
    queryFn: () =>
      adminListOrders(page, 20, statusFilter === 'ALL' ? undefined : statusFilter),
  });

  const closeDialog = useCallback(() => {
    setDialog('none');
    setTarget(null);
    setTracking('');
    setCourierCost('');
    setReason('');
    setActionError('');
  }, []);

  const done = useCallback(() => {
    queryClient.invalidateQueries({ queryKey: ['admin'] });
    closeDialog();
  }, [queryClient, closeDialog]);

  const onError = (e: unknown) =>
    setActionError(e instanceof Error ? e.message : 'Action failed');

  const statusMutation = useMutation({
    mutationFn: ({ id, status, reason, payment_reference }: {
      id: string; status: string; reason?: string; payment_reference?: string;
    }) => adminUpdateOrderStatus(id, { status, reason, payment_reference }),
    onSuccess: done,
    onError,
  });
  const shipMutation = useMutation({
    mutationFn: ({ id }: { id: string }) =>
      adminShipOrder(id, {
        carrier,
        tracking_number: tracking.trim(),
        courier_cost: courierCost ? parseInt(courierCost, 10) : undefined,
      }),
    onSuccess: done,
    onError,
  });
  const paidMutation = useMutation({
    mutationFn: (id: string) => adminMarkOrderPaid(id),
    onSuccess: done,
    onError,
  });

  const handleStatusFilter = useCallback(
    (status: OrderStatus) => {
      const params = new URLSearchParams(searchParams);
      if (status === 'ALL') params.delete('status');
      else params.set('status', status);
      params.set('page', '1');
      setSearchParams(params);
    },
    [searchParams, setSearchParams],
  );

  const handlePageChange = useCallback(
    (newPage: number) => {
      const params = new URLSearchParams(searchParams);
      params.set('page', String(newPage));
      setSearchParams(params);
    },
    [searchParams, setSearchParams],
  );

  const open = (order: OrderResponse, d: Dialogs) => {
    setTarget(order);
    setActionError('');
    setDialog(d);
  };

  if (ordersQuery.isLoading) return <LoadingSpinner />;
  if (ordersQuery.isError) {
    return (
      <ErrorMessage
        message={ordersQuery.error?.message || 'Failed to load orders'}
        onRetry={() => ordersQuery.refetch()}
      />
    );
  }

  const orders = ordersQuery.data!.data;
  const pagination = ordersQuery.data!.pagination;

  // What the admin can do next, by order state.
  const canConfirmPayment = (o: OrderResponse) => o.status === 'PAYMENT_PENDING' && o.payment_method === 'ONLINE';
  const canShip = (o: OrderResponse) => ['PAID', 'CONFIRMED', 'PROCESSING'].includes(o.status);
  const canDeliver = (o: OrderResponse) => o.status === 'SHIPPED';
  const canMarkCash = (o: OrderResponse) =>
    o.payment_method === 'COD' && o.payment_status === 'UNPAID' && !['CANCELLED', 'REFUNDED'].includes(o.status);
  const canCancel = (o: OrderResponse) => ['PAYMENT_PENDING', 'CONFIRMED', 'PAID', 'PROCESSING'].includes(o.status);
  const canRefund = (o: OrderResponse) => o.status === 'REFUND_PENDING';

  const columns = [
    {
      key: 'order_number',
      header: 'Order #',
      render: (row: OrderResponse) => (
        <button onClick={() => open(row, 'detail')} className="font-medium text-primary-600 hover:text-primary-700">
          {row.order_number}
        </button>
      ),
    },
    {
      key: 'customer',
      header: 'Customer',
      render: (row: OrderResponse) => (
        <div className="text-sm">
          <p className="text-gray-900">{row.shipping_address?.name || '—'}</p>
          <p className="text-xs text-gray-500">{row.shipping_address?.phone || ''}</p>
        </div>
      ),
    },
    {
      key: 'items',
      header: 'Items',
      render: (row: OrderResponse) => (
        <span className="text-gray-600">{row.items.reduce((n, i) => n + i.quantity, 0)} pcs</span>
      ),
    },
    {
      key: 'total',
      header: 'Total',
      render: (row: OrderResponse) => (
        <span className="font-medium text-gray-900">{formatCurrency(row.pricing.total)}</span>
      ),
    },
    {
      key: 'payment',
      header: 'Payment',
      render: (row: OrderResponse) => (
        <div className="text-xs space-y-1">
          <span className="text-gray-600">
            {row.payment_method === 'COD' ? 'Cash on delivery' : (row.payment_provider || 'Online').toUpperCase()}
          </span>
          <div>
            <span className={row.payment_status === 'PAID' ? 'badge-green' : 'badge-yellow'}>
              {row.payment_status === 'PAID' ? 'Paid' : 'Unpaid'}
            </span>
          </div>
        </div>
      ),
    },
    {
      key: 'status',
      header: 'Status',
      render: (row: OrderResponse) => <StatusBadge status={row.status} type="order" />,
    },
    {
      key: 'created_at',
      header: 'Date',
      render: (row: OrderResponse) => (
        <span className="text-gray-500 text-sm">{formatDateTime(row.created_at)}</span>
      ),
    },
    {
      key: 'actions',
      header: 'Actions',
      render: (row: OrderResponse) => (
        <div className="flex flex-wrap items-center gap-1.5">
          <button onClick={() => open(row, 'detail')} className="inline-flex items-center gap-1 rounded-lg px-2 py-1 text-xs font-medium text-gray-600 hover:bg-gray-100">
            <EyeIcon className="h-3.5 w-3.5" /> View
          </button>
          {canConfirmPayment(row) && (
            <button
              onClick={() => statusMutation.mutate({ id: row.id, status: 'PAID', reason: 'Payment verified by admin' })}
              className="rounded-lg bg-emerald-50 px-2 py-1 text-xs font-medium text-emerald-700 hover:bg-emerald-100"
            >
              Confirm payment
            </button>
          )}
          {canShip(row) && (
            <button onClick={() => open(row, 'ship')} className="rounded-lg bg-sky-50 px-2 py-1 text-xs font-medium text-sky-700 hover:bg-sky-100">
              Ship
            </button>
          )}
          {canDeliver(row) && (
            <button
              onClick={() => statusMutation.mutate({ id: row.id, status: 'DELIVERED', reason: 'Delivered' })}
              className="rounded-lg bg-emerald-50 px-2 py-1 text-xs font-medium text-emerald-700 hover:bg-emerald-100"
            >
              Delivered
            </button>
          )}
          {canMarkCash(row) && (
            <button onClick={() => paidMutation.mutate(row.id)} className="rounded-lg bg-amber-50 px-2 py-1 text-xs font-medium text-amber-700 hover:bg-amber-100">
              Cash received
            </button>
          )}
          {canRefund(row) && (
            <button onClick={() => open(row, 'refund')} className="rounded-lg bg-red-50 px-2 py-1 text-xs font-medium text-red-700 hover:bg-red-100">
              Mark refunded
            </button>
          )}
          {canCancel(row) && (
            <button onClick={() => open(row, 'cancel')} className="rounded-lg px-2 py-1 text-xs font-medium text-red-600 hover:bg-red-50">
              Cancel
            </button>
          )}
        </div>
      ),
    },
  ];

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-gray-900">Orders</h1>
        <p className="mt-1 text-sm text-gray-500">
          Confirm payments, ship orders with a tracking number, and collect cash on delivery.
        </p>
      </div>

      {(statusMutation.isError || paidMutation.isError) && dialog === 'none' && (
        <div className="rounded-lg bg-red-50 border border-red-200 p-3 text-sm text-red-700">
          {((statusMutation.error ?? paidMutation.error) as Error | null)?.message || 'Action failed'}
        </div>
      )}

      <div className="flex flex-wrap gap-2">
        {ORDER_STATUSES.map((status) => (
          <button
            key={status}
            onClick={() => handleStatusFilter(status)}
            className={cn(
              'rounded-lg px-3 py-1.5 text-xs font-medium transition-colors',
              statusFilter === status
                ? 'bg-primary-600 text-white'
                : 'bg-white text-gray-600 border border-gray-300 hover:bg-gray-50',
            )}
          >
            {status === 'ALL' ? 'All' : label(status)}
          </button>
        ))}
      </div>

      {orders.length === 0 ? (
        <EmptyState title="No orders found" description="No orders match the current filter." />
      ) : (
        <>
          <DataTable columns={columns} data={orders as unknown as Record<string, unknown>[]} />
          <div className="mt-4">
            <Pagination page={pagination.page} totalPages={pagination.total_pages} onPageChange={handlePageChange} />
          </div>
        </>
      )}

      {/* Order detail */}
      <Dialog open={dialog === 'detail'} onClose={closeDialog} className="relative z-50">
        <div className="fixed inset-0 bg-black/30" aria-hidden="true" />
        <div className="fixed inset-0 flex items-center justify-center p-4">
          <DialogPanel className="w-full max-w-2xl max-h-[90vh] overflow-y-auto rounded-2xl bg-white p-6 shadow-xl">
            {target && (
              <div className="space-y-5">
                <div className="flex items-start justify-between">
                  <div>
                    <DialogTitle className="text-lg font-semibold text-gray-900">Order {target.order_number}</DialogTitle>
                    <p className="text-xs text-gray-500">{formatDateTime(target.created_at)}</p>
                  </div>
                  <StatusBadge status={target.status} type="order" />
                </div>

                <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 text-sm">
                  <div className="rounded-lg border border-gray-200 p-3">
                    <p className="text-xs font-semibold uppercase text-gray-500 mb-1">Deliver to</p>
                    <p className="font-medium text-gray-900">{target.shipping_address?.name}</p>
                    <p className="text-gray-600">{target.shipping_address?.phone}</p>
                    <p className="text-gray-600">
                      {[target.shipping_address?.line1, target.shipping_address?.line2, target.shipping_address?.city, target.shipping_address?.state]
                        .filter(Boolean).join(', ')}
                    </p>
                  </div>
                  <div className="rounded-lg border border-gray-200 p-3">
                    <p className="text-xs font-semibold uppercase text-gray-500 mb-1">Payment</p>
                    <p className="text-gray-900">
                      {target.payment_method === 'COD' ? 'Cash on delivery' : `${(target.payment_provider || 'online').toUpperCase()} (manual)`}
                    </p>
                    <p className="text-gray-600">Status: {target.payment_status === 'PAID' ? 'Paid' : 'Unpaid'}</p>
                    {target.payment_reference && <p className="text-gray-600">TrxID: <span className="font-mono">{target.payment_reference}</span></p>}
                    {target.payer_number && <p className="text-gray-600">From: {target.payer_number}</p>}
                    {target.tracking_number && <p className="text-gray-600 mt-1">{target.carrier}: <span className="font-mono">{target.tracking_number}</span></p>}
                  </div>
                </div>

                <table className="w-full text-sm">
                  <thead>
                    <tr className="text-left text-xs uppercase text-gray-500 border-b">
                      <th className="py-2">Item</th><th className="py-2">Qty</th><th className="py-2 text-right">Total</th>
                    </tr>
                  </thead>
                  <tbody>
                    {target.items.map((it, i) => (
                      <tr key={i} className="border-b border-gray-100">
                        <td className="py-2">
                          <p className="font-medium text-gray-900">{it.name}</p>
                          <p className="text-xs text-gray-500">
                            {it.sku}{it.options && Object.keys(it.options).length > 0 ? ' · ' + Object.values(it.options).join(', ') : ''}
                          </p>
                        </td>
                        <td className="py-2">{it.quantity}</td>
                        <td className="py-2 text-right">{formatCurrency(it.pricing.total)}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>

                <div className="text-sm space-y-1 text-right">
                  <p className="text-gray-600">Subtotal {formatCurrency(target.pricing.subtotal)}</p>
                  {target.pricing.discount > 0 && <p className="text-emerald-700">Coupon {target.coupon_code} −{formatCurrency(target.pricing.discount)}</p>}
                  <p className="text-gray-600">Shipping {formatCurrency(target.pricing.shipping)}</p>
                  <p className="font-semibold text-gray-900">Total {formatCurrency(target.pricing.total)}</p>
                </div>

                <div>
                  <p className="text-xs font-semibold uppercase text-gray-500 mb-1">History</p>
                  <ul className="space-y-1 text-xs text-gray-600">
                    {target.status_history.map((h, i) => (
                      <li key={i}>{formatDateTime(h.changed_at)} — <span className="font-medium">{label(h.status)}</span>{h.reason ? ` (${h.reason})` : ''}</li>
                    ))}
                  </ul>
                </div>

                <div className="flex justify-end">
                  <button onClick={closeDialog} className="btn-secondary">Close</button>
                </div>
              </div>
            )}
          </DialogPanel>
        </div>
      </Dialog>

      {/* Ship */}
      <Dialog open={dialog === 'ship'} onClose={closeDialog} className="relative z-50">
        <div className="fixed inset-0 bg-black/30" aria-hidden="true" />
        <div className="fixed inset-0 flex items-center justify-center p-4">
          <DialogPanel className="w-full max-w-md rounded-2xl bg-white p-6 shadow-xl">
            <DialogTitle className="text-lg font-semibold text-gray-900">Ship order {target?.order_number}</DialogTitle>
            <p className="mt-1 text-sm text-gray-500">Stock was already deducted when the order was placed.</p>
            {actionError && <div className="mt-3 rounded-lg bg-red-50 border border-red-200 p-3 text-sm text-red-700">{actionError}</div>}
            <form
              className="mt-4 space-y-4"
              onSubmit={(e) => { e.preventDefault(); if (target) shipMutation.mutate({ id: target.id }); }}
            >
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
                <label className="label-field">Courier charge (৳, optional)</label>
                <input type="number" min="0" step="1" value={courierCost} onChange={(e) => setCourierCost(e.target.value)} className="input-field" placeholder="What the courier charged you" />
                <p className="mt-1 text-xs text-gray-500">Used in profit reports.</p>
              </div>
              <div className="flex justify-end gap-3 pt-2">
                <button type="button" onClick={closeDialog} className="btn-secondary">Cancel</button>
                <button type="submit" disabled={shipMutation.isPending || !tracking.trim()} className="btn-primary">
                  {shipMutation.isPending ? 'Shipping…' : 'Mark as shipped'}
                </button>
              </div>
            </form>
          </DialogPanel>
        </div>
      </Dialog>

      {/* Cancel / Refund */}
      <Dialog open={dialog === 'cancel' || dialog === 'refund'} onClose={closeDialog} className="relative z-50">
        <div className="fixed inset-0 bg-black/30" aria-hidden="true" />
        <div className="fixed inset-0 flex items-center justify-center p-4">
          <DialogPanel className="w-full max-w-md rounded-2xl bg-white p-6 shadow-xl">
            <DialogTitle className="text-lg font-semibold text-gray-900">
              {dialog === 'refund' ? 'Mark as refunded' : 'Cancel order'} {target?.order_number}
            </DialogTitle>
            <p className="mt-1 text-sm text-gray-500">
              {dialog === 'refund'
                ? 'Use this after you have sent the money back to the customer.'
                : 'Any stock taken for this order goes back to the warehouse.'}
            </p>
            {actionError && <div className="mt-3 rounded-lg bg-red-50 border border-red-200 p-3 text-sm text-red-700">{actionError}</div>}
            <form
              className="mt-4 space-y-4"
              onSubmit={(e) => {
                e.preventDefault();
                if (target) {
                  statusMutation.mutate({
                    id: target.id,
                    status: dialog === 'refund' ? 'REFUNDED' : 'CANCELLED',
                    reason: reason.trim() || undefined,
                  });
                }
              }}
            >
              <div>
                <label className="label-field">Reason</label>
                <textarea rows={3} value={reason} onChange={(e) => setReason(e.target.value)} className="input-field" placeholder="e.g. Customer asked to cancel" />
              </div>
              <div className="flex justify-end gap-3 pt-2">
                <button type="button" onClick={closeDialog} className="btn-secondary">Back</button>
                <button type="submit" disabled={statusMutation.isPending} className="btn-danger">
                  {statusMutation.isPending ? 'Saving…' : dialog === 'refund' ? 'Mark refunded' : 'Cancel order'}
                </button>
              </div>
            </form>
          </DialogPanel>
        </div>
      </Dialog>
    </div>
  );
}
