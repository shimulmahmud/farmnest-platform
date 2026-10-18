// helpers.js — Shared HTTP helpers and token management for k6.

import { check } from 'k6';
import http from 'k6/http';
import { BASE_URL, ADMIN_EMAIL, ADMIN_PASSWORD } from './config.js';

// --- Token cache (shared across VUs in the same instance) ---
let adminToken = null;
let adminTokenExpiry = 0;

const customerTokens = {}; // email -> { token, expiry }

/**
 * Login as admin and cache the access token.
 * Re-authenticates when the cached token expires.
 */
export function getAdminToken() {
  const now = Date.now() / 1000;
  if (adminToken && now < adminTokenExpiry - 30) {
    return adminToken;
  }

  const res = http.post(`${BASE_URL}/api/v1/auth/login`, JSON.stringify({
    email: ADMIN_EMAIL,
    password: ADMIN_PASSWORD,
  }), {
    headers: { 'Content-Type': 'application/json' },
    tags: { name: 'auth/login' },
  });

  check(res, {
    'admin login status 200': (r) => r.status === 200,
  'admin login has access_token': (r) => {
    const body = r.json('data') || r.json();
    return body && body.tokens && body.tokens.access_token;
  },
  });

  const body = res.json('data') || res.json();
  if (body && body.tokens) {
    adminToken = body.tokens.access_token;
    // Parse expires_at or default to 14 minutes.
    if (body.tokens.expires_at) {
      adminTokenExpiry = new Date(body.tokens.expires_at).getTime() / 1000;
    } else {
      adminTokenExpiry = now + 840;
    }
  }

  return adminToken;
}

/**
 * Login as a customer by index (0-based).
 * Caches tokens per customer.
 */
export function getCustomerToken(index) {
  const email = `loadtest-customer-${index + 1}@ecommerce.test`;
  const now = Date.now() / 1000;

  if (customerTokens[email] && now < customerTokens[email].expiry - 30) {
    return customerTokens[email].token;
  }

  const res = http.post(`${BASE_URL}/api/v1/auth/login`, JSON.stringify({
    email: email,
    password: 'LoadTest1234!',
  }), {
    headers: { 'Content-Type': 'application/json' },
    tags: { name: 'auth/login' },
  });

  const body = res.json('data') || res.json();
  if (body && body.tokens) {
    let expiry = now + 840;
    if (body.tokens.expires_at) {
      expiry = new Date(body.tokens.expires_at).getTime() / 1000;
    }
    customerTokens[email] = { token: body.tokens.access_token, expiry };
    return body.tokens.access_token;
  }

  return null;
}

/**
 * Build common headers with optional auth token.
 */
export function headers(token) {
  const h = {
    'Content-Type': 'application/json',
    'Accept': 'application/json',
  };
  if (token) {
    h['Authorization'] = `Bearer ${token}`;
  }
  return h;
}

/**
 * Build auth headers for admin.
 */
export function adminHeaders() {
  return headers(getAdminToken());
}

/**
 * Build auth headers for a customer by index.
 */
export function customerHeaders(index) {
  return headers(getCustomerToken(index));
}

/**
 * Random integer in [min, max] inclusive.
 */
export function randInt(min, max) {
  return Math.floor(Math.random() * (max - min + 1)) + min;
}

/**
 * Pick a random element from an array.
 */
export function randPick(arr) {
  return arr[Math.floor(Math.random() * arr.length)];
}

/**
 * Generate a unique-ish order item payload.
 * Uses pre-seeded product/variant IDs from the environment.
 */
export function makeOrderItem(productId, variantId, sku) {
  return {
    product_id: productId,
    variant_id: variantId,
    name: `Load Test Product ${productId.slice(-4)}`,
    sku: sku || `SKU-${productId.slice(-4)}`,
    quantity: randInt(1, 3),
    unit_price: randInt(1000, 50000),
    discount: 0,
  };
}

/**
 * Generate a valid shipping address payload.
 */
export function makeAddress() {
  return {
    line1: '123 Load Test Street',
    line2: 'Apt 4B',
    city: 'Dhaka',
    state: 'Dhaka Division',
    postal_code: '1205',
    country: 'BD',
  };
}

/**
 * Generate a full create-order payload.
 */
export function makeCreateOrderPayload(items) {
  return {
    items: items,
    shipping_address: makeAddress(),
    billing_address: makeAddress(),
  };
}
