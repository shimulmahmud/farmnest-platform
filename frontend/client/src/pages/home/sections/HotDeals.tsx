import Price from "@/components/Price";
import SectionHeader from "@/components/SectionHeader";
import WishlistButton from "@/components/WishlistButton";
import { useCart } from "@/context/CartContext";
import { useCatalog } from "@/context/CatalogContext";
import { getDeals, getPricing } from "@/lib/pricing";
import type { Product } from "@/types";
import { Check, ChevronLeft, ChevronRight, ShoppingCart } from "lucide-react";
import { useCallback, useEffect, useRef, useState } from "react";
import { Link, useNavigate } from "react-router-dom";

function DealCard({ product }: { product: Product }) {
  const { addItem } = useCart();
  const navigate = useNavigate();
  const [added, setAdded] = useState(false);
  const { discountPercent } = getPricing(product);

  const handleAdd = () => {
    if (product.hasChoices) {
      navigate(`/products/${product.id}`);
      return;
    }
    addItem(product);
    setAdded(true);
    setTimeout(() => setAdded(false), 1500);
  };

  return (
    <article className="flex min-w-full snap-start gap-4 rounded-xl bg-white p-4 shadow-soft sm:gap-5 sm:p-5 lg:min-w-[calc(50%-0.625rem)]">
      <div className="relative h-32 w-32 shrink-0 sm:h-44 sm:w-44">
        <Link
          to={`/products/${product.id}`}
          className="relative block h-full w-full overflow-hidden rounded-lg bg-surface-container-low"
        >
          <img src={product.image} alt={product.name} loading="lazy" className="h-full w-full object-cover" />
          {discountPercent > 0 && (
            <span className="absolute left-2 top-2 rounded-full bg-error px-2 py-1 text-[11px] font-bold leading-none text-white">
              -{discountPercent}%
            </span>
          )}
        </Link>
        <WishlistButton product={product} className="absolute right-2 top-2" />
      </div>

      <div className="flex min-w-0 flex-1 flex-col">
        <Link to={`/products/${product.id}`}>
          <h3 className="line-clamp-2 font-headline text-sm font-bold leading-snug text-on-surface hover:text-primary sm:text-base">
            {product.name}
          </h3>
        </Link>

        <p className="mt-1.5 flex items-baseline gap-2">
          <Price product={product} className="font-headline text-lg font-bold text-primary" />
          <span className="text-[11px] text-outline">{product.unit}</span>
        </p>



        <div className="mt-auto pt-4">
          <button
            type="button"
            onClick={handleAdd}
            disabled={!product.inStock}
            className={`inline-flex w-full items-center justify-center gap-2 rounded-lg px-4 py-2.5 text-sm font-semibold text-white transition-colors sm:w-fit ${
              added ? "bg-success" : "bg-primary hover:bg-primary-hover"
            } disabled:cursor-not-allowed disabled:bg-surface-container-highest disabled:text-outline`}
          >
            {added ? <Check className="h-4 w-4" strokeWidth={2.5} /> : <ShoppingCart className="h-4 w-4" />}
            {added ? "Added" : product.hasChoices ? "Choose option" : "Add to Cart"}
          </button>
        </div>
      </div>
    </article>
  );
}

export default function HotDeals() {
  const { products } = useCatalog();
  // Products the shop owner has put on sale (old price set in the admin panel).
  const deals = getDeals(products).slice(0, 8);

  const railRef = useRef<HTMLDivElement>(null);
  const [atStart, setAtStart] = useState(true);
  const [atEnd, setAtEnd] = useState(false);

  const updateEdges = useCallback(() => {
    const el = railRef.current;
    if (!el) return;
    setAtStart(el.scrollLeft <= 4);
    setAtEnd(el.scrollLeft + el.clientWidth >= el.scrollWidth - 4);
  }, []);

  useEffect(() => {
    updateEdges();
    window.addEventListener("resize", updateEdges);
    return () => window.removeEventListener("resize", updateEdges);
  }, [updateEdges]);

  if (deals.length === 0) return null;

  const page = (direction: 1 | -1) => {
    const el = railRef.current;
    if (el) el.scrollBy({ left: direction * el.clientWidth, behavior: "smooth" });
  };

  const arrowClass =
    "absolute top-1/2 z-10 hidden h-9 w-9 -translate-y-1/2 items-center justify-center rounded-full bg-white text-on-surface shadow-level-2 sm:flex transition-all hover:bg-primary hover:text-white disabled:pointer-events-none disabled:opacity-0";

  return (
    <section className="my-6 bg-primary-fixed/50 py-10 lg:py-12" aria-labelledby="hot-deals">
      <div className="max-w-page mx-auto px-4 sm:px-6 lg:px-8">
        <SectionHeader id="hot-deals" title="Hot Deals" to="/offers" linkLabel="View All Deals" />
        <p className="-mt-3 mb-5 text-sm text-on-surface-variant">Reduced prices, shown exactly as you pay them.</p>

        <div className="relative">
          <div
            ref={railRef}
            onScroll={updateEdges}
            className="scrollbar-hide relative flex snap-x snap-mandatory gap-5 overflow-x-auto scroll-smooth"
          >
            {deals.map((product) => (
              <DealCard key={product.id} product={product} />
            ))}
          </div>

          <button onClick={() => page(-1)} disabled={atStart} className={`${arrowClass} left-1 sm:-left-3 lg:-left-4`} aria-label="Previous deals">
            <ChevronLeft className="h-4 w-4" />
          </button>
          <button onClick={() => page(1)} disabled={atEnd} className={`${arrowClass} right-1 sm:-right-3 lg:-right-4`} aria-label="Next deals">
            <ChevronRight className="h-4 w-4" />
          </button>
        </div>
      </div>
    </section>
  );
}
