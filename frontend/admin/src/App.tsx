import React, { Suspense, useEffect } from 'react';
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { useAuthStore } from './stores/authStore';
import LoadingSpinner from './components/common/LoadingSpinner';
import AdminLayout from './components/layout/AdminLayout';

// Auth pages
const LazyLoginPage = React.lazy(() => import('./pages/auth/LoginPage'));
const LazyForgotPasswordPage = React.lazy(() => import('./pages/auth/ForgotPasswordPage'));
const LazyResetPasswordPage = React.lazy(() => import('./pages/auth/ResetPasswordPage'));

// Admin pages
const LazyAdminDashboardPage = React.lazy(() => import('./pages/admin/dashboard/DashboardPage'));
const LazyAdminProductsPage = React.lazy(() => import('./pages/admin/products/ProductsPage'));
const LazyNewProductPage = React.lazy(() => import('./pages/admin/products/ProductFormPage'));
const LazyEditProductPage = React.lazy(() => import('./pages/admin/products/ProductFormPage'));
const LazyCategoriesPage = React.lazy(() => import('./pages/admin/categories/CategoriesPage'));
const LazyAdminOrdersPage = React.lazy(() => import('./pages/admin/orders/AdminOrdersPage'));
const LazyPurchasesPage = React.lazy(() => import('./pages/admin/purchases/PurchasesPage'));
const LazySupportPage = React.lazy(() => import('./pages/admin/support/SupportPage'));
const LazyProfitPage = React.lazy(() => import('./pages/admin/reports/ProfitPage'));
const LazyInventoryPage = React.lazy(() => import('./pages/admin/inventory/InventoryPage'));
const LazyUsersPage = React.lazy(() => import('./pages/admin/users/UsersPage'));
const LazyAdminReviewsPage = React.lazy(() => import('./pages/admin/reviews/ReviewsModerationPage'));
const LazyShipmentsPage = React.lazy(() => import('./pages/admin/shipping/ShipmentsPage'));
const LazyCampaignsPage = React.lazy(() => import('./pages/admin/marketing/CampaignsPage'));
const LazyPromotionsPage = React.lazy(() => import('./pages/admin/marketing/PromotionsPage'));
const LazySegmentsPage = React.lazy(() => import('./pages/admin/marketing/SegmentsPage'));
const LazyAnalyticsPage = React.lazy(() => import('./pages/admin/analytics/AnalyticsPage'));
const LazyAuditPage = React.lazy(() => import('./pages/admin/audit/AuditLogsPage'));

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 5 * 60 * 1000,
      retry: 1,
      refetchOnWindowFocus: false,
    },
  },
});

function AuthInitializer() {
  const initialize = useAuthStore((s) => s.initialize);
  useEffect(() => {
    initialize();
  }, [initialize]);
  return null;
}

export default function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <BrowserRouter future={{ v7_startTransition: true, v7_relativeSplatPath: true }}>
        <AuthInitializer />
        <Suspense fallback={<LoadingSpinner />}>
          <Routes>
            {/* Auth pages */}
            <Route path="/login" element={<LazyLoginPage />} />
            <Route path="/forgot-password" element={<LazyForgotPasswordPage />} />
            <Route path="/reset-password" element={<LazyResetPasswordPage />} />

            {/* Admin panel */}
            <Route path="/admin" element={<AdminLayout />}>
              <Route index element={<LazyAdminDashboardPage />} />
              <Route path="products" element={<LazyAdminProductsPage />} />
              <Route path="products/new" element={<LazyNewProductPage />} />
              <Route path="products/:id/edit" element={<LazyEditProductPage />} />
              <Route path="categories" element={<LazyCategoriesPage />} />
              <Route path="orders" element={<LazyAdminOrdersPage />} />
              <Route path="inventory" element={<LazyInventoryPage />} />
              <Route path="purchases" element={<LazyPurchasesPage />} />
              <Route path="profit" element={<LazyProfitPage />} />
              <Route path="support" element={<LazySupportPage />} />
              <Route path="users" element={<LazyUsersPage />} />
              <Route path="reviews" element={<LazyAdminReviewsPage />} />
              <Route path="shipments" element={<LazyShipmentsPage />} />
              <Route path="marketing/campaigns" element={<LazyCampaignsPage />} />
              <Route path="marketing/promotions" element={<LazyPromotionsPage />} />
              <Route path="marketing/segments" element={<LazySegmentsPage />} />
              <Route path="analytics" element={<LazyAnalyticsPage />} />
              <Route path="audit" element={<LazyAuditPage />} />
            </Route>

            {/* Catch-all: redirect to admin */}
            <Route path="*" element={<Navigate to="/admin" replace />} />
          </Routes>
        </Suspense>
      </BrowserRouter>
    </QueryClientProvider>
  );
}
