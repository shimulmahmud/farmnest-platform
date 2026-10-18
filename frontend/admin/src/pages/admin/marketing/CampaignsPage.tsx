import { useState, useCallback } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  PlusIcon,
  PencilSquareIcon,
  PlayIcon,
  PauseIcon,
  ArchiveBoxIcon,
  MegaphoneIcon,
} from '@heroicons/react/24/outline';
import { Dialog, DialogPanel, DialogTitle } from '@headlessui/react';
import {
  listCampaigns,
  createCampaign,
  updateCampaign,
  transitionCampaignStatus,
} from '../../../api/marketing';
import { formatCurrency, cn, truncate } from '../../../lib/utils';
import DataTable from '../../../components/common/DataTable';
import LoadingSpinner from '../../../components/common/LoadingSpinner';
import ErrorMessage from '../../../components/common/ErrorMessage';
import EmptyState from '../../../components/common/EmptyState';
import Pagination from '../../../components/common/Pagination';
import type { CampaignResponse, CreateCampaignRequest } from '../../../types';

const CAMPAIGN_TYPES = [
  'ABANDONED_CART',
  'BROWSE_ABANDONMENT',
  'WELCOME_SERIES',
  'PROMOTIONAL',
] as const;

type CampaignType = (typeof CAMPAIGN_TYPES)[number];

const EVENT_TYPES = [
  'cart_abandoned',
  'browse_abandoned',
  'user_registered',
  'order_completed',
  'product_viewed',
] as const;

function formatCampaignType(type: string): string {
  return type
    .replace(/_/g, ' ')
    .toLowerCase()
    .replace(/\b\w/g, (c) => c.toUpperCase());
}

