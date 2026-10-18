import { NavLink } from 'react-router-dom';
import { APP_NAME } from '../../config/brand';
import { useQuery } from '@tanstack/react-query';
import { getSupportSummary } from '../../api/support';
import {
  HomeIcon,
  CubeIcon,
  FolderIcon,
  ShoppingCartIcon,
  CubeTransparentIcon,
  TruckIcon,
  UsersIcon,
  StarIcon,
  MegaphoneIcon,
  TagIcon,
  UserGroupIcon,
  ChartBarIcon,
  ClipboardDocumentListIcon,
  ChevronLeftIcon,
  ArrowDownTrayIcon,
  BanknotesIcon,
  ChatBubbleLeftRightIcon,
} from '@heroicons/react/24/outline';
import type { ReactNode } from 'react';

interface NavItem {
  label: string;
  to: string;
  icon: ReactNode;
}

interface NavGroup {
  title: string;
  items: NavItem[];
}

const navGroups: NavGroup[] = [
  {
    title: 'MAIN',
    items: [
      { label: 'Dashboard', to: '/admin', icon: <HomeIcon className="h-5 w-5" /> },
      { label: 'Products', to: '/admin/products', icon: <CubeIcon className="h-5 w-5" /> },
      { label: 'Categories', to: '/admin/categories', icon: <FolderIcon className="h-5 w-5" /> },
      { label: 'Orders', to: '/admin/orders', icon: <ShoppingCartIcon className="h-5 w-5" /> },
      { label: 'Support', to: '/admin/support', icon: <ChatBubbleLeftRightIcon className="h-5 w-5" /> },
    ],
  },
  {
    title: 'MANAGEMENT',
    items: [
      { label: 'Inventory', to: '/admin/inventory', icon: <CubeTransparentIcon className="h-5 w-5" /> },
      { label: 'Purchases', to: '/admin/purchases', icon: <ArrowDownTrayIcon className="h-5 w-5" /> },
      { label: 'Profit', to: '/admin/profit', icon: <BanknotesIcon className="h-5 w-5" /> },
      { label: 'Users', to: '/admin/users', icon: <UsersIcon className="h-5 w-5" /> },
      { label: 'Reviews', to: '/admin/reviews', icon: <StarIcon className="h-5 w-5" /> },
      { label: 'Shipping', to: '/admin/shipments', icon: <TruckIcon className="h-5 w-5" /> },
    ],
  },
  {
    title: 'MARKETING',
    items: [
      { label: 'Campaigns', to: '/admin/marketing/campaigns', icon: <MegaphoneIcon className="h-5 w-5" /> },
      { label: 'Promotions', to: '/admin/marketing/promotions', icon: <TagIcon className="h-5 w-5" /> },
      { label: 'Segments', to: '/admin/marketing/segments', icon: <UserGroupIcon className="h-5 w-5" /> },
    ],
  },
  {
    title: 'OTHER',
    items: [
      { label: 'Analytics', to: '/admin/analytics', icon: <ChartBarIcon className="h-5 w-5" /> },
      { label: 'Audit Logs', to: '/admin/audit', icon: <ClipboardDocumentListIcon className="h-5 w-5" /> },
    ],
  },
];

interface SidebarProps {
  collapsed: boolean;
  onToggle: () => void;
}

