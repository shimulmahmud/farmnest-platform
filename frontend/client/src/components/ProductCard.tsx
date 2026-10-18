import { badgeColors } from "@/config/productBadges";
import { useCart } from "@/context/CartContext";
import type { Product } from "@/types";
import { Check, ShoppingCart, Star, X } from "lucide-react";
import { useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import { StarRating } from "@/components/ui";
import AddToCartControl from "@/components/AddToCartControl";
import Price from "@/components/Price";
import { getPricing } from "@/lib/pricing";
import WishlistButton from "@/components/WishlistButton";

interface ProductCardProps {
  product: Product;
  /** `compact` is the dense shelf tile used on the home page. */
  variant?: "default" | "compact";
}

export default function ProductCard({ product, variant = "default" }: ProductCardProps) {
  return variant === "compact" ? (
    <CompactProductCard product={product} />
  ) : (
    <DefaultProductCard product={product} />
  );
}

/**
 * Dense standalone tile for product grids; lay these out with a `gap` between them.
 */
function CompactProductCard({ product }: { product: Product }) {
  const { discountPercent } = getPricing(product);

  return (
    <article className="group flex flex-col rounded-xl border border-outline-variant bg-white p-3 transition-shadow duration-300 hover:shadow-level-2 sm:p-4">
      <div className="relative">
        <Link
          to={`/products/${product.id}`}
          className="block aspect-[4/3] overflow-hidden rounded-lg bg-surface-container-low"
        >
          <img
            src={product.image}
            alt={product.name}
            loading="lazy"
            className="h-full w-full object-cover transition-transform duration-500 ease-out group-hover:scale-105"
          />
        </Link>

        {discountPercent > 0 ? (
          <span className="absolute left-2 top-2 rounded bg-error px-1.5 py-0.5 text-[10px] font-bold leading-4 text-white">
            -{discountPercent}%
          </span>
        ) : (
          product.badge && (
            <span className="absolute left-2 top-2 rounded bg-primary px-1.5 py-0.5 text-[10px] font-bold leading-4 text-white">
              {product.badge}
            </span>
          )
        )}

        <WishlistButton product={product} className="absolute right-2 top-2" />

        <div className="absolute bottom-2 right-2">
          <AddToCartControl product={product} />
        </div>
      </div>

      <div className="mt-3 flex flex-1 flex-col">
        <span className="text-[11px] text-on-surface-variant">{product.category}</span>

        <Link to={`/products/${product.id}`}>
          <h3 className="mt-0.5 line-clamp-2 min-h-[2.5rem] font-headline text-sm font-bold leading-5 text-on-surface transition-colors group-hover:text-primary">
            {product.name}
          </h3>
        </Link>

        <p className="mt-1.5 flex flex-wrap items-baseline gap-x-1.5">
          <Price product={product} />
          <span className="text-[11px] text-outline">{product.unit}</span>
        </p>

        {product.reviews > 0 && (
          <div className="mt-1 flex items-center gap-1">
            <StarRating rating={Math.round(product.rating)} size="h-3 w-3" />
            <span className="text-[11px] text-outline">({product.reviews})</span>
          </div>
        )}

        <p
          className={`mt-2 flex items-center gap-1 text-[11px] font-medium ${
            product.inStock ? "text-success-dark" : "text-error"
          }`}
        >
          {product.inStock ? (
            <Check className="h-3 w-3" strokeWidth={2.5} />
          ) : (
            <X className="h-3 w-3" strokeWidth={2.5} />
          )}
          {product.inStock ? "In Stock" : "Stock Out"}
        </p>
      </div>
    </article>
  );
}

function DefaultProductCard({ product }: { product: Product }) {
  const { addItem } = useCart();
  const navigate = useNavigate();
  const { discountPercent } = getPricing(product);
  const [added, setAdded] = useState(false);

  const handleAdd = (e: React.MouseEvent) => {
    e.preventDefault();
    e.stopPropagation();
    if (product.hasChoices) {
      navigate(`/products/${product.id}`); // pick a size / pack first
      return;
    }
    addItem(product);
    setAdded(true);
    setTimeout(() => setAdded(false), 1500);
  };

  return (
    <div className="group relative flex flex-col overflow-hidden rounded-2xl border border-stone-200/80 bg-white transition-all duration-300 hover:-translate-y-1 hover:border-primary/20 hover:shadow-xl hover:shadow-stone-900/5">
      <WishlistButton product={product} className="absolute right-3 top-3 h-9 w-9" />

      <Link
        to={`/products/${product.id}`}
        className="relative block aspect-square overflow-hidden bg-[#f7f8f3]"
      >
        <img
          src={product.image}
          alt={product.name}
          loading="lazy"
          className="h-full w-full object-cover transition-transform duration-500 ease-out group-hover:scale-110"
        />

        <div className="pointer-events-none absolute inset-0 bg-gradient-to-t from-black/5 via-transparent to-transparent" />

        {discountPercent > 0 ? (
          <span className="absolute left-3 top-3 rounded-full bg-error px-3 py-1 text-[11px] font-bold tracking-wide text-white shadow-sm">
            -{discountPercent}%
          </span>
        ) : (
          product.badge && (
            <span
              className={`absolute left-3 top-3 rounded-full border px-3 py-1 text-[11px] font-semibold tracking-wide shadow-sm backdrop-blur-sm ${
                badgeColors[product.badge] ||
                "border-stone-200 bg-white/90 text-stone-700"
              }`}
            >
              {product.badge}
            </span>
          )
        )}

        {!product.inStock && (
          <div className="absolute inset-0 flex items-center justify-center bg-white/60">
            <span className="rounded-full bg-stone-900/85 px-4 py-2 text-xs font-semibold text-white">
              Out of Stock
            </span>
          </div>
        )}
      </Link>

      <div className="flex flex-1 flex-col p-4 sm:p-5">
        <div className="mb-2 flex min-h-6 flex-wrap items-center gap-2">
          <span className="rounded-md bg-primary/5 px-2.5 py-1 text-[11px] font-semibold text-primary">
            {product.category}
          </span>

          {product.inStock && (
            <span className="flex items-center gap-1 text-[11px] font-medium text-emerald-700">
              <Check className="h-3 w-3" strokeWidth={2.5} />
              In Stock
            </span>
          )}
        </div>

        <Link to={`/products/${product.id}`} className="mb-1.5">
          <h3 className="line-clamp-2 font-headline text-sm font-semibold leading-snug text-stone-900 transition-colors duration-200 group-hover:text-primary sm:text-base">
            {product.name}
          </h3>
        </Link>

        <p className="mb-3 line-clamp-2 flex-1 text-xs leading-relaxed text-stone-500 sm:text-sm">
          {product.description}
        </p>

        {product.reviews > 0 && (
          <div className="mb-3 flex items-center gap-1.5">
            <div className="flex items-center gap-1">
              <Star className="h-3.5 w-3.5 fill-amber-400 text-amber-400" />
              <span className="text-sm font-semibold text-stone-700">
                {product.rating}
              </span>
            </div>
            <span className="text-xs text-stone-400">
              ({product.reviews} reviews)
            </span>
          </div>
        )}

        <div className="flex items-end justify-between gap-2 border-t border-stone-100 pt-3">
          <div className="min-w-0">
            <div className="flex flex-wrap items-baseline gap-x-2 font-headline text-lg font-bold leading-tight text-primary sm:text-xl">
              <Price
                product={product}
                className=""
                originalClassName="text-xs font-normal text-stone-400 line-through"
              />
            </div>
            <div className="mt-1 text-xs text-stone-400">{product.hasChoices ? `From · ${product.unit}` : product.unit}</div>
          </div>

          <button
            onClick={handleAdd}
            disabled={!product.inStock}
            className={`flex h-10 w-10 flex-shrink-0 items-center justify-center rounded-xl transition-all duration-200 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary focus-visible:ring-offset-2 ${
              added
                ? "bg-emerald-600 text-white"
                : product.inStock
                  ? "bg-primary text-white hover:scale-105 hover:bg-primary-hover active:scale-95"
                  : "cursor-not-allowed bg-stone-200 text-stone-400"
            }`}
            aria-label={added ? "Added to cart" : "Add to cart"}
            title={!product.inStock ? "Out of stock" : "Add to cart"}
          >
            {added ? (
              <Check className="h-4 w-4" strokeWidth={2.5} />
            ) : (
              <ShoppingCart className="h-4 w-4" />
            )}
          </button>
        </div>
      </div>
    </div>
  );
}
