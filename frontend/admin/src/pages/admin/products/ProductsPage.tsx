import { useState, useCallback } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Link, useSearchParams } from 'react-router-dom';
import {
  PlusIcon,
  MagnifyingGlassIcon,
  PencilSquareIcon,
  TrashIcon,
} from '@heroicons/react/24/outline';
import { adminListProducts, deleteProduct } from '../../../api/products';
import { listCategories } from '../../../api/categories';
import { formatCurrency, cn } from '../../../lib/utils';
import DataTable from '../../../components/common/DataTable';
import LoadingSpinner from '../../../components/common/LoadingSpinner';
import ErrorMessage from '../../../components/common/ErrorMessage';
import EmptyState from '../../../components/common/EmptyState';
import ConfirmDialog from '../../../components/common/ConfirmDialog';
import Pagination from '../../../components/common/Pagination';
import type { ProductResponse } from '../../../types';

const STATUS_FILTERS = ['ALL', 'ACTIVE', 'DRAFT', 'ARCHIVED'] as const;

type StatusFilter = (typeof STATUS_FILTERS)[number];


/** Cost, price and margin of the variants that are on sale. */
function priceStats(row: ProductResponse) {
  const live = row.variants.filter((v) => v.status === 'ACTIVE');
  return {
    costs: live.map((v) => v.cost_price),
    prices: live.map((v) => v.price),
    margins: live.map((v) => (v.price > 0 ? Math.round(((v.price - v.cost_price) / v.price) * 100) : 0)),
  };
}

function rangeLabel(values: number[]): string {
  if (values.length === 0) return '—';
  const lo = Math.min(...values);
  const hi = Math.max(...values);
  return lo === hi ? formatCurrency(lo) : `${formatCurrency(lo)} – ${formatCurrency(hi)}`;
}

/** Tooltip: one line per variant with buying price, selling price and profit. */
function variantBreakdown(row: ProductResponse): string {
  return row.variants
    .filter((v) => v.status === 'ACTIVE')
    .map((v) => `${v.sku}: buy ${formatCurrency(v.cost_price)} → sell ${formatCurrency(v.price)} (profit ${formatCurrency(v.price - v.cost_price)})`)
    .join('\n');
}