export default function Sidebar({ collapsed, onToggle }: SidebarProps) {
  // Number of conversations waiting for a reply (refreshes every 30 seconds).
  const summary = useQuery({ queryKey: ['admin', 'support-summary'], queryFn: getSupportSummary, refetchInterval: 30_000, retry: false });
  const supportBadge = summary.data?.open ?? 0;
  return (
    <aside
      className={
        `flex flex-col bg-white border-r border-gray-100 h-full
        transition-all duration-300 ease-in-out
        ${collapsed ? 'w-20' : 'w-64'}`
      }
    >
      {/* Logo Area */}
      <div className="flex items-center p-5 border-b border-gray-100 flex-shrink-0">
        {!collapsed ? (
          <div className="flex items-center gap-2.5">
            <div className="w-9 h-9 bg-primary-600 rounded-xl flex items-center justify-center">
              <svg className="h-5 w-5 text-white" fill="none" viewBox="0 0 24 24" strokeWidth={2} stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" d="M2.25 3h1.386c.51 0 .955.343 1.087.835l.383 1.437M7.5 14.25a3 3 0 00-3 3h15.75m-12.75-3h11.218c1.121-2.3 2.1-4.684 2.924-7.138a60.114 60.114 0 00-16.536-1.84M7.5 14.25L5.106 5.272M6 20.25a.75.75 0 11-1.5 0 .75.75 0 011.5 0zm12.75 0a.75.75 0 11-1.5 0 .75.75 0 011.5 0z" />
              </svg>
            </div>
            <span className="text-lg font-bold text-gray-900 truncate">{APP_NAME}</span>
          </div>
        ) : (
          <div className="mx-auto w-9 h-9 bg-primary-600 rounded-xl flex items-center justify-center">
            <svg className="h-5 w-5 text-white" fill="none" viewBox="0 0 24 24" strokeWidth={2} stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" d="M2.25 3h1.386c.51 0 .955.343 1.087.835l.383 1.437M7.5 14.25a3 3 0 00-3 3h15.75m-12.75-3h11.218c1.121-2.3 2.1-4.684 2.924-7.138a60.114 60.114 0 00-16.536-1.84M7.5 14.25L5.106 5.272M6 20.25a.75.75 0 11-1.5 0 .75.75 0 011.5 0zm12.75 0a.75.75 0 11-1.5 0 .75.75 0 011.5 0z" />
            </svg>
          </div>
        )}
      </div>

      {/* Navigation */}
      <nav className="flex-1 overflow-y-auto py-4 px-3">
        {navGroups.map((group, groupIndex) => (
          <div key={group.title}>
            {!collapsed && (
              <h3
                className={
                  `text-[10px] font-bold tracking-widest text-gray-400 uppercase px-3 mb-2 ${
                    groupIndex === 0 ? 'mt-0' : 'mt-6'
                  }`
                }
              >
                {group.title}
              </h3>
            )}
            {collapsed && groupIndex > 0 && (
              <div className="my-4 mx-auto w-6 border-t border-gray-200" />
            )}
            <ul className="space-y-1">
              {group.items.map((item) => (
                <li key={item.to} className="relative">
                  <NavLink
                    to={item.to}
                    end={item.to === '/admin'}
                    className={({ isActive }) =>
                      `sidebar-link flex items-center gap-3 px-3 py-2.5 rounded-xl text-sm font-medium transition-all duration-200 ${
                        isActive
                          ? 'sidebar-link-active bg-primary-600 text-white shadow-soft'
                          : 'text-gray-600 hover:bg-primary-50 hover:text-primary-700'
                      } ${collapsed ? 'justify-center' : ''}`
                    }
                    title={collapsed ? item.label : undefined}
                  >
                    {item.icon}
                    {!collapsed && <span className="truncate">{item.label}</span>}
                    {item.to === '/admin/support' && supportBadge > 0 && (
                      <span className={`ml-auto rounded-full bg-red-500 px-1.5 text-[10px] font-bold text-white ${collapsed ? 'absolute -right-1 -top-1' : ''}`}>{supportBadge}</span>
                    )}
                  </NavLink>
                </li>
              ))}
            </ul>
          </div>
        ))}
      </nav>

      {/* Collapse Toggle */}
      <div className="border-t border-gray-100 p-3 flex-shrink-0">
        <button
          onClick={onToggle}
          className="w-full flex items-center justify-center gap-2 px-3 py-2.5 rounded-xl text-sm font-medium text-gray-500 hover:bg-primary-50 hover:text-primary-700 transition-all duration-200"
          aria-label={collapsed ? 'Expand sidebar' : 'Collapse sidebar'}
        >
          <ChevronLeftIcon
            className={
              `h-5 w-5 transition-transform duration-300 ${
                collapsed ? 'rotate-180' : ''
              }`
            }
          />
          {!collapsed && <span>Collapse</span>}
        </button>
      </div>
    </aside>
  );
}
