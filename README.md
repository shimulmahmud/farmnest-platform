# FarmNest platform — backend + admin panel + storefront

```
docker-compose.yml   starts everything (this file is the one to use)
backend/             Go API + worker (MongoDB, NATS, Mailpit)  -> http://localhost:8080
frontend/admin/      Admin panel (React)                         -> http://localhost:5173
frontend/client/     Customer storefront (React)                 -> http://localhost:5174
```

Money everywhere is **whole BDT taka** (no paisa). One business, you are the only seller.

## 1. Run everything with one command

You only need [Docker](https://docs.docker.com/get-docker/) (with Compose).

```bash
docker compose up --build
```

The first start takes a few minutes (it builds the backend and both websites). When it settles:

| What | Address |
|---|---|
| Storefront (customers) | http://localhost:5174 |
| Admin panel | http://localhost:5173 |
| API | http://localhost:8080 |
| Mailpit (catches emails, e.g. the sign-up code) | http://localhost:8025 |

It starts MongoDB, NATS, Mailpit, loads demo data (7 categories, 30 products with stock, a warehouse,
2 admins, 3 customers, 3 coupons), then the API, worker and both websites.
The websites forward `/api` to the backend themselves, so there is no CORS setup.

Accounts (password for all: `Password123!`):

| Who | Email |
|---|---|
| Super admin | admin@ecommerce.com |
| Admin | manager@ecommerce.com |
| Customer | john@example.com |

Settings (ports, your bKash/Nagad/Rocket numbers) go in a `.env` file next to `docker-compose.yml`:
`cp .env.example .env` and edit it. Only wallets with a number are offered at checkout.

Everyday commands:

```bash
docker compose up -d --build        # start in the background (rebuilds after code changes)
docker compose logs -f api          # watch the backend log  (also: worker, admin, client)
docker compose down                 # stop (your data is kept)
docker compose down -v              # stop AND delete all data
docker compose run --rm -e FORCE_SEED=1 seed   # wipe and reload the demo data
```

Demo data loads only into an **empty** database, so restarting never wipes your orders.
If you ran an older version of this project before, run `docker compose down -v` once first
(old products have the old format).

## 2. Developing without Docker for the websites (optional)

Keep `docker compose up mongodb nats mailpit seed api worker` running, then in two terminals:

```bash
cd frontend/admin  && npm install && npm run dev     # http://localhost:5173
cd frontend/client && npm install && npm run dev     # http://localhost:5174
```

(Stop the `admin` and `client` containers first, or change their ports in `.env`.) The dev servers forward
`/api` to `http://localhost:8080`; change the target with `VITE_PROXY_TARGET` (client).
For a production build outside Docker set `VITE_API_URL=https://your-api/api/v1` before `npm run build`.

## 3. Try the whole flow (about 10 minutes)

1. **Storefront** (5174): browse, open a product with several options (a size / pack picker appears), add to cart.
2. **Sign up** a new customer. A verification code is emailed: open **Mailpit** (8025), copy the code, paste it in the page.
3. **Checkout → Cash on delivery.** Order is `CONFIRMED`, payment `Unpaid`, stock goes down immediately.
4. **Admin** (5173, admin@ecommerce.com): **Orders** → *Ship* (courier, tracking number, courier charge) → *Delivered* → *Cash received*.
5. **Storefront → My orders** shows the tracking number and progress. **Track order** also works for guests (order number + phone).
6. **Online order:** checkout with bKash, send money to the shown number, enter the TrxID. In admin the order waits as
   *Waiting for payment*; click **Confirm payment** → stock is taken → ship it.
7. **Cancel** a confirmed order in admin: the stock goes back.
8. **Admin → Purchases:** receive stock from a supplier (cost becomes the weighted average).
   **Admin → Profit:** sales, cost of goods, gross and net profit, best/worst products.

## Rules built in

- Checkout (cart) never touches stock; the cart lives in the browser for 1 hour after its last change.
- COD: stock is taken when the order is placed. Online: checked at checkout, taken when payment is confirmed.
  If goods sell out while a customer is paying, the order goes to *Refund pending*.
- The server decides prices, delivery fee and totals. Nothing price-related sent by the browser is trusted.
- Public product API never shows cost price or exact stock.

## Reviews

- Only customers whose order for the product is **Delivered** can review it (shown as "Verified purchase").
- A new review waits for approval: **Admin -> Reviews** -> approve. Then it appears on the product page and the
  star rating shows on product cards.
- Try it: deliver an order in admin, then open the product (or **My orders -> Write a review**) as that customer.

## Coupons

- Create them in **Admin -> Marketing -> Promotions** (percent off, fixed amount off, or free delivery), set
  minimum order, maximum discount, usage limit, dates, and optionally only certain categories. Click *Activate*.
- Demo codes already loaded: `WELCOME10` (10% off, min ৳500, max ৳300), `FLAT100` (৳100 off, min ৳1,500), `FREESHIP` (free delivery, min ৳1,000).
- Customers type the code at checkout. The server checks it (active, dates, minimum order, categories, usage limit,
  **one use per customer**) and applies the discount. The price shown is what is charged.
- A cancelled order gives its coupon use back. Discounts are subtracted in the profit report.

## Delivery charges (what customers pay for delivery)

- Set them in **Admin -> Shipping -> Delivery charges**: for Standard and Express, the charge **inside Dhaka** and
  **outside Dhaka** in taka, the text customers see, and an on/off switch. Also the order total from which Standard
  delivery is free (0 = never).
- At checkout the customer picks the delivery area and method and sees the charge. The server adds it to the order,
  so it can't be changed from the browser. Changes apply to **new orders only**; placed orders keep their charge.
- When you ship an order you can also enter **what the courier charged you**. **Shipping** then shows both numbers
  (fee collected vs courier cost) and the Profit report counts the difference.
- **Admin -> Shipping -> Shipments** lists orders to ship, in transit and delivered, with courier and tracking numbers.

## Emails and passwords

- Sign-up sends a branded email with a **Verify my email** button (and a code as a backup). The button opens the
  storefront and confirms the address by itself. A "Send the email again" link is on the sign-up screen.
- **Forgot password** is on the sign-in screen (customers) and the admin sign-in screen (staff). The email button opens
  the right site (storefront for customers, admin panel for staff).
- Order emails (confirmed, shipped...) are now really sent (through Mailpit locally) and use the same branded layout.
- Set the shop name, logo and links in `.env`: `APP_NAME`, `BRAND_LOGO_URL`, `APP_URL`, `ADMIN_URL`, `SUPPORT_EMAIL`.

## Support messages

- Customers: **Help & Support** in the account menu and the footer, or "Need help with this order?" in **My orders**.
  They write a message, see replies in the same thread (a red dot shows when we replied) and also get an email.
- Staff: **Admin -> Support**. The sidebar shows how many conversations wait for a reply. Open one, answer, close it.
  Set `SUPPORT_EMAIL` to also get an email for every new customer message.
- Rules: one conversation can be about an order (checked against the customer's own orders), up to 5 open
  conversations per customer, 2,000 characters per message. A customer writing again reopens a closed conversation.

## Going live

See `PRODUCTION.md` for the checklist (secrets, HTTPS, real email, backups, monitoring).

## Not connected yet

- Card payments and automatic bKash/Nagad gateways (wallet payments are verified by you in admin).
- Automatic refunds (you send the money, then click *Mark refunded*).
- Branding is still "FarmNest" (logo, texts in `frontend/client`).

## Troubleshooting

- **`[Violation] Permissions policy violation: unload ... bubble.js` in the browser console** comes from a browser
  extension, not from this project. Ignore it (or test in a window without extensions).
- **A page is blank or says "Request failed"**: open the browser console (F12) and the **Network** tab, click the red
  request and read its *Response*. Send that text to the developer, with the output of `docker compose logs api`.
- **Dashboard / Analytics** count real orders: *placed* = orders created in the period, *delivered* = orders marked
  Delivered, *revenue* = total of delivered orders. Product views are not tracked yet.

## Tests

```bash
cd backend && go build ./... && go test ./...
cd frontend/admin  && npm run build
cd frontend/client && npm run build
```
