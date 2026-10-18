// ============================================================================
// Shared Types
// ============================================================================

export interface User {
  id: string;
  email: string;
  first_name: string;
  last_name: string;
  role: string;
  status: string;
  email_verified: boolean;
  created_at: string;
}

export interface MessageResponse {
  message: string;
}

export interface Pagination {
  page: number;
  per_page: number;
  total: number;
  total_pages: number;
}

export interface PaginatedResponse<T> {
  data: T[];
  pagination: Pagination;
}

export interface StatusHistory {
  status: string;
  changed_at: string;
  reason: string;
}

export interface Address {
  name?: string;
  phone?: string;
  line1: string;
  line2?: string;
  city: string;
  state: string;
  postal_code: string;
  country: string;
}

export interface SEO {
  title: string;
  description: string;
  keywords: string[];
}

export interface Image {
  url: string;
  alt?: string;
  sort_order?: number;
}

// ============================================================================
// Auth Types
// ============================================================================

export interface TokenResponse {
  access_token: string;
  refresh_token: string;
  expires_at: string;
}

export interface AuthResponse {
  user: User;
  tokens: TokenResponse;
}

export interface LoginRequest {
  email: string;
  password: string;
}

export interface RegisterRequest {
  email: string;
  password: string;
  first_name: string;
  last_name: string;
}

export interface ForgotPasswordRequest {
  email: string;
}

export interface ResetPasswordRequest {
  token: string;
  new_password: string;
}

// ============================================================================
// Product Types
// ============================================================================

/** Money is whole BDT taka (no paisa). Weight is grams. */
export interface VariantResponse {
  id: string;
  sku: string;
  barcode?: string;
  options: Record<string, string>;
  price: number;
  compare_at_price?: number;
  cost_price: number;
  weight_g: number;
  requires_shipping: boolean;
  images: Image[];
  /** Sellable stock from inventory (admin list/detail only). */
  available?: number;
  status: string;
  position: number;
  created_at: string;
  updated_at: string;
}

export interface ProductResponse {
  id: string;
  vendor_id?: string;
  name: string;
  slug: string;
  description: string;
  short_description: string;
  brand: string;
  category_id: string;
  unit: string;
  currency: string;
  specs: Record<string, string>;
  tags: string[];
  images: Image[];
  variants: VariantResponse[];
  status: string;
  seo: SEO;
  sort_order: number;
  lowest_price: number;
  /** Sellable stock across variants (admin list/detail only). */
  total_available?: number;
  created_at: string;
  updated_at: string;
}

export interface CreateVariantRequest {
  sku: string;
  barcode?: string;
  options?: Record<string, string>;
  price: number;
  compare_at_price?: number;
  cost_price: number;
  weight_g?: number;
  requires_shipping?: boolean;
  images?: Image[];
}

export interface CreateProductRequest {
  status?: string;
  name: string;
  slug: string;
  description?: string;
  short_description?: string;
  brand?: string;
  category_id: string;
  unit?: string;
  currency?: string;
  specs?: Record<string, string>;
  tags?: string[];
  images?: Image[];
  variants: CreateVariantRequest[];
  seo?: SEO;
  sort_order?: number;
}

export interface UpdateProductRequest {
  name?: string;
  slug?: string;
  description?: string;
  short_description?: string;
  brand?: string;
  category_id?: string;
  unit?: string;
  currency?: string;
  specs?: Record<string, string>;
  tags?: string[];
  images?: Image[];
  variants?: CreateVariantRequest[];
  seo?: SEO;
  sort_order?: number;
  status?: string;
}

export interface SearchProductsParams {
  q?: string;
  category_id?: string;
  brand?: string;
  tags?: string;
  min_price?: number;
  max_price?: number;
  status?: string;
  sort_by?: string;
  page?: number;
  per_page?: number;
}

// ============================================================================
// Category Types
// ============================================================================

export interface CategoryResponse {
  id: string;
  name: string;
  slug: string;
  description?: string;
  parent_id?: string;
  image?: string;
  sort_order: number;
  status: string;
  seo: SEO;
  created_at: string;
  updated_at: string;
}

export interface CreateCategoryRequest {
  name: string;
  slug: string;
  description?: string;
  parent_id?: string;
  image?: string;
  sort_order?: number;
  seo?: SEO;
}

