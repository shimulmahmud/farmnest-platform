import { useState, useCallback } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  PlusIcon,
  PencilSquareIcon,
  PlayIcon,
  PauseIcon,
  ArchiveBoxIcon,
  TagIcon,
} from '@heroicons/react/24/outline';
import { Dialog, DialogPanel, DialogTitle } from '@headlessui/react';
import {
  listPromotions,
  createPromotion,
  updatePromotion,
  transitionPromotionStatus,
} from '../../../api/marketing';
import { formatCurrency, cn } from '../../../lib/utils';
import { listCategories } from '../../../api/categories';
import DataTable from '../../../components/common/DataTable';
import LoadingSpinner from '../../../components/common/LoadingSpinner';
import ErrorMessage from '../../../components/common/ErrorMessage';
import EmptyState from '../../../components/common/EmptyState';
import Pagination from '../../../components/common/Pagination';
import type {
  PromotionResponse,
  CreatePromotionRequest,
} from '../../../types';

function formatPromotionStatus(status: string): string {
  return status
    .replace(/_/g, ' ')
    .toLowerCase()
    .replace(/\b\w/g, (c) => c.toUpperCase());
}

function getStatusBadgeClass(status: string): string {
  const map: Record<string, string> = {
    DRAFT: 'bg-gray-100 text-gray-700',
    ACTIVE: 'bg-green-100 text-green-700',
    PAUSED: 'bg-yellow-100 text-yellow-700',
    ARCHIVED: 'bg-red-100 text-red-700',
  };
  return map[status] || 'bg-gray-100 text-gray-700';
}

function formatDiscount(row: PromotionResponse): string {
  if (row.discount_type === 'PERCENTAGE') {
    return `${row.discount_value}%`;
  }
  if (row.discount_type === 'FREE_SHIPPING') {
    return 'Free delivery';
  }
  return formatCurrency(row.discount_value);
}

interface PromotionForm {
  name: string;
  code: string;
  description: string;
  discount_type: 'PERCENTAGE' | 'FIXED' | 'FREE_SHIPPING';
  discount_value: string;
  min_order_cents: string;
  max_discount_cents: string;
  usage_limit: string;
  start_at: string;
  end_at: string;
  product_ids: string;
  category_ids: string;
}

const emptyForm: PromotionForm = {
  name: '',
  code: '',
  description: '',
  discount_type: 'PERCENTAGE',
  discount_value: '',
  min_order_cents: '',
  max_discount_cents: '',
  usage_limit: '',
  start_at: '',
  end_at: '',
  product_ids: '',
  category_ids: '',
};

function promotionToForm(p: PromotionResponse): PromotionForm {
  return {
    name: p.name,
    code: p.code,
    description: p.description || '',
    discount_type: p.discount_type as 'PERCENTAGE' | 'FIXED' | 'FREE_SHIPPING',
    discount_value: String(p.discount_value),
    min_order_cents: p.min_order_cents != null ? String(p.min_order_cents) : '',
    max_discount_cents: p.max_discount_cents != null ? String(p.max_discount_cents) : '',
    usage_limit: p.usage_limit != null ? String(p.usage_limit) : '',
    start_at: p.start_at ? p.start_at.slice(0, 10) : '',
    end_at: p.end_at ? p.end_at.slice(0, 10) : '',
    product_ids: p.applies_to_product_ids?.join(', ') || '',
    category_ids: p.applies_to_category_ids?.join(', ') || '',
  };
}

