import { useState, useEffect } from 'react';
import { Outlet, useNavigate, useLocation } from 'react-router-dom';
import {
  Bars3Icon,
  ArrowRightOnRectangleIcon,
} from '@heroicons/react/24/outline';
import { useAuth } from '../../hooks/useAuth';
import { useRequireAdmin } from '../../hooks/useAuth';
import Sidebar from './Sidebar';
import { cn } from '../../lib/utils';

const routeTitles: Record<string, string> = {
  '/admin': 'Dashboard',
  '/admin/products': 'Products',
  '/admin/products/new': 'New Product',
  '/admin/categories': 'Categories',
  '/admin/orders': 'Orders',
  '/admin/inventory': 'Inventory',
  '/admin/users': 'Users',
  '/admin/reviews': 'Reviews',
  '/admin/shipments': 'Shipping',
  '/admin/marketing/campaigns': 'Campaigns',
  '/admin/marketing/promotions': 'Promotions',
  '/admin/marketing/segments': 'Segments',
  '/admin/analytics': 'Analytics',
  '/admin/audit': 'Audit Logs',
};

function getPageTitle(pathname: string): string {
  if (routeTitles[pathname]) return routeTitles[pathname];
  if (pathname.startsWith('/admin/products/') && pathname.endsWith('/edit')) {
    return 'Edit Product';
  }  
  if (pathname.startsWith('/admin/orders/')) {
    return 'Order Details';
  }
  return 'Admin';
}

function getRoleBadgeClass(role: string): string {
  switch (role.toUpperCase()) {
    case 'SUPER_ADMIN':
      return 'bg-primary-100 text-primary-800';
    case 'ADMIN':
      return 'bg-primary-50 text-primary-700';
    case 'MANAGER':
      return 'bg-primary-100/50 text-primary-700';
    default:
      return 'bg-gray-100 text-gray-600';
  }
}

export default function AdminLayout() {
  useRequireAdmin();
  const { user, clearAuth } = useAuth();
  const navigate = useNavigate();
  const location = useLocation();
  const [sidebarCollapsed, setSidebarCollapsed] = useState(false);
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false);

  useEffect(() => {
    setMobileMenuOpen(false);
  }, [location.pathname]);

  const pageTitle = getPageTitle(location.pathname);

  const handleLogout = () => {
    clearAuth();
    setTimeout(() => navigate('/login', { replace: true }), 10);
  };

  const userDisplayName = user
    ? `${user.first_name} ${user.last_name}`
    : 'User';

  const userRole = user?.role ?? '';

  return (
    <div className="min-h-screen flex bg-surface-50">
      {/* Mobile sidebar overlay */}
      {mobileMenuOpen && (
        <div
          className="fixed inset-0 z-30 bg-black/30 lg:hidden"
          onClick={() => setMobileMenuOpen(false)}
          aria-hidden="true"
        />
      )}

      {/* Sidebar - desktop */}
      <div className="hidden lg:block">
        <Sidebar
          collapsed={sidebarCollapsed}
          onToggle={() => setSidebarCollapsed(!sidebarCollapsed)}
        />
      </div>

      {/* Sidebar - mobile */}
      <div
        className={cn(
          'fixed inset-y-0 left-0 z-40 lg:hidden transition-transform duration-300',
          mobileMenuOpen ? 'translate-x-0' : '-translate-x-full'
        )}
      >
        <Sidebar
          collapsed={false}
          onToggle={() => setMobileMenuOpen(false)}
        />
      </div>

      {/* Main content area */}
      <div className="flex-1 flex flex-col min-w-0">
        {/* Top bar */}
        <header className="sticky top-0 z-20 bg-white/80 backdrop-blur-xl border-b border-gray-100 h-16 flex items-center justify-between px-4 lg:px-6 shadow-nav">
          <div className="flex items-center gap-3">
            {/* Mobile hamburger */}
            <button
              className="lg:hidden p-2 -ml-2 rounded-xl text-gray-500 hover:bg-primary-50 hover:text-primary-700 transition-colors"
              onClick={() => setMobileMenuOpen(true)}
              aria-label="Open menu"
            >
              <Bars3Icon className="h-5 w-5" />
            </button>

            <h1 className="text-lg font-semibold text-gray-900">{pageTitle}</h1>
          </div>

          <div className="flex items-center gap-3">
            <span className="hidden sm:block text-sm font-medium text-gray-700">{userDisplayName}</span>
            {userRole && (
              <span
                className={cn(
                  'inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-semibold',
                  getRoleBadgeClass(userRole)
                )}
              >
                {userRole.replace(/_/g, ' ').toLowerCase().replace(/\b\w/g, c => c.toUpperCase())}
              </span>
            )}
            <button
              onClick={handleLogout}
              className="inline-flex items-center gap-1.5 rounded-xl px-3 py-1.5 text-sm font-medium text-gray-500 hover:text-red-600 transition-colors"
              title="Sign out"
            >
              <ArrowRightOnRectangleIcon className="h-4 w-4" />
              <span className="hidden sm:inline">Sign out</span>
            </button>
          </div>
        </header>

        {/* Page content */}
        <main className="flex-1 p-4 lg:p-8">
          <Outlet />
        </main>
      </div>
    </div>
  );
}
