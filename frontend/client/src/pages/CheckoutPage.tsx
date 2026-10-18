import { useEffect, useMemo, useState } from 'react';
import { Link, Navigate, useLocation } from 'react-router-dom';
import {
  AlertCircle, ArrowLeft, Banknote, Check, Loader2, LockKeyhole, Minus, Plus, ShoppingBag, Smartphone, Tag, Trash2, Truck, X,
} from 'lucide-react';
import { formatPrice } from '@/utils/format';
import { useCart } from '@/context/CartContext';
import { useAuth } from '@/context/AuthContext';
import { useCatalog } from '@/context/CatalogContext';
import { useCheckoutConfig } from '@/hooks/useCheckoutConfig';
import { walletStyle, type WalletId } from '@/config/checkout';
import { placeOrder, quoteOrder, type ApiOrder, type Quote } from '@/api/orders';
import { ApiError } from '@/api/http';

const DIVISIONS = ['Barishal', 'Chattogram', 'Dhaka', 'Khulna', 'Mymensingh', 'Rajshahi', 'Rangpur', 'Sylhet'];
/** Delivery fee for a method and area; standard delivery is free from the shop's free-delivery amount. */
function feeFor(
  m: { id: string; fee_inside_dhaka: number; fee_outside_dhaka: number },
  area: string,
  subtotal: number,
  freeFrom: number,
): number {
  if (m.id === 'standard' && freeFrom > 0 && subtotal >= freeFrom) return 0;
  return area === 'outside_dhaka' ? m.fee_outside_dhaka : m.fee_inside_dhaka;
}

const PHONE_RE = /^(?:\+?88)?01[3-9]\d{8}$/;

const inputClass = (error?: string) =>
  `w-full rounded-lg border px-3 py-2.5 text-sm focus:outline-none focus:ring-1 ${
    error ? 'border-red-400 focus:border-red-500 focus:ring-red-500' : 'border-gray-300 focus:border-primary focus:ring-primary'
  }`;

function Field({ label, error, children }: { label: string; error?: string; children: React.ReactNode }) {
  return (
    <div data-error={error ? 'true' : undefined}>
      <label className="mb-1.5 block text-sm font-semibold text-gray-700">{label}</label>
      {children}
      {error && <p className="mt-1 text-xs text-red-600">{error}</p>}
    </div>
  );
}

