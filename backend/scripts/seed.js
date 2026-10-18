// ============================================================
// Ecommerce Backend — Seed Data Script
// Run via: mongosh mongodb://localhost:27017/ecommerce --file seed.js
// Or via docker-compose (runs automatically on first start)
// ============================================================
//
// Default passwords (all users):  Password123!
//
// Test accounts:
//   Super Admin:  admin@ecommerce.com
//   Admin:        manager@ecommerce.com
//   Customer 1:   john@example.com
//   Customer 2:   jane@example.com
//   Customer 3:   bob@example.com
//
// ============================================================

const db = db.getSiblingDB("ecommerce");
// Safety: do NOT wipe a database that already has data. `docker compose up` runs this
// every time, so without this guard real orders would be deleted on each restart.
// To reset on purpose:  docker compose run --rm -e FORCE_SEED=1 seed
if (db.users.countDocuments() > 0 && !(typeof process !== "undefined" && process.env && process.env.FORCE_SEED)) {
  print("Database already has data: seed skipped. (Set FORCE_SEED=1 to wipe and re-seed.)");
  quit(0);
}
print("\n=== Seeding ecommerce database ===\n");

// Pre-computed bcrypt hash for "Password123!" (cost 12)
const PW_HASH = "$2b$12$JquIuMSXJ9Jf2skljG7y8uBS7Tc/FsjcIx3ouD4bYRgghMtf3tseS";

// Helper: ISO timestamp
const daysAgo = (d) => new Date(Date.now() - d * 86400000).toISOString();
const now = new Date().toISOString();

// UUIDs (deterministic for referential integrity)
const UUIDS = {
  // Users
  admin_user:     "a0000000-0000-4000-a000-000000000001",
  manager_user:   "a0000000-0000-4000-a000-000000000002",
  john_user:      "b0000000-0000-4000-b000-000000000001",
  jane_user:      "b0000000-0000-4000-b000-000000000002",
  bob_user:       "b0000000-0000-4000-b000-000000000003",

  // Customers
  john_customer:  "c0000000-0000-4000-c000-000000000001",
  jane_customer:  "c0000000-0000-4000-c000-000000000002",
  bob_customer:   "c0000000-0000-4000-c000-000000000003",

  // Categories
  electronics:    "d0000000-0000-4000-d000-000000000001",
  clothing:       "d0000000-0000-4000-d000-000000000002",
  accessories:    "d0000000-0000-4000-d000-000000000003",
  home:           "d0000000-0000-4000-d000-000000000004",

  // Warehouses
  warehouse_main: "e0000000-0000-4000-e000-000000000001",
  warehouse_east: "e0000000-0000-4000-e000-000000000002",

  // Products
  laptop:         "f0000000-0000-4000-f000-000000000001",
  phone:          "f0000000-0000-4000-f000-000000000002",
  headphones:     "f0000000-0000-4000-f000-000000000003",
  tshirt:         "f0000000-0000-4000-f000-000000000004",
  jeans:          "f0000000-0000-4000-f000-000000000005",
  watch:          "f0000000-0000-4000-f000-000000000006",
  backpack:       "f0000000-0000-4000-f000-000000000007",
  lamp:           "f0000000-0000-4000-f000-000000000008",

  // Orders
  order1:         "10000000-0000-4000-1000-000000000001",
  order2:         "10000000-0000-4000-1000-000000000002",
  order3:         "10000000-0000-4000-1000-000000000003",
  order4:         "10000000-0000-4000-1000-000000000004",
  order5:         "10000000-0000-4000-1000-000000000005",

  // Payments
  pay1:           "20000000-0000-4000-2000-000000000001",
  pay2:           "20000000-0000-4000-2000-000000000002",
  pay3:           "20000000-0000-4000-2000-000000000003",
  pay4:           "20000000-0000-4000-2000-000000000004",
  pay5:           "20000000-0000-4000-2000-000000000005",

  // Shipments
  ship1:          "30000000-0000-4000-3000-000000000001",
  ship2:          "30000000-0000-4000-3000-000000000002",
  ship3:          "30000000-0000-4000-3000-000000000003",

  // Reviews
  rev1:           "40000000-0000-4000-4000-000000000001",
  rev2:           "40000000-0000-4000-4000-000000000002",
  rev3:           "40000000-0000-4000-4000-000000000003",
  rev4:           "40000000-0000-4000-4000-000000000004",
  rev5:           "40000000-0000-4000-4000-000000000005",
  rev6:           "40000000-0000-4000-4000-000000000006",

  // Wishlists
  wish1:          "50000000-0000-4000-5000-000000000001",
  wish2:          "50000000-0000-4000-5000-000000000002",

  // Marketing
  segment1:       "60000000-0000-4000-6000-000000000001",
  campaign1:      "70000000-0000-4000-7000-000000000001",
  promo1:         "80000000-0000-4000-8000-000000000001",
};

// ============================================================
// 1. USERS
// ============================================================
print("Seeding users...");
db.users.drop();
db.users.insertMany([
  {
    _id: UUIDS.admin_user,
    email: "admin@ecommerce.com",
    password_hash: PW_HASH,
    first_name: "Super",
    last_name: "Admin",
    role: "SUPER_ADMIN",
    status: "ACTIVE",
    email_verified: true,
    failed_attempts: 0,
    locked_until: null,
    last_login_at: daysAgo(0.5),
    created_at: daysAgo(90),
    updated_at: daysAgo(1),
  },
  {
    _id: UUIDS.manager_user,
    email: "manager@ecommerce.com",
    password_hash: PW_HASH,
    first_name: "Store",
    last_name: "Manager",
    role: "ADMIN",
    status: "ACTIVE",
    email_verified: true,
    failed_attempts: 0,
    locked_until: null,
    last_login_at: daysAgo(2),
    created_at: daysAgo(60),
    updated_at: daysAgo(2),
  },
  {
    _id: UUIDS.john_user,
    email: "john@example.com",
    password_hash: PW_HASH,
    first_name: "John",
    last_name: "Doe",
    role: "CUSTOMER",
    status: "ACTIVE",
    email_verified: true,
    failed_attempts: 0,
    locked_until: null,
    last_login_at: daysAgo(0.2),
    created_at: daysAgo(45),
    updated_at: daysAgo(0.2),
  },
  {
    _id: UUIDS.jane_user,
    email: "jane@example.com",
    password_hash: PW_HASH,
    first_name: "Jane",
    last_name: "Smith",
    role: "CUSTOMER",
    status: "ACTIVE",
    email_verified: true,
    failed_attempts: 0,
    locked_until: null,
    last_login_at: daysAgo(5),
    created_at: daysAgo(30),
    updated_at: daysAgo(5),
  },
  {
    _id: UUIDS.bob_user,
    email: "bob@example.com",
    password_hash: PW_HASH,
    first_name: "Bob",
    last_name: "Wilson",
    role: "CUSTOMER",
    status: "ACTIVE",
    email_verified: true,
    failed_attempts: 0,
    locked_until: null,
    last_login_at: daysAgo(10),
    created_at: daysAgo(20),
    updated_at: daysAgo(10),
  },
]);
print(`  Inserted ${db.users.countDocuments()} users`);

// ============================================================
// 2. CUSTOMERS
// ============================================================
print("Seeding customers...");
db.customers.drop();
db.customers.insertMany([
  {
    _id: UUIDS.john_customer,
    user_id: UUIDS.john_user,
    phone: "+1-555-0101",
    avatar_url: "",
    date_of_birth: "1990-05-15",
    addresses: [
      {
        id: "addr-john-1",
        label: "Home",
        first_name: "John",
        last_name: "Doe",
        line1: "123 Main Street",
        line2: "Apt 4B",
        city: "New York",
        state: "NY",
        postal_code: "10001",
        country: "US",
        is_default: true,
      },
    ],
    created_at: daysAgo(45),
    updated_at: daysAgo(5),
  },
  {
    _id: UUIDS.jane_customer,
    user_id: UUIDS.jane_user,
    phone: "+1-555-0102",
    avatar_url: "",
    date_of_birth: "1988-11-22",
    addresses: [
      {
        id: "addr-jane-1",
        label: "Home",
        first_name: "Jane",
        last_name: "Smith",
        line1: "456 Oak Avenue",
        line2: "",
        city: "Los Angeles",
        state: "CA",
        postal_code: "90001",
        country: "US",
        is_default: true,
      },
    ],
    created_at: daysAgo(30),
    updated_at: daysAgo(3),
  },
  {
    _id: UUIDS.bob_customer,
    user_id: UUIDS.bob_user,
    phone: "+1-555-0103",
    avatar_url: "",
    date_of_birth: null,
    addresses: [
      {
        id: "addr-bob-1",
        label: "Work",
        first_name: "Bob",
        last_name: "Wilson",
        line1: "789 Pine Road",
        line2: "Suite 100",
        city: "Chicago",
        state: "IL",
        postal_code: "60601",
        country: "US",
        is_default: true,
      },
    ],
    created_at: daysAgo(20),
    updated_at: daysAgo(10),
  },
]);
print(`  Inserted ${db.customers.countDocuments()} customers`);

// ============================================================
// 3. CATEGORIES (demo store: mixed products, prices in whole BDT taka)
// ============================================================
print("Seeding categories...");
db.categories.drop();
db.categories.insertMany([
  { _id: "d0000000-0000-4000-d000-000000000001", name: "Electronics", slug: "electronics", description: "Phones, laptops, audio and smart devices", parent_id: null, image: "", sort_order: 1, status: "ACTIVE", seo: { title: "Electronics", description: "Phones, laptops, audio and smart devices", keywords: "electronics" }, created_at: daysAgo(30), updated_at: daysAgo(1) },
  { _id: "d0000000-0000-4000-d000-000000000002", name: "Fashion", slug: "fashion", description: "Clothing and footwear", parent_id: null, image: "", sort_order: 2, status: "ACTIVE", seo: { title: "Fashion", description: "Clothing and footwear", keywords: "fashion" }, created_at: daysAgo(30), updated_at: daysAgo(1) },
  { _id: "d0000000-0000-4000-d000-000000000003", name: "Home & Living", slug: "home-living", description: "Kitchen, lighting and everyday home items", parent_id: null, image: "", sort_order: 3, status: "ACTIVE", seo: { title: "Home & Living", description: "Kitchen, lighting and everyday home items", keywords: "home-living" }, created_at: daysAgo(30), updated_at: daysAgo(1) },
  { _id: "d0000000-0000-4000-d000-000000000004", name: "Fresh Produce", slug: "fresh-produce", description: "Farm-fresh fruit, vegetables and eggs", parent_id: null, image: "", sort_order: 4, status: "ACTIVE", seo: { title: "Fresh Produce", description: "Farm-fresh fruit, vegetables and eggs", keywords: "fresh-produce" }, created_at: daysAgo(30), updated_at: daysAgo(1) },
  { _id: "d0000000-0000-4000-d000-000000000005", name: "Grains & Staples", slug: "grains-staples", description: "Rice, oil, pulses, honey and pantry staples", parent_id: null, image: "", sort_order: 5, status: "ACTIVE", seo: { title: "Grains & Staples", description: "Rice, oil, pulses, honey and pantry staples", keywords: "grains-staples" }, created_at: daysAgo(30), updated_at: daysAgo(1) },
  { _id: "d0000000-0000-4000-d000-000000000006", name: "Farm Inputs", slug: "farm-inputs", description: "Seeds, fertilizer and farm tools", parent_id: null, image: "", sort_order: 6, status: "ACTIVE", seo: { title: "Farm Inputs", description: "Seeds, fertilizer and farm tools", keywords: "farm-inputs" }, created_at: daysAgo(30), updated_at: daysAgo(1) },
  { _id: "d0000000-0000-4000-d000-000000000007", name: "Beauty & Personal Care", slug: "beauty-care", description: "Soap, hair oil and skin care", parent_id: null, image: "", sort_order: 7, status: "ACTIVE", seo: { title: "Beauty & Personal Care", description: "Soap, hair oil and skin care", keywords: "beauty-care" }, created_at: daysAgo(30), updated_at: daysAgo(1) },

]);
print(`  Inserted ${db.categories.countDocuments()} categories`);

