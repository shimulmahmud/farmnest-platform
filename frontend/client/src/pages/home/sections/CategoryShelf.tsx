import ProductCard from "@/components/ProductCard";
import { SHELF_MAX_PRODUCTS } from "@/config/home";
import { useCatalog } from "@/context/CatalogContext";
import type { CSSProperties } from "react";
import { Link } from "react-router-dom";

export interface ShelfRowConfig {
  slug: string;
  panel: "left" | "right";
  tone: string;
}

export default function CategoryShelf({ row }: { row: ShelfRowConfig }) {
  const { products, categories } = useCatalog();
  const category = categories.find((c) => c.slug === row.slug);
  const items = products.filter((p) => p.categorySlug === row.slug).slice(0, SHELF_MAX_PRODUCTS);
  if (!category || items.length === 0) return null;

  const panelFirst = row.panel === "left";
  const cells = `repeat(${items.length}, minmax(0, 1fr))`;
  // Fewer products than a full shelf -> the panel absorbs the slack so cards keep a sensible size.
  const panelWeight = 1.2 + (SHELF_MAX_PRODUCTS - items.length) * 0.5;
  const panel = `minmax(0, ${panelWeight}fr)`;
  const template = panelFirst ? `${panel} ${cells}` : `${cells} ${panel}`;
  const panelImage = items[0].images[1] ?? items[0].image;

  return (
    <section className="max-w-page mx-auto px-4 py-4 sm:px-6 lg:px-8 lg:py-5" aria-label={category.label}>
      <div
        className="grid grid-cols-2 gap-3 sm:gap-4 lg:[grid-template-columns:var(--shelf-cols)]"
        style={{ "--shelf-cols": template } as CSSProperties}
      >
        <div className={`col-span-2 flex flex-col rounded-xl p-5 sm:p-6 lg:col-span-1 ${row.tone} ${panelFirst ? "" : "lg:order-last"}`}>
          <h3 className="font-headline text-xl font-bold text-on-surface">{category.label}</h3>
          <p className="mt-1.5 line-clamp-3 text-sm leading-relaxed text-on-surface-variant">{category.description}</p>
          <Link
            to={`/products?category=${category.slug}`}
            className="mt-4 w-fit rounded-full bg-primary px-4 py-1.5 text-xs font-bold text-white transition-colors hover:bg-primary-hover"
          >
            View all
          </Link>
          <img
            src={panelImage}
            alt=""
            loading="lazy"
            className="mt-5 h-36 w-full rounded-lg object-cover lg:mt-auto lg:h-44"
          />
        </div>

        {items.map((product) => (
          <ProductCard key={product.id} product={product} variant="compact" />
        ))}
      </div>
    </section>
  );
}
