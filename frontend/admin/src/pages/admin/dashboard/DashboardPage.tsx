import { useQuery } from '@tanstack/react-query';
import { subDays, format, parseISO } from 'date-fns';
import {
  UsersIcon,
  UserGroupIcon,
  StarIcon,
  ShoppingCartIcon,
  EyeIcon,
  ArrowTrendingUpIcon,
  CurrencyDollarIcon,
} from '@heroicons/react/24/outline';
import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
} from 'recharts';
import { Link } from 'react-router-dom';
import { getDashboardCounts } from '../../../api/admin';
import { getOverview, getDailyTrend } from '../../../api/analytics';
import { formatCurrency } from '../../../lib/utils';
import LoadingSpinner from '../../../components/common/LoadingSpinner';
import ErrorMessage from '../../../components/common/ErrorMessage';
import { useAuthStore } from '../../../stores/authStore';

function StatCard({
  label,
  value,
  icon: Icon,
  color = 'primary',
  change,
}: {
  label: string;
  value: string | number;
  icon: React.ComponentType<React.SVGProps<SVGSVGElement>>;
  color?: string;
  change?: string;
}) {
  const colorMap: Record<string, { bg: string; icon: string }> = {
    primary: { bg: 'bg-primary-50', icon: 'text-primary-600' },
    green: { bg: 'bg-green-50', icon: 'text-green-600' },
    yellow: { bg: 'bg-amber-50', icon: 'text-amber-600' },
    amber: { bg: 'bg-amber-50', icon: 'text-amber-600' },
    red: { bg: 'bg-red-50', icon: 'text-red-600' },
    purple: { bg: 'bg-purple-50', icon: 'text-purple-600' },
  };

  const colors = colorMap[color] || colorMap.primary;

  return (
    <div className="bg-white rounded-2xl p-5 border border-gray-100 shadow-soft hover:shadow-card-hover transition-shadow duration-200">
      <div className="flex items-center justify-between">
        <p className="text-sm text-gray-500">{label}</p>
        <div className={`w-12 h-12 rounded-xl flex items-center justify-center ${colors.bg}`}>
          <Icon className={`h-5 w-5 ${colors.icon}`} />
        </div>
      </div>
      <p className="mt-3 text-2xl font-bold text-gray-900">{value}</p>
      {change && (
        <p className="mt-1 text-sm font-medium text-green-600">{change}</p>
      )}
    </div>
  );
}

