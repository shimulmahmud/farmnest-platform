import { SITE_NAME } from "@/config/seo";
import { SHELF_COUNT, lightToneAt } from "@/config/home";
import { useCatalog } from "@/context/CatalogContext";
import { setJsonLd } from "@/lib/seo";
import { useEffect } from "react";
import {
  BestSellers,
  CategoryShelf,
  DeliveryBanner,
  FeaturedProducts,
  HeroSection,
  HotDeals,
  PopularCategories,
  PromoCards,
} from "@/pages/home/sections";

export default function HomePage() {
  const { products, categories, loading, error, reload } = useCatalog();
  const shelfRows = categories
    .filter((c) => products.some((p) => p.categorySlug === c.slug))
    .slice(0, SHELF_COUNT)
    .map((c, i) => ({ slug: c.slug, panel: (i % 2 === 0 ? "left" : "right") as "left" | "right", tone: lightToneAt(i + 3) }));
  useEffect(() => {
    const origin = window.location.origin;
    const removeOrg = setJsonLd('ld-organization', {
      '@context': 'https://schema.org',
      '@type': 'Organization',
      name: SITE_NAME,
      url: origin,
      logo: `${origin}/farmnest-logo.png`,
    });
    const removeSite = setJsonLd('ld-website', {
      '@context': 'https://schema.org',
      '@type': 'WebSite',
      name: SITE_NAME,
      url: origin,
      potentialAction: {
        '@type': 'SearchAction',
        target: `${origin}/products?q={search_term_string}`,
        'query-input': 'required name=search_term_string',
      },
    });
    return () => {
      removeOrg();
      removeSite();
    };
  }, []);

  return (
    <div className="bg-white animate-fade-in">
      <HeroSection />
      {!loading && error && products.length === 0 && (
        <div className="max-w-page mx-auto px-4 pt-6 sm:px-6 lg:px-8">
          <div className="flex flex-wrap items-center justify-between gap-3 rounded-xl border border-red-200 bg-red-50 p-4 text-sm text-red-700">
            <span>We could not load the products: {error}</span>
            <button onClick={reload} className="rounded-lg bg-red-600 px-4 py-1.5 font-semibold text-white">Try again</button>
          </div>
        </div>
      )}
      {loading && products.length === 0 && <div className="py-16 text-center text-sm text-gray-500" aria-busy="true">Loading products…</div>}
      <PopularCategories />
      <BestSellers />
      <PromoCards />
      <HotDeals />
      <FeaturedProducts />
      <DeliveryBanner />
      {shelfRows.map((row) => (
        <CategoryShelf key={row.slug} row={row} />
      ))}
    </div>
  );
}
