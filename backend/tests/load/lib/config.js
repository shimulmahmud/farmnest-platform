// config.js — Shared configuration for k6 load tests.
// All timing values are in seconds (k6 uses milliseconds in options).

export const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

// Default thresholds applied to every scenario.
// These are conservative baselines — tighten after first real run.
export const DEFAULT_THRESHOLDS = {
  http_req_duration: ['p(95)<500', 'p(99)<2000'],
  http_req_failed: ['rate<0.05'],
};

// Pre-seeded admin credentials (created by seed.js).
export const ADMIN_EMAIL = 'loadtest-admin@ecommerce.test';
export const ADMIN_PASSWORD = 'LoadTest1234!';

// Pre-seeded customer credentials.
export const CUSTOMER_EMAIL_PREFIX = 'loadtest-customer-';
export const CUSTOMER_PASSWORD = 'LoadTest1234!';
export const CUSTOMER_COUNT = 50; // number of pre-seeded customers

// Pre-seeded product data.
export const PRODUCT_SLUGS = [];
for (let i = 1; i <= 20; i++) {
  PRODUCT_SLUGS.push(`loadtest-product-${i}`);
}

// Pre-seeded category slugs.
export const CATEGORY_SLUGS = [];
for (let i = 1; i <= 5; i++) {
  CATEGORY_SLUGS.push(`loadtest-category-${i}`);
}

// Pagination defaults.
export const DEFAULT_PAGE = 1;
export const DEFAULT_PER_PAGE = 20;
