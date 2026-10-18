import { useState, useCallback, useMemo } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  PlusIcon,
  PencilSquareIcon,
  TrashIcon,
} from '@heroicons/react/24/outline';
import { Dialog, DialogPanel, DialogTitle } from '@headlessui/react';
import { listCategories, createCategory, updateCategory, deleteCategory } from '../../../api/categories';
import { formatDate, cn } from '../../../lib/utils';
import DataTable from '../../../components/common/DataTable';
import LoadingSpinner from '../../../components/common/LoadingSpinner';
import ErrorMessage from '../../../components/common/ErrorMessage';
import EmptyState from '../../../components/common/EmptyState';
import ConfirmDialog from '../../../components/common/ConfirmDialog';
import type { CategoryResponse, CreateCategoryRequest, UpdateCategoryRequest } from '../../../types';

interface CategoryFormData {
  name: string;
  slug: string;
  description: string;
  parent_id: string;
  sort_order: string;
}

const emptyForm: CategoryFormData = {
  name: '',
  slug: '',
  description: '',
  parent_id: '',
  sort_order: '0',
};

function slugify(text: string): string {
  return text
    .toLowerCase()
    .trim()
    .replace(/[^\w\s-]/g, '')
    .replace(/[\s_]+/g, '-')
    .replace(/-+/g, '-')
    .replace(/^-+|-+$/g, '');
}

