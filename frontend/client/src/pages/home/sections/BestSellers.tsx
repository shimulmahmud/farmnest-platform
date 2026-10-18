import ProductCard from "@/components/ProductCard";
import { BEST_SELLER_LIMIT } from "@/config/home";
import { useCatalog } from "@/context/CatalogContext";
import { useMemo, useState } from "react";

export default function BestSellers() {
  const { products, categories } = useCatalog();
  const [tab, setTab] = useState("all");

  // In-stock products first; the catalog order (set in the admin panel) decides the rest.
  const ranked = useMemo(() => [...products].sort((a, b) => Number(b.inStock) - Number(a.inStock)), [products]);

  const visible = useMemo(
    () => (tab === "all" ? ranked : ranked.filter((p) => p.categorySlug === tab)).slice(0, BEST_SELLER_LIMIT),
    [tab, ranked]
  );

  if (products.length === 0) return null;

  const tabs = [{ slug: "all", label: "All" }, ...categories.map((c) => ({ slug: c.slug, label: c.label }))];

  return (
    <section className="max-w-page mx-auto px-4 py-8 sm:px-6 lg:px-8 lg:py-10" aria-labelledby="best-sellers">
      <div className="mb-5 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <h2 id="best-sellers" className="font-headline text-xl font-bold tracking-tight text-on-surface sm:text-2xl">
          Popular Products
        </h2>

        <div className="scrollbar-hide -mx-4 flex gap-1 overflow-x-auto px-4 sm:mx-0 sm:px-0" role="group" aria-label="Filter by category">
          {tabs.map((t) => (
            <button
              key={t.slug}
              onClick={() => setTab(t.slug)}
              aria-pressed={tab === t.slug}
              className={`shrink-0 rounded-full px-3.5 py-1.5 text-xs font-semibold transition-colors ${
                tab === t.slug
                  ? "bg-primary text-white"
                  : "text-on-surface-variant hover:bg-surface-container hover:text-primary"
              }`}
            >
              {t.label}
            </button>
          ))}
        </div>
      </div>

      <div className="grid grid-cols-2 gap-3 sm:gap-4 sm:grid-cols-3 lg:grid-cols-6">
        {visible.map((product) => (
          <ProductCard key={product.id} product={product} variant="compact" />
        ))}
      </div>
    </section>
  );
}