export interface UpdateCategoryRequest {
  name?: string;
  slug?: string;
  description?: string;
  parent_id?: string;
  image?: string;
  sort_order?: number;
  seo?: SEO;
  status?: string;
}

// ============================================================================
// Order Types
// ============================================================================

export interface OrderItemPricing {
  unit_price: number;
  discount: number;
  total: number;
}

export interface OrderItemResponse {
  product_id: string;
  variant_id: string;
  name: string;
  sku: string;
  image_url?: string;
  options?: Record<string, string>;
  quantity: number;
  pricing: OrderItemPricing;
}

export interface OrderPricingResponse {
  subtotal: number;
  discount: number;
  tax: number;
  shipping: number;
  total: number;
}

export interface OrderResponse {
  id: string;
  order_number: string;
  customer_id: string;
  items: OrderItemResponse[];
  pricing: OrderPricingResponse;
  shipping_address?: Address;
  billing_address?: Address;
  payment_method: 'COD' | 'ONLINE';
  payment_status: 'UNPAID' | 'PAID';
  payment_provider?: string;
  payment_reference?: string;
  payer_number?: string;
  shipping_method?: string;
  delivery_area?: string;
  /** What the courier charged the shop (admin views only). */
  courier_cost?: number;
  coupon_code?: string;
  carrier?: string;
  tracking_number?: string;
  shipped_at?: string;
  shipment_reference?: string;
  status: string;
  status_history: StatusHistory[];
  created_at: string;
  updated_at: string;
}

export interface CreateOrderItemRequest {
  product_id: string;
  variant_id: string;
  quantity: number;
}

export interface CreateOrderRequest {
  items: CreateOrderItemRequest[];
  shipping_address: Address;
  payment_method: 'COD' | 'ONLINE';
  shipping_method?: string;
  payment_provider?: string;
  payment_reference?: string;
  payer_number?: string;
}

export interface UpdateOrderStatusRequest {
  status: string;
  reason?: string;
  payment_reference?: string;
  shipment_reference?: string;
}

// ============================================================================
// Payment Types
// ============================================================================

export interface PaymentResponse {
  id: string;
  order_id: string;
  provider: string;
  provider_reference: string;
  amount: number;
  currency: string;
  status: string;
  metadata?: Record<string, unknown>;
  created_at: string;
  updated_at: string;
}

// ============================================================================
// Inventory Types
// ============================================================================

export interface StockResponse {
  id: string;
  sku: string;
  warehouse_id: string;
  total_stock: number;
  reserved_stock: number;
  available_stock: number;
  low_stock_threshold: number;
  is_low_stock: boolean;
  created_at: string;
  updated_at: string;
}

export interface CreateStockRequest {
  sku: string;
  warehouse_id: string;
  total_stock: number;
  low_stock_threshold?: number;
}

export interface AdjustStockRequest {
  sku: string;
  warehouse_id: string;
  quantity: number;
  reason: string;
}

export interface ReserveRequest {
  sku: string;
  warehouse_id: string;
  quantity: number;
  reference: string;
  ttl_seconds?: number;
}

export interface ReservationResponse {
  id: string;
  sku: string;
  warehouse_id: string;
  quantity: number;
  status: string;
  reference: string;
  expires_at?: string;
  created_at: string;
  updated_at: string;
}

export interface ReserveResponse {
  reservation: ReservationResponse;
  stock: StockResponse;
}

export interface MovementResponse {
  id: string;
  sku: string;
  warehouse_id: string;
  type: string;
  quantity: number;
  reference: string;
  reason: string;
  created_at: string;
}

