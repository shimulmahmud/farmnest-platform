import { useState, useMemo, useEffect } from 'react';
import { useSearchParams } from 'react-router-dom';
import { ChevronLeft, ChevronRight, X, SlidersHorizontal, Check } from 'lucide-react';
import { useCatalog } from '@/context/CatalogContext';
import { sortOptions, PRODUCTS_PER_PAGE } from '@/config/filters';
import ProductCard from '@/components/ProductCard';
import { getPricing } from '@/lib/pricing';
import { formatPrice } from '@/utils/format';

export default function BrowseProductsPage() {
  const { products, categories, loading, error, reload } = useCatalog();
  const [searchParams, setSearchParams] = useSearchParams();

  // Filter choices come from what the catalog actually contains.
  const brands = useMemo(() => [...new Set(products.map((p) => p.brand).filter(Boolean))].sort(), [products]);
  const certifications = useMemo(() => [...new Set(products.flatMap((p) => p.certifications))].sort(), [products]);
  const MAX_PRICE = useMemo(() => Math.max(1000, ...products.map((p) => p.price)), [products]);
  const initialCategory = searchParams.get('category') || '';
  const initialQuery = searchParams.get('q') || '';

  const [searchQuery, setSearchQuery] = useState(initialQuery);
  const [selectedCategories, setSelectedCategories] = useState<string[]>(
    initialCategory ? [initialCategory] : []
  );
  const [selectedBrands, setSelectedBrands] = useState<string[]>([]);
  const [selectedCerts, setSelectedCerts] = useState<string[]>([]);
  const [maxPriceChoice, setMaxPriceChoice] = useState<number | null>(null);
  const maxPrice = maxPriceChoice ?? MAX_PRICE;
  const setMaxPrice = (v: number) => setMaxPriceChoice(v);
  const [sortBy, setSortBy] = useState(0);
  const [currentPage, setCurrentPage] = useState(1);
  const [mobileFiltersOpen, setMobileFiltersOpen] = useState(false);

  const productsPerPage = PRODUCTS_PER_PAGE;

  const toggleCategory = (slug: string) => {
    setSelectedCategories((prev) =>
      prev.includes(slug) ? prev.filter((c) => c !== slug) : [...prev, slug]
    );
    setCurrentPage(1);
  };

  const toggleBrand = (brand: string) => {
    setSelectedBrands((prev) =>
      prev.includes(brand) ? prev.filter((b) => b !== brand) : [...prev, brand]
    );
    setCurrentPage(1);
  };

  const toggleCert = (cert: string) => {
    setSelectedCerts((prev) =>
      prev.includes(cert) ? prev.filter((c) => c !== cert) : [...prev, cert]
    );
    setCurrentPage(1);
  };

  const clearAll = () => {
    setSelectedCategories([]);
    setSelectedBrands([]);
    setSelectedCerts([]);
    setMaxPriceChoice(null);
    setSearchQuery('');
    setSearchParams({});
    setCurrentPage(1);
  };

  const filteredProducts = useMemo(() => {
    let result = [...products];

    if (searchQuery.trim()) {
      const q = searchQuery.toLowerCase();
      result = result.filter((p) =>
        p.name.toLowerCase().includes(q) ||
        p.description.toLowerCase().includes(q) ||
        p.category.toLowerCase().includes(q)
      );
    }
    if (selectedCategories.length > 0) {
      result = result.filter((p) => selectedCategories.includes(p.categorySlug));
    }
    if (selectedBrands.length > 0) {
      result = result.filter((p) => selectedBrands.includes(p.brand));
    }
    if (selectedCerts.length > 0) {
      result = result.filter((p) =>
        selectedCerts.every((cert) => p.certifications.includes(cert))
      );
    }
    result = result.filter((p) => getPricing(p).price <= maxPrice);

    switch (sortBy) {
      case 1: result.sort((a, b) => getPricing(a).price - getPricing(b).price); break;
      case 2: result.sort((a, b) => getPricing(b).price - getPricing(a).price); break;
      case 3: result.reverse(); break;
    }

    return result;
  }, [products, searchQuery, selectedCategories, selectedBrands, selectedCerts, maxPrice, sortBy]);

  const totalPages = Math.ceil(filteredProducts.length / productsPerPage);
  const currentProducts = filteredProducts.slice(
    (currentPage - 1) * productsPerPage,
    currentPage * productsPerPage
  );

  const activeFilters = [
    ...selectedCategories.map((slug) => ({
      label: categories.find((c) => c.slug === slug)?.label || slug,
      onRemove: () => toggleCategory(slug),
    })),
    ...selectedBrands.map((brand) => ({
      label: brand,
      onRemove: () => toggleBrand(brand),
    })),
    ...selectedCerts.map((cert) => ({
      label: cert,
      onRemove: () => toggleCert(cert),
    })),
  ];

  const FilterSidebar = () => (
    <aside className="w-full md:w-64 flex-shrink-0 flex flex-col gap-3">
      {/* Categories */}
      <div className="bg-white rounded-xl p-6 shadow-sm border border-gray-100">
        <h3 className="font-headline text-lg font-bold text-on-surface mb-4 pb-3 border-b border-gray-100">
          Categories
        </h3>
        <ul className="flex flex-col gap-3">
          {categories.map((cat) => (
            <li key={cat.slug}>
              <label className="flex items-center gap-3 cursor-pointer group">
                <input
                  type="checkbox"
                  checked={selectedCategories.includes(cat.slug)}
                  onChange={() => toggleCategory(cat.slug)}
                  className="w-5 h-5 rounded border-gray-300 text-primary focus:ring-primary cursor-pointer"
                />
                <span
                  className={`text-sm font-medium transition-colors ${
                    selectedCategories.includes(cat.slug)
                      ? 'text-primary'
                      : 'text-gray-600 group-hover:text-primary'
                  }`}
                >
                  {cat.label}
                </span>
              </label>
            </li>
          ))}
        </ul>
      </div>

      {/* Price Filter */}
      <div className="bg-white rounded-xl p-6 shadow-sm border border-gray-100">
        <h3 className="font-headline text-lg font-bold text-on-surface mb-4 pb-3 border-b border-gray-100">
          Price
        </h3>
        <div className="flex flex-col gap-3">
          <input
            type="range"
            min="0"
            max={MAX_PRICE}
            step="500"
            value={maxPrice}
            onChange={(e) => setMaxPrice(Number(e.target.value))}
            className="w-full accent-primary"
          />
          <div className="flex justify-between text-sm text-gray-500 font-medium">
            <span>{formatPrice(0)}</span>
            <span>{formatPrice(maxPrice)}</span>
          </div>
        </div>
      </div>

      {/* Brand Filter */}
      {brands.length > 0 && (
      <div className="bg-white rounded-xl p-6 shadow-sm border border-gray-100">
        <h3 className="font-headline text-lg font-bold text-on-surface mb-4 pb-3 border-b border-gray-100">
          Brand
        </h3>
        <ul className="flex flex-col gap-3">
          {brands.map((brand) => (
            <li key={brand}>
              <label className="flex items-center gap-3 cursor-pointer group">
                <input
                  type="checkbox"
                  checked={selectedBrands.includes(brand)}
                  onChange={() => toggleBrand(brand)}
                  className="w-5 h-5 rounded border-gray-300 text-primary focus:ring-primary cursor-pointer"
                />
                <span className="text-sm text-gray-600 group-hover:text-primary transition-colors">
                  {brand}
                </span>
              </label>
            </li>
          ))}
        </ul>
      </div>
      )}

      {/* Certifications Filter */}
      {certifications.length > 0 && (
      <div className="bg-white rounded-xl p-6 shadow-sm border border-gray-100">
        <h3 className="font-headline text-lg font-bold text-on-surface mb-4 pb-3 border-b border-gray-100">
          Certifications
        </h3>
        <ul className="flex flex-col gap-3">
          {certifications.map((cert) => (
            <li key={cert}>
              <label className="flex items-center gap-3 cursor-pointer group">
                <input
                  type="checkbox"
                  checked={selectedCerts.includes(cert)}
                  onChange={() => toggleCert(cert)}
                  className="w-5 h-5 rounded border-gray-300 text-primary focus:ring-primary cursor-pointer"
                />
                <span
                  className={`text-sm font-medium transition-colors ${
                    selectedCerts.includes(cert)
                      ? 'text-primary'
                      : 'text-gray-600 group-hover:text-primary'
                  }`}
                >
                  {cert}
                </span>
              </label>
            </li>
          ))}
        </ul>
      </div>
      )}
    </aside>
  );

  if (loading && products.length === 0) {
    return <div className="min-h-[60vh] flex items-center justify-center text-gray-500" aria-busy="true">Loading products…</div>;
  }
  if (error && products.length === 0) {
    return (
      <div className="min-h-[60vh] flex flex-col items-center justify-center gap-3 text-center px-4">
        <p className="text-gray-700">We could not load the products: {error}</p>
        <button onClick={reload} className="rounded-lg bg-primary px-5 py-2 font-semibold text-white">Try again</button>
      </div>
    );
  }

  return (
    <div className="bg-surface-container-low min-h-screen animate-fade-in">
      {/* Page header */}
      <div className="max-w-page mx-auto px-4 sm:px-6 lg:px-8 pt-10 pb-6">
        <h1 className="font-display text-3xl md:text-4xl font-bold text-primary tracking-tight mb-2">
          Browse Products
        </h1>
        <p className="text-gray-600 text-lg">
          Discover {products.length} products, delivered across Bangladesh.
        </p>
        {searchQuery && (
          <div className="mt-4 inline-flex items-center gap-2 bg-primary/5 text-primary px-4 py-2 rounded-full text-sm font-semibold">
            Results for “{searchQuery}”
            <button onClick={() => { setSearchQuery(''); setSearchParams({}); }} className="hover:bg-primary/10 rounded-full p-0.5"><X className="w-3.5 h-3.5" /></button>
          </div>
        )}
      </div>

      <div className="max-w-page mx-auto px-4 sm:px-6 lg:px-8 pb-16 flex flex-col md:flex-row gap-8">
        {/* Desktop sidebar */}
        <div className="hidden md:block">
          <FilterSidebar />
        </div>

        {/* Mobile filter drawer */}
        {mobileFiltersOpen && (
          <div className="fixed inset-0 z-50 md:hidden">
            <div className="absolute inset-0 bg-black/30" onClick={() => setMobileFiltersOpen(false)} />
            <div className="absolute left-0 top-0 bottom-0 w-80 max-w-[85%] bg-white overflow-y-auto p-4 animate-fade-in">
              <div className="flex items-center justify-between mb-4">
                <h2 className="font-headline text-lg font-bold">Filters</h2>
                <button onClick={() => setMobileFiltersOpen(false)} className="p-2">
                  <X className="w-5 h-5" />
                </button>
              </div>
              <FilterSidebar />
            </div>
          </div>
        )}

        {/* Product grid area */}
        <div className="flex-grow flex flex-col">
          {/* Active filters & sort */}
          <div className="flex flex-col md:flex-row justify-between items-start md:items-center mb-8 gap-3 pb-4 border-b border-gray-100">
            <div className="flex flex-wrap gap-2 items-center">
              <button
                className="md:hidden flex items-center gap-2 px-3 py-1.5 border border-gray-200 rounded-lg text-sm font-medium"
                onClick={() => setMobileFiltersOpen(true)}
              >
                <SlidersHorizontal className="w-4 h-4" /> Filters
              </button>
              {activeFilters.map((filter, i) => (
                <span
                  key={i}
                  className="inline-flex items-center gap-1 px-3 py-1.5 rounded-full bg-primary-fixed text-on-primary-fixed-variant text-sm font-medium shadow-sm"
                >
                  {filter.label}
                  <button
                    onClick={filter.onRemove}
                    className="hover:text-primary hover:bg-black/5 rounded-full p-0.5 transition-colors"
                  >
                    <X className="w-3 h-3" />
                  </button>
                </span>
              ))}
              {activeFilters.length > 0 && (
                <button
                  onClick={clearAll}
                  className="text-sm text-gray-500 font-semibold hover:text-primary transition-all ml-2"
                >
                  Clear All
                </button>
              )}
            </div>
            <div className="flex items-center gap-2">
              <span className="text-sm text-gray-500 font-medium">Sort by:</span>
              <select
                value={sortBy}
                onChange={(e) => setSortBy(Number(e.target.value))}
                className="bg-white border border-gray-200 rounded-lg text-sm text-on-surface py-2 px-3 pr-8 focus:border-primary focus:ring-1 focus:ring-primary cursor-pointer shadow-sm appearance-none relative"
              >
                {sortOptions.map((option, i) => (
                  <option key={i} value={i}>{option}</option>
                ))}
              </select>
            </div>
          </div>

          {/* Results count */}
          <div className="mb-4 text-sm text-gray-500">
            Showing {currentProducts.length} of {filteredProducts.length} products
          </div>

          {/* Grid */}
          {currentProducts.length > 0 ? (
            <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-6">
              {currentProducts.map((product) => (
                <ProductCard key={product.id} product={product} />
              ))}
            </div>
          ) : (
            <div className="flex flex-col items-center justify-center py-20 text-center">
              <div className="w-16 h-16 rounded-full bg-gray-100 flex items-center justify-center mb-4">
                <Check className="w-8 h-8 text-gray-400" />
              </div>
              <h3 className="font-headline text-lg font-semibold text-gray-700 mb-2">
                No products found
              </h3>
              <p className="text-gray-500 mb-4">Try adjusting your filters to see more results.</p>
              <button
                onClick={clearAll}
                className="px-6 py-2 bg-primary text-white rounded-lg text-sm font-medium hover:bg-primary-hover transition-colors"
              >
                Clear All Filters
              </button>
            </div>
          )}

          {/* Pagination */}
          {totalPages > 1 && (
            <div className="flex justify-center items-center mt-12 gap-2">
              <button
                onClick={() => setCurrentPage((p) => Math.max(1, p - 1))}
                disabled={currentPage === 1}
                className="w-10 h-10 rounded-lg flex items-center justify-center bg-white border border-gray-200 text-gray-600 hover:bg-gray-50 transition-colors disabled:opacity-50 disabled:cursor-not-allowed shadow-sm"
              >
                <ChevronLeft className="w-5 h-5" />
              </button>
              {Array.from({ length: totalPages }, (_, i) => i + 1).map((page) => (
                <button
                  key={page}
                  onClick={() => setCurrentPage(page)}
                  className={`w-10 h-10 rounded-lg flex items-center justify-center text-sm font-medium transition-colors ${
                    currentPage === page
                      ? 'bg-primary text-white shadow-sm'
                      : 'bg-white border border-gray-200 text-gray-600 hover:border-primary hover:text-primary shadow-sm'
                  }`}
                >
                  {page}
                </button>
              ))}
              <button
                onClick={() => setCurrentPage((p) => Math.min(totalPages, p + 1))}
                disabled={currentPage === totalPages}
                className="w-10 h-10 rounded-lg flex items-center justify-center bg-white border border-gray-200 text-gray-600 hover:bg-gray-50 transition-colors disabled:opacity-50 disabled:cursor-not-allowed shadow-sm"
              >
                <ChevronRight className="w-5 h-5" />
              </button>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
