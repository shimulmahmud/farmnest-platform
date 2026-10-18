import { createContext, useContext, useEffect, useMemo, useState, type ReactNode } from 'react';
import { CART_TTL_MS } from '@/config/checkout';
import { useCatalog } from '@/context/CatalogContext';
import type { Product, ProductVariant } from '@/types';

/** One line in the cart: a specific variant of a product. */
export interface CartLine {
  product: Product;
  variant: ProductVariant;
  quantity: number;
}

interface CartContextValue {
  lines: CartLine[];
  itemCount: number;
  subtotal: number;
  /** Add one unit. Without a variant, the product's headline (cheapest in-stock) variant is used. */
  addItem: (product: Product, variant?: ProductVariant, quantity?: number) => void;
  updateQuantity: (variantId: string, quantity: number) => void;
  removeItem: (variantId: string) => void;
  clearCart: () => void;
}

const CartContext = createContext<CartContextValue | undefined>(undefined);
const STORAGE_KEY = 'farmnest-cart-v2';

interface Stored {
  savedAt: number;
  lines: { product: Product; variant: ProductVariant; quantity: number }[];
}

/**
 * The cart lives in the browser (localStorage) and expires 1 hour after its last change
 * (CART_TTL_MS). Nothing here touches stock: stock only moves when an order is placed.
 */
function loadInitial(): Stored['lines'] {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) return [];
    const parsed = JSON.parse(raw) as Stored;
    if (!parsed || !Array.isArray(parsed.lines)) return [];
    if (Date.now() - parsed.savedAt > CART_TTL_MS) {
      localStorage.removeItem(STORAGE_KEY);
      return [];
    }
    return parsed.lines.filter((l) => l && l.product && l.variant && typeof l.quantity === 'number');
  } catch {
    return [];
  }
}

export const headlineVariant = (product: Product): ProductVariant | undefined =>
  product.variants.filter((v) => v.inStock).sort((a, b) => a.price - b.price)[0];

export function CartProvider({ children }: { children: ReactNode }) {
  const [stored, setStored] = useState(loadInitial);
  const { products, loading } = useCatalog();

  // Persist with a fresh timestamp on every change (so the 1 hour counts from the last edit).
  useEffect(() => {
    try {
      if (stored.length === 0) localStorage.removeItem(STORAGE_KEY);
      else localStorage.setItem(STORAGE_KEY, JSON.stringify({ savedAt: Date.now(), lines: stored }));
    } catch {
      /* storage unavailable (private mode) */
    }
  }, [stored]);

  // Expire an open tab too, not just on reload.
  useEffect(() => {
    if (stored.length === 0) return;
    const timer = window.setInterval(() => {
      try {
        const raw = localStorage.getItem(STORAGE_KEY);
        const savedAt = raw ? (JSON.parse(raw) as Stored).savedAt : 0;
        if (Date.now() - savedAt > CART_TTL_MS) setStored([]);
      } catch {
        /* ignore */
      }
    }, 60_000);
    return () => window.clearInterval(timer);
  }, [stored.length]);

  // Always show LIVE price/stock from the catalog, never a stale copy saved in the browser.
  const lines = useMemo<CartLine[]>(() => {
    if (loading || products.length === 0) return stored;
    const out: CartLine[] = [];
    for (const line of stored) {
      const product = products.find((p) => p.id === line.product.id);
      const variant = product?.variants.find((v) => v.id === line.variant.id);
      if (product && variant) out.push({ product, variant, quantity: line.quantity });
      // product or variant no longer sold: drop the line
    }
    return out;
  }, [stored, products, loading]);

  const value = useMemo<CartContextValue>(() => {
    const updateQuantity = (variantId: string, quantity: number) =>
      setStored((cur) =>
        quantity < 1
          ? cur.filter((l) => l.variant.id !== variantId)
          : cur.map((l) => (l.variant.id === variantId ? { ...l, quantity } : l)),
      );

    return {
      lines,
      itemCount: lines.reduce((n, l) => n + l.quantity, 0),
      subtotal: lines.reduce((n, l) => n + l.variant.price * l.quantity, 0),
      addItem: (product, variant, quantity = 1) => {
        const v = variant ?? headlineVariant(product);
        if (!v || !v.inStock) return;
        setStored((cur) => {
          const existing = cur.find((l) => l.variant.id === v.id);
          if (existing) return cur.map((l) => (l.variant.id === v.id ? { ...l, quantity: l.quantity + quantity } : l));
          return [...cur, { product, variant: v, quantity }];
        });
      },
      updateQuantity,
      removeItem: (variantId) => updateQuantity(variantId, 0),
      clearCart: () => setStored([]),
    };
  }, [lines]);

  return <CartContext.Provider value={value}>{children}</CartContext.Provider>;
}

export function useCart(): CartContextValue {
  const context = useContext(CartContext);
  if (!context) throw new Error('useCart must be used inside CartProvider');
  return context;
}