// ============================================================
// 4. WAREHOUSES
// ============================================================
print("Seeding warehouses...");
db.warehouses.drop();
db.warehouses.insertMany([
  { _id: UUIDS.warehouse_main, name: "Dhaka Main Warehouse", code: "DHK-01", address: "Tejgaon Industrial Area, Dhaka", is_active: true, created_at: daysAgo(30), updated_at: daysAgo(1) },
]);
print(`  Inserted ${db.warehouses.countDocuments()} warehouses`);

// ============================================================
// 5. PRODUCTS (one document per product, variants embedded)
//    Money = whole BDT taka. Weight = grams. Stock lives in `inventory`, not here.
// ============================================================
print("Seeding products...");
db.products.drop();
const P = [
  {"_id": "1a731122-880e-efda-9e5b-2867a6f9bc92", "vendor_id": "", "name": "Galaxy Nova X 5G", "slug": "galaxy-nova-x", "description": "Flagship smartphone with a 6.7-inch AMOLED display, 108MP camera system and a 5000mAh battery.", "short_description": "6.7\" AMOLED, 108MP camera, 5G", "brand": "NovaTech", "category_id": "d0000000-0000-4000-d000-000000000001", "unit": "piece", "specs": {"display": "6.7 inch AMOLED", "battery": "5000mAh", "warranty": "1 year"}, "tags": ["phone", "smartphone", "5g"], "images": [{"url": "https://picsum.photos/seed/galaxy-nova-x/800/800", "alt": "Galaxy Nova X 5G", "sort_order": 1}], "currency": "BDT", "variants": [{"_id": "e9284f36-609f-0623-158a-5c5a1c255354", "sku": "GN-NOVA-X-256-BLK", "options": {"color": "Midnight Black", "storage": "256GB"}, "price": 89990, "cost_price": 72000, "weight_g": 200, "requires_shipping": true, "images": [], "status": "ACTIVE", "position": 0, "compare_at_price": 99990, "length_mm": 165, "width_mm": 78, "height_mm": 9}, {"_id": "48447793-fb3a-d46c-a0d4-f7874f4add27", "sku": "GN-NOVA-X-512-BLK", "options": {"color": "Midnight Black", "storage": "512GB"}, "price": 104990, "cost_price": 84000, "weight_g": 200, "requires_shipping": true, "images": [], "status": "ACTIVE", "position": 1, "compare_at_price": 114990, "length_mm": 165, "width_mm": 78, "height_mm": 9}], "status": "ACTIVE", "seo": {"title": "Galaxy Nova X 5G", "description": "6.7\" AMOLED, 108MP camera, 5G", "keywords": ["phone", "smartphone", "5g"]}, "sort_order": 1, "lowest_price": 89990},
  {"_id": "61d88b41-09c6-897c-4e17-72590439e436", "vendor_id": "", "name": "ProBook Ultra 15 Laptop", "slug": "probook-ultra-15", "description": "High-performance 15.6-inch laptop with 32GB RAM and fast SSD storage for professionals.", "short_description": "15.6\" laptop, 32GB RAM, 512GB SSD", "brand": "TechPro", "category_id": "d0000000-0000-4000-d000-000000000001", "unit": "piece", "specs": {"processor": "Intel Core Ultra 7", "ram": "32GB", "screen": "15.6 inch", "os": "Windows 11"}, "tags": ["laptop", "computer", "professional"], "images": [{"url": "https://picsum.photos/seed/probook-ultra-15/800/800", "alt": "ProBook Ultra 15 Laptop", "sort_order": 1}], "currency": "BDT", "variants": [{"_id": "5dccdd7b-66fd-1f34-3622-fb4796d7ee30", "sku": "PB-U15-512-SLV", "options": {"color": "Silver", "storage": "512GB"}, "price": 149990, "cost_price": 125000, "weight_g": 1800, "requires_shipping": true, "images": [], "status": "ACTIVE", "position": 0, "compare_at_price": 164990, "length_mm": 350, "width_mm": 240, "height_mm": 20}, {"_id": "839725f6-bf83-ecb3-295b-a1baa0d18cb9", "sku": "PB-U15-1TB-GRY", "options": {"color": "Space Gray", "storage": "1TB"}, "price": 169990, "cost_price": 142000, "weight_g": 1800, "requires_shipping": true, "images": [], "status": "ACTIVE", "position": 1, "compare_at_price": 184990, "length_mm": 350, "width_mm": 240, "height_mm": 20}], "status": "ACTIVE", "seo": {"title": "ProBook Ultra 15 Laptop", "description": "15.6\" laptop, 32GB RAM, 512GB SSD", "keywords": ["laptop", "computer", "professional"]}, "sort_order": 2, "lowest_price": 149990},
  {"_id": "95b3eb03-920a-2e30-8df1-0debd692e829", "vendor_id": "", "name": "SoundWave Pro ANC Headphones", "slug": "soundwave-pro-anc", "description": "Wireless over-ear headphones with active noise cancellation and 30-hour battery life.", "short_description": "Over-ear ANC, 30hr battery", "brand": "SoundWave", "category_id": "d0000000-0000-4000-d000-000000000001", "unit": "piece", "specs": {"battery": "30 hours", "connectivity": "Bluetooth 5.3"}, "tags": ["headphones", "audio", "wireless", "anc"], "images": [{"url": "https://picsum.photos/seed/soundwave-pro-anc/800/800", "alt": "SoundWave Pro ANC Headphones", "sort_order": 1}], "currency": "BDT", "variants": [{"_id": "94758241-0835-3de4-6c7a-818cc2e428d1", "sku": "SW-PRO-ANC-BLK", "options": {"color": "Matte Black"}, "price": 24990, "cost_price": 15000, "weight_g": 250, "requires_shipping": true, "images": [], "status": "ACTIVE", "position": 0, "compare_at_price": 29990}, {"_id": "f6d0a62e-9627-d414-0ba2-763144bc234d", "sku": "SW-PRO-ANC-WHT", "options": {"color": "Cream White"}, "price": 24990, "cost_price": 15000, "weight_g": 250, "requires_shipping": true, "images": [], "status": "ACTIVE", "position": 1, "compare_at_price": 29990}], "status": "ACTIVE", "seo": {"title": "SoundWave Pro ANC Headphones", "description": "Over-ear ANC, 30hr battery", "keywords": ["headphones", "audio", "wireless", "anc"]}, "sort_order": 3, "lowest_price": 24990},
  {"_id": "0f54be55-0ecc-f090-eaf9-e089c9da6f43", "vendor_id": "", "name": "VoltMax 20000mAh Power Bank", "slug": "powerbank-20000", "description": "Slim 20000mAh power bank with USB-C PD and dual USB-A outputs.", "short_description": "20000mAh, 22.5W fast charge", "brand": "VoltMax", "category_id": "d0000000-0000-4000-d000-000000000001", "unit": "piece", "specs": {"capacity": "20000mAh", "output": "22.5W"}, "tags": ["powerbank", "charger", "accessory"], "images": [{"url": "https://picsum.photos/seed/powerbank-20000/800/800", "alt": "VoltMax 20000mAh Power Bank", "sort_order": 1}], "currency": "BDT", "variants": [{"_id": "64a75aef-1dfb-b571-1c8e-43af8babfb3c", "sku": "VM-PB-20K-BLK", "options": {"color": "Black"}, "price": 3290, "cost_price": 2100, "weight_g": 430, "requires_shipping": true, "images": [], "status": "ACTIVE", "position": 0, "compare_at_price": 3990}, {"_id": "fc3e7285-a1e1-3704-c1e6-3656da65d3f7", "sku": "VM-PB-20K-WHT", "options": {"color": "White"}, "price": 3290, "cost_price": 2100, "weight_g": 430, "requires_shipping": true, "images": [], "status": "ACTIVE", "position": 1, "compare_at_price": 3990}], "status": "ACTIVE", "seo": {"title": "VoltMax 20000mAh Power Bank", "description": "20000mAh, 22.5W fast charge", "keywords": ["powerbank", "charger", "accessory"]}, "sort_order": 4, "lowest_price": 3290},
  {"_id": "e19a8c54-da4a-0871-0c48-7afa87b19ff8", "vendor_id": "", "name": "Chrono Titanium Smart Watch", "slug": "chrono-titanium-watch", "description": "Titanium smartwatch with always-on AMOLED display, GPS and health monitoring.", "short_description": "AMOLED, GPS, 7-day battery", "brand": "Chrono", "category_id": "d0000000-0000-4000-d000-000000000001", "unit": "piece", "specs": {"case": "Grade-5 titanium", "water_resistance": "100m"}, "tags": ["watch", "smartwatch", "gps"], "images": [{"url": "https://picsum.photos/seed/chrono-titanium-watch/800/800", "alt": "Chrono Titanium Smart Watch", "sort_order": 1}], "currency": "BDT", "variants": [{"_id": "097630ad-e54e-39b5-8a50-e833e2aa1d1e", "sku": "CH-TIT-40", "options": {"case_size": "40mm", "band": "Leather"}, "price": 39990, "cost_price": 27000, "weight_g": 55, "requires_shipping": true, "images": [], "status": "ACTIVE", "position": 0, "compare_at_price": 44990}, {"_id": "ad2c83ee-7776-f86a-4645-bf1d14de49c7", "sku": "CH-TIT-44", "options": {"case_size": "44mm", "band": "Titanium"}, "price": 42990, "cost_price": 29000, "weight_g": 65, "requires_shipping": true, "images": [], "status": "ACTIVE", "position": 1, "compare_at_price": 47990}], "status": "ACTIVE", "seo": {"title": "Chrono Titanium Smart Watch", "description": "AMOLED, GPS, 7-day battery", "keywords": ["watch", "smartwatch", "gps"]}, "sort_order": 5, "lowest_price": 39990},
  {"_id": "f2a17799-bef3-f72a-569a-904819b82c41", "vendor_id": "", "name": "Classic Cotton Crew Tee", "slug": "classic-cotton-tee", "description": "Soft pre-shrunk cotton crew neck t-shirt for everyday wear.", "short_description": "100% cotton, relaxed fit", "brand": "BasicsCo", "category_id": "d0000000-0000-4000-d000-000000000002", "unit": "piece", "specs": {"fabric": "100% cotton", "fit": "Relaxed"}, "tags": ["tshirt", "cotton", "casual"], "images": [{"url": "https://picsum.photos/seed/classic-cotton-tee/800/800", "alt": "Classic Cotton Crew Tee", "sort_order": 1}], "currency": "BDT", "variants": [{"_id": "999be3f1-e1a5-4ef1-f6da-c84ca4811eab", "sku": "BC-CCT-WHT-M", "options": {"color": "White", "size": "M"}, "price": 699, "cost_price": 300, "weight_g": 200, "requires_shipping": true, "images": [], "status": "ACTIVE", "position": 0}, {"_id": "5f9bfd43-7ec1-b94e-6c54-563b80b93127", "sku": "BC-CCT-WHT-L", "options": {"color": "White", "size": "L"}, "price": 699, "cost_price": 300, "weight_g": 220, "requires_shipping": true, "images": [], "status": "ACTIVE", "position": 1}, {"_id": "a2703b9c-3190-3ace-0f98-2c248fea754e", "sku": "BC-CCT-BLK-M", "options": {"color": "Black", "size": "M"}, "price": 699, "cost_price": 300, "weight_g": 200, "requires_shipping": true, "images": [], "status": "ACTIVE", "position": 2}, {"_id": "c4738b34-70fd-411f-fa57-3bc5c39cc7aa", "sku": "BC-CCT-BLK-L", "options": {"color": "Black", "size": "L"}, "price": 699, "cost_price": 300, "weight_g": 220, "requires_shipping": true, "images": [], "status": "ACTIVE", "position": 3}], "status": "ACTIVE", "seo": {"title": "Classic Cotton Crew Tee", "description": "100% cotton, relaxed fit", "keywords": ["tshirt", "cotton", "casual"]}, "sort_order": 6, "lowest_price": 699},
  {"_id": "5a64db36-ed01-84a6-a824-bfcc6afb1911", "vendor_id": "", "name": "Slim Fit Stretch Denim Jeans", "slug": "slim-fit-denim", "description": "Slim-fit stretch denim jeans with a modern tapered leg.", "short_description": "Slim fit, stretch denim, indigo", "brand": "DenimLab", "category_id": "d0000000-0000-4000-d000-000000000002", "unit": "piece", "specs": {"fabric": "98% cotton, 2% elastane"}, "tags": ["jeans", "denim", "slim fit"], "images": [{"url": "https://picsum.photos/seed/slim-fit-denim/800/800", "alt": "Slim Fit Stretch Denim Jeans", "sort_order": 1}], "currency": "BDT", "variants": [{"_id": "708bb62b-0a46-736d-3192-96c5abb14e1b", "sku": "DL-SFD-IND-30", "options": {"color": "Indigo", "size": "30"}, "price": 2490, "cost_price": 1200, "weight_g": 450, "requires_shipping": true, "images": [], "status": "ACTIVE", "position": 0, "compare_at_price": 2990}, {"_id": "62d5d768-3abe-a008-5cf3-b2f8b022ebfe", "sku": "DL-SFD-IND-32", "options": {"color": "Indigo", "size": "32"}, "price": 2490, "cost_price": 1200, "weight_g": 480, "requires_shipping": true, "images": [], "status": "ACTIVE", "position": 1, "compare_at_price": 2990}, {"_id": "df69f6af-3b09-7106-13c5-39edb99c7457", "sku": "DL-SFD-IND-34", "options": {"color": "Indigo", "size": "34"}, "price": 2490, "cost_price": 1200, "weight_g": 500, "requires_shipping": true, "images": [], "status": "ACTIVE", "position": 2, "compare_at_price": 2990}], "status": "ACTIVE", "seo": {"title": "Slim Fit Stretch Denim Jeans", "description": "Slim fit, stretch denim, indigo", "keywords": ["jeans", "denim", "slim fit"]}, "sort_order": 7, "lowest_price": 2490},
  {"_id": "be92878c-f1a3-0356-f95c-f2a8d3e8127c", "vendor_id": "", "name": "Embroidered Cotton Panjabi", "slug": "cotton-panjabi", "description": "Comfortable cotton panjabi with chest embroidery, ideal for Eid and weddings.", "short_description": "Festive cotton panjabi", "brand": "Aarong Style", "category_id": "d0000000-0000-4000-d000-000000000002", "unit": "piece", "specs": {"fabric": "Cotton", "care": "Machine wash cold"}, "tags": ["panjabi", "festive", "men"], "images": [{"url": "https://picsum.photos/seed/cotton-panjabi/800/800", "alt": "Embroidered Cotton Panjabi", "sort_order": 1}], "currency": "BDT", "variants": [{"_id": "e752f35a-7eda-a48c-38a7-6ea146ec39ef", "sku": "AS-PNJ-WHT-M", "options": {"color": "Off White", "size": "M"}, "price": 2990, "cost_price": 1500, "weight_g": 380, "requires_shipping": true, "images": [], "status": "ACTIVE", "position": 0, "compare_at_price": 3490}, {"_id": "1771c3a7-39e9-af3e-6a74-3b8717cad208", "sku": "AS-PNJ-WHT-L", "options": {"color": "Off White", "size": "L"}, "price": 2990, "cost_price": 1500, "weight_g": 400, "requires_shipping": true, "images": [], "status": "ACTIVE", "position": 1, "compare_at_price": 3490}, {"_id": "7a09dd7b-0a32-851b-cc33-e7f835c97682", "sku": "AS-PNJ-NVY-L", "options": {"color": "Navy", "size": "L"}, "price": 3190, "cost_price": 1600, "weight_g": 400, "requires_shipping": true, "images": [], "status": "ACTIVE", "position": 2, "compare_at_price": 3690}], "status": "ACTIVE", "seo": {"title": "Embroidered Cotton Panjabi", "description": "Festive cotton panjabi", "keywords": ["panjabi", "festive", "men"]}, "sort_order": 8, "lowest_price": 2990},
  {"_id": "4ee5d011-ecd1-0738-b525-9458b57ec912", "vendor_id": "", "name": "Everyday Canvas Sneakers", "slug": "canvas-sneakers", "description": "Breathable canvas sneakers with cushioned insole.", "short_description": "Lightweight canvas sneakers", "brand": "StepUp", "category_id": "d0000000-0000-4000-d000-000000000002", "unit": "pair", "specs": {"upper": "Canvas", "sole": "Rubber"}, "tags": ["shoes", "sneakers", "casual"], "images": [{"url": "https://picsum.photos/seed/canvas-sneakers/800/800", "alt": "Everyday Canvas Sneakers", "sort_order": 1}], "currency": "BDT", "variants": [{"_id": "8c002ccb-b106-65a9-4e89-75b1bd2e4d22", "sku": "SU-CNV-WHT-41", "options": {"color": "White", "size": "41"}, "price": 1890, "cost_price": 950, "weight_g": 700, "requires_shipping": true, "images": [], "status": "ACTIVE", "position": 0}, {"_id": "77c34b0a-4b1a-ca89-09c4-848964fd22c8", "sku": "SU-CNV-WHT-42", "options": {"color": "White", "size": "42"}, "price": 1890, "cost_price": 950, "weight_g": 720, "requires_shipping": true, "images": [], "status": "ACTIVE", "position": 1}, {"_id": "8f5c110f-b6ab-7359-40ed-4e3d96f51e56", "sku": "SU-CNV-BLK-42", "options": {"color": "Black", "size": "42"}, "price": 1890, "cost_price": 950, "weight_g": 720, "requires_shipping": true, "images": [], "status": "ACTIVE", "position": 2}], "status": "ACTIVE", "seo": {"title": "Everyday Canvas Sneakers", "description": "Lightweight canvas sneakers", "keywords": ["shoes", "sneakers", "casual"]}, "sort_order": 9, "lowest_price": 1890},
  {"_id": "4ecd13da-2ff2-481a-5ece-93629d0d3658", "vendor_id": "", "name": "Smart Rice Cooker 1.8L", "slug": "rice-cooker-18l", "description": "Electric rice cooker with non-stick inner pot and automatic keep-warm.", "short_description": "1.8L, keep-warm, non-stick", "brand": "HomeChef", "category_id": "d0000000-0000-4000-d000-000000000003", "unit": "piece", "specs": {"capacity": "1.8L", "power": "700W", "warranty": "1 year"}, "tags": ["kitchen", "rice cooker", "appliance"], "images": [{"url": "https://picsum.photos/seed/rice-cooker-18l/800/800", "alt": "Smart Rice Cooker 1.8L", "sort_order": 1}], "currency": "BDT", "variants": [{"_id": "d2819872-c2e8-8de5-7910-b8ffc240c04b", "sku": "HC-RC-18-WHT", "options": {"color": "White"}, "price": 3990, "cost_price": 2600, "weight_g": 2800, "requires_shipping": true, "images": [], "status": "ACTIVE", "position": 0, "compare_at_price": 4590, "length_mm": 300, "width_mm": 300, "height_mm": 260}], "status": "ACTIVE", "seo": {"title": "Smart Rice Cooker 1.8L", "description": "1.8L, keep-warm, non-stick", "keywords": ["kitchen", "rice cooker", "appliance"]}, "sort_order": 10, "lowest_price": 3990},
  {"_id": "9adb14fd-c471-b794-d174-5f422cd67dda", "vendor_id": "", "name": "Nordic Arc Floor Lamp", "slug": "nordic-arc-lamp", "description": "Minimalist arc floor lamp with linen drum shade and foot dimmer.", "short_description": "Brushed brass arc, linen shade", "brand": "LumiHome", "category_id": "d0000000-0000-4000-d000-000000000003", "unit": "piece", "specs": {"bulb": "E27", "material": "Brass"}, "tags": ["lamp", "lighting", "nordic"], "images": [{"url": "https://picsum.photos/seed/nordic-arc-lamp/800/800", "alt": "Nordic Arc Floor Lamp", "sort_order": 1}], "currency": "BDT", "variants": [{"_id": "170212e2-f665-1d1b-318b-dbeafd998c00", "sku": "LH-NAF-BRS", "options": {"finish": "Brushed Brass"}, "price": 8990, "cost_price": 5200, "weight_g": 5500, "requires_shipping": true, "images": [], "status": "ACTIVE", "position": 0, "compare_at_price": 10990, "length_mm": 350, "width_mm": 240, "height_mm": 1800}, {"_id": "ae62dcd3-591e-cebc-d319-32edd2e11e40", "sku": "LH-NAF-BLK", "options": {"finish": "Matte Black"}, "price": 8490, "cost_price": 4900, "weight_g": 5500, "requires_shipping": true, "images": [], "status": "ACTIVE", "position": 1, "compare_at_price": 10490, "length_mm": 350, "width_mm": 240, "height_mm": 1800}], "status": "ACTIVE", "seo": {"title": "Nordic Arc Floor Lamp", "description": "Brushed brass arc, linen shade", "keywords": ["lamp", "lighting", "nordic"]}, "sort_order": 11, "lowest_price": 8490},
  {"_id": "f0ee471d-e928-1392-4f79-716ac81d461d", "vendor_id": "", "name": "Insulated Steel Water Bottle", "slug": "steel-water-bottle", "description": "Double-wall vacuum insulated stainless steel bottle.", "short_description": "Keeps cold 24h, hot 12h", "brand": "HydroMate", "category_id": "d0000000-0000-4000-d000-000000000003", "unit": "piece", "specs": {"material": "304 stainless steel"}, "tags": ["bottle", "steel", "insulated"], "images": [{"url": "https://picsum.photos/seed/steel-water-bottle/800/800", "alt": "Insulated Steel Water Bottle", "sort_order": 1}], "currency": "BDT", "variants": [{"_id": "bb424f1c-bddd-d21f-6baa-b5a6d2837823", "sku": "HM-ISB-750-BLU", "options": {"color": "Blue", "capacity": "750ml"}, "price": 1190, "cost_price": 600, "weight_g": 380, "requires_shipping": true, "images": [], "status": "ACTIVE", "position": 0}, {"_id": "b3af7c8d-ce7d-6810-ba33-098b879f0782", "sku": "HM-ISB-1000-BLK", "options": {"color": "Black", "capacity": "1L"}, "price": 1390, "cost_price": 700, "weight_g": 450, "requires_shipping": true, "images": [], "status": "ACTIVE", "position": 1}], "status": "ACTIVE", "seo": {"title": "Insulated Steel Water Bottle", "description": "Keeps cold 24h, hot 12h", "keywords": ["bottle", "steel", "insulated"]}, "sort_order": 12, "lowest_price": 1190},
  {"_id": "0a832c77-b355-e487-c2e7-5de65688c348", "vendor_id": "", "name": "Rajshahi Himsagar Mango", "slug": "himsagar-mango", "description": "Hand-picked Himsagar mangoes from Rajshahi, naturally ripened and formalin-free.", "short_description": "Chemical-free Himsagar, sweet and fiberless", "brand": "Rajshahi Orchards", "category_id": "d0000000-0000-4000-d000-000000000004", "unit": "kg", "specs": {"origin": "Rajshahi", "grade": "A", "shelf_life_days": "5"}, "tags": ["mango", "fruit", "seasonal", "rajshahi"], "images": [{"url": "https://picsum.photos/seed/himsagar-mango/800/800", "alt": "Rajshahi Himsagar Mango", "sort_order": 1}], "currency": "BDT", "variants": [{"_id": "b0e94f3a-7be1-0b00-b6e4-79c10f2b1d92", "sku": "RO-HIM-2KG", "options": {"pack": "2kg"}, "price": 460, "cost_price": 300, "weight_g": 2000, "requires_shipping": true, "images": [], "status": "ACTIVE", "position": 0}, {"_id": "bbc03296-f94c-5fc4-231f-07051242b5d8", "sku": "RO-HIM-5KG", "options": {"pack": "5kg"}, "price": 1100, "cost_price": 720, "weight_g": 5000, "requires_shipping": true, "images": [], "status": "ACTIVE", "position": 1}, {"_id": "92235b73-028f-9e86-f0be-2dbcce6f7d00", "sku": "RO-HIM-10KG", "options": {"pack": "10kg"}, "price": 2100, "cost_price": 1400, "weight_g": 10000, "requires_shipping": true, "images": [], "status": "ACTIVE", "position": 2}], "status": "ACTIVE", "seo": {"title": "Rajshahi Himsagar Mango", "description": "Chemical-free Himsagar, sweet and fiberless", "keywords": ["mango", "fruit", "seasonal", "rajshahi"]}, "sort_order": 13, "lowest_price": 460},
  {"_id": "0acd2b9a-4425-7dfe-433d-22ff6649db9b", "vendor_id": "", "name": "Fresh Country Tomato", "slug": "fresh-tomato", "description": "Firm, ripe country tomatoes harvested within 24 hours of dispatch.", "short_description": "Farm-fresh, pesticide-safe tomatoes", "brand": "Bogura Farms", "category_id": "d0000000-0000-4000-d000-000000000004", "unit": "kg", "specs": {"origin": "Bogura", "grade": "A"}, "tags": ["vegetable", "tomato", "fresh"], "images": [{"url": "https://picsum.photos/seed/fresh-tomato/800/800", "alt": "Fresh Country Tomato", "sort_order": 1}], "currency": "BDT", "variants": [{"_id": "80291546-bb54-5083-10df-8d02a7e64060", "sku": "BF-TOM-1KG", "options": {"pack": "1kg"}, "price": 80, "cost_price": 50, "weight_g": 1000, "requires_shipping": true, "images": [], "status": "ACTIVE", "position": 0}, {"_id": "97db6569-062e-04c4-76ff-9670d5e6f877", "sku": "BF-TOM-5KG", "options": {"pack": "5kg"}, "price": 380, "cost_price": 240, "weight_g": 5000, "requires_shipping": true, "images": [], "status": "ACTIVE", "position": 1}], "status": "ACTIVE", "seo": {"title": "Fresh Country Tomato", "description": "Farm-fresh, pesticide-safe tomatoes", "keywords": ["vegetable", "tomato", "fresh"]}, "sort_order": 14, "lowest_price": 80},
  {"_id": "a89bdfc3-60c1-9766-d328-e58b402b53c5", "vendor_id": "", "name": "Desi Red Potato", "slug": "red-potato", "description": "Firm red potatoes from Munshiganj, ideal for curry and frying.", "short_description": "Cold-stored red potato", "brand": "Munshiganj Growers", "category_id": "d0000000-0000-4000-d000-000000000004", "unit": "kg", "specs": {"origin": "Munshiganj"}, "tags": ["vegetable", "potato"], "images": [{"url": "https://picsum.photos/seed/red-potato/800/800", "alt": "Desi Red Potato", "sort_order": 1}], "currency": "BDT", "variants": [{"_id": "8aadc7f0-5f4e-2b96-f0af-b0ee66907d8d", "sku": "MG-POT-2KG", "options": {"pack": "2kg"}, "price": 110, "cost_price": 70, "weight_g": 2000, "requires_shipping": true, "images": [], "status": "ACTIVE", "position": 0}, {"_id": "aaa41c3a-39dd-5262-c3fe-921e096907e3", "sku": "MG-POT-10KG", "options": {"pack": "10kg"}, "price": 520, "cost_price": 340, "weight_g": 10000, "requires_shipping": true, "images": [], "status": "ACTIVE", "position": 1}], "status": "ACTIVE", "seo": {"title": "Desi Red Potato", "description": "Cold-stored red potato", "keywords": ["vegetable", "potato"]}, "sort_order": 15, "lowest_price": 110},
  {"_id": "00071cd4-7126-97e5-8f54-15e0450eee17", "vendor_id": "", "name": "Fresh Green Chili", "slug": "green-chili", "description": "Fresh, hot green chilies picked and packed on the same day.", "short_description": "Spicy green chili", "brand": "Jashore Greens", "category_id": "d0000000-0000-4000-d000-000000000004", "unit": "kg", "specs": {"origin": "Jashore", "heat": "High"}, "tags": ["vegetable", "chili", "spice"], "images": [{"url": "https://picsum.photos/seed/green-chili/800/800", "alt": "Fresh Green Chili", "sort_order": 1}], "currency": "BDT", "variants": [{"_id": "11a93288-2528-e512-a520-6aa74cba279a", "sku": "JG-CHL-250G", "options": {"pack": "250g"}, "price": 40, "cost_price": 24, "weight_g": 250, "requires_shipping": true, "images": [], "status": "ACTIVE", "position": 0}, {"_id": "802205a9-db91-024c-7520-fa2b496ffeff", "sku": "JG-CHL-1KG", "options": {"pack": "1kg"}, "price": 150, "cost_price": 95, "weight_g": 1000, "requires_shipping": true, "images": [], "status": "ACTIVE", "position": 1}], "status": "ACTIVE", "seo": {"title": "Fresh Green Chili", "description": "Spicy green chili", "keywords": ["vegetable", "chili", "spice"]}, "sort_order": 16, "lowest_price": 40},
  {"_id": "fecca790-ceab-a471-6a19-ad713b900fa3", "vendor_id": "", "name": "Farm Fresh Desi Eggs", "slug": "farm-eggs-dozen", "description": "Free-range desi eggs, packed in protective trays.", "short_description": "Free-range desi eggs", "brand": "Savar Poultry", "category_id": "d0000000-0000-4000-d000-000000000004", "unit": "dozen", "specs": {"origin": "Savar", "type": "Free-range"}, "tags": ["eggs", "protein", "fresh"], "images": [{"url": "https://picsum.photos/seed/farm-eggs-dozen/800/800", "alt": "Farm Fresh Desi Eggs", "sort_order": 1}], "currency": "BDT", "variants": [{"_id": "45280ffb-bea7-1c5d-e633-defec68c7027", "sku": "SP-EGG-12", "options": {"pack": "12 pcs"}, "price": 240, "cost_price": 170, "weight_g": 900, "requires_shipping": true, "images": [], "status": "ACTIVE", "position": 0}, {"_id": "0da897d0-e2de-2916-55ee-f5f186c20f67", "sku": "SP-EGG-30", "options": {"pack": "30 pcs"}, "price": 580, "cost_price": 410, "weight_g": 2200, "requires_shipping": true, "images": [], "status": "ACTIVE", "position": 1}], "status": "ACTIVE", "seo": {"title": "Farm Fresh Desi Eggs", "description": "Free-range desi eggs", "keywords": ["eggs", "protein", "fresh"]}, "sort_order": 17, "lowest_price": 240},
  {"_id": "57de8367-9a96-01b2-f039-5edba196faba", "vendor_id": "", "name": "Premium Miniket Rice", "slug": "miniket-rice", "description": "Premium miniket rice from Dinajpur, aged and polish-free.", "short_description": "Polish-free, aromatic miniket", "brand": "Dinajpur Mills", "category_id": "d0000000-0000-4000-d000-000000000005", "unit": "kg", "specs": {"origin": "Dinajpur", "type": "Miniket"}, "tags": ["rice", "staple", "miniket"], "images": [{"url": "https://picsum.photos/seed/miniket-rice/800/800", "alt": "Premium Miniket Rice", "sort_order": 1}], "currency": "BDT", "variants": [{"_id": "9c300a95-00b9-6b47-98a2-978934edee0a", "sku": "DM-MNK-5KG", "options": {"pack": "5kg"}, "price": 430, "cost_price": 330, "weight_g": 5000, "requires_shipping": true, "images": [], "status": "ACTIVE", "position": 0}, {"_id": "fe319eb7-d1de-ffa8-22cc-54740d80e56e", "sku": "DM-MNK-25KG", "options": {"pack": "25kg"}, "price": 2050, "cost_price": 1600, "weight_g": 25000, "requires_shipping": true, "images": [], "status": "ACTIVE", "position": 1}], "status": "ACTIVE", "seo": {"title": "Premium Miniket Rice", "description": "Polish-free, aromatic miniket", "keywords": ["rice", "staple", "miniket"]}, "sort_order": 18, "lowest_price": 430},
  {"_id": "1bf77dc8-7cd7-e4a5-bb80-b428be2eabd7", "vendor_id": "", "name": "Cold-Pressed Mustard Oil", "slug": "mustard-oil", "description": "Pure mustard oil pressed in traditional wooden ghani, no additives.", "short_description": "Traditional kolhu-pressed mustard oil", "brand": "Pabna Ghani", "category_id": "d0000000-0000-4000-d000-000000000005", "unit": "litre", "specs": {"origin": "Pabna", "process": "Cold-pressed"}, "tags": ["oil", "mustard", "cooking"], "images": [{"url": "https://picsum.photos/seed/mustard-oil/800/800", "alt": "Cold-Pressed Mustard Oil", "sort_order": 1}], "currency": "BDT", "variants": [{"_id": "3b9ac2a7-374b-dc29-e567-644198b3ed3b", "sku": "PG-MO-1L", "options": {"pack": "1L"}, "price": 320, "cost_price": 240, "weight_g": 920, "requires_shipping": true, "images": [], "status": "ACTIVE", "position": 0}, {"_id": "b6fc9b87-a30a-4a12-e2cd-2d61bf705daf", "sku": "PG-MO-5L", "options": {"pack": "5L"}, "price": 1550, "cost_price": 1180, "weight_g": 4600, "requires_shipping": true, "images": [], "status": "ACTIVE", "position": 1}], "status": "ACTIVE", "seo": {"title": "Cold-Pressed Mustard Oil", "description": "Traditional kolhu-pressed mustard oil", "keywords": ["oil", "mustard", "cooking"]}, "sort_order": 19, "lowest_price": 320},
  {"_id": "78c99416-739a-f71c-c935-847d3e143b5a", "vendor_id": "", "name": "Desi Masur Dal", "slug": "masur-dal", "description": "Unpolished red lentils, cleaned and sorted for quick cooking.", "short_description": "Cleaned, unpolished red lentils", "brand": "Pabna Pulses", "category_id": "d0000000-0000-4000-d000-000000000005", "unit": "kg", "specs": {"origin": "Pabna"}, "tags": ["lentil", "dal", "pulse"], "images": [{"url": "https://picsum.photos/seed/masur-dal/800/800", "alt": "Desi Masur Dal", "sort_order": 1}], "currency": "BDT", "variants": [{"_id": "133e4713-dcdb-2285-322a-62e4ec56431d", "sku": "PP-MSR-1KG", "options": {"pack": "1kg"}, "price": 150, "cost_price": 110, "weight_g": 1000, "requires_shipping": true, "images": [], "status": "ACTIVE", "position": 0}, {"_id": "fd27ab4d-c16f-8830-580f-4ed296506016", "sku": "PP-MSR-5KG", "options": {"pack": "5kg"}, "price": 720, "cost_price": 540, "weight_g": 5000, "requires_shipping": true, "images": [], "status": "ACTIVE", "position": 1}], "status": "ACTIVE", "seo": {"title": "Desi Masur Dal", "description": "Cleaned, unpolished red lentils", "keywords": ["lentil", "dal", "pulse"]}, "sort_order": 20, "lowest_price": 150},
  {"_id": "9b2098bf-2fed-0af1-ec8d-b0f8c9c6e8b1", "vendor_id": "", "name": "Sundarban Wild Honey", "slug": "sundarban-honey", "description": "Raw wild honey collected from the Sundarbans, unprocessed.", "short_description": "Raw wild honey, no sugar added", "brand": "Khalisha Honey", "category_id": "d0000000-0000-4000-d000-000000000005", "unit": "kg", "specs": {"origin": "Sundarbans", "type": "Raw"}, "tags": ["honey", "natural", "sundarban"], "images": [{"url": "https://picsum.photos/seed/sundarban-honey/800/800", "alt": "Sundarban Wild Honey", "sort_order": 1}], "currency": "BDT", "variants": [{"_id": "2d8bcb6c-849d-5ceb-edd0-d0fc73222ca7", "sku": "KH-HNY-500G", "options": {"pack": "500g"}, "price": 650, "cost_price": 470, "weight_g": 650, "requires_shipping": true, "images": [], "status": "ACTIVE", "position": 0}, {"_id": "9ea0f2d6-d72d-bfd1-ab53-a8f1c9257ab7", "sku": "KH-HNY-1KG", "options": {"pack": "1kg"}, "price": 1250, "cost_price": 900, "weight_g": 1250, "requires_shipping": true, "images": [], "status": "ACTIVE", "position": 1}], "status": "ACTIVE", "seo": {"title": "Sundarban Wild Honey", "description": "Raw wild honey, no sugar added", "keywords": ["honey", "natural", "sundarban"]}, "sort_order": 21, "lowest_price": 650},
  {"_id": "d559020b-74a9-4e45-902a-d7e467fe50d1", "vendor_id": "", "name": "Chattogram Dried Shutki", "slug": "ctg-dried-fish", "description": "Hygienically sun-dried shutki, vacuum packed.", "short_description": "Sun-dried loitta shutki", "brand": "Coastal Foods", "category_id": "d0000000-0000-4000-d000-000000000005", "unit": "kg", "specs": {"origin": "Chattogram"}, "tags": ["dried fish", "shutki", "coastal"], "images": [{"url": "https://picsum.photos/seed/ctg-dried-fish/800/800", "alt": "Chattogram Dried Shutki", "sort_order": 1}], "currency": "BDT", "variants": [{"_id": "ad22e475-4510-4a98-3e4b-3c40562c434f", "sku": "CF-SHT-250G", "options": {"pack": "250g"}, "price": 380, "cost_price": 260, "weight_g": 300, "requires_shipping": true, "images": [], "status": "ACTIVE", "position": 0}, {"_id": "08703ded-01b3-53e2-270b-84d77d7024d9", "sku": "CF-SHT-500G", "options": {"pack": "500g"}, "price": 720, "cost_price": 500, "weight_g": 550, "requires_shipping": true, "images": [], "status": "ACTIVE", "position": 1}], "status": "ACTIVE", "seo": {"title": "Chattogram Dried Shutki", "description": "Sun-dried loitta shutki", "keywords": ["dried fish", "shutki", "coastal"]}, "sort_order": 22, "lowest_price": 380},
  {"_id": "7fb18339-40bb-1d45-c494-34eb44ece668", "vendor_id": "", "name": "Hybrid Boro Rice Seed", "slug": "hybrid-rice-seed", "description": "Certified high-yield hybrid rice seed suited for boro season.", "short_description": "High-yield hybrid boro seed", "brand": "BRRI Select", "category_id": "d0000000-0000-4000-d000-000000000006", "unit": "kg", "specs": {"season": "Boro", "germination": "90%+"}, "tags": ["seed", "rice", "hybrid"], "images": [{"url": "https://picsum.photos/seed/hybrid-rice-seed/800/800", "alt": "Hybrid Boro Rice Seed", "sort_order": 1}], "currency": "BDT", "variants": [{"_id": "74f5d1b4-f428-e76b-eab6-ca39588e094b", "sku": "BS-HRS-1KG", "options": {"pack": "1kg"}, "price": 450, "cost_price": 340, "weight_g": 1000, "requires_shipping": true, "images": [], "status": "ACTIVE", "position": 0}, {"_id": "3a267de8-9f76-14f2-6651-9d1e84fb0272", "sku": "BS-HRS-5KG", "options": {"pack": "5kg"}, "price": 2100, "cost_price": 1600, "weight_g": 5000, "requires_shipping": true, "images": [], "status": "ACTIVE", "position": 1}], "status": "ACTIVE", "seo": {"title": "Hybrid Boro Rice Seed", "description": "High-yield hybrid boro seed", "keywords": ["seed", "rice", "hybrid"]}, "sort_order": 23, "lowest_price": 450},
  {"_id": "d452e1fd-ed74-efff-6242-e56a7e875c9b", "vendor_id": "", "name": "Organic Vermicompost", "slug": "organic-compost", "description": "Vermicompost made from cow dung and kitchen waste, odour-free.", "short_description": "Nutrient-rich organic compost", "brand": "GreenSoil", "category_id": "d0000000-0000-4000-d000-000000000006", "unit": "kg", "specs": {"npk": "2-1-1", "type": "Organic"}, "tags": ["fertilizer", "organic", "compost"], "images": [{"url": "https://picsum.photos/seed/organic-compost/800/800", "alt": "Organic Vermicompost", "sort_order": 1}], "currency": "BDT", "variants": [{"_id": "acd7a9c0-d4fe-1b29-7ae5-6ed926e4239c", "sku": "GS-VCP-5KG", "options": {"pack": "5kg"}, "price": 320, "cost_price": 220, "weight_g": 5000, "requires_shipping": true, "images": [], "status": "ACTIVE", "position": 0}, {"_id": "86081c63-1e22-b96e-a484-bb89aaaa9fe0", "sku": "GS-VCP-25KG", "options": {"pack": "25kg"}, "price": 1450, "cost_price": 1050, "weight_g": 25000, "requires_shipping": true, "images": [], "status": "ACTIVE", "position": 1}], "status": "ACTIVE", "seo": {"title": "Organic Vermicompost", "description": "Nutrient-rich organic compost", "keywords": ["fertilizer", "organic", "compost"]}, "sort_order": 24, "lowest_price": 320},
  {"_id": "4d8f4069-c5c6-9f1a-2804-41c38692dc4c", "vendor_id": "", "name": "Drip Irrigation Starter Kit", "slug": "drip-irrigation-kit", "description": "Complete drip kit with tubing, drippers and connectors for small plots.", "short_description": "Covers up to 100 plants", "brand": "AquaFarm", "category_id": "d0000000-0000-4000-d000-000000000006", "unit": "kit", "specs": {"coverage": "100 plants", "tubing": "50m"}, "tags": ["irrigation", "drip", "farm"], "images": [{"url": "https://picsum.photos/seed/drip-irrigation-kit/800/800", "alt": "Drip Irrigation Starter Kit", "sort_order": 1}], "currency": "BDT", "variants": [{"_id": "77175b3d-8a9b-689f-84c8-c5114bf87cc8", "sku": "AF-DRP-100", "options": {"coverage": "100 plants"}, "price": 2890, "cost_price": 2000, "weight_g": 3200, "requires_shipping": true, "images": [], "status": "ACTIVE", "position": 0, "compare_at_price": 3390}, {"_id": "708d7519-6bf5-10bc-b210-1bcc5d3ff122", "sku": "AF-DRP-500", "options": {"coverage": "500 plants"}, "price": 11900, "cost_price": 8800, "weight_g": 14500, "requires_shipping": true, "images": [], "status": "ACTIVE", "position": 1, "compare_at_price": 13500}], "status": "ACTIVE", "seo": {"title": "Drip Irrigation Starter Kit", "description": "Covers up to 100 plants", "keywords": ["irrigation", "drip", "farm"]}, "sort_order": 25, "lowest_price": 2890},
  {"_id": "7468fc73-1bcb-99a9-c391-acf9ddd298eb", "vendor_id": "", "name": "Knapsack Sprayer 16L", "slug": "knapsack-sprayer-16l", "description": "Durable 16-litre knapsack sprayer with adjustable nozzle.", "short_description": "Manual 16L pesticide/water sprayer", "brand": "AgroPro", "category_id": "d0000000-0000-4000-d000-000000000006", "unit": "piece", "specs": {"capacity": "16L", "type": "Manual"}, "tags": ["sprayer", "tool", "farm"], "images": [{"url": "https://picsum.photos/seed/knapsack-sprayer-16l/800/800", "alt": "Knapsack Sprayer 16L", "sort_order": 1}], "currency": "BDT", "variants": [{"_id": "48da515b-90d0-05ea-2ae0-3235cd8dc467", "sku": "AP-SPR-16L", "options": {"capacity": "16L"}, "price": 2250, "cost_price": 1600, "weight_g": 3500, "requires_shipping": true, "images": [], "status": "ACTIVE", "position": 0, "compare_at_price": 2590}], "status": "ACTIVE", "seo": {"title": "Knapsack Sprayer 16L", "description": "Manual 16L pesticide/water sprayer", "keywords": ["sprayer", "tool", "farm"]}, "sort_order": 26, "lowest_price": 2250},
  {"_id": "c13fa1e1-458a-5d92-65a5-4a4a93d4d319", "vendor_id": "", "name": "Garden Hand Tool Set (5 pcs)", "slug": "garden-tool-set", "description": "Five-piece steel garden tool set with ergonomic grips.", "short_description": "Trowel, fork, pruner and more", "brand": "AgroPro", "category_id": "d0000000-0000-4000-d000-000000000006", "unit": "set", "specs": {"pieces": "5", "material": "Carbon steel"}, "tags": ["tools", "garden", "farm"], "images": [{"url": "https://picsum.photos/seed/garden-tool-set/800/800", "alt": "Garden Hand Tool Set (5 pcs)", "sort_order": 1}], "currency": "BDT", "variants": [{"_id": "f0d5139f-e7eb-7b19-5858-5ec154f6e165", "sku": "AP-GTS-5", "options": {"set": "5 pcs"}, "price": 1290, "cost_price": 850, "weight_g": 1400, "requires_shipping": true, "images": [], "status": "ACTIVE", "position": 0, "compare_at_price": 1590}], "status": "ACTIVE", "seo": {"title": "Garden Hand Tool Set (5 pcs)", "description": "Trowel, fork, pruner and more", "keywords": ["tools", "garden", "farm"]}, "sort_order": 27, "lowest_price": 1290},
  {"_id": "b690af1b-b343-f6fb-0d0e-4e75aa361854", "vendor_id": "", "name": "Herbal Neem Soap", "slug": "neem-soap", "description": "Handmade herbal soap with neem and tulsi, free from parabens.", "short_description": "Antibacterial neem & tulsi soap", "brand": "Prokriti", "category_id": "d0000000-0000-4000-d000-000000000007", "unit": "piece", "specs": {"weight": "100g", "type": "Handmade"}, "tags": ["soap", "herbal", "neem"], "images": [{"url": "https://picsum.photos/seed/neem-soap/800/800", "alt": "Herbal Neem Soap", "sort_order": 1}], "currency": "BDT", "variants": [{"_id": "7df227e8-bb22-5e30-b482-f9d441c060d0", "sku": "PK-NMS-1", "options": {"pack": "1 bar"}, "price": 90, "cost_price": 50, "weight_g": 110, "requires_shipping": true, "images": [], "status": "ACTIVE", "position": 0}, {"_id": "2ee3e0ed-756a-d5e3-bd59-89d5111ec560", "sku": "PK-NMS-4", "options": {"pack": "4 bars"}, "price": 330, "cost_price": 190, "weight_g": 440, "requires_shipping": true, "images": [], "status": "ACTIVE", "position": 1}], "status": "ACTIVE", "seo": {"title": "Herbal Neem Soap", "description": "Antibacterial neem & tulsi soap", "keywords": ["soap", "herbal", "neem"]}, "sort_order": 28, "lowest_price": 90},
  {"_id": "ada21d64-8d3f-1cd0-d1a5-bd6a7ae49edc", "vendor_id": "", "name": "Pure Coconut Hair Oil", "slug": "coconut-hair-oil", "description": "Pure cold-pressed coconut oil for hair and skin.", "short_description": "Cold-pressed coconut oil", "brand": "Prokriti", "category_id": "d0000000-0000-4000-d000-000000000007", "unit": "piece", "specs": {"process": "Cold-pressed"}, "tags": ["oil", "hair", "coconut"], "images": [{"url": "https://picsum.photos/seed/coconut-hair-oil/800/800", "alt": "Pure Coconut Hair Oil", "sort_order": 1}], "currency": "BDT", "variants": [{"_id": "0bcfed6d-eeb8-4d8a-df15-d57ccf784550", "sku": "PK-CHO-200", "options": {"size": "200ml"}, "price": 290, "cost_price": 190, "weight_g": 230, "requires_shipping": true, "images": [], "status": "ACTIVE", "position": 0}, {"_id": "b63b28b6-c467-10f7-eeaa-9f206227ac5b", "sku": "PK-CHO-500", "options": {"size": "500ml"}, "price": 640, "cost_price": 430, "weight_g": 560, "requires_shipping": true, "images": [], "status": "ACTIVE", "position": 1}], "status": "ACTIVE", "seo": {"title": "Pure Coconut Hair Oil", "description": "Cold-pressed coconut oil", "keywords": ["oil", "hair", "coconut"]}, "sort_order": 29, "lowest_price": 290},
  {"_id": "f8b53b21-e459-0014-b953-22517ace2b40", "vendor_id": "", "name": "Charcoal Face Wash", "slug": "charcoal-face-wash", "description": "Activated charcoal face wash that removes oil and impurities.", "short_description": "Deep cleansing, oil control", "brand": "ClearSkin", "category_id": "d0000000-0000-4000-d000-000000000007", "unit": "piece", "specs": {"skin_type": "Oily"}, "tags": ["facewash", "skincare"], "images": [{"url": "https://picsum.photos/seed/charcoal-face-wash/800/800", "alt": "Charcoal Face Wash", "sort_order": 1}], "currency": "BDT", "variants": [{"_id": "f0e4ad66-8fc7-8212-db8b-7ef8a3d6a5f7", "sku": "CS-CFW-100", "options": {"size": "100ml"}, "price": 420, "cost_price": 260, "weight_g": 130, "requires_shipping": true, "images": [], "status": "ACTIVE", "position": 0, "compare_at_price": 490}], "status": "ACTIVE", "seo": {"title": "Charcoal Face Wash", "description": "Deep cleansing, oil control", "keywords": ["facewash", "skincare"]}, "sort_order": 30, "lowest_price": 420},
];
const stamp = daysAgo(7);
P.forEach((p) => {
  p.created_at = stamp; p.updated_at = stamp;
  p.variants.forEach((v) => { v.created_at = stamp; v.updated_at = stamp; });
});
db.products.insertMany(P);
print(`  Inserted ${db.products.countDocuments()} products`);