export default function CategoriesPage() {
  const queryClient = useQueryClient();
  const [dialogOpen, setDialogOpen] = useState(false);
  const [editingId, setEditingId] = useState<string | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<CategoryResponse | null>(null);
  const [form, setForm] = useState<CategoryFormData>(emptyForm);
  const [autoSlug, setAutoSlug] = useState(true);

  const categoriesQuery = useQuery({
    queryKey: ['categories'],
    queryFn: listCategories,
  });

  const createMutation = useMutation({
    mutationFn: createCategory,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['categories'] });
      handleCloseDialog();
    },
  });

  const updateMutation = useMutation({
    mutationFn: ({ id, data }: { id: string; data: UpdateCategoryRequest }) =>
      updateCategory(id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['categories'] });
      handleCloseDialog();
    },
  });

  const deleteMutation = useMutation({
    mutationFn: deleteCategory,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['categories'] });
      setDeleteTarget(null);
    },
  });

  const categories = useMemo(() => categoriesQuery.data?.data ?? [], [categoriesQuery.data]);

  const categoryMap = useMemo(() => {
    const map = new Map<string, string>();
    for (const cat of categories) {
      map.set(cat.id, cat.name);
    }
    return map;
  }, [categories]);

  // Auto-generate slug from name
  const handleNameChange = useCallback((name: string) => {
    setForm((prev) => ({
      ...prev,
      name,
      slug: autoSlug ? slugify(name) : prev.slug,
    }));
  }, [autoSlug]);

  const handleOpenCreate = useCallback(() => {
    setEditingId(null);
    setForm(emptyForm);
    setAutoSlug(true);
    setDialogOpen(true);
  }, []);

  const handleOpenEdit = useCallback((cat: CategoryResponse) => {
    setEditingId(cat.id);
    setForm({
      name: cat.name,
      slug: cat.slug,
      description: cat.description || '',
      parent_id: cat.parent_id || '',
      sort_order: String(cat.sort_order),
    });
    setAutoSlug(false);
    setDialogOpen(true);
  }, []);

  const handleCloseDialog = useCallback(() => {
    setDialogOpen(false);
    setEditingId(null);
    setForm(emptyForm);
    setAutoSlug(true);
  }, []);

  const handleSubmit = useCallback(
    (e: React.FormEvent) => {
      e.preventDefault();
      if (editingId) {
        const data: UpdateCategoryRequest = {
          name: form.name.trim(),
          slug: form.slug.trim(),
          description: form.description.trim() || undefined,
          parent_id: form.parent_id || undefined,
          sort_order: parseInt(form.sort_order, 10) || 0,
        };
        updateMutation.mutate({ id: editingId, data });
      } else {
        const data: CreateCategoryRequest = {
          name: form.name.trim(),
          slug: form.slug.trim(),
          description: form.description.trim() || undefined,
          parent_id: form.parent_id || undefined,
          sort_order: parseInt(form.sort_order, 10) || 0,
        };
        createMutation.mutate(data);
      }
    },
    [editingId, form, createMutation, updateMutation],
  );

  const isSubmitting = createMutation.isPending || updateMutation.isPending;
  const mutationError = createMutation.error || updateMutation.error;

  if (categoriesQuery.isLoading) {
    return <LoadingSpinner />;
  }

  if (categoriesQuery.isError) {
    return (
      <ErrorMessage
        message={categoriesQuery.error?.message || 'Failed to load categories'}
        onRetry={() => categoriesQuery.refetch()}
      />
    );
  }

  const columns = [
    {
      key: 'name',
      header: 'Name',
      render: (row: CategoryResponse) => (
        <div>
          <p className="font-medium text-gray-900">{row.name}</p>
          <p className="text-xs text-gray-500">{row.slug}</p>
        </div>
      ),
    },
    {
      key: 'description',
      header: 'Description',
      render: (row: CategoryResponse) => (
        <span className="text-gray-600 text-sm">{row.description || '—'}</span>
      ),
    },
    {
      key: 'parent_id',
      header: 'Parent',
      render: (row: CategoryResponse) => (
        <span className="text-gray-600">{row.parent_id ? categoryMap.get(row.parent_id) || row.parent_id : '—'}</span>
      ),
    },
    {
      key: 'sort_order',
      header: 'Sort Order',
      render: (row: CategoryResponse) => (
        <span className="text-gray-600">{row.sort_order}</span>
      ),
    },
    {
      key: 'status',
      header: 'Status',
      render: (row: CategoryResponse) => (
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
      key: 'created_at',
      header: 'Created',
      render: (row: CategoryResponse) => (
        <span className="text-gray-500 text-sm">{formatDate(row.created_at)}</span>
      ),
    },
    {
      key: 'actions',
      header: 'Actions',
      render: (row: CategoryResponse) => (
        <div className="flex items-center gap-2">
          <button
            onClick={(e) => {
              e.stopPropagation();
              handleOpenEdit(row);
            }}
            className="inline-flex items-center gap-1 rounded-lg px-2.5 py-1.5 text-xs font-medium text-gray-600 hover:bg-gray-100 transition-colors"
          >
            <PencilSquareIcon className="h-3.5 w-3.5" />
            Edit
          </button>
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
      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Categories</h1>
          <p className="mt-1 text-sm text-gray-500">Manage your product categories</p>
        </div>
        <button
          onClick={handleOpenCreate}
          className="btn-primary inline-flex items-center gap-2"
        >
          <PlusIcon className="h-4 w-4" />
          New Category
        </button>
      </div>

      {/* Mutation Error */}
      {mutationError && (
        <ErrorMessage message={mutationError.message || 'Something went wrong'} />
      )}

      {/* Table */}
      {categories.length === 0 ? (
        <EmptyState
          title="No categories yet"
          description="Get started by creating your first category."
          action={
            <button
              onClick={handleOpenCreate}
              className="btn-primary inline-flex items-center gap-2"
            >
              <PlusIcon className="h-4 w-4" />
              New Category
            </button>
          }
        />
      ) : (
        <DataTable columns={columns} data={categories as unknown as Record<string, unknown>[]} />
      )}

      {/* Create / Edit Dialog */}
      <Dialog open={dialogOpen} onClose={handleCloseDialog} className="relative z-50">
        <div className="fixed inset-0 bg-black/30" aria-hidden="true" />
        <div className="fixed inset-0 flex items-center justify-center p-4">
          <DialogPanel className="w-full max-w-lg rounded-2xl bg-white p-6 shadow-xl max-h-[90vh] overflow-y-auto">
            <DialogTitle className="text-lg font-semibold text-gray-900">
              {editingId ? 'Edit Category' : 'New Category'}
            </DialogTitle>

            {mutationError && dialogOpen && (
              <div className="mt-3 rounded-lg bg-red-50 border border-red-200 p-3">
                <p className="text-sm text-red-700">{mutationError.message}</p>
              </div>
            )}

            <form onSubmit={handleSubmit} className="mt-4 space-y-4">
              <div>
                <label htmlFor="cat-name" className="label-field">
                  Name <span className="text-red-500">*</span>
                </label>
                <input
                  id="cat-name"
                  type="text"
                  required
                  value={form.name}
                  onChange={(e) => handleNameChange(e.target.value)}
                  placeholder="Category name"
                  className="input-field"
                />
              </div>

              <div>
                <label htmlFor="cat-slug" className="label-field">
                  Slug
                </label>
                <div className="flex gap-2">
                  <input
                    id="cat-slug"
                    type="text"
                    required
                    value={form.slug}
                    onChange={(e) => {
                      setForm((prev) => ({ ...prev, slug: e.target.value }));
                      setAutoSlug(false);
                    }}
                    placeholder="category-slug"
                    className="input-field"
                  />
                  <button
                    type="button"
                    onClick={() => {
                      setAutoSlug(true);
                      setForm((prev) => ({ ...prev, slug: slugify(prev.name) }));
                    }}
                    className="btn-secondary btn-sm whitespace-nowrap"
                  >
                    Auto
                  </button>
                </div>
              </div>

              <div>
                <label htmlFor="cat-desc" className="label-field">Description</label>
                <textarea
                  id="cat-desc"
                  rows={3}
                  value={form.description}
                  onChange={(e) => setForm((prev) => ({ ...prev, description: e.target.value }))}
                  placeholder="Optional description"
                  className="input-field"
                />
              </div>

              <div>
                <label htmlFor="cat-parent" className="label-field">Parent Category</label>
                <select
                  id="cat-parent"
                  value={form.parent_id}
                  onChange={(e) => setForm((prev) => ({ ...prev, parent_id: e.target.value }))}
                  className="input-field"
                >
                  <option value="">None (Root Category)</option>
                  {categories
                    .filter((c) => c.id !== editingId)
                    .map((cat) => (
                      <option key={cat.id} value={cat.id}>
                        {cat.name}
                      </option>
                    ))}
                </select>
              </div>

              <div>
                <label htmlFor="cat-sort" className="label-field">Sort Order</label>
                <input
                  id="cat-sort"
                  type="number"
                  value={form.sort_order}
                  onChange={(e) => setForm((prev) => ({ ...prev, sort_order: e.target.value }))}
                  placeholder="0"
                  className="input-field"
                />
              </div>

              <div className="flex items-center justify-end gap-3 pt-2">
                <button type="button" onClick={handleCloseDialog} className="btn-secondary">
                  Cancel
                </button>
                <button type="submit" disabled={isSubmitting} className="btn-primary">
                  {isSubmitting ? 'Saving...' : editingId ? 'Update' : 'Create'}
                </button>
              </div>
            </form>
          </DialogPanel>
        </div>
      </Dialog>

      {/* Delete Confirmation */}
      <ConfirmDialog
        open={deleteTarget !== null}
        onClose={() => setDeleteTarget(null)}
        onConfirm={() => {
          if (deleteTarget) {
            deleteMutation.mutate(deleteTarget.id);
          }
        }}
        title="Delete Category"
        message={`Are you sure you want to delete "${deleteTarget?.name}"? This action cannot be undone.`}
        confirmLabel="Delete"
        danger
      />
    </div>
  );
}
