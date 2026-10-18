import { categoryIcons } from "@/config/categoryIcons";
import { toneAt } from "@/config/home";
import { useCatalog } from "@/context/CatalogContext";
import { Sprout } from "lucide-react";
import { Link } from "react-router-dom";
import SectionHeader from "@/components/SectionHeader";

export default function PopularCategories() {
  const { products, categories } = useCatalog();
  if (categories.length === 0) return null;

  return (
    <section id="categories" className="max-w-page mx-auto scroll-mt-20 px-4 py-8 sm:px-6 lg:px-8 lg:py-10" aria-labelledby="popular-categories">
      <SectionHeader id="popular-categories" title="Shop by Category" to="/products" linkLabel="View All Products" />

      <div className="grid grid-cols-3 gap-x-3 gap-y-7 sm:grid-cols-4 lg:grid-cols-6">
        {categories.map((cat, index) => {
          const Icon = categoryIcons[cat.slug] ?? Sprout;
          const image = cat.image || products.find((p) => p.categorySlug === cat.slug)?.image;

          return (
            <Link key={cat.slug} to={`/products?category=${cat.slug}`} className="group flex flex-col items-center gap-3 text-center">
              <span
                className={`flex h-24 w-24 items-center justify-center rounded-full p-2 transition-transform duration-300 group-hover:scale-105 sm:h-28 sm:w-28 lg:h-32 lg:w-32 ${
                  toneAt(index)
                }`}
              >
                {image ? (
                  <img src={image} alt="" loading="lazy" className="h-full w-full rounded-full object-cover" />
                ) : (
                  <Icon className="h-9 w-9 text-primary" />
                )}
              </span>
              <span>
                <span className="block text-sm font-semibold text-on-surface transition-colors group-hover:text-primary">
                  {cat.label}
                </span>
                <span className="block text-xs text-on-surface-variant">{cat.count} items</span>
              </span>
            </Link>
          );
        })}
      </div>
    </section>
  );
}