// ============================================================
// 6. INVENTORY (stock per SKU per warehouse)
// ============================================================
print("Seeding inventory...");
db.inventory.drop();
db.inventory.insertMany([
  { _id: "inv-gn-nova-x-256-blk", sku: "GN-NOVA-X-256-BLK", warehouse_id: UUIDS.warehouse_main, total_stock: 30, reserved_stock: 0, low_stock_threshold: 5, created_at: daysAgo(7), updated_at: daysAgo(1) },
  { _id: "inv-gn-nova-x-512-blk", sku: "GN-NOVA-X-512-BLK", warehouse_id: UUIDS.warehouse_main, total_stock: 15, reserved_stock: 0, low_stock_threshold: 5, created_at: daysAgo(7), updated_at: daysAgo(1) },
  { _id: "inv-pb-u15-512-slv", sku: "PB-U15-512-SLV", warehouse_id: UUIDS.warehouse_main, total_stock: 12, reserved_stock: 0, low_stock_threshold: 5, created_at: daysAgo(7), updated_at: daysAgo(1) },
  { _id: "inv-pb-u15-1tb-gry", sku: "PB-U15-1TB-GRY", warehouse_id: UUIDS.warehouse_main, total_stock: 8, reserved_stock: 0, low_stock_threshold: 5, created_at: daysAgo(7), updated_at: daysAgo(1) },
  { _id: "inv-sw-pro-anc-blk", sku: "SW-PRO-ANC-BLK", warehouse_id: UUIDS.warehouse_main, total_stock: 60, reserved_stock: 0, low_stock_threshold: 6, created_at: daysAgo(7), updated_at: daysAgo(1) },
  { _id: "inv-sw-pro-anc-wht", sku: "SW-PRO-ANC-WHT", warehouse_id: UUIDS.warehouse_main, total_stock: 40, reserved_stock: 0, low_stock_threshold: 5, created_at: daysAgo(7), updated_at: daysAgo(1) },
  { _id: "inv-vm-pb-20k-blk", sku: "VM-PB-20K-BLK", warehouse_id: UUIDS.warehouse_main, total_stock: 120, reserved_stock: 0, low_stock_threshold: 12, created_at: daysAgo(7), updated_at: daysAgo(1) },
  { _id: "inv-vm-pb-20k-wht", sku: "VM-PB-20K-WHT", warehouse_id: UUIDS.warehouse_main, total_stock: 80, reserved_stock: 0, low_stock_threshold: 8, created_at: daysAgo(7), updated_at: daysAgo(1) },
  { _id: "inv-ch-tit-40", sku: "CH-TIT-40", warehouse_id: UUIDS.warehouse_main, total_stock: 25, reserved_stock: 0, low_stock_threshold: 5, created_at: daysAgo(7), updated_at: daysAgo(1) },
  { _id: "inv-ch-tit-44", sku: "CH-TIT-44", warehouse_id: UUIDS.warehouse_main, total_stock: 18, reserved_stock: 0, low_stock_threshold: 5, created_at: daysAgo(7), updated_at: daysAgo(1) },
  { _id: "inv-bc-cct-wht-m", sku: "BC-CCT-WHT-M", warehouse_id: UUIDS.warehouse_main, total_stock: 150, reserved_stock: 0, low_stock_threshold: 15, created_at: daysAgo(7), updated_at: daysAgo(1) },
  { _id: "inv-bc-cct-wht-l", sku: "BC-CCT-WHT-L", warehouse_id: UUIDS.warehouse_main, total_stock: 150, reserved_stock: 0, low_stock_threshold: 15, created_at: daysAgo(7), updated_at: daysAgo(1) },
  { _id: "inv-bc-cct-blk-m", sku: "BC-CCT-BLK-M", warehouse_id: UUIDS.warehouse_main, total_stock: 100, reserved_stock: 0, low_stock_threshold: 10, created_at: daysAgo(7), updated_at: daysAgo(1) },
  { _id: "inv-bc-cct-blk-l", sku: "BC-CCT-BLK-L", warehouse_id: UUIDS.warehouse_main, total_stock: 100, reserved_stock: 0, low_stock_threshold: 10, created_at: daysAgo(7), updated_at: daysAgo(1) },
  { _id: "inv-dl-sfd-ind-30", sku: "DL-SFD-IND-30", warehouse_id: UUIDS.warehouse_main, total_stock: 60, reserved_stock: 0, low_stock_threshold: 6, created_at: daysAgo(7), updated_at: daysAgo(1) },
  { _id: "inv-dl-sfd-ind-32", sku: "DL-SFD-IND-32", warehouse_id: UUIDS.warehouse_main, total_stock: 80, reserved_stock: 0, low_stock_threshold: 8, created_at: daysAgo(7), updated_at: daysAgo(1) },
  { _id: "inv-dl-sfd-ind-34", sku: "DL-SFD-IND-34", warehouse_id: UUIDS.warehouse_main, total_stock: 60, reserved_stock: 0, low_stock_threshold: 6, created_at: daysAgo(7), updated_at: daysAgo(1) },
  { _id: "inv-as-pnj-wht-m", sku: "AS-PNJ-WHT-M", warehouse_id: UUIDS.warehouse_main, total_stock: 40, reserved_stock: 0, low_stock_threshold: 5, created_at: daysAgo(7), updated_at: daysAgo(1) },
  { _id: "inv-as-pnj-wht-l", sku: "AS-PNJ-WHT-L", warehouse_id: UUIDS.warehouse_main, total_stock: 40, reserved_stock: 0, low_stock_threshold: 5, created_at: daysAgo(7), updated_at: daysAgo(1) },
  { _id: "inv-as-pnj-nvy-l", sku: "AS-PNJ-NVY-L", warehouse_id: UUIDS.warehouse_main, total_stock: 30, reserved_stock: 0, low_stock_threshold: 5, created_at: daysAgo(7), updated_at: daysAgo(1) },
  { _id: "inv-su-cnv-wht-41", sku: "SU-CNV-WHT-41", warehouse_id: UUIDS.warehouse_main, total_stock: 50, reserved_stock: 0, low_stock_threshold: 5, created_at: daysAgo(7), updated_at: daysAgo(1) },
  { _id: "inv-su-cnv-wht-42", sku: "SU-CNV-WHT-42", warehouse_id: UUIDS.warehouse_main, total_stock: 50, reserved_stock: 0, low_stock_threshold: 5, created_at: daysAgo(7), updated_at: daysAgo(1) },
  { _id: "inv-su-cnv-blk-42", sku: "SU-CNV-BLK-42", warehouse_id: UUIDS.warehouse_main, total_stock: 40, reserved_stock: 0, low_stock_threshold: 5, created_at: daysAgo(7), updated_at: daysAgo(1) },
  { _id: "inv-hc-rc-18-wht", sku: "HC-RC-18-WHT", warehouse_id: UUIDS.warehouse_main, total_stock: 45, reserved_stock: 0, low_stock_threshold: 5, created_at: daysAgo(7), updated_at: daysAgo(1) },
  { _id: "inv-lh-naf-brs", sku: "LH-NAF-BRS", warehouse_id: UUIDS.warehouse_main, total_stock: 20, reserved_stock: 0, low_stock_threshold: 5, created_at: daysAgo(7), updated_at: daysAgo(1) },
  { _id: "inv-lh-naf-blk", sku: "LH-NAF-BLK", warehouse_id: UUIDS.warehouse_main, total_stock: 15, reserved_stock: 0, low_stock_threshold: 5, created_at: daysAgo(7), updated_at: daysAgo(1) },
  { _id: "inv-hm-isb-750-blu", sku: "HM-ISB-750-BLU", warehouse_id: UUIDS.warehouse_main, total_stock: 90, reserved_stock: 0, low_stock_threshold: 9, created_at: daysAgo(7), updated_at: daysAgo(1) },
  { _id: "inv-hm-isb-1000-blk", sku: "HM-ISB-1000-BLK", warehouse_id: UUIDS.warehouse_main, total_stock: 70, reserved_stock: 0, low_stock_threshold: 7, created_at: daysAgo(7), updated_at: daysAgo(1) },
  { _id: "inv-ro-him-2kg", sku: "RO-HIM-2KG", warehouse_id: UUIDS.warehouse_main, total_stock: 100, reserved_stock: 0, low_stock_threshold: 10, created_at: daysAgo(7), updated_at: daysAgo(1) },
  { _id: "inv-ro-him-5kg", sku: "RO-HIM-5KG", warehouse_id: UUIDS.warehouse_main, total_stock: 60, reserved_stock: 0, low_stock_threshold: 6, created_at: daysAgo(7), updated_at: daysAgo(1) },
  { _id: "inv-ro-him-10kg", sku: "RO-HIM-10KG", warehouse_id: UUIDS.warehouse_main, total_stock: 30, reserved_stock: 0, low_stock_threshold: 5, created_at: daysAgo(7), updated_at: daysAgo(1) },
  { _id: "inv-bf-tom-1kg", sku: "BF-TOM-1KG", warehouse_id: UUIDS.warehouse_main, total_stock: 200, reserved_stock: 0, low_stock_threshold: 20, created_at: daysAgo(7), updated_at: daysAgo(1) },
  { _id: "inv-bf-tom-5kg", sku: "BF-TOM-5KG", warehouse_id: UUIDS.warehouse_main, total_stock: 80, reserved_stock: 0, low_stock_threshold: 8, created_at: daysAgo(7), updated_at: daysAgo(1) },
  { _id: "inv-mg-pot-2kg", sku: "MG-POT-2KG", warehouse_id: UUIDS.warehouse_main, total_stock: 150, reserved_stock: 0, low_stock_threshold: 15, created_at: daysAgo(7), updated_at: daysAgo(1) },
  { _id: "inv-mg-pot-10kg", sku: "MG-POT-10KG", warehouse_id: UUIDS.warehouse_main, total_stock: 60, reserved_stock: 0, low_stock_threshold: 6, created_at: daysAgo(7), updated_at: daysAgo(1) },
  { _id: "inv-jg-chl-250g", sku: "JG-CHL-250G", warehouse_id: UUIDS.warehouse_main, total_stock: 180, reserved_stock: 0, low_stock_threshold: 18, created_at: daysAgo(7), updated_at: daysAgo(1) },
  { _id: "inv-jg-chl-1kg", sku: "JG-CHL-1KG", warehouse_id: UUIDS.warehouse_main, total_stock: 90, reserved_stock: 0, low_stock_threshold: 9, created_at: daysAgo(7), updated_at: daysAgo(1) },
  { _id: "inv-sp-egg-12", sku: "SP-EGG-12", warehouse_id: UUIDS.warehouse_main, total_stock: 120, reserved_stock: 0, low_stock_threshold: 12, created_at: daysAgo(7), updated_at: daysAgo(1) },
  { _id: "inv-sp-egg-30", sku: "SP-EGG-30", warehouse_id: UUIDS.warehouse_main, total_stock: 60, reserved_stock: 0, low_stock_threshold: 6, created_at: daysAgo(7), updated_at: daysAgo(1) },
  { _id: "inv-dm-mnk-5kg", sku: "DM-MNK-5KG", warehouse_id: UUIDS.warehouse_main, total_stock: 200, reserved_stock: 0, low_stock_threshold: 20, created_at: daysAgo(7), updated_at: daysAgo(1) },
  { _id: "inv-dm-mnk-25kg", sku: "DM-MNK-25KG", warehouse_id: UUIDS.warehouse_main, total_stock: 50, reserved_stock: 0, low_stock_threshold: 5, created_at: daysAgo(7), updated_at: daysAgo(1) },
  { _id: "inv-pg-mo-1l", sku: "PG-MO-1L", warehouse_id: UUIDS.warehouse_main, total_stock: 150, reserved_stock: 0, low_stock_threshold: 15, created_at: daysAgo(7), updated_at: daysAgo(1) },
  { _id: "inv-pg-mo-5l", sku: "PG-MO-5L", warehouse_id: UUIDS.warehouse_main, total_stock: 50, reserved_stock: 0, low_stock_threshold: 5, created_at: daysAgo(7), updated_at: daysAgo(1) },
  { _id: "inv-pp-msr-1kg", sku: "PP-MSR-1KG", warehouse_id: UUIDS.warehouse_main, total_stock: 200, reserved_stock: 0, low_stock_threshold: 20, created_at: daysAgo(7), updated_at: daysAgo(1) },
  { _id: "inv-pp-msr-5kg", sku: "PP-MSR-5KG", warehouse_id: UUIDS.warehouse_main, total_stock: 60, reserved_stock: 0, low_stock_threshold: 6, created_at: daysAgo(7), updated_at: daysAgo(1) },
  { _id: "inv-kh-hny-500g", sku: "KH-HNY-500G", warehouse_id: UUIDS.warehouse_main, total_stock: 80, reserved_stock: 0, low_stock_threshold: 8, created_at: daysAgo(7), updated_at: daysAgo(1) },
  { _id: "inv-kh-hny-1kg", sku: "KH-HNY-1KG", warehouse_id: UUIDS.warehouse_main, total_stock: 50, reserved_stock: 0, low_stock_threshold: 5, created_at: daysAgo(7), updated_at: daysAgo(1) },
  { _id: "inv-cf-sht-250g", sku: "CF-SHT-250G", warehouse_id: UUIDS.warehouse_main, total_stock: 70, reserved_stock: 0, low_stock_threshold: 7, created_at: daysAgo(7), updated_at: daysAgo(1) },
  { _id: "inv-cf-sht-500g", sku: "CF-SHT-500G", warehouse_id: UUIDS.warehouse_main, total_stock: 40, reserved_stock: 0, low_stock_threshold: 5, created_at: daysAgo(7), updated_at: daysAgo(1) },
  { _id: "inv-bs-hrs-1kg", sku: "BS-HRS-1KG", warehouse_id: UUIDS.warehouse_main, total_stock: 150, reserved_stock: 0, low_stock_threshold: 15, created_at: daysAgo(7), updated_at: daysAgo(1) },
  { _id: "inv-bs-hrs-5kg", sku: "BS-HRS-5KG", warehouse_id: UUIDS.warehouse_main, total_stock: 60, reserved_stock: 0, low_stock_threshold: 6, created_at: daysAgo(7), updated_at: daysAgo(1) },
  { _id: "inv-gs-vcp-5kg", sku: "GS-VCP-5KG", warehouse_id: UUIDS.warehouse_main, total_stock: 120, reserved_stock: 0, low_stock_threshold: 12, created_at: daysAgo(7), updated_at: daysAgo(1) },
  { _id: "inv-gs-vcp-25kg", sku: "GS-VCP-25KG", warehouse_id: UUIDS.warehouse_main, total_stock: 50, reserved_stock: 0, low_stock_threshold: 5, created_at: daysAgo(7), updated_at: daysAgo(1) },
  { _id: "inv-af-drp-100", sku: "AF-DRP-100", warehouse_id: UUIDS.warehouse_main, total_stock: 35, reserved_stock: 0, low_stock_threshold: 5, created_at: daysAgo(7), updated_at: daysAgo(1) },
  { _id: "inv-af-drp-500", sku: "AF-DRP-500", warehouse_id: UUIDS.warehouse_main, total_stock: 12, reserved_stock: 0, low_stock_threshold: 5, created_at: daysAgo(7), updated_at: daysAgo(1) },
  { _id: "inv-ap-spr-16l", sku: "AP-SPR-16L", warehouse_id: UUIDS.warehouse_main, total_stock: 40, reserved_stock: 0, low_stock_threshold: 5, created_at: daysAgo(7), updated_at: daysAgo(1) },
  { _id: "inv-ap-gts-5", sku: "AP-GTS-5", warehouse_id: UUIDS.warehouse_main, total_stock: 60, reserved_stock: 0, low_stock_threshold: 6, created_at: daysAgo(7), updated_at: daysAgo(1) },
  { _id: "inv-pk-nms-1", sku: "PK-NMS-1", warehouse_id: UUIDS.warehouse_main, total_stock: 300, reserved_stock: 0, low_stock_threshold: 30, created_at: daysAgo(7), updated_at: daysAgo(1) },
  { _id: "inv-pk-nms-4", sku: "PK-NMS-4", warehouse_id: UUIDS.warehouse_main, total_stock: 120, reserved_stock: 0, low_stock_threshold: 12, created_at: daysAgo(7), updated_at: daysAgo(1) },
  { _id: "inv-pk-cho-200", sku: "PK-CHO-200", warehouse_id: UUIDS.warehouse_main, total_stock: 150, reserved_stock: 0, low_stock_threshold: 15, created_at: daysAgo(7), updated_at: daysAgo(1) },
  { _id: "inv-pk-cho-500", sku: "PK-CHO-500", warehouse_id: UUIDS.warehouse_main, total_stock: 80, reserved_stock: 0, low_stock_threshold: 8, created_at: daysAgo(7), updated_at: daysAgo(1) },
  { _id: "inv-cs-cfw-100", sku: "CS-CFW-100", warehouse_id: UUIDS.warehouse_main, total_stock: 140, reserved_stock: 0, low_stock_threshold: 14, created_at: daysAgo(7), updated_at: daysAgo(1) },

]);
print(`  Inserted ${db.inventory.countDocuments()} stock items`);

