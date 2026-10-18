import { Link, useParams } from 'react-router-dom';
import { useEffect, useMemo, useState } from 'react';
import { ArrowLeft, Check, ShoppingCart, Star, Truck, X } from 'lucide-react';
import { formatPrice } from '@/utils/format';
import { headlineVariant, useCart } from '@/context/CartContext';
import { useCatalog } from '@/context/CatalogContext';
import { QuantitySelector } from '@/components/ui';
import ProductGallery from '@/components/ProductGallery';
import ProductReviews from '@/components/ProductReviews';
import NotFound from '@/pages/NotFoundPage';
import { getPricing } from '@/lib/pricing';
import { useProductSeo } from '@/hooks/useProductSeo';

export default function ProductDetailPage() {
  const { id } = useParams();
  const { products, loading, error, reload } = useCatalog();
  const product = products.find((p) => p.id === id);
  const { addItem, lines } = useCart();
  const [qty, setQty] = useState(1);
  const [variantId, setVariantId] = useState<string | null>(null);
  const [justAdded, setJustAdded] = useState(false);
  useProductSeo(product);

  // Preselect the headline variant (cheapest in stock) when the product loads or changes.
  useEffect(() => {
    if (product) {
      setVariantId(headlineVariant(product)?.id ?? product.variants[0]?.id ?? null);
      setQty(1);
    }
  }, [product?.id]); // eslint-disable-line react-hooks/exhaustive-deps

  const variant = useMemo(
    () => product?.variants.find((v) => v.id === variantId) ?? product?.variants[0],
    [product, variantId],
  );

  if (loading && !product) {
    return <div className="min-h-[60vh] flex items-center justify-center text-gray-500" aria-busy="true">Loading product…</div>;
  }
  if (error && !product) {
    return (
      <div className="min-h-[60vh] flex flex-col items-center justify-center gap-3 text-center px-4">
        <p className="text-gray-700">{error}</p>
        <button onClick={reload} className="rounded-lg bg-primary px-5 py-2 font-semibold text-white">Try again</button>
      </div>
    );
  }
  if (!product || !variant) return <NotFound />;

  const galleryImages = [variant.image, ...product.images].filter((x, i, a): x is string => Boolean(x) && a.indexOf(x) === i);
  const related = products.filter((p) => p.categorySlug === product.categorySlug && p.id !== product.id).slice(0, 4);
  const inCartQty = lines.find((l) => l.variant.id === variant.id)?.quantity ?? 0;
  const discountPercent = variant.originalPrice ? Math.round((1 - variant.price / variant.originalPrice) * 100) : 0;
  const specEntries = Object.entries(product.specs).filter(([k]) => k !== 'certifications');

  const handleAdd = () => {
    addItem(product, variant, qty);
    setJustAdded(true);
    window.setTimeout(() => setJustAdded(false), 1500);
  };

  return (
    <div className="bg-white pb-20">
      <div className="max-w-6xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        <Link to="/products" className="inline-flex items-center gap-2 text-sm font-semibold text-gray-600 hover:text-primary mb-6">
          <ArrowLeft className="w-4 h-4" /> Back to Products
        </Link>

        <div className="grid md:grid-cols-2 gap-10">
          <ProductGallery images={galleryImages.length ? galleryImages : [product.image]} alt={product.name} />

          <div className="flex flex-col">
            <span className="text-xs font-bold uppercase tracking-wide text-primary">{product.category}</span>
            <h1 className="font-headline text-3xl font-bold mt-2">{product.name}</h1>
            {product.brand && <p className="text-sm text-gray-500 mt-1">by {product.brand}</p>}
            {product.reviews > 0 && (
              <div className="flex items-center gap-2 mt-3">
                <Star className="w-4 h-4 fill-amber-400 text-amber-400" />
                <strong className="text-sm">{product.rating}</strong>
                <a href="#reviews" className="text-sm text-gray-400 hover:text-primary">({product.reviews} reviews)</a>
              </div>
            )}
            <p className="text-gray-600 leading-relaxed mt-5 whitespace-pre-line">{product.description}</p>

            {product.certifications.length > 0 && (
              <div className="flex flex-wrap gap-2 mt-4">
                {product.certifications.map((cert) => (
                  <span key={cert} className="text-xs font-semibold bg-green-50 text-green-700 px-3 py-1 rounded-full border border-green-200">{cert}</span>
                ))}
              </div>
            )}

            {product.variants.length > 1 && (
              <div className="mt-6">
                <p className="text-sm font-semibold text-gray-700 mb-2">Choose an option</p>
                <div className="flex flex-wrap gap-2">
                  {product.variants.map((v) => {
                    const selected = v.id === variant.id;
                    return (
                      <button
                        key={v.id}
                        type="button"
                        onClick={() => { setVariantId(v.id); setQty(1); }}
                        disabled={!v.inStock}
                        className={`rounded-lg border px-4 py-2 text-sm font-medium transition-colors ${
                          selected ? 'border-primary bg-primary/5 text-primary' : 'border-gray-200 text-gray-700 hover:border-primary/50'
                        } ${!v.inStock ? 'cursor-not-allowed opacity-50 line-through' : ''}`}
                        aria-pressed={selected}
                      >
                        {v.label}
                        <span className="ml-2 text-xs text-gray-500">{formatPrice(v.price)}</span>
                      </button>
                    );
                  })}
                </div>
              </div>
            )}

            <div className="flex items-end gap-3 mt-6">
              <span className="font-headline text-4xl font-bold text-primary">{formatPrice(variant.price)}</span>
              {variant.originalPrice && <span className="text-xl text-gray-400 line-through mb-0.5">{formatPrice(variant.originalPrice)}</span>}
              <span className="text-gray-500 mb-1">/ {product.unit}</span>
              {discountPercent > 0 && <span className="mb-1.5 rounded bg-error px-2 py-1 text-xs font-bold text-white">-{discountPercent}%</span>}
            </div>

            <div className={`flex items-center gap-2 mt-4 text-sm ${variant.inStock ? 'text-green-700' : 'text-error'}`}>
              {variant.inStock ? <Check className="w-4 h-4" /> : <X className="w-4 h-4" />}
              {variant.inStock ? 'In stock — ready to ship' : 'Out of stock'}
            </div>

            <div className="flex items-center gap-4 mt-8">
              <QuantitySelector quantity={qty} onDecrease={() => setQty((q) => Math.max(1, q - 1))} onIncrease={() => setQty((q) => q + 1)} />
              <button
                onClick={handleAdd}
                disabled={!variant.inStock}
                className="flex-1 bg-primary text-white font-semibold py-3 rounded-lg hover:bg-primary-hover transition-colors flex items-center justify-center gap-2 disabled:cursor-not-allowed disabled:bg-gray-300"
              >
                {justAdded ? <Check className="w-5 h-5" /> : <ShoppingCart className="w-5 h-5" />}
                {!variant.inStock ? 'Out of stock' : justAdded ? 'Added' : 'Add to Cart'}
              </button>
            </div>
            {inCartQty > 0 && <p className="mt-2 text-xs text-gray-500">{inCartQty} of this option already in your cart.</p>}

            {specEntries.length > 0 && (
              <dl className="mt-8 divide-y divide-gray-100 rounded-lg border border-gray-100 text-sm">
                {specEntries.map(([k, v]) => (
                  <div key={k} className="flex justify-between gap-4 px-4 py-2.5">
                    <dt className="text-gray-500 capitalize">{k.replace(/_/g, ' ')}</dt>
                    <dd className="font-medium text-gray-800 text-right">{v}</dd>
                  </div>
                ))}
              </dl>
            )}

            <div className="mt-6 flex items-start gap-3 bg-gray-50 rounded-lg p-4 text-sm text-gray-600">
              <Truck className="w-5 h-5 text-primary flex-shrink-0" />
              <span>Cash on delivery or mobile wallet. Delivery charge is shown at checkout.</span>
            </div>
          </div>
        </div>

        <ProductReviews product={product} />

        {related.length > 0 && (
          <section className="mt-16">
            <h2 className="font-headline text-2xl font-bold mb-6">Related Products</h2>
            <div className="grid sm:grid-cols-2 lg:grid-cols-4 gap-5">
              {related.map((p) => (
                <Link key={p.id} to={`/products/${p.id}`} className="group rounded-xl border border-gray-100 overflow-hidden hover:shadow-md transition-shadow bg-white">
                  <div className="aspect-square overflow-hidden bg-gray-50"><img src={p.image} alt={p.name} className="w-full h-full object-cover group-hover:scale-105 transition-transform duration-300" /></div>
                  <div className="p-3"><h3 className="font-headline text-sm font-semibold line-clamp-2 group-hover:text-primary transition-colors">{p.name}</h3><strong className="block text-primary mt-2">{p.hasChoices ? 'From ' : ''}{formatPrice(getPricing(p).price)}</strong></div>
                </Link>
              ))}
            </div>
          </section>
        )}
      </div>
    </div>
  );
}
