import { useState, useCallback } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  PlusIcon,
  PencilSquareIcon,
  TrashIcon,
  UserGroupIcon,
} from '@heroicons/react/24/outline';
import { Dialog, DialogPanel, DialogTitle } from '@headlessui/react';
import {
  listSegments,
  createSegment,
  updateSegment,
  deleteSegment,
} from '../../../api/marketing';
import { cn, truncate } from '../../../lib/utils';
import DataTable from '../../../components/common/DataTable';
import LoadingSpinner from '../../../components/common/LoadingSpinner';
import ErrorMessage from '../../../components/common/ErrorMessage';
import EmptyState from '../../../components/common/EmptyState';
import Pagination from '../../../components/common/Pagination';
import ConfirmDialog from '../../../components/common/ConfirmDialog';
import type {
  SegmentResponse,
  SegmentRule,
  CreateSegmentRequest,
} from '../../../types';

const SEGMENT_TYPES = ['DYNAMIC', 'STATIC'] as const;
const OPERATORS = ['eq', 'neq', 'gt', 'lt', 'contains'] as const;

type SegmentType = (typeof SEGMENT_TYPES)[number];
type Operator = (typeof OPERATORS)[number];

interface RuleRow {
  field: string;
  operator: Operator;
  value: string;
}

interface SegmentForm {
  name: string;
  type: SegmentType;
  description: string;
  rules: RuleRow[];
}

const emptyRule = (): RuleRow => ({ field: '', operator: 'eq', value: '' });

const emptyForm: SegmentForm = {
  name: '',
  type: 'DYNAMIC',
  description: '',
  rules: [emptyRule()],
};

function segmentToForm(s: SegmentResponse): SegmentForm {
  return {
    name: s.name,
    type: s.type as SegmentType,
    description: s.description || '',
    rules: s.rules.map((r) => ({
      field: r.field,
      operator: (r.operator || 'eq') as Operator,
      value: String(r.value ?? ''),
    })),
  };
}

function formToRules(f: SegmentForm): SegmentRule[] {
  return f.rules
    .filter((r) => r.field.trim())
    .map((r) => ({
      field: r.field.trim(),
      operator: r.operator,
      value: r.value.trim() || '',
    }));
}

function formatSegmentType(type: string): string {
  return type.charAt(0).toUpperCase() + type.slice(1).toLowerCase();
}

function formatRulesPreview(rules: SegmentRule[]): string {
  if (rules.length === 0) return 'No rules';
  return rules
    .slice(0, 2)
    .map(
      (r) =>
        `${r.field} ${r.operator} ${typeof r.value === 'string' && r.value.length > 15 ? r.value.slice(0, 15) + '...' : r.value}`,
    )
    .join(', ') + (rules.length > 2 ? ` +${rules.length - 2} more` : '');
}

