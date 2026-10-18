import type { Category, Product } from '@/types';

export interface SearchResult {
  type: 'product' | 'category';
  id: string;
  label: string;
  sublabel: string;
  image?: string;
  path: string;
}

export function searchAll(query: string, products: Product[], categories: Category[]): SearchResult[] {
  const q = query.trim().toLowerCase();
  if (!q) return [];

  const results: SearchResult[] = [];

  products
    .filter(
      (p) =>
        p.name.toLowerCase().includes(q) ||
        p.description.toLowerCase().includes(q) ||
        p.category.toLowerCase().includes(q) ||
        p.brand.toLowerCase().includes(q) ||
        p.tags.some((t) => t.toLowerCase().includes(q)),
    )
    .slice(0, 5)
    .forEach((p) => {
      results.push({ type: 'product', id: p.id, label: p.name, sublabel: p.category, image: p.image, path: `/products/${p.id}` });
    });

  categories
    .filter((c) => c.label.toLowerCase().includes(q) || c.description.toLowerCase().includes(q))
    .slice(0, 3)
    .forEach((c) => {
      results.push({ type: 'category', id: c.slug, label: c.label, sublabel: c.description, path: `/products?category=${c.slug}` });
    });

  return results;
}