export interface WarehouseResponse {
  id: string;
  name: string;
  code: string;
  address: Address;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

export interface CreateWarehouseRequest {
  name: string;
  code: string;
  address: Address;
}

// ============================================================================
// Review Types
// ============================================================================

export interface ReviewResponse {
  id: string;
  product_id: string;
  customer_id: string;
  author_name?: string;
  rating: number;
  title: string;
  comment: string;
  images: Image[];
  verified_purchase: boolean;
  status: string;
  created_at: string;
  updated_at: string;
}

export interface CreateReviewRequest {
  rating: number;
  title: string;
  comment?: string;
  images?: Image[];
}

export interface UpdateReviewRequest {
  rating?: number;
  title?: string;
  comment?: string;
  images?: Image[];
}

// ============================================================================
// Shipping Types
// ============================================================================

export interface ShippingRateResponse {
  method: string;
  carrier: string;
  service_level: string;
  cost_cents: number;
  currency: string;
  estimated_days: number;
  estimated_date?: string;
}

export interface TrackingEventResponse {
  status: string;
  description: string;
  location?: string;
  timestamp: string;
}

export interface ShipmentResponse {
  id: string;
  shipment_number: string;
  order_id: string;
  customer_id: string;
  method: string;
  carrier: string;
  tracking_number?: string;
  provider_ref?: string;
  destination: Address;
  packages: unknown[];
  cost_cents: number;
  currency: string;
  estimated_days: number;
  estimated_delivery?: string;
  actual_delivery?: string;
  tracking_events: TrackingEventResponse[];
  status: string;
  status_history: StatusHistory[];
  created_at: string;
  updated_at: string;
}

export interface CreateShipmentRequest {
  order_id: string;
  customer_id: string;
  method: string;
  carrier: string;
  destination: Address;
  packages: unknown[];
  currency?: string;
}

// ============================================================================
// Wishlist Types
// ============================================================================

export interface WishlistItemResponse {
  id: string;
  product_id: string;
  variant_id?: string;
  added_at: string;
}

export interface WishlistResponse {
  id: string;
  customer_id: string;
  items: WishlistItemResponse[];
  created_at: string;
  updated_at: string;
}

export interface AddWishlistItemRequest {
  product_id: string;
  variant_id?: string;
}

// ============================================================================
// Marketing Types
// ============================================================================

export interface CampaignMetrics {
  events_received: number;
  customers_reached: number;
  messages_sent: number;
  conversions: number;
  revenue_cents: number;
}

export interface CampaignResponse {
  id: string;
  name: string;
  type: string;
  status: string;
  description?: string;
  rules: unknown;
  metrics: CampaignMetrics;
  start_at?: string;
  end_at?: string;
  created_at: string;
  updated_at: string;
}

export interface CreateCampaignRequest {
  name: string;
  type: string;
  description?: string;
  rules: unknown;
  start_at?: string;
  end_at?: string;
}

export interface UpdateCampaignRequest {
  name?: string;
  type?: string;
  description?: string;
  rules?: unknown;
  start_at?: string;
  end_at?: string;
}

export interface SegmentRule {
  field: string;
  operator: string;
  value: unknown;
}

export interface SegmentResponse {
  id: string;
  name: string;
  type: string;
  description?: string;
  rules: SegmentRule[];
  customer_ids?: string[];
  customer_count: number;
  created_at: string;
  updated_at: string;
}

export interface CreateSegmentRequest {
  name: string;
  type: string;
  description?: string;
  rules: SegmentRule[];
}

export interface UpdateSegmentRequest {
  name?: string;
  type?: string;
  description?: string;
  rules?: SegmentRule[];
}

export interface PromotionResponse {
  id: string;
  name: string;
  code: string;
  description?: string;
  status: string;
  discount_type: string;
  discount_value: number;
  min_order_cents?: number;
  max_discount_cents?: number;
  usage_limit?: number;
  usage_count: number;
  applies_to_product_ids?: string[];
  applies_to_category_ids?: string[];
  campaign_id?: string;
  start_at?: string;
  end_at?: string;
  created_at: string;
  updated_at: string;
}

export interface CreatePromotionRequest {
  name: string;
  code: string;
  description?: string;
  discount_type: string;
  discount_value: number;
  min_order_cents?: number;
  max_discount_cents?: number;
  usage_limit?: number;
  applies_to_product_ids?: string[];
  applies_to_category_ids?: string[];
  campaign_id?: string;
  start_at?: string;
  end_at?: string;
}

export interface UpdatePromotionRequest {
  name?: string;
  code?: string;
  description?: string;
  discount_type?: string;
  discount_value?: number;
  min_order_cents?: number;
  max_discount_cents?: number;
  usage_limit?: number;
  applies_to_product_ids?: string[];
  applies_to_category_ids?: string[];
  campaign_id?: string;
  start_at?: string;
  end_at?: string;
}

// ============================================================================
// Analytics Types
// ============================================================================

export interface OverviewResponse {
  product_views: number;
  unique_viewers: number;
  carts_created: number;
  checkouts_started: number;
  orders_completed: number;
  conversion_rate: number;
  revenue_cents: number;
  start_at: string;
  end_at: string;
}

export interface TopProductResponse {
  product_id: string;
  views: number;
}

export interface DailyCountResponse {
  date: string;
  count: number;
}

// ============================================================================
// Admin Types
// ============================================================================

export interface DashboardCounts {
  total_users: number;
  total_customers: number;
  pending_reviews: number;
}

export interface CustomerResponse {
  id: string;
  user_id: string;
  phone?: string;
  avatar_url?: string;
  date_of_birth?: string;
  address_count: number;
  created_at: string;
  updated_at: string;
}

export interface AuditEntryResponse {
  id: string;
  actor_id: string;
  action: string;
  resource_type: string;
  resource_id: string;
  old_value?: unknown;
  new_value?: unknown;
  details?: unknown;
  ip_address: string;
  request_id?: string;
  created_at: string;
}

export interface ReviewModerationResponse {
  id: string;
  product_id: string;
  customer_id: string;
  author_name?: string;
  rating: number;
  title: string;
  comment: string;
  status: string;
  verified_purchase: boolean;
  created_at: string;
  updated_at: string;
}

export interface UpdateUserRoleRequest {
  role: string;
}

export interface UpdateUserStatusRequest {
  status: string;
}

export interface ModerateReviewRequest {
  action: 'approve' | 'reject' | 'hide';
  reason?: string;
}

export interface HealthResponse {
  status: string;
  timestamp: string;
  version?: string;
  uptime?: string;
}

export interface ReadyResponse {
  status: string;
  checks: Record<string, { status: string; message?: string }>;
}

// ============================================================================
// Fulfillment, Purchases & Profit (single-business back office)
// ============================================================================

export interface ShipOrderRequest {
  carrier: string;
  tracking_number: string;
  /** What the courier charged you, whole BDT (optional, feeds profit reports). */
  courier_cost?: number;
}

export interface PurchaseItemInput {
  sku: string;
  quantity: number;
  unit_cost: number;
}

export interface ReceiveStockRequest {
  supplier_name: string;
  reference?: string;
  warehouse_id: string;
  note?: string;
  items: PurchaseItemInput[];
}

export interface PurchaseItem {
  sku: string;
  quantity: number;
  unit_cost: number;
  line_total: number;
  stock_before: number;
  stock_after: number;
  cost_before: number;
  cost_after: number;
}

export interface PurchaseResponse {
  id: string;
  number: string;
  supplier_name: string;
  reference: string;
  warehouse_id: string;
  items: PurchaseItem[];
  total_cost: number;
  note?: string;
  received_at: string;
  created_at: string;
}

export interface ProfitSummary {
  orders: number;
  units_sold: number;
  items_revenue: number;
  discounts: number;
  shipping_charged: number;
  cost_of_goods: number;
  courier_cost: number;
  gross_profit: number;
  net_profit: number;
  margin_percent: number;
  items_missing_cost: number;
}

export interface ProfitPoint {
  period: string;
  orders: number;
  revenue: number;
  cost_of_goods: number;
  gross_profit: number;
  margin_percent: number;
}

export interface ProfitReport {
  from: string;
  to: string;
  statuses: string[];
  summary: ProfitSummary;
  trend: ProfitPoint[];
}

export interface ProductProfit {
  sku: string;
  name: string;
  units_sold: number;
  revenue: number;
  cost_of_goods: number;
  gross_profit: number;
  margin_percent: number;
}

export interface OrderProfit {
  order_id: string;
  order_number: string;
  created_at: string;
  status: string;
  payment_method: string;
  payment_status: string;
  items_revenue: number;
  discount: number;
  cost_of_goods: number;
  shipping_charged: number;
  courier_cost: number;
  gross_profit: number;
  net_profit: number;
}

// ============================================================================
// Delivery charges (what customers pay for delivery; set by the shop)
// ============================================================================

export interface DeliveryMethod {
  id: 'standard' | 'express';
  label: string;
  detail: string;
  enabled: boolean;
  fee_inside_dhaka: number;
  fee_outside_dhaka: number;
}

export interface DeliverySettings {
  methods: DeliveryMethod[];
  /** Standard delivery is free from this item total (whole BDT). 0 = never free. */
  free_shipping_threshold: number;
  areas: { id: string; label: string }[];
  updated_at?: string;
}
