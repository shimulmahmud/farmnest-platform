# Product model conventions

- Money: integer whole BDT taka. No paisa/cents anywhere: `price: 1299` means 1,299 BDT. Currency is per product (`currency`, default BDT).
- Weight: integer grams (`weight_g`). Dimensions: integer millimetres.
- Variant selectors: `options` (color, size, grade). Product info: `specs`. Sold-by unit: `unit` ("piece", "kg", "dozen"...).
- Stock is NOT on the product. The `inventory` collection (by SKU) is the single source of truth.
  Public API exposes only `in_stock`; admin/vendor use the inventory endpoints for numbers.
- Variant IDs are stable: edits match by SKU. Removed variants are ARCHIVED, never deleted.
- Statuses: product DRAFT|ACTIVE|ARCHIVED; variant ACTIVE|INACTIVE|ARCHIVED; vendor PENDING|APPROVED|SUSPENDED.

## Audiences
| Route | Who | Sees |
|---|---|---|
| GET /api/v1/products, /search, /:slug | public | ACTIVE only; no cost, no stock numbers, no vendor_id |
| /api/v1/vendor/products | role VENDOR, approved vendor | own products only (vendor_id from token, never request) |
| /api/v1/admin/products, /admin/vendors | admin | everything |

## Orders
POST /api/v1/orders takes only product_id, variant_id, quantity. Price, name, SKU, vendor, weight,
options are copied from the catalog into an immutable order-item snapshot.

## Vendor onboarding
1. User registers normally. 2. Admin: POST /api/v1/admin/vendors {owner_user_id,name,slug,commission_bps}
(user becomes VENDOR, vendor is PENDING). 3. Admin: PATCH /api/v1/admin/vendors/:id/status {"status":"APPROVED"}.

## Selling by weight (agriculture)
No fractional quantities. Sell pack sizes as variants (`500g`, `1kg`, `5kg`) with whole-number quantities,
so stock, price and shipping weight stay exact.

## Vendor money
Each order item freezes the vendor's `commission_bps` at purchase. Orders carry `vendor_totals`
(subtotal, commission, payout per vendor, whole BDT); platform-owned items are excluded.
Ordering from a non-APPROVED vendor is rejected.

## Daily operations (single-business model)
- **Receive stock:** `POST /api/v1/admin/purchases` {supplier_name, reference, warehouse_id, items[{sku, quantity, unit_cost}]}.
  Adds stock, sets each variant's `cost_price` to the weighted average, records the purchase.
- **Checkout never touches stock.** The cart / checkout session lives in the frontend (1-hour cache). Stock only
  moves when an order is placed and completed:
  - **COD** (`payment_method: "COD"`): order is CONFIRMED, `payment_status: UNPAID`, stock is deducted at once
    (409 INSUFFICIENT_STOCK if any line is short, so zero stock cannot be ordered). After delivery call
    `POST /api/v1/admin/orders/:id/mark-paid` when the cash is collected.
  - **ONLINE** (default): order stays PAYMENT_PENDING, `payment_status: UNPAID`, stock is only *checked*, not taken.
    When payment completes the order becomes PAID and the stock is deducted. If the goods sold out while the
    customer was paying, the order goes to REFUND_PENDING (paid, to be refunded).
  - **Cancelling** a COD or paid order adds the units back to the same warehouse. Cancelling an unpaid online order
    changes no stock. Shipping never touches stock.
- **Ship:** `POST /api/v1/admin/orders/:id/ship` {carrier, tracking_number}. PAID/PROCESSING -> SHIPPED, tracking saved (stock was already deducted at order time).
- **Reorder:** `GET /api/v1/admin/inventory/low-stock`.

## Profit reports (admin)
- `GET /api/v1/admin/reports/profit` summary + daily/monthly trend; `/profit/products` best or worst products;
  `/profit/orders` profit per order. Params: `from`, `to` (YYYY-MM-DD, Bangladesh time, inclusive), `statuses`
  (default SHIPPED,DELIVERED).
- Gross profit = item revenue - cost of goods. Each order item freezes the variant's `cost_price` when the order is
  placed, so later cost changes never rewrite past profit. Net profit = gross + shipping charged - courier cost
  (send `courier_cost` when shipping).
- Not included: payment-gateway fees, ads, packaging, refunds. `items_missing_cost > 0` flags orders placed before
  costs were recorded (profit overstated).
- Keep `cost_price` current by receiving stock through purchases (weighted-average cost).
