# Going live

`docker compose up` (the plain `docker-compose.yml`) is for trying the shop on your own computer.
For real customers use **`docker-compose.prod.yml`**. It already does the hard parts for you:

- HTTPS with free, self-renewing certificates (Caddy)
- a private database with a password (nothing but ports 80/443 is open to the internet)
- **no demo data and no demo passwords**
- real email through your provider (no Mailpit)
- a daily backup of the database and your uploaded pictures

## Quick start on a server

You need a Linux server (2 GB memory is enough to start) with Docker and Docker Compose, and two names
(for example `shop.yourdomain.com` and `admin.yourdomain.com`) whose DNS **A records point at the server's IP**.
Ports 80 and 443 must be open.

```bash
# 1. Settings
cp .env.production.example .env.production
nano .env.production            # fill in every REQUIRED line (secrets: openssl rand -hex 32)

# 2. Start everything (the first start builds the site and takes a few minutes)
docker compose -f docker-compose.prod.yml --env-file .env.production up -d --build

# 3. Create YOUR admin account (there is no demo account here)
docker compose -f docker-compose.prod.yml --env-file .env.production run --rm \
  --entrypoint /bin/createadmin \
  -e ADMIN_EMAIL=you@yourdomain.com \
  -e ADMIN_PASSWORD='a long password phrase of 12+ characters' \
  -e ADMIN_FIRST_NAME=Your -e ADMIN_LAST_NAME=Name api
```

Then open `https://admin.yourdomain.com`, sign in, and look at **Products, Categories, Delivery charges
(Shipping), Promotions**. Add your categories and products (use **Upload picture** for photos) and receive stock in
**Purchases**. Your shop is at `https://shop.yourdomain.com`.

Forgot your admin password? Run step 3 again with `-e ADMIN_RESET=1`.

### Everyday commands
```bash
docker compose -f docker-compose.prod.yml --env-file .env.production ps            # is everything running?
docker compose -f docker-compose.prod.yml --env-file .env.production logs -f api    # watch the backend (also: worker, caddy)
docker compose -f docker-compose.prod.yml --env-file .env.production up -d --build  # update after changing code/settings
docker compose -f docker-compose.prod.yml --env-file .env.production down           # stop (data is kept)
```
Never run `down -v` on the live shop: it deletes the database and the uploaded pictures.

## Backups and restoring
- A backup runs when the shop starts and then every 24 hours. Files go to the **`backups/`** folder next to
  `docker-compose.prod.yml`: `db-<date>.gz` (database) and `uploads-<date>.tgz` (pictures). Files older than
  `BACKUP_KEEP_DAYS` (default 14) are deleted.
- **Copy `backups/` to another machine or cloud storage regularly.** A backup on the same server is lost with the server.
- Restore the database (this replaces current data):
  ```bash
  docker compose -f docker-compose.prod.yml --env-file .env.production exec -T mongodb \
    mongorestore -u "$MONGO_USER" -p "$MONGO_PASSWORD" --authenticationDatabase admin \
    --archive --gzip --drop < backups/db-YYYYMMDD-HHMMSS.gz
  ```
- Restore pictures: `docker compose ... cp backups/uploads-....tgz api:/tmp/u.tgz`, then
  `docker compose ... exec api tar xzf /tmp/u.tgz -C /data/uploads`.
- **Practise a restore on a spare machine once.** A backup you never restored is a guess.

## What is still up to you

### Before opening the shop
- [ ] **Email:** use a real provider and add **SPF, DKIM and DMARC** records on your domain, or emails land in spam.
      Then send yourself a sign-up email, a password reset, an order email and a support reply.
- [ ] **Wallet numbers:** set `PAYMENT_BKASH_NUMBER` etc. in `.env.production` (no number = wallet hidden).
- [ ] **Delivery charges:** set them in Admin -> Shipping -> Delivery charges.
- [ ] **Pictures, logo, texts:** upload real product photos; set `APP_NAME` and `BRAND_LOGO_URL`; edit the
      About / Terms / Privacy / Refund pages for your business (get legal advice for your country).
- [ ] **Do one full test order** as a customer: COD and wallet; ship, deliver, cancel one and check the stock came back.

### Payments and orders
- bKash/Nagad/Rocket are confirmed by you by hand. Decide who checks them, how often, and what happens when a TrxID
  does not match (cancel and tell the customer).
- Decide cash-on-delivery rules (refused parcels, return shipping) and refund rules, and publish them.
- Automatic gateway and card payments are not connected yet.

### Keeping it healthy
- [ ] Add an uptime check on `https://shop.../health` (UptimeRobot, Better Stack...) that alerts your phone.
- [ ] Watch disk space (database, pictures, backups, logs).
- [ ] Add error tracking (for example Sentry) if you want to hear about problems before customers do.
- [ ] Keep the server updated (`apt upgrade`), and rebuild the shop now and then for security updates.
- [ ] Limit the admin address to your office/VPN if you can, and use long passwords for every staff account.

### Good to know
- The storefront loads the whole catalog at start. That is fine for a few hundred products; beyond that, switch the
  product list and search to server-side paging.
- SMS and push notifications are still log-only (no real sending).
- Uploaded pictures live on this server's disk (a volume). For heavy traffic, move them to a CDN/S3 later.