// ============================================================
// 7-11. Orders, payments, shipments, reviews, wishlists start EMPTY:
//       place real orders from the storefront to try the whole flow.
// ============================================================
["orders", "payments", "shipments", "reviews", "wishlists", "purchases", "vendors"].forEach((c) => db[c].drop());
print("  Orders, payments, shipments, reviews, wishlists: empty (start fresh)");

// ============================================================
// 12. MARKETING
// ============================================================
print("Seeding marketing data...");

// Segments
db.segments.drop();
db.segments.insertMany([
  {
    _id: UUIDS.segment1,
    name: "High-Value Customers",
    type: "DYNAMIC",
    description: "Customers who have spent more than $500 total",
    rules: [
      { field: "total_spent", operator: "gte", value: 50000 },
    ],
    customer_ids: [],
    customer_count: 2,
    created_at: daysAgo(30),
    updated_at: daysAgo(1),
  },
  {
    _id: "60000000-0000-4000-6000-000000000002",
    name: "Cart Abandoners",
    type: "DYNAMIC",
    description: "Customers who abandoned their cart in the last 7 days",
    rules: [
      { field: "event_type", operator: "eq", value: "cart_abandoned" },
    ],
    customer_ids: [],
    customer_count: 5,
    created_at: daysAgo(14),
    updated_at: daysAgo(1),
  },
]);
print(`  Inserted ${db.segments.countDocuments()} segments`);

