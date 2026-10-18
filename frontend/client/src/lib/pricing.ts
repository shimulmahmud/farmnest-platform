import type { Product } from '@/types';

export interface Pricing {
  /** What the customer pays. */
  price: number;
  /** Old price, only set while the product is on sale. */
  originalPrice: number | null;
  discountPercent: number;
}

/**
 * Prices come from the backend catalog: `price` is what is charged, `compare_at_price`
 * (the old price) is only shown struck through. Discounts are set by the shop owner in the
 * admin panel, so what you see here is exactly what the server will charge.
 */
export function getPricing(product: Product): Pricing {
  const original = product.originalPrice;
  if (original && original > product.price) {
    return {
      price: product.price,
      originalPrice: original,
      discountPercent: Math.round((1 - product.price / original) * 100),
    };
  }
  return { price: product.price, originalPrice: null, discountPercent: 0 };
}

/** Products currently on sale. */
export const getDeals = (products: Product[]): Product[] =>
  products
    .filter((p) => p.originalPrice !== null && p.originalPrice > p.price)
    .sort((a, b) => getPricing(b).discountPercent - getPricing(a).discountPercent);
