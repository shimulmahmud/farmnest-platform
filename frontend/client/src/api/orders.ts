import { get, getPage, post } from './http';

export interface CheckoutConfig {
  currency: string;
  shipping_methods: { id: string; label: string; detail: string; fee_inside_dhaka: number; fee_outside_dhaka: number }[];
  delivery_areas: { id: string; label: string }[];
  free_shipping_threshold: number;
  cod_enabled: boolean;
  wallets: { id: string; label: string; number: string; type: string }[];
}

export interface ApiOrderItem {
  product_id: string;
  variant_id: string;
  name: string;
  sku: string;
  image_url?: string;
  options?: Record<string, string>;
  quantity: number;
  pricing: { unit_price: number; discount: number; total: number };
}

export interface ApiOrder {
  id: string;
  order_number: string;
  items: ApiOrderItem[];
  pricing: { subtotal: number; discount: number; tax: number; shipping: number; total: number };
  shipping_address?: { name?: string; phone?: string; line1: string; line2?: string; city: string; state: string; postal_code?: string };
  payment_method: 'COD' | 'ONLINE';
  payment_status: 'UNPAID' | 'PAID';
  payment_provider?: string;
  payment_reference?: string;
  shipping_method?: string;
  delivery_area?: string;
  coupon_code?: string;
  carrier?: string;
  tracking_number?: string;
  status: string;
  status_history: { status: string; changed_at: string; reason: string }[];
  created_at: string;
}

export interface PlaceOrderInput {
  items: { product_id: string; variant_id: string; quantity: number }[];
  shipping_address: {
    name: string;
    phone: string;
    line1: string;
    line2?: string;
    city: string;
    state: string;
    postal_code?: string;
    country: string;
  };
  shipping_method: string;
  delivery_area: string;
  payment_method: 'COD' | 'ONLINE';
  payment_provider?: string;
  payment_reference?: string;
  payer_number?: string;
  coupon_code?: string;
}

export interface Quote {
  subtotal: number;
  discount: number;
  shipping: number;
  total: number;
  shipping_method: string;
  coupon?: { code: string; applied: boolean; message: string; free_shipping: boolean };
}

export const fetchCheckoutConfig = () => get<CheckoutConfig>('/checkout/config');

/** What the server would charge right now for this cart (prices, delivery, coupon). Does not touch stock. */
export const quoteOrder = (input: { items: PlaceOrderInput['items']; shipping_method: string; delivery_area: string; coupon_code?: string }) =>
  post<Quote>('/checkout/quote', input);

export const placeOrder = (input: PlaceOrderInput) => post<ApiOrder>('/orders', input);

export const fetchMyOrders = (page = 1) => getPage<ApiOrder>('/orders', { page, per_page: 20 }, true);

export const fetchMyOrder = (id: string) => get<ApiOrder>(`/orders/${id}`, undefined, true);

export interface TrackedOrder {
  order_number: string;
  status: string;
  payment_method: string;
  payment_status: string;
  created_at: string;
  pricing: ApiOrder['pricing'];
  items: { name: string; image_url?: string; quantity: number; total: number }[];
  carrier?: string;
  tracking_number?: string;
  city?: string;
  status_history: { status: string; changed_at: string; reason: string }[];
}

export const trackOrder = (orderNumber: string, phone: string) =>
  get<TrackedOrder>('/track-order', { order_number: orderNumber, phone });