// Campaigns
db.campaigns.drop();
db.campaigns.insertMany([
  {
    _id: UUIDS.campaign1,
    name: "Summer Sale Abandoned Cart Recovery",
    type: "ABANDONED_CART",
    status: "ACTIVE",
    description: "Recover abandoned carts during summer sale with a 10% discount reminder",
    rules: {
      segment_ids: ["60000000-0000-4000-6000-000000000002"],
      event_types: ["cart_abandoned"],
      product_ids: [],
      category_ids: [],
      min_abandoned_minutes: 30,
      max_emails_per_user: 3,
    },
    metrics: {
      events_received: 142,
      customers_reached: 89,
      messages_sent: 67,
      conversions: 23,
      revenue_cents: 456780,
    },
    start_at: daysAgo(14),
    end_at: daysAgo(-14),
    created_at: daysAgo(14),
    updated_at: daysAgo(1),
  },
]);
print(`  Inserted ${db.campaigns.countDocuments()} campaigns`);

// Promotions
db.promotions.drop();
db.promotions.insertMany([
  {
    _id: UUIDS.promo1,
    name: "Welcome 10% Off",
    code: "WELCOME10",
    description: "10% off your order (max \u09f3300), minimum order \u09f3500",
    status: "ACTIVE",
    discount_type: "PERCENTAGE",
    discount_value: 10,
    min_order_cents: 500,      // whole BDT (field name is historical)
    max_discount_cents: 300,   // whole BDT
    usage_limit: 500,
    usage_count: 0,
    applies_to_product_ids: [],
    applies_to_category_ids: [],
    campaign_id: UUIDS.campaign1,
    start_at: daysAgo(14),
    end_at: daysAgo(-60),
    created_at: daysAgo(14),
    updated_at: daysAgo(1),
  },
  {
    _id: "80000000-0000-4000-8000-000000000002",
    name: "Free Delivery",
    code: "FREESHIP",
    description: "Free delivery on orders of \u09f31,000 or more",
    status: "ACTIVE",
    discount_type: "FREE_SHIPPING",
    discount_value: 0,
    min_order_cents: 1000,
    max_discount_cents: 0,
    usage_limit: 0,
    usage_count: 0,
    applies_to_product_ids: [],
    applies_to_category_ids: [],
    campaign_id: null,
    start_at: daysAgo(30),
    end_at: null,
    created_at: daysAgo(30),
    updated_at: daysAgo(1),
  },
  {
    _id: "80000000-0000-4000-8000-000000000003",
    name: "Flat \u09f3100 Off",
    code: "FLAT100",
    description: "\u09f3100 off orders of \u09f31,500 or more",
    status: "ACTIVE",
    discount_type: "FIXED",
    discount_value: 100,
    min_order_cents: 1500,
    max_discount_cents: 0,
    usage_limit: 0,
    usage_count: 0,
    applies_to_product_ids: [],
    applies_to_category_ids: [],
    campaign_id: null,
    start_at: daysAgo(30),
    end_at: null,
    created_at: daysAgo(30),
    updated_at: daysAgo(1),
  },
]);
print(`  Inserted ${db.promotions.countDocuments()} promotions`);

