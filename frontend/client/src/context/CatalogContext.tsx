import { createContext, useCallback, useContext, useEffect, useMemo, useState, type ReactNode } from 'react';
import { fetchCatalog } from '@/api/catalog';
import type { Category, Product } from '@/types';

interface CatalogValue {
  products: Product[];
  categories: Category[];
  loading: boolean;
  error: string | null;
  reload: () => void;
  /** Find by slug (the id used in URLs). */
  getProduct: (id: string) => Product | undefined;
}

const CatalogContext = createContext<CatalogValue | undefined>(undefined);

/** Loads the whole catalog (products + categories) from the backend once, and on demand. */
export function CatalogProvider({ children }: { children: ReactNode }) {
  const [products, setProducts] = useState<Product[]>([]);
  const [categories, setCategories] = useState<Category[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [nonce, setNonce] = useState(0);

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    fetchCatalog()
      .then((data) => {
        if (cancelled) return;
        setProducts(data.products);
        setCategories(data.categories);
        setError(null);
      })
      .catch((e: unknown) => {
        if (!cancelled) setError(e instanceof Error ? e.message : 'Could not load products');
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [nonce]);

  const reload = useCallback(() => setNonce((n) => n + 1), []);

  const value = useMemo<CatalogValue>(
    () => ({
      products,
      categories,
      loading,
      error,
      reload,
      getProduct: (id: string) => products.find((p) => p.id === id),
    }),
    [products, categories, loading, error, reload],
  );

  return <CatalogContext.Provider value={value}>{children}</CatalogContext.Provider>;
}

export function useCatalog(): CatalogValue {
  const ctx = useContext(CatalogContext);
  if (!ctx) throw new Error('useCatalog must be used inside CatalogProvider');
  return ctx;
}