export default function ProductsPage() {
  const queryClient = useQueryClient();
  const [searchParams, setSearchParams] = useSearchParams();

  const page = Number(searchParams.get('page')) || 1;
  const statusFilter = (searchParams.get('status') as StatusFilter) || 'ALL';
  const searchQuery = searchParams.get('q') || '';

  const [searchInput, setSearchInput] = useState(searchQuery);
  const [deleteTarget, setDeleteTarget] = useState<ProductResponse | null>(null);

  const productsQuery = useQuery({
    queryKey: ['admin', 'products', page, statusFilter, searchQuery],
    queryFn: () =>
      adminListProducts(page, 20, {
        status: statusFilter === 'ALL' ? undefined : statusFilter,
        search: searchQuery || undefined,
      }),
  });

  const categoriesQuery = useQuery({ queryKey: ['categories'], queryFn: listCategories });
  const categoryName = (id: string) =>
    categoriesQuery.data?.data.find((c) => c.id === id)?.name ?? '—';

  const deleteMutation = useMutation({
    mutationFn: deleteProduct,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin', 'products'] });
      setDeleteTarget(null);
    },
  });

  const handleSearch = useCallback(
    (e: React.FormEvent) => {
      e.preventDefault();
      const params = new URLSearchParams(searchParams);
      if (searchInput.trim()) {
        params.set('q', searchInput.trim());
      } else {
        params.delete('q');
      }
      params.set('page', '1');
      setSearchParams(params);
    },
    [searchInput, searchParams, setSearchParams],
  );

  const handleStatusFilter = useCallback(
    (status: StatusFilter) => {
      const params = new URLSearchParams(searchParams);
      if (status === 'ALL') {
        params.delete('status');
      } else {
        params.set('status', status);
      }
      params.set('page', '1');
      setSearchParams(params);
    },
    [searchParams, setSearchParams],
  );

  const handlePageChange = useCallback(
    (newPage: number) => {
      const params = new URLSearchParams(searchParams);
      params.set('page', String(newPage));
      setSearchParams(params);
    },
    [searchParams, setSearchParams],
  );

  if (productsQuery.isLoading) {
    return <LoadingSpinner />;
  }

  if (productsQuery.isError) {
    return (
      <ErrorMessage
        message={productsQuery.error?.message || 'Failed to load products'}
        onRetry={() => productsQuery.refetch()}
      />
    );
  }

  const products = productsQuery.data!.data;
  const pagination = productsQuery.data!.pagination;

  const columns = [
    {
      key: 'name',
      header: 'Name',
      render: (row: ProductResponse) => (
        <div>
          <p className="font-medium text-gray-900">{row.name}</p>
          <p className="text-xs text-gray-500">{row.slug}</p>
        </div>
      ),
    },
    {
      key: 'category_id',
      header: 'Category',
      render: (row: ProductResponse) => (
        <span className="text-gray-600">{categoryName(row.category_id)}</span>
      ),
    },
    {
      key: 'brand',
      header: 'Brand',
      render: (row: ProductResponse) => (
        <span className="text-gray-600">{row.brand || '—'}</span>
      ),
    },
    {
      key: 'variants',
      header: 'Variants',
      render: (row: ProductResponse) => (
        <span className="badge-gray">{row.variants.length}</span>
      ),
    },
    {
      key: 'buying_price',
      header: 'Buying price',
      render: (row: ProductResponse) => {
        const { costs } = priceStats(row);
        return (
          <span className="text-gray-700" title={variantBreakdown(row)}>
            {rangeLabel(costs)}
          </span>
        );
      },
    },
    {
      key: 'selling_price',
      header: 'Selling price',
      render: (row: ProductResponse) => {
        const { prices } = priceStats(row);
        return (
          <span className="font-medium text-gray-900" title={variantBreakdown(row)}>
            {rangeLabel(prices)}
          </span>
        );
      },
    },
    {
      key: 'margin',
      header: 'Margin',
      render: (row: ProductResponse) => {
        const { margins } = priceStats(row);
        if (margins.length === 0) return <span className="text-gray-400">—</span>;
        const lo = Math.min(...margins);
        const hi = Math.max(...margins);
        const tone = lo < 0 ? 'badge-red' : lo < 15 ? 'badge-yellow' : 'badge-green';
        return (
          <span className={tone} title={variantBreakdown(row)}>
            {lo === hi ? `${lo}%` : `${lo}% – ${hi}%`}
          </span>
        );
      },
    },
    {
      key: 'total_available',
      header: 'Stock',
      render: (row: ProductResponse) => {
        const n = row.total_available ?? 0;
        return (
          <span className={n > 10 ? 'badge-green' : n > 0 ? 'badge-yellow' : 'badge-red'}>
            {n > 0 ? `${n} ${row.unit || 'pcs'}` : 'Out of stock'}
          </span>
        );
      },
    },
    {
      key: 'status',
      header: 'Status',
      render: (row: ProductResponse) => (
        <span className={cn(
          row.status === 'ACTIVE' && 'badge-green',
          row.status === 'DRAFT' && 'badge-yellow',
          row.status === 'ARCHIVED' && 'badge-gray',
        )}>
          {row.status.charAt(0) + row.status.slice(1).toLowerCase()}
        </span>
      ),
    },
    {
      key: 'actions',
      header: 'Actions',
      render: (row: ProductResponse) => (
        <div className="flex items-center gap-2">
          <Link
            to={`/admin/products/${row.id}/edit`}
            className="inline-flex items-center gap-1 rounded-lg px-2.5 py-1.5 text-xs font-medium text-gray-600 hover:bg-gray-100 transition-colors"
          >
            <PencilSquareIcon className="h-3.5 w-3.5" />
            Edit
          </Link>
          <button
            onClick={(e) => {
              e.stopPropagation();
              setDeleteTarget(row);
            }}
            className="inline-flex items-center gap-1 rounded-lg px-2.5 py-1.5 text-xs font-medium text-red-600 hover:bg-red-50 transition-colors"
          >
            <TrashIcon className="h-3.5 w-3.5" />
            Delete
          </button>
        </div>
      ),
    },
  ];

  return (
    <div className="space-y-6">
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Products</h1>
          <p className="mt-1 text-sm text-gray-500">
            Manage your product catalog
          </p>
        </div>
        <Link to="/admin/products/new" className="btn-primary inline-flex items-center gap-2">
          <PlusIcon className="h-4 w-4" />
          New Product
        </Link>
      </div>

      {/* Search Bar */}
      <form onSubmit={handleSearch} className="flex gap-3">
        <div className="relative flex-1">
          <MagnifyingGlassIcon className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-gray-400" />
          <input
            type="text"
            value={searchInput}
            onChange={(e) => setSearchInput(e.target.value)}
            placeholder="Search products..."
            className="input-field pl-10"
          />
        </div>
        <button type="submit" className="btn-primary">
          Search
        </button>
      </form>

      {/* Status Filter Tabs */}
      <div className="flex flex-wrap gap-2">
        {STATUS_FILTERS.map((status) => (
          <button
            key={status}
            onClick={() => handleStatusFilter(status)}
            className={cn(
              'rounded-lg px-3 py-1.5 text-xs font-medium transition-colors',
              statusFilter === status
                ? 'bg-primary-600 text-white'
                : 'bg-white text-gray-600 border border-gray-300 hover:bg-gray-50',
            )}
          >
            {status.charAt(0) + status.slice(1).toLowerCase()}
          </button>
        ))}
      </div>

      {/* Products Table */}
      {products.length === 0 ? (
        <EmptyState
          title="No products found"
          description={
            searchQuery
              ? 'Try adjusting your search or filter.'
              : 'Get started by creating your first product.'
          }
          action={
            !searchQuery ? (
              <Link to="/admin/products/new" className="btn-primary inline-flex items-center gap-2">
                <PlusIcon className="h-4 w-4" />
                New Product
              </Link>
            ) : undefined
          }
        />
      ) : (
        <>
          <DataTable columns={columns} data={products as unknown as Record<string, unknown>[]} />
          <div className="mt-4">
            <Pagination
              page={pagination.page}
              totalPages={pagination.total_pages}
              onPageChange={handlePageChange}
            />
          </div>
        </>
      )}

      {/* Delete Confirmation */}
      <ConfirmDialog
        open={deleteTarget !== null}
        onClose={() => setDeleteTarget(null)}
        onConfirm={() => {
          if (deleteTarget) {
            deleteMutation.mutate(deleteTarget.id);
          }
        }}
        title="Delete Product"
        message={`Are you sure you want to delete "${deleteTarget?.name}"? This action cannot be undone.`}
        confirmLabel="Delete"
        danger
      />
    </div>
  );
}