export default function CheckoutPage() {
  const { lines, subtotal, updateQuantity, removeItem, clearCart } = useCart();
  const { user, loading: authLoading } = useAuth();
  const { reload: reloadCatalog } = useCatalog();
  const { config, error: configError } = useCheckoutConfig();
  const location = useLocation();

  const [shippingId, setShippingId] = useState('standard');
  const [area, setArea] = useState('inside_dhaka'); // delivery charges differ inside / outside Dhaka
  const [payment, setPayment] = useState<'COD' | WalletId>('COD');
  const [trxId, setTrxId] = useState('');
  const [payerNumber, setPayerNumber] = useState('');

  const [name, setName] = useState('');
  const [phone, setPhone] = useState('');
  const [address, setAddress] = useState('');
  const [city, setCity] = useState('');
  const [division, setDivision] = useState('');
  const [postcode, setPostcode] = useState('');
  const [errors, setErrors] = useState<Record<string, string>>({});

  // Coupon: the code the customer typed, and the one the server accepted.
  const [couponInput, setCouponInput] = useState('');
  const [couponCode, setCouponCode] = useState('');
  const [couponMessage, setCouponMessage] = useState<{ ok: boolean; text: string } | null>(null);
  const [quote, setQuote] = useState<Quote | null>(null);
  const [quoting, setQuoting] = useState(false);

  const [submitting, setSubmitting] = useState(false);
  const [submitError, setSubmitError] = useState('');
  const [placed, setPlaced] = useState<ApiOrder | null>(null);

  // Delivery fee preview. The server recalculates it, so this can never differ from what is charged.
  const delivery = useMemo(() => {
    const method = config?.shipping_methods.find((m) => m.id === shippingId);
    if (!config || !method) return 0;
    return feeFor(method, area, subtotal, config.free_shipping_threshold);
  }, [config, shippingId, area, subtotal]);
  // The server's quote (coupon included) is the truth; the local numbers are only a fallback.
  const discount = quote?.discount ?? 0;
  const deliveryShown = quote ? quote.shipping : delivery;
  const total = quote ? quote.total : subtotal + delivery;

  // If the shop switched off the method that was selected, fall back to the first available one.
  useEffect(() => {
    if (config && config.shipping_methods.length > 0 && !config.shipping_methods.some((m) => m.id === shippingId)) {
      setShippingId(config.shipping_methods[0].id);
    }
  }, [config, shippingId]);

  const wallets = config?.wallets ?? [];
  const wallet = wallets.find((w) => w.id === payment);
  const hasUnavailable = lines.some((l) => !l.variant.inStock);

  const lineKey = lines.map((l) => `${l.variant.id}x${l.quantity}`).join(',');
  useEffect(() => {
    if (!user || lines.length === 0) {
      setQuote(null);
      return;
    }
    let cancelled = false;
    setQuoting(true);
    quoteOrder({
      items: lines.map((l) => ({ product_id: l.product.backendId, variant_id: l.variant.id, quantity: l.quantity })),
      shipping_method: shippingId,
      delivery_area: area,
      coupon_code: couponCode || undefined,
    })
      .then((q) => {
        if (cancelled) return;
        setQuote(q);
        if (couponCode && q.coupon) {
          setCouponMessage({ ok: q.coupon.applied, text: q.coupon.message });
          if (!q.coupon.applied) setCouponCode(''); // not valid for this cart
        }
      })
      .catch(() => { if (!cancelled) setQuote(null); })
      .finally(() => { if (!cancelled) setQuoting(false); });
    return () => { cancelled = true; };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [user, lineKey, shippingId, area, couponCode]);

  const applyCoupon = () => {
    const code = couponInput.trim().toUpperCase();
    if (!code) return;
    setCouponMessage(null);
    setCouponCode(code); // the quote effect validates it with the server
  };
  const removeCoupon = () => {
    setCouponCode('');
    setCouponInput('');
    setCouponMessage(null);
  };

  if (placed) {
    return (
      <div className="min-h-screen bg-[#f7f8f6] flex items-center justify-center px-4">
        <div className="max-w-md w-full bg-white border border-gray-200 rounded-2xl p-10 text-center shadow-sm">
          <div className="mx-auto mb-5 w-14 h-14 rounded-full bg-primary/10 text-primary flex items-center justify-center"><Check /></div>
          <h1 className="font-headline text-3xl font-bold text-primary mb-3">Order placed</h1>
          <p className="text-gray-700">Your order number</p>
          <p className="font-mono text-lg font-semibold text-gray-900 mb-4">{placed.order_number}</p>
          <p className="text-gray-600 leading-relaxed mb-7">
            {placed.payment_method === 'COD'
              ? 'Pay in cash when your order arrives.'
              : 'We will confirm your payment shortly and then ship your order.'}
            {' '}Keep your order number and phone number to track the delivery.
          </p>
          <div className="flex flex-col gap-3 sm:flex-row sm:justify-center">
            <Link to="/orders" className="inline-flex items-center justify-center rounded-lg bg-primary text-white px-6 py-3 font-semibold hover:bg-primary-hover transition-colors">My orders</Link>
            <Link to="/products" className="inline-flex items-center justify-center rounded-lg border border-gray-300 px-6 py-3 font-semibold text-gray-700 hover:bg-gray-50">Continue shopping</Link>
          </div>
        </div>
      </div>
    );
  }

  if (authLoading) return <div className="min-h-screen flex items-center justify-center text-gray-500" aria-busy="true">Loading…</div>;
  if (!user) return <Navigate to="/signin" replace state={{ from: location.pathname }} />;

  if (!lines.length) {
    return (
      <div className="min-h-screen bg-[#f7f8f6] flex items-center justify-center px-4">
        <div className="text-center">
          <ShoppingBag className="mx-auto w-12 h-12 text-primary mb-4" />
          <h1 className="font-headline text-3xl font-bold text-primary mb-2">Your cart is empty</h1>
          <p className="text-gray-600 mb-6">Your cart is kept for 1 hour. Add some products to check out.</p>
          <Link to="/products" className="rounded-lg bg-primary text-white px-6 py-3 font-semibold">Browse products</Link>
        </div>
      </div>
    );
  }

  const validate = () => {
    const e: Record<string, string> = {};
    if (!name.trim()) e.name = 'Full name is required';
    if (!phone.trim()) e.phone = 'Phone number is required';
    else if (!PHONE_RE.test(phone.replace(/[\s-]/g, ''))) e.phone = 'Enter a valid mobile number, e.g. 01712345678';
    if (!division) e.division = 'Select your division';
    if (!city.trim()) e.city = 'City / district is required';
    if (!address.trim()) e.address = 'Street address is required';
    if (postcode.trim() && !/^\d{4}$/.test(postcode.trim())) e.postcode = 'Enter a 4-digit postcode';
    if (payment !== 'COD') {
      if (!trxId.trim()) e.trx = 'Enter the transaction ID (TrxID) from your payment';
      if (!PHONE_RE.test(payerNumber.replace(/[\s-]/g, ''))) e.payer = 'Enter the number you paid from';
    }
    return e;
  };

  const handlePlaceOrder = async () => {
    setSubmitError('');
    const e = validate();
    setErrors(e);
    if (Object.keys(e).length > 0) {
      document.querySelector('[data-error="true"]')?.scrollIntoView({ behavior: 'smooth', block: 'center' });
      return;
    }
    if (hasUnavailable) {
      setSubmitError('Remove the out-of-stock items from your cart first.');
      return;
    }

    setSubmitting(true);
    try {
      const order = await placeOrder({
        items: lines.map((l) => ({ product_id: l.product.backendId, variant_id: l.variant.id, quantity: l.quantity })),
        shipping_address: {
          name: name.trim(),
          phone: phone.replace(/[\s-]/g, ''),
          line1: address.trim(),
          city: city.trim(),
          state: division,
          postal_code: postcode.trim() || undefined,
          country: 'BD',
        },
        shipping_method: shippingId,
        delivery_area: area,
        coupon_code: quote?.coupon?.applied ? quote.coupon.code : undefined,
        payment_method: payment === 'COD' ? 'COD' : 'ONLINE',
        payment_provider: payment === 'COD' ? undefined : payment,
        payment_reference: payment === 'COD' ? undefined : trxId.trim(),
        payer_number: payment === 'COD' ? undefined : payerNumber.replace(/[\s-]/g, ''),
      });
      clearCart();
      reloadCatalog(); // stock changed
      setPlaced(order);
    } catch (err) {
      if (err instanceof ApiError && err.code === 'INSUFFICIENT_STOCK') {
        setSubmitError(`${err.message}. Please reduce the quantity or remove the item.`);
        reloadCatalog();
      } else {
        setSubmitError(err instanceof Error ? err.message : 'Could not place your order. Please try again.');
      }
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className="min-h-screen bg-[#f7f8f6]">
      <div className="max-w-6xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        <Link to="/products" className="inline-flex items-center gap-2 text-sm font-semibold text-gray-600 hover:text-primary mb-6">
          <ArrowLeft className="w-4 h-4" /> Continue shopping
        </Link>
        <h1 className="font-headline text-3xl font-bold text-primary mb-8 flex items-center gap-3"><LockKeyhole className="w-7 h-7" /> Checkout</h1>

        <div className="grid gap-8 lg:grid-cols-[1fr_380px]">
          <div className="space-y-6">
            {/* Delivery details */}
            <section className="bg-white rounded-2xl border border-gray-100 p-6 shadow-sm">
              <h2 className="font-headline text-xl font-bold mb-5">Delivery details</h2>
              <div className="grid gap-4 sm:grid-cols-2">
                <Field label="Full name" error={errors.name}>
                  <input value={name} onChange={(e) => setName(e.target.value)} className={inputClass(errors.name)} autoComplete="name" />
                </Field>
                <Field label="Mobile number" error={errors.phone}>
                  <input value={phone} onChange={(e) => setPhone(e.target.value)} placeholder="01712345678" className={inputClass(errors.phone)} autoComplete="tel" inputMode="tel" />
                </Field>
                <Field label="Division" error={errors.division}>
                  <select value={division} onChange={(e) => setDivision(e.target.value)} className={inputClass(errors.division)}>
                    <option value="">Select division</option>
                    {DIVISIONS.map((d) => <option key={d}>{d}</option>)}
                  </select>
                </Field>
                <Field label="City / District" error={errors.city}>
                  <input value={city} onChange={(e) => setCity(e.target.value)} className={inputClass(errors.city)} />
                </Field>
                <div className="sm:col-span-2">
                  <Field label="Street address" error={errors.address}>
                    <input value={address} onChange={(e) => setAddress(e.target.value)} placeholder="House, road, area" className={inputClass(errors.address)} autoComplete="street-address" />
                  </Field>
                </div>
                <Field label="Postcode (optional)" error={errors.postcode}>
                  <input value={postcode} onChange={(e) => setPostcode(e.target.value)} placeholder="1205" className={inputClass(errors.postcode)} inputMode="numeric" />
                </Field>
              </div>
            </section>

            {/* Delivery method */}
            <section className="bg-white rounded-2xl border border-gray-100 p-6 shadow-sm">
              <h2 className="font-headline text-xl font-bold mb-5 flex items-center gap-2"><Truck className="w-5 h-5 text-primary" /> Delivery method</h2>
              {configError && <p className="text-sm text-red-600 mb-3">{configError}</p>}
              <div className="mb-4 grid gap-3 sm:grid-cols-2" role="radiogroup" aria-label="Delivery area">
                {(config?.delivery_areas ?? []).map((a) => (
                  <label key={a.id} className={`flex cursor-pointer items-center gap-3 rounded-xl border p-3 text-sm font-semibold ${area === a.id ? 'border-primary bg-primary/5' : 'border-gray-200'}`}>
                    <input type="radio" name="area" checked={area === a.id} onChange={() => setArea(a.id)} className="accent-primary" />
                    {a.label}
                  </label>
                ))}
              </div>
              <div className="space-y-3">
                {(config?.shipping_methods ?? []).map((m) => {
                  const fee = feeFor(m, area, subtotal, config!.free_shipping_threshold);
                  return (
                    <label key={m.id} className={`flex cursor-pointer items-center justify-between rounded-xl border p-4 ${shippingId === m.id ? 'border-primary bg-primary/5' : 'border-gray-200'}`}>
                      <span className="flex items-center gap-3">
                        <input type="radio" name="shipping" checked={shippingId === m.id} onChange={() => setShippingId(m.id)} className="accent-primary" />
                        <span><span className="block font-semibold text-sm">{m.label}</span><span className="block text-xs text-gray-500">{m.detail}</span></span>
                      </span>
                      <strong className="text-sm">{fee === 0 ? 'Free' : formatPrice(fee)}</strong>
                    </label>
                  );
                })}
              </div>
              {config && config.free_shipping_threshold > 0 && subtotal < config.free_shipping_threshold && (
                <p className="mt-3 text-xs text-gray-500">Add {formatPrice(config.free_shipping_threshold - subtotal)} more for free standard delivery.</p>
              )}
            </section>

            {/* Payment */}
            <section className="bg-white rounded-2xl border border-gray-100 p-6 shadow-sm">
              <h2 className="font-headline text-xl font-bold mb-5">Payment</h2>
              <div className="space-y-3">
                {config?.cod_enabled !== false && (
                  <label className={`flex cursor-pointer items-center gap-3 rounded-xl border p-4 ${payment === 'COD' ? 'border-primary bg-primary/5' : 'border-gray-200'}`}>
                    <input type="radio" name="payment" checked={payment === 'COD'} onChange={() => setPayment('COD')} className="accent-primary" />
                    <Banknote className="w-5 h-5 text-green-700" />
                    <span><span className="block font-semibold text-sm">Cash on Delivery</span><span className="block text-xs text-gray-500">Pay in cash when you receive your order</span></span>
                  </label>
                )}
                {wallets.map((w) => {
                  const style = walletStyle[w.id as WalletId];
                  return (
                    <label key={w.id} className={`flex cursor-pointer items-center gap-3 rounded-xl border p-4 ${payment === w.id ? 'border-primary bg-primary/5' : 'border-gray-200'}`}>
                      <input type="radio" name="payment" checked={payment === w.id} onChange={() => setPayment(w.id as WalletId)} className="accent-primary" />
                      <Smartphone className={`w-5 h-5 ${style?.accent ?? 'text-gray-600'}`} />
                      <span><span className="block font-semibold text-sm">{w.label}</span><span className="block text-xs text-gray-500">{style?.desc ?? 'Send money, then enter the TrxID'}</span></span>
                    </label>
                  );
                })}
              </div>

              {wallet && (
                <div className="mt-5 rounded-xl bg-amber-50 border border-amber-200 p-4 text-sm text-amber-900">
                  <p className="font-semibold">How to pay with {wallet.label}</p>
                  <ol className="mt-2 list-decimal pl-5 space-y-1">
                    <li>Open your {wallet.label} app and choose <strong>Send Money</strong>.</li>
                    <li>Send <strong>{formatPrice(total)}</strong> to <strong className="font-mono">{wallet.number}</strong> ({wallet.type}).</li>
                    <li>Enter the transaction ID and the number you paid from below.</li>
                  </ol>
                  <div className="mt-4 grid gap-3 sm:grid-cols-2">
                    <Field label="Transaction ID (TrxID)" error={errors.trx}>
                      <input value={trxId} onChange={(e) => setTrxId(e.target.value)} className={inputClass(errors.trx)} placeholder="e.g. 9AB3C4D5EF" />
                    </Field>
                    <Field label={`Your ${wallet.label} number`} error={errors.payer}>
                      <input value={payerNumber} onChange={(e) => setPayerNumber(e.target.value)} className={inputClass(errors.payer)} placeholder="01712345678" inputMode="tel" />
                    </Field>
                  </div>
                  <p className="mt-3 text-xs">We verify the payment manually, then ship your order.</p>
                </div>
              )}
            </section>
          </div>

          {/* Summary */}
          <aside className="lg:sticky lg:top-6 h-fit">
            <div className="bg-white rounded-2xl border border-gray-100 p-6 shadow-sm">
              <h2 className="font-headline text-xl font-bold mb-5">Order summary</h2>
              <div className="space-y-4 max-h-80 overflow-y-auto pr-1">
                {lines.map(({ product, variant, quantity }) => (
                  <div key={variant.id} className="flex gap-3">
                    <img src={variant.image || product.image} alt={product.name} className="w-16 h-16 rounded-lg object-cover flex-shrink-0" />
                    <div className="flex-1 min-w-0">
                      <p className="text-sm font-semibold line-clamp-2">{product.name}</p>
                      {product.hasChoices && <p className="text-xs text-gray-500">{variant.label}</p>}
                      {!variant.inStock && <p className="text-xs font-semibold text-red-600">Out of stock</p>}
                      <div className="mt-1.5 flex items-center justify-between">
                        <div className="flex items-center rounded-md border border-gray-200">
                          <button type="button" onClick={() => updateQuantity(variant.id, quantity - 1)} className="p-1.5 text-gray-500 hover:text-primary" aria-label="Decrease quantity"><Minus className="w-3.5 h-3.5" /></button>
                          <span className="w-7 text-center text-sm">{quantity}</span>
                          <button type="button" onClick={() => updateQuantity(variant.id, quantity + 1)} className="p-1.5 text-gray-500 hover:text-primary" aria-label="Increase quantity"><Plus className="w-3.5 h-3.5" /></button>
                        </div>
                        <strong className="text-sm text-primary">{formatPrice(variant.price * quantity)}</strong>
                      </div>
                    </div>
                    <button type="button" onClick={() => removeItem(variant.id)} className="self-start p-1 text-gray-400 hover:text-red-600" aria-label="Remove item"><Trash2 className="w-4 h-4" /></button>
                  </div>
                ))}
              </div>

              {/* Coupon */}
              <div className="mt-5 border-t border-gray-100 pt-4">
                {quote?.coupon?.applied ? (
                  <div className="flex items-center justify-between rounded-lg bg-emerald-50 px-3 py-2 text-sm text-emerald-800">
                    <span className="flex items-center gap-2"><Tag className="w-4 h-4" /> <strong>{quote.coupon.code}</strong> · {quote.coupon.message}</span>
                    <button type="button" onClick={removeCoupon} aria-label="Remove coupon" className="text-emerald-700 hover:text-red-600"><X className="w-4 h-4" /></button>
                  </div>
                ) : (
                  <form onSubmit={(e) => { e.preventDefault(); applyCoupon(); }} className="flex gap-2">
                    <input value={couponInput} onChange={(e) => setCouponInput(e.target.value)} placeholder="Coupon code" className="min-w-0 flex-1 rounded-lg border border-gray-300 px-3 py-2 text-sm uppercase focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary" aria-label="Coupon code" />
                    <button type="submit" disabled={quoting || !couponInput.trim()} className="rounded-lg border border-primary px-4 py-2 text-sm font-semibold text-primary hover:bg-primary/5 disabled:opacity-50">Apply</button>
                  </form>
                )}
                {couponMessage && !couponMessage.ok && <p className="mt-2 text-xs text-red-600">{couponMessage.text}</p>}
              </div>

              <dl className="mt-4 space-y-2 border-t border-gray-100 pt-4 text-sm">
                <div className="flex justify-between"><dt className="text-gray-600">Subtotal</dt><dd>{formatPrice(subtotal)}</dd></div>
                {discount > 0 && <div className="flex justify-between text-emerald-700"><dt>Coupon {quote?.coupon?.code}</dt><dd>−{formatPrice(discount)}</dd></div>}
                <div className="flex justify-between"><dt className="text-gray-600">Delivery</dt><dd>{deliveryShown === 0 ? 'Free' : formatPrice(deliveryShown)}</dd></div>
                <div className="flex justify-between border-t border-gray-100 pt-3 text-base font-bold"><dt>Total</dt><dd className="text-primary">{formatPrice(total)}</dd></div>
              </dl>

              {submitError && (
                <div className="mt-4 flex items-start gap-2 rounded-lg border border-red-200 bg-red-50 px-3 py-2.5 text-sm text-red-700">
                  <AlertCircle className="w-4 h-4 mt-0.5 flex-shrink-0" /><span>{submitError}</span>
                </div>
              )}

              <button
                type="button"
                onClick={handlePlaceOrder}
                disabled={submitting || !config || hasUnavailable}
                className="mt-5 flex w-full items-center justify-center gap-2 rounded-lg bg-primary py-3.5 font-semibold text-white transition-colors hover:bg-primary-hover disabled:opacity-60"
              >
                {submitting && <Loader2 className="w-4 h-4 animate-spin" />}
                {payment === 'COD' ? 'Place order' : 'Place order & confirm payment'}
              </button>
              <p className="mt-3 text-center text-xs text-gray-400">Prices, coupon and stock are checked again when you place the order.</p>
            </div>
          </aside>
        </div>
      </div>
    </div>
  );
}
