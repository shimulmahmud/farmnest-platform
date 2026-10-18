import { getPage } from './http';
import type { Category, Product, ProductVariant } from '@/types';

// ---- Backend shapes (public API: no cost, no exact stock) -------------------
interface ApiImage { url: string; alt?: string }
interface ApiVariant {
  id: string;
  sku: string;
  options: Record<string, string> | null;
  price: number;
  compare_at_price?: number;
  weight_g: number;
  images: ApiImage[] | null;
  in_stock: boolean;
}
interface ApiProduct {
  id: string;
  name: string;
  slug: string;
  description: string;
  short_description: string;
  brand: string;
  category_id: string;
  unit: string;
  specs: Record<string, string> | null;
  tags: string[] | null;
  images: ApiImage[] | null;
  variants: ApiVariant[] | null;
  lowest_price: number;
  in_stock: boolean;
  rating?: number;
  review_count?: number;
}
interface ApiCategory {
  id: string;
  name: string;
  slug: string;
  description: string;
  image: string;
  sort_order: number;
  status: string;
}

const PLACEHOLDER = 'https://placehold.co/600x600/f1f5ef/5a6b4a?text=No+image';

const unitLabel = (unit: string): string => {
  const map: Record<string, string> = { piece: 'piece', kg: 'per kg', g: 'per g', litre: 'per litre', dozen: 'per dozen', pack: 'per pack', pair: 'per pair', set: 'per set', kit: 'per kit' };
  return map[unit] ?? unit;
};

const variantLabel = (options: Record<string, string> | null): string => {
  const values = Object.values(options ?? {}).filter(Boolean);
  return values.length ? values.join(' · ') : 'Standard';
};

export function toProduct(p: ApiProduct, categories: Category[]): Product {
  const cat = categories.find((c) => c.id === p.category_id);
  const variants: ProductVariant[] = (p.variants ?? []).map((v) => ({
    id: v.id,
    sku: v.sku,
    label: variantLabel(v.options),
    options: v.options ?? {},
    price: v.price,
    originalPrice: v.compare_at_price && v.compare_at_price > v.price ? v.compare_at_price : null,
    inStock: v.in_stock,
    weightG: v.weight_g,
    image: v.images?.[0]?.url,
  }));

  // The "headline" variant: cheapest one that is in stock, else cheapest overall.
  const pool = variants.filter((v) => v.inStock);
  const headline = (pool.length ? pool : variants).slice().sort((a, b) => a.price - b.price)[0];

  const images = (p.images ?? []).map((i) => i.url).filter(Boolean);
  const certs = (p.specs?.certifications ?? '').split(',').map((s) => s.trim()).filter(Boolean);
  const discount = headline?.originalPrice ? Math.round((1 - headline.price / headline.originalPrice) * 100) : 0;

  return {
    id: p.slug, // used in URLs, wishlist and cart keys
    backendId: p.id,
    name: p.name,
    category: cat?.label ?? 'Uncategorized',
    categorySlug: cat?.slug ?? 'uncategorized',
    price: headline?.price ?? p.lowest_price,
    originalPrice: headline?.originalPrice ?? null,
    unit: unitLabel(p.unit || 'piece'),
    rating: p.rating ?? 0,
    reviews: p.review_count ?? 0,
    badge: discount > 0 ? `${discount}% OFF` : undefined,
    image: images[0] ?? headline?.image ?? PLACEHOLDER,
    images: images.length ? images : [headline?.image ?? PLACEHOLDER],
    description: p.description || p.short_description || '',
    shortDescription: p.short_description || '',
    inStock: p.in_stock,
    brand: p.brand || '',
    certifications: certs,
    specs: p.specs ?? {},
    tags: p.tags ?? [],
    variants,
    hasChoices: variants.length > 1,
  };
}

export function toCategory(c: ApiCategory, counts: Record<string, number>): Category {
  return {
    id: c.id,
    slug: c.slug,
    label: c.name,
    icon: c.slug,
    count: counts[c.id] ?? 0,
    description: c.description || '',
    image: c.image || undefined,
  };
}

/** Load every page of a public list endpoint. */
async function loadAll<T>(path: string, query: Record<string, string | number> = {}): Promise<T[]> {
  const out: T[] = [];
  for (let page = 1; page <= 50; page++) {
    const res = await getPage<T>(path, { ...query, page, per_page: 100 });
    out.push(...res.items);
    if (page >= res.totalPages) break;
  }
  return out;
}

export async function fetchCatalog(): Promise<{ categories: Category[]; products: Product[] }> {
  const [apiCategories, apiProducts] = await Promise.all([
    loadAll<ApiCategory>('/categories', { status: 'ACTIVE' }),
    loadAll<ApiProduct>('/products'),
  ]);

  const counts: Record<string, number> = {};
  for (const p of apiProducts) counts[p.category_id] = (counts[p.category_id] ?? 0) + 1;

  const categories = apiCategories
    .slice()
    .sort((a, b) => a.sort_order - b.sort_order)
    .map((c) => toCategory(c, counts));

  return { categories, products: apiProducts.map((p) => toProduct(p, categories)) };
}
