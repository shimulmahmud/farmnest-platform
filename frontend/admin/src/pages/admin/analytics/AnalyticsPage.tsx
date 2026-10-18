import { useState, useMemo, useCallback } from 'react';
import { useQuery } from '@tanstack/react-query';
import {
  EyeIcon,
  CreditCardIcon,
  CheckCircleIcon,
  ArrowTrendingUpIcon,
  CurrencyDollarIcon,
  ChartBarIcon,
} from '@heroicons/react/24/outline';
import {
  BarChart,
  Bar,
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
} from 'recharts';
import { getOverview, getTopProducts, getDailyTrend } from '../../../api/analytics';
import { formatCurrency, formatDate } from '../../../lib/utils';
import LoadingSpinner from '../../../components/common/LoadingSpinner';
import ErrorMessage from '../../../components/common/ErrorMessage';
import DataTable from '../../../components/common/DataTable';

function getDaysAgo(days: number): string {
  const d = new Date();
  d.setDate(d.getDate() - days);
  return d.toISOString().split('T')[0];
}

function getToday(): string {
  return new Date().toISOString().split('T')[0];
}

interface StatCardData {
  label: string;
  value: string;
  icon: typeof EyeIcon;
  color: string;
}

export default function AnalyticsPage() {
  const [startAt, setStartAt] = useState(getDaysAgo(30));
  const [endAt, setEndAt] = useState(getToday());

  const overviewQuery = useQuery({
    queryKey: ['analytics', 'overview', startAt, endAt],
    queryFn: () => getOverview(startAt, endAt),
  });

  const orderTrendQuery = useQuery({
    queryKey: ['analytics', 'trend', 'order_completed', startAt, endAt],
    queryFn: () => getDailyTrend(startAt, endAt, 'order_completed'),
    enabled: !!startAt && !!endAt,
  });

  const pageViewTrendQuery = useQuery({
    queryKey: ['analytics', 'trend', 'order_created', startAt, endAt],
    queryFn: () => getDailyTrend(startAt, endAt, 'order_created'),
    enabled: !!startAt && !!endAt,
  });

  const topProductsQuery = useQuery({
    queryKey: ['analytics', 'top-products', startAt, endAt],
    queryFn: () => getTopProducts(10, startAt, endAt),
  });

  const handleApplyRange = useCallback((e: React.FormEvent) => {
    e.preventDefault();
  }, []);

  const stats: StatCardData[] = useMemo(() => {
    const o = overviewQuery.data;
    if (!o) return [];
    return [
      {
        label: 'Orders Placed',
        value: o.checkouts_started.toLocaleString(),
        icon: CreditCardIcon,
        color: 'bg-orange-50 text-orange-600',
      },
      {
        label: 'Orders Delivered',
        value: o.orders_completed.toLocaleString(),
        icon: CheckCircleIcon,
        color: 'bg-green-50 text-green-600',
      },
      {
        label: 'Delivered Rate',
        value: `${o.conversion_rate.toFixed(1)}%`,
        icon: ArrowTrendingUpIcon,
        color: 'bg-emerald-50 text-emerald-600',
      },
      {
        label: 'Revenue (delivered)',
        value: formatCurrency(o.revenue_cents),
        icon: CurrencyDollarIcon,
        color: 'bg-teal-50 text-teal-600',
      },
    ];
  }, [overviewQuery.data]);

  const topProductsData = useMemo(() => {
    if (!topProductsQuery.data) return [];
    return topProductsQuery.data.map((p, idx) => ({ ...p, _rank: idx + 1 }));
  }, [topProductsQuery.data]);

  const topProductColumns = [
    {
      key: 'rank',
      header: '#',
      render: (row: Record<string, unknown>) => (
        <span className="text-sm font-medium text-gray-500">{String(row._rank ?? '')}</span>
      ),
    },
    {
      key: 'product_id',
      header: 'Product ID',
      render: (row: Record<string, unknown>) => (
        <span className="text-sm font-mono text-gray-700">{String(row.product_id ?? '')}</span>
      ),
    },
    {
      key: 'views',
      header: 'Views',
      render: (row: Record<string, unknown>) => (
        <span className="text-sm font-semibold text-gray-900">
          {Number(row.views ?? 0).toLocaleString()}
        </span>
      ),
    },
  ];

  const isOverviewLoading =
    overviewQuery.isLoading &&
    orderTrendQuery.isLoading &&
    pageViewTrendQuery.isLoading;

  const overviewError =
    overviewQuery.error || orderTrendQuery.error || pageViewTrendQuery.error;

  if (isOverviewLoading && !overviewQuery.data) return <LoadingSpinner />;

  if (overviewError && !overviewQuery.data) {
    return (
      <ErrorMessage
        message={overviewError?.message || 'Failed to load analytics'}
        onRetry={() => {
          overviewQuery.refetch();
          orderTrendQuery.refetch();
          pageViewTrendQuery.refetch();
          topProductsQuery.refetch();
        }}
      />
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Analytics</h1>
          <p className="mt-1 text-sm text-gray-500">
            Overview of store performance and customer behavior
          </p>
        </div>

        <form onSubmit={handleApplyRange} className="flex items-end gap-3">
          <div>
            <label htmlFor="start-date" className="block text-xs font-medium text-gray-600 mb-1">
              Start
            </label>
            <input
              id="start-date"
              type="date"
              value={startAt}
              onChange={(e) => setStartAt(e.target.value)}
              className="input-field !py-1.5 !text-sm"
            />
          </div>
          <div>
            <label htmlFor="end-date" className="block text-xs font-medium text-gray-600 mb-1">
              End
            </label>
            <input
              id="end-date"
              type="date"
              value={endAt}
              onChange={(e) => setEndAt(e.target.value)}
              className="input-field !py-1.5 !text-sm"
            />
          </div>
        </form>
      </div>

      {/* Stat Cards */}
      {overviewQuery.isLoading && !overviewQuery.data ? (
        <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 xl:grid-cols-4 gap-4">
          {Array.from({ length: 4 }).map((_, i) => (
            <div key={i} className="card animate-pulse">
              <div className="h-4 bg-gray-200 rounded w-16 mb-2" />
              <div className="h-6 bg-gray-200 rounded w-20" />
            </div>
          ))}
        </div>
      ) : (
        <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 xl:grid-cols-4 gap-4">
          {stats.map((stat) => {
            const Icon = stat.icon;
            return (
              <div key={stat.label} className="stat-card">
                <div className="flex items-center justify-between mb-3">
                  <span className="text-xs font-medium text-gray-500">{stat.label}</span>
                  <div className={`rounded-lg p-1.5 ${stat.color}`}>
                    <Icon className="h-4 w-4" />
                  </div>
                </div>
                <p className="text-lg font-bold text-gray-900">{stat.value}</p>
              </div>
            );
          })}
        </div>
      )}

      {/* Charts Row */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Daily Orders Bar Chart */}
        <div className="card">
          <div className="flex items-center gap-2 mb-4">
            <ChartBarIcon className="h-5 w-5 text-primary-600" />
            <h2 className="text-base font-semibold text-gray-900">Orders delivered per day</h2>
          </div>
          {orderTrendQuery.isLoading ? (
            <div className="h-64 flex items-center justify-center">
              <LoadingSpinner size="sm" />
            </div>
          ) : orderTrendQuery.isError ? (
            <div className="h-64 flex items-center justify-center">
              <ErrorMessage
                message={orderTrendQuery.error?.message || 'Failed to load trend'}
                onRetry={() => orderTrendQuery.refetch()}
              />
            </div>
          ) : (
            <ResponsiveContainer width="100%" height={260}>
              <BarChart data={orderTrendQuery.data || []}>
                <CartesianGrid strokeDasharray="3 3" stroke="#f0f0f0" />
                <XAxis
                  dataKey="date"
                  tick={{ fontSize: 11 }}
                  tickFormatter={(v) => {
                    try {
                      return formatDate(v);
                    } catch {
                      return v;
                    }
                  }}
                />
                <YAxis tick={{ fontSize: 11 }} allowDecimals={false} />
                <Tooltip
                  labelFormatter={(label) => {
                    try {
                      return formatDate(label as string);
                    } catch {
                      return label as string;
                    }
                  }}
                  contentStyle={{
                    borderRadius: '8px',
                    border: '1px solid #e5e7eb',
                    boxShadow: '0 2px 8px rgba(0,0,0,0.08)',
                  }}
                />
                <Bar dataKey="count" fill="#4f46e5" radius={[4, 4, 0, 0]} name="Orders" />
              </BarChart>
            </ResponsiveContainer>
          )}
        </div>

        {/* Daily Page Views Line Chart */}
        <div className="card">
          <div className="flex items-center gap-2 mb-4">
            <EyeIcon className="h-5 w-5 text-primary-600" />
            <h2 className="text-base font-semibold text-gray-900">Orders placed per day</h2>
          </div>
          {pageViewTrendQuery.isLoading ? (
            <div className="h-64 flex items-center justify-center">
              <LoadingSpinner size="sm" />
            </div>
          ) : pageViewTrendQuery.isError ? (
            <div className="h-64 flex items-center justify-center">
              <ErrorMessage
                message={pageViewTrendQuery.error?.message || 'Failed to load trend'}
                onRetry={() => pageViewTrendQuery.refetch()}
              />
            </div>
          ) : (
            <ResponsiveContainer width="100%" height={260}>
              <LineChart data={pageViewTrendQuery.data || []}>
                <CartesianGrid strokeDasharray="3 3" stroke="#f0f0f0" />
                <XAxis
                  dataKey="date"
                  tick={{ fontSize: 11 }}
                  tickFormatter={(v) => {
                    try {
                      return formatDate(v);
                    } catch {
                      return v;
                    }
                  }}
                />
                <YAxis tick={{ fontSize: 11 }} allowDecimals={false} />
                <Tooltip
                  labelFormatter={(label) => {
                    try {
                      return formatDate(label as string);
                    } catch {
                      return label as string;
                    }
                  }}
                  contentStyle={{
                    borderRadius: '8px',
                    border: '1px solid #e5e7eb',
                    boxShadow: '0 2px 8px rgba(0,0,0,0.08)',
                  }}
                />
                <Line
                  type="monotone"
                  dataKey="count"
                  stroke="#0891b2"
                  strokeWidth={2}
                  dot={false}
                  name="Page Views"
                />
              </LineChart>
            </ResponsiveContainer>
          )}
        </div>
      </div>

      {/* Top Products Table */}
      <div className="card">
        <div className="flex items-center gap-2 mb-4">
          <ChartBarIcon className="h-5 w-5 text-primary-600" />
          <h2 className="text-base font-semibold text-gray-900">Most viewed products <span className="text-xs font-normal text-gray-400">(needs storefront view tracking)</span></h2>
        </div>
        {topProductsQuery.isLoading ? (
          <div className="py-8">
            <LoadingSpinner size="sm" />
          </div>
        ) : topProductsQuery.isError ? (
          <ErrorMessage
            message={topProductsQuery.error?.message || 'Failed to load top products'}
            onRetry={() => topProductsQuery.refetch()}
          />
        ) : topProductsQuery.data && topProductsQuery.data.length > 0 ? (
          <DataTable
            columns={topProductColumns}
            data={topProductsData}
            emptyMessage="No product data available."
          />
        ) : (
          <div className="text-center py-8 text-sm text-gray-500">
            No product data available for the selected date range.
          </div>
        )}
      </div>
    </div>
  );
}
