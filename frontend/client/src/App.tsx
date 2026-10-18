import { BrowserRouter, Routes, Route, Navigate, useLocation } from 'react-router-dom';
import { Suspense, lazy, useState } from 'react';
import Navbar from '@/components/Navbar';
import Footer from '@/components/Footer';
import CartDrawer from '@/components/CartDrawer';
import ScrollToTop from '@/components/ScrollToTop';
import RouteMeta from '@/components/RouteMeta';
import HomePage from '@/pages/HomePage';
const BrowseProductsPage = lazy(() => import('@/pages/BrowseProductsPage'));
const ProductDetailPage = lazy(() => import('@/pages/ProductDetailPage'));
const CheckoutPage = lazy(() => import('@/pages/CheckoutPage'));
const OffersPage = lazy(() => import('@/pages/OffersPage'));
const TechnologyPage = lazy(() => import('@/pages/TechnologyPage'));
const TrackOrderPage = lazy(() => import('@/pages/TrackOrderPage'));
const AboutPage = lazy(() => import('@/pages/AboutPage'));
const CareersPage = lazy(() => import('@/pages/CareersPage'));
const PartnersPage = lazy(() => import('@/pages/PartnersPage'));
const PrivacyPage = lazy(() => import('@/pages/PrivacyPage'));
const TermsPage = lazy(() => import('@/pages/TermsPage'));
const AuthPage = lazy(() => import('@/pages/AuthPage'));
const NotFoundPage = lazy(() => import('@/pages/NotFoundPage'));
const WishlistPage = lazy(() => import('@/pages/WishlistPage'));
const MyOrdersPage = lazy(() => import('@/pages/MyOrdersPage'));
const SupportPage = lazy(() => import('@/pages/SupportPage'));
const SupportTicketPage = lazy(() => import('@/pages/SupportTicketPage'));
const ForgotPasswordPage = lazy(() => import('@/pages/ForgotPasswordPage'));
const ResetPasswordPage = lazy(() => import('@/pages/ResetPasswordPage'));
const VerifyEmailPage = lazy(() => import('@/pages/VerifyEmailPage'));
import { CartProvider } from '@/context/CartContext';
import { CatalogProvider } from '@/context/CatalogContext';
import { AuthProvider } from '@/context/AuthContext';
import { WishlistProvider } from '@/context/WishlistContext';

function AppShell() {
  const [cartOpen, setCartOpen] = useState(false);
  const location = useLocation();
  const isStandalone = location.pathname === '/checkout' || location.pathname === '/signin' || location.pathname === '/signup' ||
    location.pathname === '/forgot-password' || location.pathname === '/reset-password' || location.pathname === '/verify-email';

  return (
    <>
      <ScrollToTop />
      <RouteMeta />
      {!isStandalone && <Navbar onCartClick={() => setCartOpen(true)} />}
      <main className={isStandalone ? '' : 'flex-1'}>
        <Suspense fallback={<div className="min-h-[60vh]" aria-busy="true" />}>
        <Routes>
          <Route path="/" element={<HomePage />} />
          <Route path="/products" element={<BrowseProductsPage />} />
          <Route path="/products/:id" element={<ProductDetailPage />} />
          <Route path="/categories" element={<Navigate to="/#categories" replace />} />
          <Route path="/reviews" element={<Navigate to="/products" replace />} />
          <Route path="/offers" element={<OffersPage />} />
          <Route path="/wishlist" element={<WishlistPage />} />
          <Route path="/technology" element={<TechnologyPage />} />
          <Route path="/signin" element={<AuthPage initialTab="signin" />} />
          <Route path="/signup" element={<AuthPage initialTab="signup" />} />
          <Route path="/forgot-password" element={<ForgotPasswordPage />} />
          <Route path="/reset-password" element={<ResetPasswordPage />} />
          <Route path="/verify-email" element={<VerifyEmailPage />} />
          <Route path="/checkout" element={<CheckoutPage />} />
          <Route path="/track-order" element={<TrackOrderPage />} />
          <Route path="/orders" element={<MyOrdersPage />} />
          <Route path="/support" element={<SupportPage />} />
          <Route path="/support/:id" element={<SupportTicketPage />} />
          <Route path="/about" element={<AboutPage />} />
          <Route path="/careers" element={<CareersPage />} />
          <Route path="/partners" element={<PartnersPage />} />
          <Route path="/privacy" element={<PrivacyPage />} />
          <Route path="/terms" element={<TermsPage />} />
          <Route path="*" element={<NotFoundPage />} />
        </Routes>
        </Suspense>
      </main>
      {!isStandalone && <Footer />}
      {!isStandalone && <CartDrawer open={cartOpen} onClose={() => setCartOpen(false)} />}
    </>
  );
}

export default function App() {
  return (
    <BrowserRouter>
      <AuthProvider>
        <CatalogProvider>
          <CartProvider>
            <WishlistProvider>
              <div className="min-h-screen flex flex-col bg-white">
                <AppShell />
              </div>
            </WishlistProvider>
          </CartProvider>
        </CatalogProvider>
      </AuthProvider>
    </BrowserRouter>
  );
}
