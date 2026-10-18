import ProductCard from '@/components/ProductCard';
import SectionHeader from '@/components/SectionHeader';
import { useCatalog } from '@/context/CatalogContext';
import { useCheckoutConfig } from '@/hooks/useCheckoutConfig';
import { getDeals, getPricing } from '@/lib/pricing';
import { formatPrice } from '@/utils/format';
import { ArrowRight, Tag, Truck } from 'lucide-react';
import { Link } from 'react-router-dom';

/** Deals = products the shop owner has reduced (old price struck through in the admin panel). */
export default function OffersPage() {
  const { products, loading } = useCatalog();
  const { config } = useCheckoutConfig();
  const deals = getDeals(products);
  const best = deals[0] ? getPricing(deals[0]).discountPercent : 0;

  return (
    <div className="bg-white animate-fade-in">
      <section className="max-w-page mx-auto px-4 pt-5 sm:px-6 lg:px-8">
        <div className="relative overflow-hidden rounded-2xl bg-gradient-to-r from-primary-dark via-primary-container to-primary px-6 py-12 sm:px-10 lg:px-14 lg:py-16">
          <span className="inline-flex items-center gap-1.5 rounded bg-accent px-2.5 py-1 text-xs font-bold text-on-secondary-fixed">
            <Tag className="h-3.5 w-3.5" /> Deals
          </span>
          <h1 className="mt-5 font-headline text-4xl font-semibold leading-[1.05] tracking-tight text-white sm:text-5xl">
            Today&apos;s best prices
          </h1>
          <p className="mt-4 max-w-lg text-sm leading-relaxed text-white/85 sm:text-base">
            {best > 0
              ? `Save up to ${best}% on selected products. Prices shown are exactly what you pay — no codes needed.`
              : 'Reduced prices appear here as soon as they are available. Check back soon.'}
          </p>
          <Link
            to="/products"
            className="mt-7 inline-flex items-center gap-2 rounded-full bg-accent px-6 py-3 text-sm font-bold text-on-secondary-fixed transition-colors hover:bg-accent-hover"
          >
            Browse all products <ArrowRight className="h-4 w-4" />
          </Link>
        </div>
      </section>

      <section className="max-w-page mx-auto px-4 py-10 sm:px-6 lg:px-8 lg:py-12" aria-labelledby="deal-products">
        <SectionHeader id="deal-products" title="Products on sale" to="/products" linkLabel="View All Products" />
        {loading && deals.length === 0 ? (
          <p className="py-10 text-center text-sm text-gray-500" aria-busy="true">Loading deals…</p>
        ) : deals.length === 0 ? (
          <p className="py-10 text-center text-sm text-gray-500">No products are on sale right now.</p>
        ) : (
          <div className="grid grid-cols-2 gap-3 sm:gap-4 sm:grid-cols-3 lg:grid-cols-5">
            {deals.map((product) => (
              <ProductCard key={product.id} product={product} variant="compact" />
            ))}
          </div>
        )}
      </section>

      <section className="max-w-page mx-auto px-4 py-8 sm:px-6 lg:px-8">
        <div className="flex flex-col items-center justify-between gap-4 rounded-2xl bg-primary px-6 py-6 text-center sm:flex-row sm:text-left lg:px-8">
          <p className="flex items-center gap-3 font-semibold text-white">
            <Truck className="h-6 w-6 flex-shrink-0" />
            {config && config.free_shipping_threshold > 0
              ? `Free delivery on orders above ${formatPrice(config.free_shipping_threshold)}`
              : 'Delivery across Bangladesh'}
          </p>
          <Link to="/products" className="rounded-full bg-white px-5 py-2.5 text-sm font-bold text-primary transition-colors hover:bg-primary-fixed">
            Start Shopping
          </Link>
        </div>
      </section>
    </div>
  );
}
