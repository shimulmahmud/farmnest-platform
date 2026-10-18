import { getPricing } from "@/lib/pricing";
import type { Product } from "@/types";
import { formatPrice } from "@/utils/format";

interface PriceProps {
  product: Product;
  /** Classes for the current (sale) price. */
  className?: string;
  /** Classes for the struck-through list price. */
  originalClassName?: string;
}

/** Current price, with the list price struck through while a discount applies. */
export default function Price({
  product,
  className = "font-headline text-base font-bold text-primary",
  originalClassName = "text-xs text-outline line-through",
}: PriceProps) {
  const { price, originalPrice } = getPricing(product);

  return (
    <>
      <span className={className}>{formatPrice(price)}</span>
      {originalPrice !== null && (
        <>
          <span className={originalClassName} aria-hidden="true">{formatPrice(originalPrice)}</span>
          <span className="sr-only">, was {formatPrice(originalPrice)}</span>
        </>
      )}
    </>
  );
}
