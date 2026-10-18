import { useState } from 'react';
import { Link } from 'react-router-dom';
import { AlertCircle, Loader2, PackageSearch, Truck } from 'lucide-react';
import { trackOrder, type TrackedOrder } from '@/api/orders';
import { ApiError } from '@/api/http';
import OrderTimeline, { statusLabel, statusTone } from '@/components/OrderTimeline';
import { formatPrice } from '@/utils/format';

export default function TrackOrderPage() {
  const [orderNumber, setOrderNumber] = useState('');
  const [phone, setPhone] = useState('');
  const [order, setOrder] = useState<TrackedOrder | null>(null);
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setOrder(null);
    if (!orderNumber.trim() || !phone.trim()) {
      setError('Enter your order number and the mobile number used for delivery.');
      return;
    }
    setLoading(true);
    try {
      setOrder(await trackOrder(orderNumber.trim(), phone.trim()));
    } catch (err) {
      setError(
        err instanceof ApiError && err.status === 404
          ? 'We could not find an order with those details. Check the order number and phone number.'
          : err instanceof Error ? err.message : 'Something went wrong',
      );
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="bg-[#f7f8f6] min-h-[70vh] py-12">
      <div className="max-w-3xl mx-auto px-4 sm:px-6">
        <div className="text-center mb-8">
          <PackageSearch className="mx-auto w-10 h-10 text-primary mb-3" />
          <h1 className="font-headline text-3xl font-bold text-primary">Track your order</h1>
          <p className="text-gray-600 mt-2">Enter your order number and the mobile number you gave for delivery.</p>
          <p className="text-sm text-gray-500 mt-1">Signed in? See all your orders in <Link to="/orders" className="text-primary font-semibold hover:underline">My orders</Link>.</p>
        </div>

        <form onSubmit={handleSubmit} className="bg-white rounded-2xl border border-gray-100 p-6 shadow-sm grid gap-4 sm:grid-cols-[1fr_1fr_auto] items-end">
          <div>
            <label className="block text-sm font-semibold text-gray-700 mb-1.5">Order number</label>
            <input value={orderNumber} onChange={(e) => setOrderNumber(e.target.value)} placeholder="e.g. ORD-20261009-ABC123" className="w-full rounded-lg border border-gray-300 px-3 py-2.5 text-sm focus:outline-none focus:border-primary focus:ring-1 focus:ring-primary" />
          </div>
          <div>
            <label className="block text-sm font-semibold text-gray-700 mb-1.5">Mobile number</label>
            <input value={phone} onChange={(e) => setPhone(e.target.value)} placeholder="01712345678" inputMode="tel" className="w-full rounded-lg border border-gray-300 px-3 py-2.5 text-sm focus:outline-none focus:border-primary focus:ring-1 focus:ring-primary" />
          </div>
          <button type="submit" disabled={loading} className="flex items-center justify-center gap-2 rounded-lg bg-primary px-6 py-2.5 font-semibold text-white hover:bg-primary-hover disabled:opacity-60">
            {loading && <Loader2 className="w-4 h-4 animate-spin" />} Track
          </button>
        </form>

        {error && (
          <div className="mt-5 flex items-start gap-2 rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700">
            <AlertCircle className="w-4 h-4 mt-0.5 flex-shrink-0" /><span>{error}</span>
          </div>
        )}

        {order && (
          <div className="mt-6 space-y-5">
            <div className="bg-white rounded-2xl border border-gray-100 p-6 shadow-sm">
              <div className="flex flex-wrap items-start justify-between gap-3">
                <div>
                  <p className="text-xs text-gray-500">Order</p>
                  <p className="font-mono text-lg font-semibold">{order.order_number}</p>
                  <p className="text-xs text-gray-500 mt-1">
                    Placed {new Date(order.created_at).toLocaleDateString('en-GB', { dateStyle: 'long' })}
                    {order.city ? ` · Delivering to ${order.city}` : ''}
                  </p>
                </div>
                <span className={`rounded-full border px-3 py-1 text-xs font-semibold ${statusTone(order.status)}`}>{statusLabel(order.status)}</span>
              </div>

              {order.tracking_number && (
                <div className="mt-4 flex items-center gap-3 rounded-lg bg-sky-50 p-3 text-sm text-sky-900">
                  <Truck className="w-5 h-5 flex-shrink-0" />
                  <span>Courier: <strong>{order.carrier}</strong> · Tracking number: <strong className="font-mono">{order.tracking_number}</strong></span>
                </div>
              )}

              <p className="mt-4 text-sm text-gray-600">
                Payment: {order.payment_method === 'COD' ? 'Cash on delivery' : 'Online'} · {order.payment_status === 'PAID' ? 'Paid' : 'Not paid yet'}
              </p>
            </div>

            <div className="bg-white rounded-2xl border border-gray-100 p-6 shadow-sm">
              <h2 className="font-headline text-lg font-bold mb-4">Progress</h2>
              <OrderTimeline history={order.status_history} />
            </div>

            <div className="bg-white rounded-2xl border border-gray-100 p-6 shadow-sm">
              <h2 className="font-headline text-lg font-bold mb-4">Items</h2>
              <ul className="divide-y divide-gray-100">
                {order.items.map((it, i) => (
                  <li key={i} className="flex items-center gap-3 py-3">
                    {it.image_url && <img src={it.image_url} alt="" className="w-14 h-14 rounded-lg object-cover" />}
                    <div className="flex-1 min-w-0">
                      <p className="text-sm font-semibold line-clamp-2">{it.name}</p>
                      <p className="text-xs text-gray-500">Qty {it.quantity}</p>
                    </div>
                    <strong className="text-sm">{formatPrice(it.total)}</strong>
                  </li>
                ))}
              </ul>
              <dl className="mt-3 space-y-1 border-t border-gray-100 pt-3 text-sm">
                <div className="flex justify-between"><dt className="text-gray-600">Subtotal</dt><dd>{formatPrice(order.pricing.subtotal)}</dd></div>
                {order.pricing.discount > 0 && <div className="flex justify-between text-emerald-700"><dt>Coupon discount</dt><dd>−{formatPrice(order.pricing.discount)}</dd></div>}
                <div className="flex justify-between"><dt className="text-gray-600">Delivery</dt><dd>{order.pricing.shipping === 0 ? 'Free' : formatPrice(order.pricing.shipping)}</dd></div>
                <div className="flex justify-between font-bold"><dt>Total</dt><dd className="text-primary">{formatPrice(order.pricing.total)}</dd></div>
              </dl>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
