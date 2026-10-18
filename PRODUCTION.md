# Going live: checklist

The demo setup (`docker compose up --build`) is meant for trying things out. Before real customers use the shop,
work through this list. Items marked **MUST** are not optional.

## 1. Secrets and accounts (MUST)
- [ ] **Do not load the demo data.** `backend/scripts/seed.js` creates demo admins with the password `Password123!`.
      On a real server, start without the `seed` service, then create your own admin (or change every demo password
      and delete the demo accounts) before the site is reachable.
- [ ] Set a long random `JWT_SECRET` (and any other secret) in `backend/.env.docker` (or your host's secret store).
      `.env` files must **never** be committed to git; add them to `.gitignore`.
- [ ] Turn on MongoDB authentication (a username/password in the connection string) and do not publish port 27017.
- [ ] Change the default `mailpit` setup (it catches emails instead of sending them): see section 3.

## 2. HTTPS and addresses (MUST)
- [ ] Put a reverse proxy with HTTPS in front (Caddy, Traefik, or nginx with Let's Encrypt).
      Example: `shop.yourdomain.com` -> storefront, `admin.yourdomain.com` -> admin panel.
- [ ] Set `APP_URL=https://shop.yourdomain.com` and `ADMIN_URL=https://admin.yourdomain.com` (buttons in emails use these).
- [ ] Only the proxy should be reachable from the internet; keep MongoDB, NATS and the API port private.
- [ ] Consider limiting the admin panel to your office/VPN IP, or adding a second login step.

## 3. Email that really arrives (MUST)
- [ ] Use a mail provider (Amazon SES, Brevo, SendGrid, Mailgun...) and set `SMTP_HOST`, `SMTP_PORT`,
      `SMTP_USERNAME`, `SMTP_PASSWORD`, `SMTP_FROM` in `backend/.env.docker`.
- [ ] Send from your own domain and add its **SPF, DKIM and DMARC** records, or emails land in spam.
- [ ] Set `APP_NAME`, `BRAND_LOGO_URL` (full https:// address), `SUPPORT_EMAIL` so emails look like your shop.
- [ ] Send yourself a sign-up email, a password reset, an order email and a support reply, in Gmail and one other mailbox.

## 4. Backups (MUST)
- [ ] Back up MongoDB every day (for example `mongodump` to another machine or cloud storage) and keep 14+ days.
- [ ] **Test a restore** on another machine. A backup you never restored is a guess.
- [ ] Back up your uploaded/product images and the `.env` secrets (somewhere private).

## 5. Payments and orders
- [ ] bKash/Nagad/Rocket: you confirm payments by hand. Write down who checks them, how often, and what happens when a
      TrxID does not match (cancel the order and tell the customer).
- [ ] Decide your cash-on-delivery rules: refused parcels, return shipping, who restocks.
- [ ] Decide refund and return rules, and put them on the Terms/Refund pages.
- [ ] Automatic gateway payments and card payments are not connected yet.

## 6. Monitoring
- [ ] Add an uptime check on `https://shop.../` and `https://shop.../health` (UptimeRobot, Better Stack...) with an alert to your phone.
- [ ] Watch disk space (MongoDB and logs grow). Set log rotation.
- [ ] Add error tracking (for example Sentry) so you hear about problems before customers do.
- [ ] `docker compose logs -f api worker` is your first stop when something breaks.

## 7. Content and images
- [ ] Replace the placeholder product pictures. Products take image **addresses**; for your own photos use an image
      host or CDN (Cloudinary, S3 + CloudFront...). A built-in upload button is a good next feature.
- [ ] Read the About, Terms, Privacy pages and edit them for your business (get legal advice for your country).
- [ ] Replace the logo, favicon and the "FarmNest" texts if you use another name.

## 8. Growing later
- The storefront loads the whole catalog at start. That is fine for a few hundred products; beyond that, switch the
  product list and search to server-side paging.
- SMS and push notifications are still log-only (no real sending).
- Reviews, coupons and support messaging are built; product view tracking and abandoned-cart tracking are not.
- Keep dependencies updated (`go get -u`, `npm audit`) and rebuild regularly.

## 9. Before the first real order
1. Place a COD order and an online (wallet) order yourself. Ship, deliver and mark paid.
2. Cancel one and check the stock came back.
3. Review the Profit page and compare with your own numbers.
4. Send a support message and answer it from the admin panel.
5. Ask a friend to do the sign-up and reset-password flow on their phone.