export default function DashboardPage() {
  const user = useAuthStore((s) => s.user);

  const thirtyDaysAgo = format(subDays(new Date(), 30), 'yyyy-MM-dd');
  const today = format(new Date(), 'yyyy-MM-dd');
  const sevenDaysAgo = format(subDays(new Date(), 7), 'yyyy-MM-dd');

  const countsQuery = useQuery({
    queryKey: ['admin', 'dashboard', 'counts'],
    queryFn: getDashboardCounts,
  });

  const overviewQuery = useQuery({
    queryKey: ['analytics', 'overview', thirtyDaysAgo, today],
    queryFn: () => getOverview(thirtyDaysAgo, today),
  });

  const trendQuery = useQuery({
    queryKey: ['analytics', 'daily-trend', sevenDaysAgo, today],
    queryFn: () => getDailyTrend(sevenDaysAgo, today),
  });

  if (countsQuery.isLoading || overviewQuery.isLoading) {
    return (
      <div className="card">
        <LoadingSpinner />
      </div>
    );
  }

  if (countsQuery.isError) {
    return (
      <div className="card">
        <ErrorMessage
          message={countsQuery.error?.message || 'Failed to load dashboard counts'}
          onRetry={() => countsQuery.refetch()}
        />
      </div>
    );
  }

  if (overviewQuery.isError) {
    return (
      <div className="card">
        <ErrorMessage
          message={overviewQuery.error?.message || 'Failed to load analytics overview'}
          onRetry={() => overviewQuery.refetch()}
        />
      </div>
    );
  }

  const counts = countsQuery.data!;
  const overview = overviewQuery.data!;

  const chartData = (trendQuery.data ?? []).map((d) => ({
    date: format(parseISO(d.date), 'MMM dd'),
    count: d.count,
  }));

  const quickLinks = [
    { label: 'Products', href: '/admin/products', icon: ShoppingCartIcon },
    { label: 'Categories', href: '/admin/categories', icon: UserGroupIcon },
    { label: 'Orders', href: '/admin/orders', icon: ShoppingCartIcon },
    { label: 'Inventory', href: '/admin/inventory', icon: CurrencyDollarIcon },
    { label: 'Users', href: '/admin/users', icon: UsersIcon },
    { label: 'Reviews', href: '/admin/reviews', icon: StarIcon },
    { label: 'Shipments', href: '/admin/shipments', icon: ShoppingCartIcon },
    { label: 'Analytics', href: '/admin/analytics', icon: ArrowTrendingUpIcon },
  ];

  return (
    <div className="space-y-6">
      {/* Page Header */}
      <div>
        <span className="section-label">Overview</span>
        <h1 className="mt-2 text-2xl font-bold text-gray-900">Dashboard</h1>
        <p className="mt-1 text-gray-500">
          Welcome back, {user?.first_name || 'Admin'}
        </p>
      </div>

      {/* Stats Row 1: 4 cards */}
      <div className="grid grid-cols-2 lg:grid-cols-4 gap-5">
        <StatCard
          label="Total Users"
          value={counts.total_users.toLocaleString()}
          icon={UsersIcon}
          color="primary"
        />
        <StatCard
          label="Total Customers"
          value={counts.total_customers.toLocaleString()}
          icon={UserGroupIcon}
          color="green"
        />
        <StatCard
          label="Pending Reviews"
          value={counts.pending_reviews.toLocaleString()}
          icon={StarIcon}
          color="amber"
        />
        <StatCard
          label="Orders Delivered (30 days)"
          value={overview.orders_completed.toLocaleString()}
          icon={ShoppingCartIcon}
          color="purple"
        />
      </div>

      {/* Stats Row 2: 3 cards */}
      <div className="grid grid-cols-1 sm:grid-cols-3 gap-5">
        <StatCard
          label="Orders Placed (30 days)"
          value={overview.checkouts_started.toLocaleString()}
          icon={EyeIcon}
          color="primary"
        />
        <StatCard
          label="Delivered Rate"
          value={`${overview.conversion_rate.toFixed(1)}%`}
          icon={ArrowTrendingUpIcon}
          color="green"
        />
        <StatCard
          label="Revenue (delivered, 30 days)"
          value={formatCurrency(overview.revenue_cents)}
          icon={CurrencyDollarIcon}
          color="purple"
        />
      </div>

      {/* Revenue Trend Chart */}
      <div className="card">
        <div className="flex items-center justify-between">
          <h2 className="text-lg font-semibold text-gray-900">
            Orders Delivered per Day (Last 7 Days)
          </h2>
          <span className="text-xs text-gray-400 font-medium px-3 py-1 rounded-full bg-gray-50 border border-gray-100">
            Last 7 days
          </span>
        </div>
        <div className="mt-4">
          {trendQuery.isLoading ? (
            <div className="h-72 flex items-center justify-center">
              <LoadingSpinner size="sm" />
            </div>
          ) : trendQuery.isError ? (
            <div className="h-72 flex items-center justify-center">
              <ErrorMessage
                message={trendQuery.error?.message || 'Failed to load trend data'}
                onRetry={() => trendQuery.refetch()}
              />
            </div>
          ) : chartData.length === 0 ? (
            <div className="h-72 flex items-center justify-center text-sm text-gray-400">
              No data available for the selected period.
            </div>
          ) : (
            <div className="h-72">
              <ResponsiveContainer width="100%" height="100%">
                <LineChart data={chartData}>
                  <CartesianGrid strokeDasharray="3 3" stroke="#f0f0f0" />
                  <XAxis
                    dataKey="date"
                    tick={{ fontSize: 12, fill: '#9ca3af' }}
                    tickLine={false}
                    axisLine={{ stroke: '#e5e7eb' }}
                  />
                  <YAxis
                    tick={{ fontSize: 12, fill: '#9ca3af' }}
                    tickLine={false}
                    axisLine={{ stroke: '#e5e7eb' }}
                    allowDecimals={false}
                  />
                  <Tooltip
                    contentStyle={{
                      borderRadius: '12px',
                      border: '1px solid #e5e7eb',
                      boxShadow: '0 4px 12px rgba(0,0,0,0.08)',
                    }}
                  />
                  <Line
                    type="monotone"
                    dataKey="count"
                    stroke="#1a4d2e"
                    strokeWidth={2}
                    dot={{ r: 4, fill: '#1a4d2e', strokeWidth: 0 }}
                    activeDot={{ r: 6, fill: '#1a4d2e', strokeWidth: 0 }}
                    name="Orders"
                  />
                </LineChart>
              </ResponsiveContainer>
            </div>
          )}
        </div>
      </div>

      {/* Quick Actions */}
      <div className="card">
        <h2 className="text-lg font-semibold text-gray-900 mb-4">Quick Actions</h2>
        <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
          {quickLinks.map((link) => (
            <Link
              key={link.href}
              to={link.href}
              className="flex items-center gap-3 rounded-xl border border-gray-100 p-3 text-sm font-medium text-gray-700 hover:bg-primary-50 hover:border-primary-200 hover:text-primary-700 transition-colors duration-150"
            >
              <link.icon className="h-5 w-5 text-gray-400" />
              {link.label}
            </Link>
          ))}
        </div>
      </div>
    </div>
  );
}