// ============================================================
// Summary
// ============================================================
print("\n=== Seed Complete ===");
print(`  Users:       ${db.users.countDocuments()}`);
print(`  Customers:   ${db.customers.countDocuments()}`);
print(`  Categories:  ${db.categories.countDocuments()}`);
print(`  Warehouses:  ${db.warehouses.countDocuments()}`);
print(`  Products:    ${db.products.countDocuments()}`);
print(`  Inventory:   ${db.inventory.countDocuments()}`);
print(`  Orders:      ${db.orders.countDocuments()}`);
print(`  Payments:    ${db.payments.countDocuments()}`);
print(`  Shipments:   ${db.shipments.countDocuments()}`);
print(`  Reviews:     ${db.reviews.countDocuments()}`);
print(`  Wishlists:   ${db.wishlists.countDocuments()}`);
print(`  Segments:    ${db.segments.countDocuments()}`);
print(`  Campaigns:   ${db.campaigns.countDocuments()}`);
print(`  Promotions:  ${db.promotions.countDocuments()}`);
print("\nTest Accounts (password: Password123!):");
print("  Super Admin:  admin@ecommerce.com");
print("  Admin:        manager@ecommerce.com");
print("  Customer:     john@example.com");
print("  Customer:     jane@example.com");
print("  Customer:     bob@example.com");
print("====================\n");