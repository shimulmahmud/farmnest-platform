import ProductCard from "@/components/ProductCard";
import { FEATURED_LIMIT } from "@/config/home";
import { useCatalog } from "@/context/CatalogContext";
import SectionHeader from "@/components/SectionHeader";

export default function FeaturedProducts() {
  const { products } = useCatalog();
  const featured = [...products].reverse().slice(0, FEATURED_LIMIT); // newest first
  if (featured.length === 0) return null;

  return (
    <section className="max-w-page mx-auto px-4 py-8 sm:px-6 lg:px-8 lg:py-10" aria-labelledby="featured-products">
      <SectionHeader id="featured-products" title="New Arrivals" to="/products" linkLabel="View All Products" />

      <div className="grid grid-cols-2 gap-3 sm:gap-4 sm:grid-cols-4">
        {featured.map((product) => (
          <ProductCard key={product.id} product={product} variant="compact" />
        ))}
      </div>
    </section>
  );
}
