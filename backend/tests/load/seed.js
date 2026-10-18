// seed.js — Seed MongoDB with test data for k6 load tests.
// Run: k6 run tests/load/seed.js
// Prerequisite: API server and MongoDB must be running.
//
// This script creates:
//   - 1 admin user
//   - 50 customer users
//   - 5 categories
//   - 20 products (each with 1 variant)
//   - 20 inventory stock entries
//
// All entities use the "loadtest-" prefix for easy cleanup after tests.

import http from 'k6/http';
import { check, sleep } from 'k6';
import { BASE_URL } from './lib/config.js';

export const options = {
  vus: 1,
  iterations: 1,
  noConnectionReuse: false,
};

const ADMIN_EMAIL = 'loadtest-admin@ecommerce.test';
const ADMIN_PASSWORD = 'LoadTest1234!';
const CUSTOMER_COUNT = 50;
const CATEGORY_COUNT = 5;
const PRODUCT_COUNT = 20;

export default function () {
  const jsonHeaders = { 'Content-Type': 'application/json' };

  // --- 1. Register admin ---
  console.log('Seeding admin user...');
  const adminRes = http.post(`${BASE_URL}/api/v1/auth/register`, JSON.stringify({
    email: ADMIN_EMAIL,
    password: ADMIN_PASSWORD,
    first_name: 'LoadTest',
    last_name: 'Admin',
  }), { headers: jsonHeaders });

  if (adminRes.status === 201 || adminRes.status === 409) {
    console.log('Admin user ready (created or already exists).');
  } else {
    console.error(`Admin registration failed: ${adminRes.status} ${adminRes.body}`);
  }

  // --- 2. Register customers ---
  console.log(`Seeding ${CUSTOMER_COUNT} customer users...`);
  let customerIds = [];
  for (let i = 1; i <= CUSTOMER_COUNT; i++) {
    const email = `loadtest-customer-${i}@ecommerce.test`;
    const res = http.post(`${BASE_URL}/api/v1/auth/register`, JSON.stringify({
      email: email,
      password: 'LoadTest1234!',
      first_name: `Customer${i}`,
      last_name: 'LoadTest',
    }), { headers: jsonHeaders });

    if (res.status === 201) {
      const body = res.json('data') || res.json();
      if (body && body.user && body.user.id) {
        customerIds.push(body.user.id);
      }
    }
    // 409 (already exists) is fine for re-runs.
  }
  console.log(`Customers seeded: ${customerIds.length} new, ${CUSTOMER_COUNT} total expected.`);

  // --- 3. Login as admin ---
  const loginRes = http.post(`${BASE_URL}/api/v1/auth/login`, JSON.stringify({
    email: ADMIN_EMAIL,
    password: ADMIN_PASSWORD,
  }), { headers: jsonHeaders });

  if (loginRes.status !== 200) {
    console.error(`Admin login failed: ${loginRes.status} ${loginRes.body}`);
    return;
  }

  const loginBody = loginRes.json('data') || loginRes.json();
  const adminToken = loginBody.tokens.access_token;
  const authHeaders = {
    'Content-Type': 'application/json',
    'Authorization': `Bearer ${adminToken}`,
  };

  // --- 4. Create categories ---
  console.log(`Seeding ${CATEGORY_COUNT} categories...`);
  let categoryIds = [];
  for (let i = 1; i <= CATEGORY_COUNT; i++) {
    const res = http.post(`${BASE_URL}/api/v1/admin/categories`, JSON.stringify({
      name: `LoadTest Category ${i}`,
      slug: `loadtest-category-${i}`,
      description: `Category ${i} for load testing`,
    }), { headers: authHeaders });

    if (res.status === 201) {
      const body = res.json('data') || res.json();
      if (body && body.id) {
        categoryIds.push(body.id);
      }
    }
  }
  console.log(`Categories seeded: ${categoryIds.length} new.`);

  if (categoryIds.length === 0) {
    // Try to fetch existing ones.
    const listRes = http.get(`${BASE_URL}/api/v1/categories`, { headers: { 'Accept': 'application/json' } });
    if (listRes.status === 200) {
      const body = listRes.json('data') || listRes.json();
      if (Array.isArray(body)) {
        for (const cat of body) {
          if (cat.slug && cat.slug.startsWith('loadtest-category-')) {
            categoryIds.push(cat.id);
          }
        }
      }
    }
  }

  // --- 5. Create products ---
  console.log(`Seeding ${PRODUCT_COUNT} products...`);
  let productData = []; // { id, slug, variant_id, sku }
  for (let i = 1; i <= PRODUCT_COUNT; i++) {
    const catId = categoryIds[(i - 1) % Math.max(categoryIds.length, 1)] || '000000000000000000000000';
    const slug = `loadtest-product-${i}`;
    const sku = `LT-SKU-${String(i).padStart(4, '0')}`;
    const price = 1000 + i * 500;

    const res = http.post(`${BASE_URL}/api/v1/admin/products`, JSON.stringify({
      name: `LoadTest Product ${i}`,
      slug: slug,
      description: `Product ${i} created for load testing. This is a sample product description with some text.`,
      short_description: `Short desc for product ${i}`,
      brand: `TestBrand${(i % 5) + 1}`,
      category_id: catId,
      tags: [`tag${(i % 3) + 1}`, `loadtest`],
      variants: [{
        sku: sku,
        price: price,
        cost_price: Math.floor(price * 0.6),
        stock: 1000,
        weight: 0.5 + i * 0.1,
        attributes: { size: 'M', color: ['red', 'blue', 'green'][i % 3] },
      }],
    }), { headers: authHeaders });

    if (res.status === 201) {
      const body = res.json('data') || res.json();
      if (body && body.id) {
        let variantId = '';
        if (body.variants && body.variants.length > 0) {
          variantId = body.variants[0].id;
        }
        productData.push({ id: body.id, slug: slug, variant_id: variantId, sku: sku });
      }
    }
  }
  console.log(`Products seeded: ${productData.length} new.`);

  // --- 6. Create inventory stock entries ---
  console.log('Seeding inventory stock entries...');
  // First create a warehouse.
  const whRes = http.post(`${BASE_URL}/api/v1/admin/inventory/warehouses`, JSON.stringify({
    name: 'LoadTest Warehouse',
    location: 'Dhaka, Bangladesh',
  }), { headers: authHeaders });

  let warehouseId = '000000000000000000000000';
  if (whRes.status === 201) {
    const whBody = whRes.json('data') || whRes.json();
    if (whBody && whBody.id) {
      warehouseId = whBody.id;
    }
  }

  for (const prod of productData) {
    http.post(`${BASE_URL}/api/v1/admin/inventory/stock`, JSON.stringify({
      sku: prod.sku,
      warehouse_id: warehouseId,
      quantity: 10000,
    }), { headers: authHeaders });
  }

  // --- 7. Output seed summary as env for downstream scripts ---
  console.log('=== SEED COMPLETE ===');
  console.log(`Categories: ${categoryIds.length}`);
  console.log(`Products: ${productData.length}`);
  console.log(`Warehouse: ${warehouseId}`);
  console.log('Seed data is ready for load tests.');
}

export function handleSummary(data) {
  return {
    'stdout': 'Seed completed. Review logs above for counts.\n',
  };
}
