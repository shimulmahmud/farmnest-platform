import ProductCard from '@/components/ProductCard';
import { useCart } from '@/context/CartContext';
import { useWishlist } from '@/context/WishlistContext';
import { useCatalog } from '@/context/CatalogContext';
import { Check, Heart, ShoppingCart, Trash2 } from 'lucide-react';
import { useState } from 'react';
import { Link } from 'react-router-dom';

export default function WishlistPage() {
  const { ids, clear } = useWishlist();
  const { products } = useCatalog();
  const { addItem } = useCart();
  const [added, setAdded] = useState(false);

  // Resolve ids against the catalog, dropping any that no longer exist.
  const items = ids.map((id) => products.find((p) => p.id === id)).filter((p) => p !== undefined);
  const available = items.filter((p) => p.inStock);

  const addAll = () => {
    // Products with several options need a choice on their own page, so only add single-option ones.
    available.filter((p) => !p.hasChoices).forEach((p) => addItem(p));
    setAdded(true);
    setTimeout(() => setAdded(false), 1800);
  };

  return (
    <div className="bg-white animate-fade-in">
      <div className="max-w-page mx-auto px-4 py-10 sm:px-6 lg:px-8 lg:py-12">
        <div className="mb-6 flex flex-wrap items-end justify-between gap-4">
          <div>
            <h1 className="font-headline text-2xl font-bold tracking-tight text-on-surface sm:text-3xl">My Wishlist</h1>
            <p className="mt-1 text-sm text-on-surface-variant">
              {items.length === 0 ? 'Nothing saved yet.' : `${items.length} saved ${items.length === 1 ? 'product' : 'products'}`}
            </p>
          </div>

          {items.length > 0 && (
            <div className="flex flex-wrap gap-3">
              <button
                type="button"
                onClick={addAll}
                disabled={available.length === 0}
                className="inline-flex items-center gap-2 rounded-lg bg-primary px-4 py-2.5 text-sm font-semibold text-white transition-colors hover:bg-primary-hover disabled:cursor-not-allowed disabled:bg-surface-container-highest disabled:text-outline"
              >
                {added ? <Check className="h-4 w-4" strokeWidth={2.5} /> : <ShoppingCart className="h-4 w-4" />}
                {added ? 'Added to cart' : 'Add all to cart'}
              </button>
              <button
                type="button"
                onClick={clear}
                className="inline-flex items-center gap-2 rounded-lg border border-outline-variant px-4 py-2.5 text-sm font-semibold text-on-surface-variant transition-colors hover:border-error hover:text-error"
              >
                <Trash2 className="h-4 w-4" /> Clear wishlist
              </button>
            </div>
          )}
        </div>

        {items.length === 0 ? (
          <div className="flex flex-col items-center rounded-xl border border-dashed border-outline-variant px-6 py-16 text-center">
            <span className="flex h-14 w-14 items-center justify-center rounded-full bg-primary-fixed text-primary">
              <Heart className="h-6 w-6" />
            </span>
            <h2 className="mt-4 font-headline text-lg font-bold text-on-surface">Your wishlist is empty</h2>
            <p className="mt-1 max-w-sm text-sm text-on-surface-variant">
              Tap the heart on any product to save it here for later.
            </p>
            <Link
              to="/products"
              className="mt-6 rounded-lg bg-primary px-5 py-2.5 text-sm font-semibold text-white transition-colors hover:bg-primary-hover"
            >
              Browse Products
            </Link>
          </div>
        ) : (
          <div className="grid grid-cols-2 gap-3 sm:gap-4 sm:grid-cols-3 lg:grid-cols-5">
            {items.map((product) => (
              <ProductCard key={product.id} product={product} variant="compact" />
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
