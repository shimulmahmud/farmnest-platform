import { Link, useLocation, useNavigate } from 'react-router-dom';
import { useEffect, useRef, useState } from 'react';
import {
  Headset,
  Search, ShoppingCart, Menu, X, Sprout, User, LogOut, ChevronDown, Tag, Truck, Headphones, LayoutGrid, Heart, Zap,
} from 'lucide-react';
import { useCart } from '@/context/CartContext';
import { useAuth } from '@/context/AuthContext';
import { useWishlist } from '@/context/WishlistContext';
import { searchAll, type SearchResult } from '@/lib/search';
import { navLinks } from '@/config/navigation';
import { useCatalog } from '@/context/CatalogContext';
import { useCheckoutConfig } from '@/hooks/useCheckoutConfig';
import { unreadCount } from '@/api/support';
import { categoryIcons } from '@/config/categoryIcons';
import { contactInfo } from '@/config/footer';
import { formatPrice } from '@/utils/format';

export default function Navbar({ onCartClick }: { onCartClick: () => void }) {
  const location = useLocation();
  const navigate = useNavigate();
  const { itemCount } = useCart();
  const { user, signOut } = useAuth();
  const [supportUnread, setSupportUnread] = useState(0);

  // Red dot when the shop has replied to one of your messages.
  useEffect(() => {
    if (!user) { setSupportUnread(0); return; }
    let alive = true;
    const check = () => unreadCount().then((n) => alive && setSupportUnread(n)).catch(() => undefined);
    check();
    const timer = window.setInterval(check, 60_000);
    return () => { alive = false; window.clearInterval(timer); };
  }, [user]);
  const { products, categories } = useCatalog();
  const { config: checkoutConfig } = useCheckoutConfig();
  const { count: wishlistCount } = useWishlist();
  const [mobileOpen, setMobileOpen] = useState(false);
  const [query, setQuery] = useState('');
  const [searchCategory, setSearchCategory] = useState('');
  const [userMenuOpen, setUserMenuOpen] = useState(false);
  const [categoryMenuOpen, setCategoryMenuOpen] = useState(false);
  const [searchResults, setSearchResults] = useState<SearchResult[]>([]);
  const [searchFocused, setSearchFocused] = useState(false);
  const userMenuRef = useRef<HTMLDivElement>(null);
  const categoryMenuRef = useRef<HTMLDivElement>(null);
  const searchRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const onClick = (e: MouseEvent) => {
      const target = e.target as Node;
      if (userMenuRef.current && !userMenuRef.current.contains(target)) setUserMenuOpen(false);
      if (categoryMenuRef.current && !categoryMenuRef.current.contains(target)) setCategoryMenuOpen(false);
      if (searchRef.current && !searchRef.current.contains(target)) setSearchFocused(false);
    };
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        setUserMenuOpen(false);
        setCategoryMenuOpen(false);
        setSearchFocused(false);
      }
    };
    document.addEventListener('mousedown', onClick);
    document.addEventListener('keydown', onKey);
    return () => {
      document.removeEventListener('mousedown', onClick);
      document.removeEventListener('keydown', onKey);
    };
  }, []);

  useEffect(() => {
    setMobileOpen(false);
    setCategoryMenuOpen(false);
  }, [location.pathname, location.search]);

  const handleSignOut = async () => {
    await signOut();
    setUserMenuOpen(false);
    navigate('/');
  };

  const isActive = (path: string) =>
    path === '/' ? location.pathname === '/' : location.pathname.startsWith(path);

  const submitSearch = (e: React.FormEvent) => {
    e.preventDefault();
    const q = query.trim();
    if (!q && !searchCategory) return;
    const params = new URLSearchParams();
    if (q) params.set('q', q);
    if (searchCategory) params.set('category', searchCategory);
    navigate(`/products?${params.toString()}`);
    setQuery('');
    setSearchResults([]);
    setSearchFocused(false);
    setMobileOpen(false);
  };

  const handleSearchChange = (value: string) => {
    setQuery(value);
    setSearchResults(searchAll(value, products, categories));
  };

  const resultIcon = (type: SearchResult['type']) => {
    if (type === 'category') return <Tag className="w-4 h-4 text-gray-400" />;
    return null;
  };

  const cartButton = (
    <button
      onClick={onCartClick}
      className="relative p-2.5 text-on-surface hover:text-primary hover:bg-surface-container-low rounded-lg transition-colors"
      aria-label={`Open cart${itemCount > 0 ? `, ${itemCount} ${itemCount === 1 ? 'item' : 'items'}` : ''}`}
    >
      <ShoppingCart className="w-5 h-5" />
      {itemCount > 0 && (
        <span className="absolute top-0 right-0 min-w-4 h-4 px-1 bg-primary text-white text-[10px] font-semibold rounded-full flex items-center justify-center">
          {itemCount}
        </span>
      )}
    </button>
  );

  const wishlistLink = (
    <Link
      to="/wishlist"
      className="relative p-2.5 text-on-surface hover:text-primary hover:bg-surface-container-low rounded-lg transition-colors"
      aria-label={`Wishlist${wishlistCount > 0 ? `, ${wishlistCount} ${wishlistCount === 1 ? 'item' : 'items'}` : ''}`}
    >
      <Heart className="w-5 h-5" />
      {wishlistCount > 0 && (
        <span className="absolute top-0 right-0 min-w-4 h-4 px-1 bg-primary text-white text-[10px] font-semibold rounded-full flex items-center justify-center">
          {wishlistCount}
        </span>
      )}
    </Link>
  );

  return (
    <>
      {/* Announcement bar — delivery threshold and payment options */}
      <div className="bg-primary text-white text-xs py-2">
        <div className="max-w-page mx-auto px-4 sm:px-6 lg:px-8 flex items-center justify-center md:justify-between gap-4">
          <span className="hidden md:flex items-center gap-1.5 font-medium">
            <Truck className="w-3.5 h-3.5" /> {checkoutConfig && checkoutConfig.free_shipping_threshold > 0 ? <>Free delivery on orders above {formatPrice(checkoutConfig.free_shipping_threshold)}</> : <>Delivery across Bangladesh</>}
          </span>
          <Link to="/offers" className="font-medium hover:underline flex items-center gap-1.5">
            <Sprout className="w-3.5 h-3.5 md:hidden" />
            Cash on delivery &amp; bKash / Nagad / Rocket accepted
          </Link>
        </div>
      </div>

      {/* Brand + search row. Sticky on small screens (it is the only header there). */}
      <header className="sticky top-0 z-50 border-b border-outline-variant/60 bg-white lg:relative lg:z-[51]">
        <div className="max-w-page mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex items-center justify-between gap-4 h-16 lg:h-20">
            <Link to="/" className="flex items-center gap-2.5 group flex-shrink-0">
              <img src="/farmnest-logo.png" alt="" className="w-10 h-10 rounded-xl object-cover group-hover:scale-105 transition-transform" />
              <span className="font-headline text-2xl font-extrabold text-primary tracking-tight">FarmNest</span>
            </Link>

            {/* Search with category filter */}
            <div className="hidden lg:block relative flex-1 max-w-xl" ref={searchRef}>
              <form onSubmit={submitSearch} className="flex items-stretch rounded-lg border border-outline-variant bg-surface-container-low focus-within:border-primary focus-within:bg-white transition-colors">
                <label className="sr-only" htmlFor="search-category">Search category</label>
                <select
                  id="search-category"
                  value={searchCategory}
                  onChange={(e) => setSearchCategory(e.target.value)}
                  className="w-36 shrink-0 cursor-pointer rounded-l-lg border-r border-outline-variant bg-transparent px-3 text-sm text-on-surface-variant focus:outline-none"
                >
                  <option value="">All categories</option>
                  {categories.map((c) => (
                    <option key={c.slug} value={c.slug}>{c.label}</option>
                  ))}
                </select>
                <input
                  type="text"
                  value={query}
                  onChange={(e) => handleSearchChange(e.target.value)}
                  onFocus={() => setSearchFocused(true)}
                  placeholder="Search products, categories..."
                  aria-label="Search products"
                  className="min-w-0 flex-1 bg-transparent px-4 py-3 text-sm focus:outline-none"
                />
                <button type="submit" className="px-4 text-on-surface-variant hover:text-primary transition-colors" aria-label="Search">
                  <Search className="w-4 h-4" />
                </button>
              </form>

              {searchFocused && query.trim() && (
                <div className="absolute top-full mt-2 left-0 right-0 bg-white rounded-xl border border-gray-100 shadow-lg py-2 z-50 max-h-96 overflow-y-auto animate-fade-in">
                  {searchResults.length === 0 ? (
                    <div className="px-4 py-3 text-sm text-gray-400">No results for "{query}"</div>
                  ) : (
                    searchResults.map((r) => (
                      <Link
                        key={`${r.type}-${r.id}`}
                        to={r.path}
                        onClick={() => { setQuery(''); setSearchResults([]); setSearchFocused(false); }}
                        className="flex items-center gap-3 px-4 py-2.5 hover:bg-gray-50 transition-colors"
                      >
                        {r.image ? (
                          <img src={r.image} alt="" className="w-10 h-10 rounded-md object-cover flex-shrink-0" />
                        ) : (
                          <div className="w-10 h-10 rounded-md bg-gray-100 flex items-center justify-center flex-shrink-0">{resultIcon(r.type)}</div>
                        )}
                        <div className="min-w-0">
                          <p className="text-sm font-semibold text-gray-800 truncate">{r.label}</p>
                          <p className="text-xs text-gray-400 truncate">{r.sublabel}</p>
                        </div>
                      </Link>
                    ))
                  )}
                  <div className="border-t border-gray-100 mt-1 pt-1">
                    <button
                      onClick={submitSearch}
                      className="w-full text-left px-4 py-2 text-sm text-primary font-semibold hover:bg-primary/5 transition-colors"
                    >
                      See all results for "{query}"
                    </button>
                  </div>
                </div>
              )}
            </div>

            <div className="hidden lg:flex items-center gap-2.5 flex-shrink-0">
              <span className="flex h-10 w-10 items-center justify-center rounded-full bg-primary-fixed text-primary">
                <Headphones className="w-5 h-5" />
              </span>
              <div className="leading-tight">
                <a href={`tel:${contactInfo.phone.replace(/\s/g, '')}`} className="block font-headline text-sm font-bold text-primary hover:underline">
                  {contactInfo.phone}
                </a>
                <span className="text-xs text-on-surface-variant">Customer support</span>
              </div>
            </div>

            <div className="flex items-center gap-1 lg:hidden">
              {wishlistLink}
              {cartButton}
              <button
                className="p-2 text-gray-700"
                onClick={() => setMobileOpen(!mobileOpen)}
                aria-label="Toggle menu"
                aria-expanded={mobileOpen}
              >
                {mobileOpen ? <X className="w-6 h-6" /> : <Menu className="w-6 h-6" />}
              </button>
            </div>
          </div>
        </div>

        {mobileOpen && (
          <div className="lg:hidden border-t border-gray-100 bg-white animate-fade-in max-h-[calc(100vh-7rem)] overflow-y-auto">
            <div className="px-4 py-4 space-y-2">
              <form onSubmit={submitSearch} className="relative mb-4">
                <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
                <input
                  type="text"
                  value={query}
                  onChange={(e) => handleSearchChange(e.target.value)}
                  placeholder="Search products, categories..."
                  aria-label="Search products"
                  className="pl-9 pr-4 py-3 w-full bg-gray-50 border border-gray-200 rounded-lg text-sm focus:outline-none focus:border-primary"
                />
              </form>
              {navLinks.map((link) => (
                <Link
                  key={link.path}
                  to={link.path}
                  className={`flex items-center gap-2 px-4 py-3 rounded-lg font-label-md transition-colors ${
                    link.highlight
                      ? 'text-error font-semibold bg-error/5'
                      : isActive(link.path)
                        ? 'text-primary font-semibold bg-primary/5'
                        : 'text-gray-600 hover:bg-gray-50'
                  }`}
                >
                  {link.highlight && <Zap className="w-4 h-4 fill-current" />}
                  {link.label}
                </Link>
              ))}
              <div className="pt-3 border-t border-gray-100">
                <p className="px-4 pb-2 text-xs font-semibold text-gray-400">Shop by category</p>
                <div className="grid grid-cols-2 gap-1">
                  {categories.map((c) => {
                    const Icon = categoryIcons[c.slug] ?? Sprout;
                    return (
                      <Link key={c.slug} to={`/products?category=${c.slug}`} className="flex items-center gap-2 px-4 py-2.5 rounded-lg text-sm text-gray-600 hover:bg-gray-50">
                        <Icon className="w-4 h-4 text-primary" /> {c.label}
                      </Link>
                    );
                  })}
                </div>
              </div>
              <div className="flex items-center gap-3 pt-3 border-t border-gray-100">
                {user ? (
                  <button onClick={handleSignOut} className="flex-1 py-3 bg-red-50 text-red-600 rounded-lg font-label-md font-semibold flex items-center justify-center gap-2">
                    <LogOut className="w-4 h-4" /> Sign Out
                  </button>
                ) : (
                  <Link to="/signin" className="flex-1 py-3 bg-primary text-white rounded-lg font-label-md font-semibold flex items-center justify-center gap-2">
                    <User className="w-4 h-4" /> Sign In
                  </Link>
                )}
              </div>
            </div>
          </div>
        )}
      </header>

      {/* Navigation row (desktop). This is the part that stays pinned while scrolling. */}
      <div className="hidden lg:block sticky top-0 z-50 border-b border-outline-variant/60 bg-white">
        <nav className="max-w-page mx-auto px-4 sm:px-6 lg:px-8 flex items-center gap-8 h-14" aria-label="Main">
          <div className="relative" ref={categoryMenuRef}>
            <button
              onClick={() => setCategoryMenuOpen((open) => !open)}
              aria-expanded={categoryMenuOpen}
              aria-haspopup="true"
              className="flex w-64 items-center justify-between gap-3 rounded-lg bg-primary px-4 py-2.5 text-sm font-semibold text-white transition-colors hover:bg-primary-hover"
            >
              <span className="flex items-center gap-2.5"><LayoutGrid className="w-4 h-4" /> Shop by Categories</span>
              <ChevronDown className={`w-4 h-4 transition-transform ${categoryMenuOpen ? 'rotate-180' : ''}`} />
            </button>
            {categoryMenuOpen && (
              <div className="absolute left-0 top-full mt-2 w-64 rounded-xl border border-gray-100 bg-white py-2 shadow-lg animate-fade-in">
                {categories.map((c) => {
                  const Icon = categoryIcons[c.slug] ?? Sprout;
                  return (
                    <Link
                      key={c.slug}
                      to={`/products?category=${c.slug}`}
                      className="flex items-center justify-between gap-3 px-4 py-2.5 text-sm text-gray-700 transition-colors hover:bg-primary/5 hover:text-primary"
                    >
                      <span className="flex items-center gap-2.5"><Icon className="w-4 h-4 text-primary" /> {c.label}</span>
                      <span className="text-xs text-gray-400">{c.count}</span>
                    </Link>
                  );
                })}
                <Link to="/products" className="mt-1 block border-t border-gray-100 px-4 py-2.5 text-sm font-semibold text-primary hover:bg-primary/5">
                  All products
                </Link>
              </div>
            )}
          </div>

          <div className="flex items-center gap-1">
            {navLinks.map((link) => (
              <Link
                key={link.path}
                to={link.path}
                aria-current={isActive(link.path) ? 'page' : undefined}
                className={`flex items-center gap-1.5 px-3.5 py-2 rounded-lg text-sm transition-colors ${
                  link.highlight
                    ? `font-semibold text-error hover:bg-error/5 ${isActive(link.path) ? 'bg-error/5' : ''}`
                    : isActive(link.path)
                      ? 'text-primary font-semibold bg-primary/5'
                      : 'text-on-surface hover:text-primary hover:bg-surface-container-low'
                }`}
              >
                {link.highlight && <Zap className="w-3.5 h-3.5 fill-current" />}
                {link.label}
              </Link>
            ))}
          </div>

          <div className="ml-auto flex items-center gap-2">
            {wishlistLink}
            {cartButton}
            {user ? (
              <div className="relative" ref={userMenuRef}>
                <button
                  onClick={() => setUserMenuOpen(!userMenuOpen)}
                  aria-expanded={userMenuOpen}
                  aria-label="Account menu"
                  className="flex items-center gap-2 px-2 py-1.5 rounded-lg hover:bg-gray-50 transition-colors"
                >
                  <div className="relative w-8 h-8 rounded-full bg-primary text-white flex items-center justify-center text-sm font-bold">
                    {(user.email ?? 'U')[0].toUpperCase()}
                    {supportUnread > 0 && <span className="absolute -right-0.5 -top-0.5 h-3 w-3 rounded-full border-2 border-white bg-red-500" aria-label="New support reply" />}
                  </div>
                  <ChevronDown className="w-4 h-4 text-gray-400" />
                </button>
                {userMenuOpen && (
                  <div className="absolute right-0 top-full mt-2 w-60 bg-white rounded-xl border border-gray-100 shadow-lg py-2 z-50 animate-fade-in">
                    <div className="px-4 py-2.5 border-b border-gray-100">
                      <span className="block text-xs text-gray-400">Signed in as</span>
                      <span className="block text-sm font-semibold text-gray-700 truncate">{user.email}</span>
                    </div>
                    <Link to="/orders" onClick={() => setUserMenuOpen(false)} className="w-full flex items-center gap-2 px-4 py-2.5 text-sm text-gray-600 hover:bg-gray-50 hover:text-primary transition-colors">
                      <Truck className="w-4 h-4" /> My Orders
                    </Link>
                    <Link to="/support" onClick={() => setUserMenuOpen(false)} className="w-full flex items-center justify-between gap-2 px-4 py-2.5 text-sm text-gray-600 hover:bg-gray-50 hover:text-primary transition-colors">
                      <span className="flex items-center gap-2"><Headset className="w-4 h-4" /> Help &amp; Support</span>
                      {supportUnread > 0 && <span className="rounded-full bg-red-500 px-1.5 text-[10px] font-bold text-white">{supportUnread}</span>}
                    </Link>
                    <button onClick={handleSignOut} className="w-full flex items-center gap-2 px-4 py-2.5 text-sm text-gray-600 hover:bg-gray-50 hover:text-red-600 transition-colors">
                      <LogOut className="w-4 h-4" /> Sign Out
                    </button>
                  </div>
                )}
              </div>
            ) : (
              <Link to="/signin" className="flex items-center gap-2 rounded-lg px-3 py-2 text-sm font-semibold text-on-surface hover:text-primary hover:bg-surface-container-low transition-colors">
                <User className="w-5 h-5" /> Sign in
              </Link>
            )}
          </div>
        </nav>
      </div>
    </>
  );
}
