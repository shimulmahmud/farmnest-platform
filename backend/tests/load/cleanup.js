// cleanup.js — Remove all load test seed data from MongoDB via API.
// Run: k6 run tests/load/cleanup.js
//
// Deletes all users, products, categories, and orders with the "loadtest-" prefix.

import http from 'k6/http';
import { check } from 'k6';
import { BASE_URL } from './lib/config.js';

export const options = {
  vus: 1,
  iterations: 1,
};

export default function () {
  const jsonHeaders = { 'Content-Type': 'application/json' };

  // Login as admin.
  const loginRes = http.post(`${BASE_URL}/api/v1/auth/login`, JSON.stringify({
    email: 'loadtest-admin@ecommerce.test',
    password: 'LoadTest1234!',
  }), { headers: jsonHeaders });

  if (loginRes.status !== 200) {
    console.error('Cannot login as admin for cleanup. Manual cleanup required.');
    return;
  }

  const loginBody = loginRes.json('data') || loginRes.json();
  const token = loginBody.tokens.access_token;
  const authHeaders = {
    'Content-Type': 'application/json',
    'Authorization': `Bearer ${token}`,
  };

  // List and delete loadtest products.
  let page = 1;
  let deleted = 0;
  while (true) {
    const res = http.get(
      `${BASE_URL}/api/v1/products?page=${page}&per_page=100`,
      { headers: { 'Accept': 'application/json' } },
    );
    if (res.status !== 200) break;
    const body = res.json('data') || res.json();
    const items = Array.isArray(body) ? body : (body.items || []);
    if (items.length === 0) break;

    let hasMore = false;
    for (const p of items) {
      if (p.slug && p.slug.startsWith('loadtest-')) {
        http.del(`${BASE_URL}/api/v1/admin/products/${p.id}`, { headers: authHeaders });
        deleted++;
      } else if (p.slug && !p.slug.startsWith('loadtest-')) {
        hasMore = true;
      }
    }
    page++;
    if (page > 10) break; // safety limit
  }
  console.log(`Deleted ${deleted} loadtest products.`);

  // List and delete loadtest categories.
  page = 1;
  deleted = 0;
  while (true) {
    const res = http.get(`${BASE_URL}/api/v1/categories?page=${page}&per_page=100`,
      { headers: { 'Accept': 'application/json' } });
    if (res.status !== 200) break;
    const body = res.json('data') || res.json();
    const items = Array.isArray(body) ? body : (body.items || []);
    if (items.length === 0) break;

    for (const c of items) {
      if (c.slug && c.slug.startsWith('loadtest-category-')) {
        http.del(`${BASE_URL}/api/v1/admin/categories/${c.id}`, { headers: authHeaders });
        deleted++;
      }
    }
    page++;
    if (page > 5) break;
  }
  console.log(`Deleted ${deleted} loadtest categories.`);
  console.log('Cleanup complete. Note: user records must be deleted directly from MongoDB.');
  console.log('  db.users.deleteMany({ email: /loadtest-/ })');
}

export function handleSummary() {
  return { 'stdout': 'Cleanup completed. Review logs above.
' };
}