function formatCampaignStatus(status: string): string {
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

interface CampaignForm {
  name: string;
  type: CampaignType;
  description: string;
  eventTypes: string[];
  segmentId: string;
}

const emptyForm: CampaignForm = {
  name: '',
  type: 'PROMOTIONAL',
  description: '',
  eventTypes: [],
  segmentId: '',
};

function parseRules(rules: unknown): { eventTypes: string[]; segmentId: string } {
  if (!rules || typeof rules !== 'object') return { eventTypes: [], segmentId: '' };
  const r = rules as Record<string, unknown>;
  return {
    eventTypes: Array.isArray(r.event_types) ? (r.event_types as string[]) : [],
    segmentId: typeof r.segment_id === 'string' ? r.segment_id : '',
  };
}

export default function CampaignsPage() {
  const queryClient = useQueryClient();
  const [page, setPage] = useState(1);

  const [formDialogOpen, setFormDialogOpen] = useState(false);
  const [editingCampaign, setEditingCampaign] = useState<CampaignResponse | null>(null);
  const [form, setForm] = useState<CampaignForm>(emptyForm);

  const campaignsQuery = useQuery({
    queryKey: ['admin', 'campaigns', page],
    queryFn: () => listCampaigns(page, 20),
  });

  const createMutation = useMutation({
    mutationFn: (data: CreateCampaignRequest) => createCampaign(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin', 'campaigns'] });
      setFormDialogOpen(false);
      setForm(emptyForm);
    },
  });

  const updateMutation = useMutation({
    mutationFn: ({ id, data }: { id: string; data: CreateCampaignRequest }) =>
      updateCampaign(id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin', 'campaigns'] });
      setFormDialogOpen(false);
      setEditingCampaign(null);
      setForm(emptyForm);
    },
  });

  const transitionMutation = useMutation({
    mutationFn: ({ id, status }: { id: string; status: string }) =>
      transitionCampaignStatus(id, { status }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin', 'campaigns'] });
    },
  });

  const campaigns = campaignsQuery.data?.data ?? [];
  const pagination = campaignsQuery.data?.pagination;

  const buildRules = (f: CampaignForm): Record<string, unknown> => ({
    event_types: f.eventTypes,
    segment_id: f.segmentId || undefined,
  });

  const handleOpenCreate = useCallback(() => {
    setEditingCampaign(null);
    setForm(emptyForm);
    setFormDialogOpen(true);
  }, []);

  const handleOpenEdit = useCallback((campaign: CampaignResponse) => {
    setEditingCampaign(campaign);
    const parsed = parseRules(campaign.rules);
    setForm({
      name: campaign.name,
      type: campaign.type as CampaignType,
      description: campaign.description || '',
      eventTypes: parsed.eventTypes,
      segmentId: parsed.segmentId,
    });
    setFormDialogOpen(true);
  }, []);

  const handleCloseFormDialog = useCallback(() => {
    setFormDialogOpen(false);
    setEditingCampaign(null);
    setForm(emptyForm);
  }, []);

  const handleSubmitForm = useCallback(
    (e: React.FormEvent) => {
      e.preventDefault();
      const payload: CreateCampaignRequest = {
        name: form.name,
        type: form.type,
        description: form.description || undefined,
        rules: buildRules(form),
      };

      if (editingCampaign) {
        updateMutation.mutate({ id: editingCampaign.id, data: payload });
      } else {
        createMutation.mutate(payload);
      }
    },
    [form, editingCampaign, createMutation, updateMutation],
  );

  const handleToggleEventType = useCallback((eventType: string) => {
    setForm((prev) => ({
      ...prev,
      eventTypes: prev.eventTypes.includes(eventType)
        ? prev.eventTypes.filter((t) => t !== eventType)
        : [...prev.eventTypes, eventType],
    }));
  }, []);

  const getNextTransitions = (status: string): { label: string; value: string; icon: typeof PlayIcon }[] => {
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

  if (campaignsQuery.isLoading) return <LoadingSpinner />;

  if (campaignsQuery.isError) {
    return (
      <ErrorMessage
        message={campaignsQuery.error?.message || 'Failed to load campaigns'}
        onRetry={() => campaignsQuery.refetch()}
      />
    );
  }

  const columns = [
    {
      key: 'name',
      header: 'Name',
      render: (row: CampaignResponse) => (
        <div>
          <span className="font-medium text-gray-900">{truncate(row.name, 30)}</span>
          {row.description && (
            <p className="text-xs text-gray-500 mt-0.5">{truncate(row.description, 50)}</p>
          )}
        </div>
      ),
    },
    {
      key: 'type',
      header: 'Type',
      render: (row: CampaignResponse) => (
        <span className="text-sm text-gray-600">{formatCampaignType(row.type)}</span>
      ),
    },
    {
      key: 'status',
      header: 'Status',
      render: (row: CampaignResponse) => (
        <span
          className={cn(
            'inline-flex rounded-full px-2.5 py-0.5 text-xs font-medium',
            getStatusBadgeClass(row.status),
          )}
        >
          {formatCampaignStatus(row.status)}
        </span>
      ),
    },
    {
      key: 'metrics',
      header: 'Sent',
      render: (row: CampaignResponse) => (
        <span className="text-sm text-gray-700">
          {row.metrics.messages_sent.toLocaleString()}
        </span>
      ),
    },
    {
      key: 'conversions',
      header: 'Converted',
      render: (row: CampaignResponse) => (
        <span className="text-sm text-gray-700">
          {row.metrics.conversions.toLocaleString()}
        </span>
      ),
    },
    {
      key: 'revenue',
      header: 'Revenue',
      render: (row: CampaignResponse) => (
        <span className="text-sm font-medium text-gray-900">
          {formatCurrency(row.metrics.revenue_cents)}
        </span>
      ),
    },
    {
      key: 'actions',
      header: 'Actions',
      render: (row: CampaignResponse) => {
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
          <h1 className="text-2xl font-bold text-gray-900">Campaigns</h1>
          <p className="mt-1 text-sm text-gray-500">
            Manage marketing campaigns and track performance
          </p>
        </div>
        <button onClick={handleOpenCreate} className="btn-primary btn-sm">
          <PlusIcon className="h-4 w-4 mr-1" />
          Create Campaign
        </button>
      </div>

      {campaigns.length === 0 ? (
        <EmptyState
          icon={<MegaphoneIcon className="h-10 w-10 text-gray-400" />}
          title="No campaigns found"
          description="Create your first marketing campaign to get started."
          action={
            <button onClick={handleOpenCreate} className="btn-primary btn-sm">
              <PlusIcon className="h-4 w-4 mr-1" />
              Create Campaign
            </button>
          }
        />
      ) : (
        <>
          <DataTable
            columns={columns}
            data={campaigns as unknown as Record<string, unknown>[]}
            emptyMessage="No campaigns found."
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

      {/* Create/Edit Campaign Dialog */}
      <Dialog open={formDialogOpen} onClose={handleCloseFormDialog} className="relative z-50">
        <div className="fixed inset-0 bg-black/30" aria-hidden="true" />
        <div className="fixed inset-0 flex items-center justify-center p-4">
          <DialogPanel className="w-full max-w-lg rounded-2xl bg-white p-6 shadow-xl max-h-[90vh] overflow-y-auto">
            <DialogTitle className="text-lg font-semibold text-gray-900">
              {editingCampaign ? 'Edit Campaign' : 'Create Campaign'}
            </DialogTitle>

            {formError && (
              <div className="mt-3 rounded-lg bg-red-50 border border-red-200 p-3">
                <p className="text-sm text-red-700">{formError}</p>
              </div>
            )}

            <form onSubmit={handleSubmitForm} className="mt-4 space-y-4">
              <div>
                <label htmlFor="campaign-name" className="label-field">
                  Name <span className="text-red-500">*</span>
                </label>
                <input
                  id="campaign-name"
                  type="text"
                  required
                  value={form.name}
                  onChange={(e) => setForm((f) => ({ ...f, name: e.target.value }))}
                  className="input-field"
                  placeholder="e.g. Holiday Sale Campaign"
                />
              </div>

              <div>
                <label htmlFor="campaign-type" className="label-field">
                  Type <span className="text-red-500">*</span>
                </label>
                <select
                  id="campaign-type"
                  required
                  value={form.type}
                  onChange={(e) => setForm((f) => ({ ...f, type: e.target.value as CampaignType }))}
                  className="input-field"
                >
                  {CAMPAIGN_TYPES.map((t) => (
                    <option key={t} value={t}>
                      {formatCampaignType(t)}
                    </option>
                  ))}
                </select>
              </div>

              <div>
                <label htmlFor="campaign-desc" className="label-field">
                  Description
                </label>
                <textarea
                  id="campaign-desc"
                  rows={3}
                  value={form.description}
                  onChange={(e) => setForm((f) => ({ ...f, description: e.target.value }))}
                  className="input-field"
                  placeholder="Describe the campaign..."
                />
              </div>

              <div>
                <label className="label-field">Event Types (Triggers)</label>
                <div className="mt-1 flex flex-wrap gap-2">
                  {EVENT_TYPES.map((et) => (
                    <label
                      key={et}
                      className="inline-flex items-center gap-1.5 cursor-pointer"
                    >
                      <input
                        type="checkbox"
                        checked={form.eventTypes.includes(et)}
                        onChange={() => handleToggleEventType(et)}
                        className="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500"
                      />
                      <span className="text-sm text-gray-700">
                        {et.replace(/_/g, ' ').replace(/\b\w/g, (c) => c.toUpperCase())}
                      </span>
                    </label>
                  ))}
                </div>
              </div>

              <div>
                <label htmlFor="segment-id" className="label-field">
                  Segment ID
                </label>
                <input
                  id="segment-id"
                  type="text"
                  value={form.segmentId}
                  onChange={(e) => setForm((f) => ({ ...f, segmentId: e.target.value }))}
                  className="input-field"
                  placeholder="Optional: target a specific segment"
                />
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
                    : editingCampaign
                      ? 'Update Campaign'
                      : 'Create Campaign'}
                </button>
              </div>
            </form>
          </DialogPanel>
        </div>
      </Dialog>
    </div>
  );
}
