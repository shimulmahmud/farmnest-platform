import { Link } from "react-router-dom";
import { QuantitySelector } from "@/components/ui";
import { headlineVariant, useCart } from "@/context/CartContext";
import type { Product } from "@/types";
import { ShoppingCart } from "lucide-react";

/**
 * Round cart button that becomes a quantity stepper once the product is in the cart.
 * Products with several variants (sizes, packs...) send the buyer to the product page to choose.
 */
export default function AddToCartControl({ product }: { product: Product }) {
  const { lines, addItem, updateQuantity } = useCart();
  const headline = headlineVariant(product);
  const inCart = lines.filter((line) => line.product.id === product.id);

  // Single-variant product already in the cart: show a stepper.
  if (!product.hasChoices && inCart.length > 0) {
    const line = inCart[0];
    return (
      <div className="rounded-md bg-white shadow-level-2">
        <QuantitySelector
          size="sm"
          quantity={line.quantity}
          onDecrease={() => updateQuantity(line.variant.id, line.quantity - 1)}
          onIncrease={() => updateQuantity(line.variant.id, line.quantity + 1)}
        />
      </div>
    );
  }

  const base =
    "flex h-9 w-9 items-center justify-center rounded-full shadow-level-2 transition-all focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary focus-visible:ring-offset-2";

  if (!product.inStock || !headline) {
    return (
      <button type="button" disabled className={`${base} cursor-not-allowed bg-surface-container-high text-outline`} aria-label="Out of stock" title="Out of stock">
        <ShoppingCart className="h-4 w-4" />
      </button>
    );
  }

  if (product.hasChoices) {
    return (
      <Link
        to={`/products/${product.id}`}
        className={`${base} bg-primary text-white hover:scale-105 hover:bg-primary-hover active:scale-95`}
        aria-label={`Choose options for ${product.name}`}
        title="Choose options"
      >
        <ShoppingCart className="h-4 w-4" />
      </Link>
    );
  }

  return (
    <button
      type="button"
      onClick={() => addItem(product, headline)}
      className={`${base} bg-primary text-white hover:scale-105 hover:bg-primary-hover active:scale-95`}
      aria-label={`Add ${product.name} to cart`}
      title="Add to cart"
    >
      <ShoppingCart className="h-4 w-4" />
    </button>
  );
}
