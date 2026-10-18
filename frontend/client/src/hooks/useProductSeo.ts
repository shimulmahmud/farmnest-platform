import { SITE_NAME } from '@/config/seo';
import { getPricing } from '@/lib/pricing';
import { applyPageMeta, setJsonLd } from '@/lib/seo';
import type { Product } from '@/types';
import { useEffect } from 'react';

const trim = (text: string, max = 155) => (text.length > max ? `${text.slice(0, max - 1).trimEnd()}…` : text);

/** Title, description, share image and schema.org Product data for a product page. */
export function useProductSeo(product: Product | undefined) {
  useEffect(() => {
    if (!product) {
      applyPageMeta({ title: 'Product not found', noindex: true });
      return;
    }

    const { price } = getPricing(product);
    applyPageMeta({
      title: `${product.name} - ${product.category}`,
      description: trim(`${product.description} Buy online from ${SITE_NAME} for ৳${price}.`),
      image: product.image,
      type: 'product',
    });

    const cleanup = setJsonLd('ld-product', {
      '@context': 'https://schema.org',
      '@type': 'Product',
      name: product.name,
      image: product.images.length > 0 ? product.images : [product.image],
      description: product.description,
      sku: product.variants[0]?.sku ?? product.id,
      category: product.category,
      brand: { '@type': 'Brand', name: product.brand },
      ...(product.reviews > 0 ? { aggregateRating: { '@type': 'AggregateRating', ratingValue: product.rating, reviewCount: product.reviews } } : {}),
      offers: {
        '@type': 'Offer',
        url: `${window.location.origin}${window.location.pathname}`,
        priceCurrency: 'BDT',
        price,
        itemCondition: 'https://schema.org/NewCondition',
        availability: product.inStock ? 'https://schema.org/InStock' : 'https://schema.org/OutOfStock',
      },
    });
    return cleanup;
  }, [product]);
}