function formToRequest(f: PromotionForm): CreatePromotionRequest {
  return {
    name: f.name,
    code: f.code,
    description: f.description || undefined,
    discount_type: f.discount_type,
    discount_value: f.discount_type === 'FREE_SHIPPING' ? 0 : parseInt(f.discount_value, 10) || 0,
    min_order_cents: f.min_order_cents ? parseInt(f.min_order_cents, 10) : undefined,
    max_discount_cents: f.max_discount_cents ? parseInt(f.max_discount_cents, 10) : undefined,
    usage_limit: f.usage_limit ? parseInt(f.usage_limit, 10) : undefined,
    start_at: f.start_at || undefined,
    end_at: f.end_at || undefined,
    applies_to_product_ids: f.product_ids
      ? f.product_ids.split(',').map((s) => s.trim()).filter(Boolean)
      : undefined,
    applies_to_category_ids: f.category_ids
      ? f.category_ids.split(',').map((s) => s.trim()).filter(Boolean)
      : undefined,
  };
}

export default function PromotionsPage() {
  const queryClient = useQueryClient();
  const categoriesQuery = useQuery({ queryKey: ['categories'], queryFn: listCategories });
  const [page, setPage] = useState(1);

  const [formDialogOpen, setFormDialogOpen] = useState(false);
  const [editingPromotion, setEditingPromotion] = useState<PromotionResponse | null>(null);
  const [form, setForm] = useState<PromotionForm>(emptyForm);

  const promotionsQuery = useQuery({
    queryKey: ['admin', 'promotions', page],
    queryFn: () => listPromotions(page, 20),
  });

  const createMutation = useMutation({
    mutationFn: (data: CreatePromotionRequest) => createPromotion(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin', 'promotions'] });
      setFormDialogOpen(false);
      setForm(emptyForm);
    },
  });

  const updateMutation = useMutation({
    mutationFn: ({ id, data }: { id: string; data: CreatePromotionRequest }) =>
      updatePromotion(id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin', 'promotions'] });
      setFormDialogOpen(false);
      setEditingPromotion(null);
      setForm(emptyForm);
    },
  });

  const transitionMutation = useMutation({
    mutationFn: ({ id, status }: { id: string; status: string }) =>
      transitionPromotionStatus(id, { status }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin', 'promotions'] });
    },
  });

  const promotions = promotionsQuery.data?.data ?? [];
  const pagination = promotionsQuery.data?.pagination;

  const handleOpenCreate = useCallback(() => {
    setEditingPromotion(null);
    setForm(emptyForm);
    setFormDialogOpen(true);
  }, []);

  const handleOpenEdit = useCallback((promo: PromotionResponse) => {
    setEditingPromotion(promo);
    setForm(promotionToForm(promo));
    setFormDialogOpen(true);
  }, []);

  const handleCloseFormDialog = useCallback(() => {
    setFormDialogOpen(false);
    setEditingPromotion(null);
    setForm(emptyForm);
  }, []);

  const handleSubmitForm = useCallback(
    (e: React.FormEvent) => {
      e.preventDefault();
      const payload = formToRequest(form);
      if (editingPromotion) {
        updateMutation.mutate({ id: editingPromotion.id, data: payload });
      } else {
        createMutation.mutate(payload);
      }
    },
    [form, editingPromotion, createMutation, updateMutation],
  );

  const getNextTransitions = (
    status: string,
  ): { label: string; value: string; icon: typeof PlayIcon }[] => {
    switch (status) {
      case 'DRAFT':
        return [{ label: 'Activate', value: 'ACTIVE', icon: PlayIcon }];
      case 'ACTIVE':
        return [
          { label: 'Pause', value: 'PAUSED', icon: PauseIcon },
          { label: 'Archive', value: 'ARCHIVED', icon: ArchiveBoxIcon },
        ];
      case 'PAUSED':
        return [
          { label: 'Resume', value: 'ACTIVE', icon: PlayIcon },
          { label: 'Archive', value: 'ARCHIVED', icon: ArchiveBoxIcon },
        ];
      default:
        return [];
    }
  };

  const isFormPending = createMutation.isPending || updateMutation.isPending;
  const formError = createMutation.error?.message || updateMutation.error?.message;

  if (promotionsQuery.isLoading) return <LoadingSpinner />;

  if (promotionsQuery.isError) {
    return (
      <ErrorMessage
        message={promotionsQuery.error?.message || 'Failed to load promotions'}
        onRetry={() => promotionsQuery.refetch()}
      />
    );
  }

  const columns = [
    {
      key: 'name',
      header: 'Name',
      render: (row: PromotionResponse) => (
        <span className="font-medium text-gray-900">{row.name}</span>
      ),
    },
    {
      key: 'code',
      header: 'Code',
      render: (row: PromotionResponse) => (
        <span className="font-mono text-sm bg-gray-100 px-2 py-0.5 rounded">{row.code}</span>
      ),
    },
    {
      key: 'discount',
      header: 'Discount',
      render: (row: PromotionResponse) => (
        <span className="text-sm font-semibold text-green-700">{formatDiscount(row)}</span>
      ),
    },
    {
      key: 'status',
      header: 'Status',
      render: (row: PromotionResponse) => (
        <span
          className={cn(
            'inline-flex rounded-full px-2.5 py-0.5 text-xs font-medium',
            getStatusBadgeClass(row.status),
          )}
        >
          {formatPromotionStatus(row.status)}
        </span>
      ),
    },
    {
      key: 'usage',
      header: 'Usage',
      render: (row: PromotionResponse) => (
        <span className="text-sm text-gray-700">
          {row.usage_count}
          {row.usage_limit != null ? ` / ${row.usage_limit}` : ' / ∞'}
        </span>
      ),
    },
    {
      key: 'actions',
      header: 'Actions',
      render: (row: PromotionResponse) => {
        const transitions = getNextTransitions(row.status);
        return (
          <div className="flex items-center gap-1">
            <button
              onClick={(e) => {
                e.stopPropagation();
                handleOpenEdit(row);
              }}
              className="inline-flex items-center gap-1 rounded-lg px-2 py-1.5 text-xs font-medium text-primary-600 hover:bg-primary-50 transition-colors"
            >
              <PencilSquareIcon className="h-3.5 w-3.5" />
              Edit
            </button>
            {transitions.map((t) => {
              const Icon = t.icon;
              return (
                <button
                  key={t.value}
                  onClick={(e) => {
                    e.stopPropagation();
                    transitionMutation.mutate({ id: row.id, status: t.value });
                  }}
                  disabled={transitionMutation.isPending}
                  className={cn(
                    'inline-flex items-center gap-1 rounded-lg px-2 py-1.5 text-xs font-medium transition-colors',
                    t.value === 'ACTIVE' && 'text-green-600 hover:bg-green-50',
                    t.value === 'PAUSED' && 'text-yellow-600 hover:bg-yellow-50',
                    t.value === 'ARCHIVED' && 'text-gray-600 hover:bg-gray-100',
                  )}
                >
                  <Icon className="h-3.5 w-3.5" />
                  {t.label}
                </button>
              );
            })}
          </div>
        );
      },
    },
  ];

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Promotions</h1>
          <p className="mt-1 text-sm text-gray-500">
            Create and manage discount codes and promotions
          </p>
        </div>
        <button onClick={handleOpenCreate} className="btn-primary btn-sm">
          <PlusIcon className="h-4 w-4 mr-1" />
          Create Promotion
        </button>
      </div>

      {promotions.length === 0 ? (
        <EmptyState
          icon={<TagIcon className="h-10 w-10 text-gray-400" />}
          title="No promotions found"
          description="Create your first promotion to offer discounts."
          action={
            <button onClick={handleOpenCreate} className="btn-primary btn-sm">
              <PlusIcon className="h-4 w-4 mr-1" />
              Create Promotion
            </button>
          }
        />
      ) : (
        <>
          <DataTable
            columns={columns}
            data={promotions as unknown as Record<string, unknown>[]}
            emptyMessage="No promotions found."
          />
          {pagination && (
            <div className="mt-4">
              <Pagination
                page={pagination.page}
                totalPages={pagination.total_pages}
                onPageChange={setPage}
              />
            </div>
          )}
        </>
      )}

      {/* Create/Edit Promotion Dialog */}
      <Dialog open={formDialogOpen} onClose={handleCloseFormDialog} className="relative z-50">
        <div className="fixed inset-0 bg-black/30" aria-hidden="true" />
        <div className="fixed inset-0 flex items-center justify-center p-4">
          <DialogPanel className="w-full max-w-lg rounded-2xl bg-white p-6 shadow-xl max-h-[90vh] overflow-y-auto">
            <DialogTitle className="text-lg font-semibold text-gray-900">
              {editingPromotion ? 'Edit Promotion' : 'Create Promotion'}
            </DialogTitle>

            {formError && (
              <div className="mt-3 rounded-lg bg-red-50 border border-red-200 p-3">
                <p className="text-sm text-red-700">{formError}</p>
              </div>
            )}

            <form onSubmit={handleSubmitForm} className="mt-4 space-y-4">
              <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                <div>
                  <label htmlFor="promo-name" className="label-field">
                    Name <span className="text-red-500">*</span>
                  </label>
                  <input
                    id="promo-name"
                    type="text"
                    required
                    value={form.name}
                    onChange={(e) => setForm((f) => ({ ...f, name: e.target.value }))}
                    className="input-field"
                    placeholder="e.g. Summer Sale"
                  />
                </div>
                <div>
                  <label htmlFor="promo-code" className="label-field">
                    Code <span className="text-red-500">*</span>
                  </label>
                  <input
                    id="promo-code"
                    type="text"
                    required
                    value={form.code}
                    onChange={(e) => setForm((f) => ({ ...f, code: e.target.value.toUpperCase() }))}
                    className="input-field"
                    placeholder="e.g. SUMMER20"
                  />
                </div>
              </div>

              <div>
                <label htmlFor="promo-desc" className="label-field">
                  Description
                </label>
                <textarea
                  id="promo-desc"
                  rows={2}
                  value={form.description}
                  onChange={(e) => setForm((f) => ({ ...f, description: e.target.value }))}
                  className="input-field"
                  placeholder="Brief description..."
                />
              </div>

              <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                <div>
                  <label htmlFor="discount-type" className="label-field">
                    Discount Type <span className="text-red-500">*</span>
                  </label>
                  <select
                    id="discount-type"
                    required
                    value={form.discount_type}
                    onChange={(e) =>
                      setForm((f) => ({
                        ...f,
                        discount_type: e.target.value as 'PERCENTAGE' | 'FIXED' | 'FREE_SHIPPING',
                      }))
                    }
                    className="input-field"
                  >
                    <option value="PERCENTAGE">Percentage (%)</option>
                    <option value="FIXED">Fixed amount (৳)</option>
                    <option value="FREE_SHIPPING">Free delivery</option>
                  </select>
                </div>
                <div>
                  <label htmlFor="discount-value" className="label-field">
                    {form.discount_type === 'PERCENTAGE' ? 'Percent off' : form.discount_type === 'FIXED' ? 'Amount off (৳)' : 'Value'} {form.discount_type !== 'FREE_SHIPPING' && <span className="text-red-500">*</span>}
                  </label>
                  <input
                    id="discount-value"
                    type="number"
                    required={form.discount_type !== 'FREE_SHIPPING'}
                    disabled={form.discount_type === 'FREE_SHIPPING'}
                    min="0"
                    step="1"
                    value={form.discount_value}
                    onChange={(e) => setForm((f) => ({ ...f, discount_value: e.target.value }))}
                    className="input-field"
                    placeholder={form.discount_type === 'PERCENTAGE' ? 'e.g. 10' : 'e.g. 500'}
                  />
                </div>
              </div>

              <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                <div>
                  <label htmlFor="min-order" className="label-field">
                    Minimum order (৳)
                  </label>
                  <input
                    id="min-order"
                    type="number"
                    min="0"
                    value={form.min_order_cents}
                    onChange={(e) => setForm((f) => ({ ...f, min_order_cents: e.target.value }))}
                    className="input-field"
                    placeholder="e.g. 5000 ($50.00)"
                  />
                </div>
                <div>
                  <label htmlFor="max-discount" className="label-field">
                    Maximum discount (৳, percent coupons)
                  </label>
                  <input
                    id="max-discount"
                    type="number"
                    min="0"
                    value={form.max_discount_cents}
                    onChange={(e) => setForm((f) => ({ ...f, max_discount_cents: e.target.value }))}
                    className="input-field"
                    placeholder="e.g. 2000 ($20.00)"
                  />
                </div>
              </div>

              <div>
                <label htmlFor="usage-limit" className="label-field">
                  Usage Limit
                </label>
                <input
                  id="usage-limit"
                  type="number"
                  min="0"
                  value={form.usage_limit}
                  onChange={(e) => setForm((f) => ({ ...f, usage_limit: e.target.value }))}
                  className="input-field"
                  placeholder="Leave empty for unlimited"
                />
              </div>

              <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                <div>
                  <label htmlFor="start-at" className="label-field">
                    Start Date
                  </label>
                  <input
                    id="start-at"
                    type="date"
                    value={form.start_at}
                    onChange={(e) => setForm((f) => ({ ...f, start_at: e.target.value }))}
                    className="input-field"
                  />
                </div>
                <div>
                  <label htmlFor="end-at" className="label-field">
                    End Date
                  </label>
                  <input
                    id="end-at"
                    type="date"
                    value={form.end_at}
                    onChange={(e) => setForm((f) => ({ ...f, end_at: e.target.value }))}
                    className="input-field"
                  />
                </div>
              </div>

              <div>
                <label htmlFor="product-ids" className="label-field">
                  Only for these product IDs <span className="text-xs font-normal text-gray-400">(optional)</span>
                </label>
                <input
                  id="product-ids"
                  type="text"
                  value={form.product_ids}
                  onChange={(e) => setForm((f) => ({ ...f, product_ids: e.target.value }))}
                  className="input-field"
                  placeholder="Comma-separated, leave empty for all"
                />
              </div>

              <div>
                <label htmlFor="category-ids" className="label-field">
                  Only for these categories <span className="text-xs font-normal text-gray-400">(none ticked = whole store)</span>
                </label>
                <div className="flex flex-wrap gap-2">
                  {(categoriesQuery.data?.data ?? []).map((c) => {
                    const ids = form.category_ids.split(',').map((s) => s.trim()).filter(Boolean);
                    const on = ids.includes(c.id);
                    return (
                      <label key={c.id} className={cn('cursor-pointer rounded-full border px-3 py-1 text-xs font-medium', on ? 'border-primary-600 bg-primary-50 text-primary-700' : 'border-gray-300 text-gray-600')}>
                        <input
                          type="checkbox"
                          className="sr-only"
                          checked={on}
                          onChange={() => {
                            const next = on ? ids.filter((x) => x !== c.id) : [...ids, c.id];
                            setForm((f) => ({ ...f, category_ids: next.join(', ') }));
                          }}
                        />
                        {c.name}
                      </label>
                    );
                  })}
                </div>
              </div>

              <div className="flex items-center justify-end gap-3 pt-2">
                <button
                  type="button"
                  onClick={handleCloseFormDialog}
                  className="btn-secondary btn-sm"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  disabled={isFormPending || !form.name || !form.code}
                  className="btn-primary btn-sm"
                >
                  {isFormPending
                    ? 'Saving...'
                    : editingPromotion
                      ? 'Update Promotion'
                      : 'Create Promotion'}
                </button>
              </div>
            </form>
          </DialogPanel>
        </div>
      </Dialog>
    </div>
  );
}
