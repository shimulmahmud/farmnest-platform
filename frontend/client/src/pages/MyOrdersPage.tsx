import { useEffect, useState } from 'react';
import { Link, Navigate, useLocation } from 'react-router-dom';
import { AlertCircle, ChevronDown, Loader2, PackageOpen, Truck } from 'lucide-react';
import { fetchMyOrders, type ApiOrder } from '@/api/orders';
import { useAuth } from '@/context/AuthContext';
import { useCatalog } from '@/context/CatalogContext';
import OrderTimeline, { statusLabel, statusTone } from '@/components/OrderTimeline';
import { formatPrice } from '@/utils/format';

export default function MyOrdersPage() {
  const { user, loading: authLoading } = useAuth();
  const location = useLocation();
  const { products } = useCatalog();
  const [orders, setOrders] = useState<ApiOrder[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [open, setOpen] = useState<string | null>(null);

  useEffect(() => {
    if (!user) return;
    setLoading(true);
    fetchMyOrders()
      .then((page) => setOrders(page.items))
      .catch((e: unknown) => setError(e instanceof Error ? e.message : 'Could not load your orders'))
      .finally(() => setLoading(false));
  }, [user]);

  if (authLoading) return <div className="min-h-[60vh] flex items-center justify-center text-gray-500" aria-busy="true">Loading…</div>;
  if (!user) return <Navigate to="/signin" replace state={{ from: location.pathname }} />;

  return (
    <div className="bg-[#f7f8f6] min-h-[70vh] py-10">
      <div className="max-w-3xl mx-auto px-4 sm:px-6">
        <h1 className="font-headline text-3xl font-bold text-primary mb-6">My orders</h1>

        {loading && <p className="flex items-center gap-2 text-gray-500"><Loader2 className="w-4 h-4 animate-spin" /> Loading…</p>}
        {error && (
          <div className="flex items-start gap-2 rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700">
            <AlertCircle className="w-4 h-4 mt-0.5 flex-shrink-0" /><span>{error}</span>
          </div>
        )}
        {!loading && !error && orders.length === 0 && (
          <div className="text-center bg-white rounded-2xl border border-gray-100 p-10">
            <PackageOpen className="mx-auto w-10 h-10 text-gray-300 mb-3" />
            <p className="font-semibold text-gray-700">You have not placed any orders yet.</p>
            <Link to="/products" className="mt-4 inline-block rounded-lg bg-primary px-5 py-2.5 font-semibold text-white">Start shopping</Link>
          </div>
        )}

        <div className="space-y-4">
          {orders.map((o) => {
            const expanded = open === o.id;
            return (
              <article key={o.id} className="bg-white rounded-2xl border border-gray-100 shadow-sm">
                <button onClick={() => setOpen(expanded ? null : o.id)} className="flex w-full items-center justify-between gap-3 p-5 text-left" aria-expanded={expanded}>
                  <div>
                    <p className="font-mono font-semibold">{o.order_number}</p>
                    <p className="text-xs text-gray-500">
                      {new Date(o.created_at).toLocaleDateString('en-GB', { dateStyle: 'medium' })} · {o.items.reduce((n, i) => n + i.quantity, 0)} items · {formatPrice(o.pricing.total)}
                    </p>
                  </div>
                  <span className="flex items-center gap-3">
                    <span className={`rounded-full border px-3 py-1 text-xs font-semibold ${statusTone(o.status)}`}>{statusLabel(o.status)}</span>
                    <ChevronDown className={`w-4 h-4 text-gray-400 transition-transform ${expanded ? 'rotate-180' : ''}`} />
                  </span>
                </button>

                {expanded && (
                  <div className="border-t border-gray-100 p-5 space-y-5">
                    {o.tracking_number && (
                      <div className="flex items-center gap-3 rounded-lg bg-sky-50 p-3 text-sm text-sky-900">
                        <Truck className="w-5 h-5 flex-shrink-0" />
                        <span>Courier: <strong>{o.carrier}</strong> · Tracking number: <strong className="font-mono">{o.tracking_number}</strong></span>
                      </div>
                    )}
                    <ul className="divide-y divide-gray-100">
                      {o.items.map((it, i) => (
                        <li key={i} className="flex items-center gap-3 py-2.5">
                          {it.image_url && <img src={it.image_url} alt="" className="w-12 h-12 rounded-lg object-cover" />}
                          <div className="flex-1 min-w-0">
                            <p className="text-sm font-semibold line-clamp-2">{it.name}</p>
                            <p className="text-xs text-gray-500">
                              {it.options && Object.keys(it.options).length > 0 ? Object.values(it.options).join(' · ') + ' · ' : ''}Qty {it.quantity}
                            </p>
                          </div>
                          <strong className="text-sm">{formatPrice(it.pricing.total)}</strong>
                          {o.status === 'DELIVERED' && products.find((p) => p.backendId === it.product_id) && (
                            <Link to={`/products/${products.find((p) => p.backendId === it.product_id)!.id}#reviews`} className="ml-2 text-xs font-semibold text-primary hover:underline whitespace-nowrap">Write a review</Link>
                          )}
                        </li>
                      ))}
                    </ul>
                    <p className="text-sm text-gray-600">
                      Payment: {o.payment_method === 'COD' ? 'Cash on delivery' : `${(o.payment_provider ?? 'online').toUpperCase()}`} · {o.payment_status === 'PAID' ? 'Paid' : 'Unpaid'}
                      {o.pricing.discount > 0 ? ` · Coupon ${o.coupon_code ?? ''} −${formatPrice(o.pricing.discount)}` : ''}
                      {' · '}Delivery {o.pricing.shipping === 0 ? 'free' : formatPrice(o.pricing.shipping)}
                    </p>
                    <OrderTimeline history={o.status_history} />
                    <Link to={`/support?order=${encodeURIComponent(o.order_number)}`} className="inline-block text-sm font-semibold text-primary hover:underline">Need help with this order?</Link>
                  </div>
                )}
              </article>
            );
          })}
        </div>
      </div>
    </div>
  );
}
