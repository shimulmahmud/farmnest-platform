import { PROMO_COUNT, lightToneAt } from "@/config/home";
import { useCatalog } from "@/context/CatalogContext";
import { formatPrice } from "@/utils/format";
import { Link } from "react-router-dom";

/** One promo per category (the first few that have products). */
export default function PromoCards() {
  const { products, categories } = useCatalog();

  const cards = categories
    .map((category) => ({ category, items: products.filter((p) => p.categorySlug === category.slug) }))
    .filter((c) => c.items.length > 0)
    .slice(0, PROMO_COUNT);

  if (cards.length === 0) return null;

  return (
    <section className="max-w-page mx-auto px-4 py-4 sm:px-6 lg:px-8 lg:py-6" aria-label="Featured collections">
      <div className="grid gap-4 md:grid-cols-3">
        {cards.map(({ category, items }, index) => {
          const fromPrice = Math.min(...items.map((p) => p.price));
          return (
            <Link
              key={category.slug}
              to={`/products?category=${category.slug}`}
              className={`group relative flex min-h-[170px] overflow-hidden rounded-xl ${lightToneAt(index)}`}
            >
              <img
                src={category.image || items[0].image}
                alt=""
                loading="lazy"
                className="fade-left absolute inset-y-0 right-0 h-full w-1/2 object-cover transition-transform duration-500 group-hover:scale-105"
              />
              <div className="relative z-10 flex flex-col justify-center p-6">
                <span className="text-xs font-semibold text-on-surface-variant">Featured</span>
                <h3 className="mt-1 font-headline text-xl font-bold text-on-surface">{category.label}</h3>
                <p className="mt-1 text-sm text-on-surface-variant">From {formatPrice(fromPrice)}</p>
                <span className="mt-4 text-sm font-semibold text-primary underline underline-offset-4">
                  Shop {category.label.toLowerCase()}
                </span>
              </div>
            </Link>
          );
        })}
      </div>
    </section>
  );
}