export default function SegmentsPage() {
  const queryClient = useQueryClient();
  const [page, setPage] = useState(1);

  const [formDialogOpen, setFormDialogOpen] = useState(false);
  const [editingSegment, setEditingSegment] = useState<SegmentResponse | null>(null);
  const [form, setForm] = useState<SegmentForm>(emptyForm);

  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [deleteTarget, setDeleteTarget] = useState<SegmentResponse | null>(null);

  const segmentsQuery = useQuery({
    queryKey: ['admin', 'segments', page],
    queryFn: () => listSegments(page, 20),
  });

  const createMutation = useMutation({
    mutationFn: (data: CreateSegmentRequest) => createSegment(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin', 'segments'] });
      setFormDialogOpen(false);
      setForm(emptyForm);
    },
  });

  const updateMutation = useMutation({
    mutationFn: ({ id, data }: { id: string; data: CreateSegmentRequest }) =>
      updateSegment(id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin', 'segments'] });
      setFormDialogOpen(false);
      setEditingSegment(null);
      setForm(emptyForm);
    },
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => deleteSegment(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin', 'segments'] });
    },
  });

  const segments = segmentsQuery.data?.data ?? [];
  const pagination = segmentsQuery.data?.pagination;

  const handleOpenCreate = useCallback(() => {
    setEditingSegment(null);
    setForm(emptyForm);
    setFormDialogOpen(true);
  }, []);

  const handleOpenEdit = useCallback((segment: SegmentResponse) => {
    setEditingSegment(segment);
    setForm(segmentToForm(segment));
    setFormDialogOpen(true);
  }, []);

  const handleCloseFormDialog = useCallback(() => {
    setFormDialogOpen(false);
    setEditingSegment(null);
    setForm(emptyForm);
  }, []);

  const handleSubmitForm = useCallback(
    (e: React.FormEvent) => {
      e.preventDefault();
      const payload: CreateSegmentRequest = {
        name: form.name,
        type: form.type,
        description: form.description || undefined,
        rules: formToRules(form),
      };
      if (editingSegment) {
        updateMutation.mutate({ id: editingSegment.id, data: payload });
      } else {
        createMutation.mutate(payload);
      }
    },
    [form, editingSegment, createMutation, updateMutation],
  );

  const handleAddRule = useCallback(() => {
    setForm((prev) => ({ ...prev, rules: [...prev.rules, emptyRule()] }));
  }, []);

  const handleRemoveRule = useCallback((index: number) => {
    setForm((prev) => ({
      ...prev,
      rules: prev.rules.filter((_, i) => i !== index),
    }));
  }, []);

  const handleRuleChange = useCallback(
    (index: number, field: keyof RuleRow, value: string) => {
      setForm((prev) => {
        const rules = [...prev.rules];
        rules[index] = { ...rules[index], [field]: value };
        return { ...prev, rules };
      });
    },
    [],
  );

  const handleOpenDelete = useCallback((segment: SegmentResponse) => {
    setDeleteTarget(segment);
    setDeleteDialogOpen(true);
  }, []);

  const handleConfirmDelete = useCallback(() => {
    if (deleteTarget) {
      deleteMutation.mutate(deleteTarget.id);
    }
  }, [deleteTarget, deleteMutation]);

  const isFormPending = createMutation.isPending || updateMutation.isPending;
  const formError = createMutation.error?.message || updateMutation.error?.message;

  if (segmentsQuery.isLoading) return <LoadingSpinner />;

  if (segmentsQuery.isError) {
    return (
      <ErrorMessage
        message={segmentsQuery.error?.message || 'Failed to load segments'}
        onRetry={() => segmentsQuery.refetch()}
      />
    );
  }

  const columns = [
    {
      key: 'name',
      header: 'Name',
      render: (row: SegmentResponse) => (
        <div>
          <span className="font-medium text-gray-900">{row.name}</span>
          {row.description && (
            <p className="text-xs text-gray-500 mt-0.5">{truncate(row.description, 40)}</p>
          )}
        </div>
      ),
    },
    {
      key: 'type',
      header: 'Type',
      render: (row: SegmentResponse) => (
        <span
          className={cn(
            'inline-flex rounded-full px-2.5 py-0.5 text-xs font-medium',
            row.type === 'DYNAMIC'
              ? 'bg-blue-100 text-blue-700'
              : 'bg-purple-100 text-purple-700',
          )}
        >
          {formatSegmentType(row.type)}
        </span>
      ),
    },
    {
      key: 'customer_count',
      header: 'Customers',
      render: (row: SegmentResponse) => (
        <span className="text-sm font-medium text-gray-900">
          {row.customer_count.toLocaleString()}
        </span>
      ),
    },
    {
      key: 'rules',
      header: 'Rules',
      render: (row: SegmentResponse) => (
        <span className="text-xs text-gray-600 font-mono">
          {formatRulesPreview(row.rules)}
        </span>
      ),
    },
    {
      key: 'actions',
      header: 'Actions',
      render: (row: SegmentResponse) => (
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
          <button
            onClick={(e) => {
              e.stopPropagation();
              handleOpenDelete(row);
            }}
            className="inline-flex items-center gap-1 rounded-lg px-2 py-1.5 text-xs font-medium text-red-600 hover:bg-red-50 transition-colors"
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
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Segments</h1>
          <p className="mt-1 text-sm text-gray-500">
            Create and manage customer segments for targeting
          </p>
        </div>
        <button onClick={handleOpenCreate} className="btn-primary btn-sm">
          <PlusIcon className="h-4 w-4 mr-1" />
          Create Segment
        </button>
      </div>

      {segments.length === 0 ? (
        <EmptyState
          icon={<UserGroupIcon className="h-10 w-10 text-gray-400" />}
          title="No segments found"
          description="Create your first customer segment."
          action={
            <button onClick={handleOpenCreate} className="btn-primary btn-sm">
              <PlusIcon className="h-4 w-4 mr-1" />
              Create Segment
            </button>
          }
        />
      ) : (
        <>
          <DataTable
            columns={columns}
            data={segments as any[]}
            emptyMessage="No segments found."
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

      {/* Create/Edit Segment Dialog */}
      <Dialog open={formDialogOpen} onClose={handleCloseFormDialog} className="relative z-50">
        <div className="fixed inset-0 bg-black/30" aria-hidden="true" />
        <div className="fixed inset-0 flex items-center justify-center p-4">
          <DialogPanel className="w-full max-w-lg rounded-2xl bg-white p-6 shadow-xl max-h-[90vh] overflow-y-auto">
            <DialogTitle className="text-lg font-semibold text-gray-900">
              {editingSegment ? 'Edit Segment' : 'Create Segment'}
            </DialogTitle>

            {formError && (
              <div className="mt-3 rounded-lg bg-red-50 border border-red-200 p-3">
                <p className="text-sm text-red-700">{formError}</p>
              </div>
            )}

            <form onSubmit={handleSubmitForm} className="mt-4 space-y-4">
              <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                <div>
                  <label htmlFor="segment-name" className="label-field">
                    Name <span className="text-red-500">*</span>
                  </label>
                  <input
                    id="segment-name"
                    type="text"
                    required
                    value={form.name}
                    onChange={(e) => setForm((f) => ({ ...f, name: e.target.value }))}
                    className="input-field"
                    placeholder="e.g. High Spenders"
                  />
                </div>
                <div>
                  <label htmlFor="segment-type" className="label-field">
                    Type <span className="text-red-500">*</span>
                  </label>
                  <select
                    id="segment-type"
                    required
                    value={form.type}
                    onChange={(e) =>
                      setForm((f) => ({ ...f, type: e.target.value as SegmentType }))
                    }
                    className="input-field"
                  >
                    {SEGMENT_TYPES.map((t) => (
                      <option key={t} value={t}>
                        {formatSegmentType(t)}
                      </option>
                    ))}
                  </select>
                </div>
              </div>

              <div>
                <label htmlFor="segment-desc" className="label-field">
                  Description
                </label>
                <textarea
                  id="segment-desc"
                  rows={2}
                  value={form.description}
                  onChange={(e) => setForm((f) => ({ ...f, description: e.target.value }))}
                  className="input-field"
                  placeholder="Describe this segment..."
                />
              </div>

              <div>
                <div className="flex items-center justify-between mb-2">
                  <label className="label-field mb-0">Rules</label>
                  <button
                    type="button"
                    onClick={handleAddRule}
                    className="inline-flex items-center gap-1 rounded-lg px-2 py-1 text-xs font-medium text-primary-600 hover:bg-primary-50 transition-colors"
                  >
                    <PlusIcon className="h-3.5 w-3.5" />
                    Add Rule
                  </button>
                </div>
                <div className="space-y-2 max-h-60 overflow-y-auto">
                  {form.rules.map((rule, idx) => (
                    <div key={idx} className="flex items-center gap-2">
                      <input
                        type="text"
                        value={rule.field}
                        onChange={(e) => handleRuleChange(idx, 'field', e.target.value)}
                        className="input-field flex-1 min-w-0"
                        placeholder="Field (e.g. total_spent)"
                      />
                      <select
                        value={rule.operator}
                        onChange={(e) =>
                          handleRuleChange(idx, 'operator', e.target.value)
                        }
                        className="input-field w-28 shrink-0"
                      >
                        {OPERATORS.map((op) => (
                          <option key={op} value={op}>
                            {op}
                          </option>
                        ))}
                      </select>
                      <input
                        type="text"
                        value={rule.value}
                        onChange={(e) => handleRuleChange(idx, 'value', e.target.value)}
                        className="input-field flex-1 min-w-0"
                        placeholder="Value"
                      />
                      <button
                        type="button"
                        onClick={() => handleRemoveRule(idx)}
                        disabled={form.rules.length <= 1}
                        className="p-1.5 rounded-lg text-gray-400 hover:text-red-500 hover:bg-red-50 transition-colors disabled:opacity-30 disabled:cursor-not-allowed"
                      >
                        <TrashIcon className="h-4 w-4" />
                      </button>
                    </div>
                  ))}
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
                  disabled={isFormPending || !form.name}
                  className="btn-primary btn-sm"
                >
                  {isFormPending
                    ? 'Saving...'
                    : editingSegment
                      ? 'Update Segment'
                      : 'Create Segment'}
                </button>
              </div>
            </form>
          </DialogPanel>
        </div>
      </Dialog>

      {/* Delete Confirmation Dialog */}
      <ConfirmDialog
        open={deleteDialogOpen}
        onClose={() => setDeleteDialogOpen(false)}
        onConfirm={handleConfirmDelete}
        title="Delete Segment"
        message={`Are you sure you want to delete "${deleteTarget?.name}"? This action cannot be undone.`}
        confirmLabel="Delete"
        danger
      />
    </div>
  );
}
