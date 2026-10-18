import { useWishlist } from "@/context/WishlistContext";
import type { Product } from "@/types";
import { Heart } from "lucide-react";

interface WishlistButtonProps {
  product: Pick<Product, "id" | "name">;
  /** Positioning classes, e.g. "absolute right-2 top-2". */
  className?: string;
}

export default function WishlistButton({ product, className = "" }: WishlistButtonProps) {
  const { has, toggle } = useWishlist();
  const active = has(product.id);

  return (
    <button
      type="button"
      onClick={(e) => {
        e.preventDefault();
        e.stopPropagation();
        toggle(product.id);
      }}
      aria-pressed={active}
      aria-label={active ? `Remove ${product.name} from wishlist` : `Add ${product.name} to wishlist`}
      title={active ? "Remove from wishlist" : "Add to wishlist"}
      className={`z-10 flex h-8 w-8 items-center justify-center rounded-full bg-white/90 shadow-level-1 transition-all hover:scale-110 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary ${className}`}
    >
      <Heart className={`h-4 w-4 ${active ? "fill-error text-error" : "text-on-surface-variant"}`} />
    </button>
  );
}
