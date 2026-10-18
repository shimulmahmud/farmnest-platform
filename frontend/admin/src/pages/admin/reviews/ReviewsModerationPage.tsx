import { useState, useCallback } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  CheckCircleIcon,
  XCircleIcon,
  EyeSlashIcon,
  ChatBubbleLeftRightIcon,
} from '@heroicons/react/24/outline';
import { StarIcon as StarSolidIcon } from '@heroicons/react/24/solid';
import { Dialog, DialogPanel, DialogTitle } from '@headlessui/react';
import { listReviewsModeration, moderateReview } from '../../../api/admin';
import { formatDate, cn, truncate } from '../../../lib/utils';
import DataTable from '../../../components/common/DataTable';
import LoadingSpinner from '../../../components/common/LoadingSpinner';
import ErrorMessage from '../../../components/common/ErrorMessage';
import EmptyState from '../../../components/common/EmptyState';
import Pagination from '../../../components/common/Pagination';
import StatusBadge from '../../../components/common/StatusBadge';
import type { ReviewModerationResponse } from '../../../types';

const REVIEW_STATUSES = ['ALL', 'PENDING', 'APPROVED', 'REJECTED', 'HIDDEN'] as const;
type ReviewStatusFilter = (typeof REVIEW_STATUSES)[number];

function StarRating({ rating }: { rating: number }) {
  return (
    <div className="inline-flex items-center gap-0.5">
      {[1, 2, 3, 4, 5].map((star) => (
        <StarSolidIcon
          key={star}
          className={cn(
            'h-4 w-4',
            star <= rating ? 'text-yellow-400' : 'text-gray-200',
          )}
        />
      ))}
    </div>
  );
}

export default function ReviewsModerationPage() {
  const queryClient = useQueryClient();
  const [page, setPage] = useState(1);
  const [statusFilter, setStatusFilter] = useState<ReviewStatusFilter>('ALL');

  const [moderateDialogOpen, setModerateDialogOpen] = useState(false);
  const [selectedReview, setSelectedReview] = useState<ReviewModerationResponse | null>(null);
  const [moderateAction, setModerateAction] = useState<'' | 'APPROVED' | 'REJECTED' | 'HIDDEN'>('');
  const [reason, setReason] = useState('');

  const reviewsQuery = useQuery({
    queryKey: ['admin', 'reviews-moderation', page, statusFilter],
    queryFn: () =>
      listReviewsModeration(
        page,
        20,
        statusFilter === 'ALL' ? undefined : statusFilter,
      ),
  });

  const moderateMutation = useMutation({
    mutationFn: ({ id, action, reason }: { id: string; action: 'approve' | 'reject' | 'hide'; reason?: string }) =>
      moderateReview(id, { action, reason: reason || undefined }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin', 'reviews-moderation'] });
      setModerateDialogOpen(false);
      setSelectedReview(null);
      setModerateAction('');
      setReason('');
    },
  });

  const reviews = reviewsQuery.data?.data ?? [];
  const pagination = reviewsQuery.data?.pagination;

  const handleOpenModerateDialog = useCallback(
    (review: ReviewModerationResponse, action: 'APPROVED' | 'REJECTED' | 'HIDDEN') => {
      setSelectedReview(review);
      setModerateAction(action);
      setReason('');
      setModerateDialogOpen(true);
    },
    [],
  );

  const handleCloseModerateDialog = useCallback(() => {
    setModerateDialogOpen(false);
    setSelectedReview(null);
    setModerateAction('');
    setReason('');
  }, []);

  const handleSubmitModerate = useCallback(
    (e: React.FormEvent) => {
      e.preventDefault();
      if (!selectedReview || !moderateAction) return;
      const action = moderateAction === 'APPROVED' ? 'approve' : moderateAction === 'REJECTED' ? 'reject' : 'hide';
      moderateMutation.mutate({ id: selectedReview.id, action, reason: reason.trim() });
    },
    [selectedReview, moderateAction, moderateMutation, reason],
  );

  const handleStatusFilter = useCallback((status: ReviewStatusFilter) => {
    setStatusFilter(status);
    setPage(1);
  }, []);

  if (reviewsQuery.isLoading) return <LoadingSpinner />;

  if (reviewsQuery.isError) {
    return (
      <ErrorMessage
        message={reviewsQuery.error?.message || 'Failed to load reviews'}
        onRetry={() => reviewsQuery.refetch()}
      />
    );
  }

  const actionLabel: Record<string, string> = {
    APPROVED: 'Approve',
    REJECTED: 'Reject',
    HIDDEN: 'Hide',
  };

  const ActionIcon: Record<string, typeof CheckCircleIcon> = {
    APPROVED: CheckCircleIcon,
    REJECTED: XCircleIcon,
    HIDDEN: EyeSlashIcon,
  };

  const columns = [
    {
      key: 'product_id',
      header: 'Product ID',
      render: (row: ReviewModerationResponse) => (
        <span className="text-gray-600 text-sm font-mono">
          {row.product_id.length > 8 ? row.product_id.slice(0, 8) + '...' : row.product_id}
        </span>
      ),
    },
    {
      key: 'customer_id',
      header: 'Customer ID',
      render: (row: ReviewModerationResponse) => (
        <span className="text-gray-600 text-sm font-mono">
          {row.author_name || (row.customer_id.length > 8 ? row.customer_id.slice(0, 8) + '...' : row.customer_id)}
        </span>
      ),
    },
    {
      key: 'rating',
      header: 'Rating',
      render: (row: ReviewModerationResponse) => <StarRating rating={row.rating} />,
    },
    {
      key: 'title',
      header: 'Title',
      render: (row: ReviewModerationResponse) => (
        <span className="font-medium text-gray-900">{truncate(row.title, 40)}</span>
      ),
    },
    {
      key: 'status',
      header: 'Status',
      render: (row: ReviewModerationResponse) => (
        <StatusBadge status={row.status} type="review" />
      ),
    },
    {
      key: 'verified_purchase',
      header: 'Verified',
      render: (row: ReviewModerationResponse) => (
        <span className={row.verified_purchase ? 'text-green-600 font-medium' : 'text-gray-400'}>
          {row.verified_purchase ? 'Yes' : 'No'}
        </span>
      ),
    },
    {
      key: 'created_at',
      header: 'Date',
      render: (row: ReviewModerationResponse) => (
        <span className="text-gray-500 text-sm">{formatDate(row.created_at)}</span>
      ),
    },
    {
      key: 'actions',
      header: 'Actions',
      render: (row: ReviewModerationResponse) => {
        if (row.status !== 'PENDING') {
          return <span className="text-xs text-gray-400">No actions</span>;
        }
        return (
          <div className="flex items-center gap-1">
            {(['APPROVED', 'REJECTED', 'HIDDEN'] as const).map((action) => {
              const Icon = ActionIcon[action];
              return (
                <button
                  key={action}
                  onClick={(e) => {
                    e.stopPropagation();
                    handleOpenModerateDialog(row, action);
                  }}
                  className={cn(
                    'inline-flex items-center gap-1 rounded-lg px-2 py-1.5 text-xs font-medium transition-colors',
                    action === 'APPROVED' && 'text-green-600 hover:bg-green-50',
                    action === 'REJECTED' && 'text-red-600 hover:bg-red-50',
                    action === 'HIDDEN' && 'text-gray-600 hover:bg-gray-100',
                  )}
                >
                  <Icon className="h-3.5 w-3.5" />
                  {actionLabel[action]}
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
      <div>
        <h1 className="text-2xl font-bold text-gray-900">Reviews Moderation</h1>
        <p className="mt-1 text-sm text-gray-500">
          Review, approve, reject, or hide customer reviews
        </p>
      </div>

      <div className="flex flex-wrap gap-2">
        {REVIEW_STATUSES.map((status) => (
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
            {status === 'ALL'
              ? 'All'
              : status
                  .replace(/_/g, ' ')
                  .toLowerCase()
                  .replace(/\b\w/g, (c) => c.toUpperCase())}
          </button>
        ))}
      </div>

      {reviews.length === 0 ? (
        <EmptyState
          icon={<ChatBubbleLeftRightIcon className="h-10 w-10 text-gray-400" />}
          title="No reviews found"
          description="No reviews match the current filter."
        />
      ) : (
        <>
          <DataTable
            columns={columns}
            data={reviews as unknown as Record<string, unknown>[]}
            emptyMessage="No reviews found."
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

      {/* Moderate Review Dialog */}
      <Dialog
        open={moderateDialogOpen}
        onClose={handleCloseModerateDialog}
        className="relative z-50"
      >
        <div className="fixed inset-0 bg-black/30" aria-hidden="true" />
        <div className="fixed inset-0 flex items-center justify-center p-4">
          <DialogPanel className="w-full max-w-md rounded-2xl bg-white p-6 shadow-xl">
            <DialogTitle className="text-lg font-semibold text-gray-900">
              {moderateAction === 'APPROVED' && 'Approve Review'}
              {moderateAction === 'REJECTED' && 'Reject Review'}
              {moderateAction === 'HIDDEN' && 'Hide Review'}
            </DialogTitle>

            {selectedReview && (
              <div className="mt-3 space-y-2">
                <div className="text-sm text-gray-500">
                  <span className="font-medium text-gray-900">{selectedReview.title}</span>
                </div>
                <div className="flex items-center gap-3 text-xs text-gray-500">
                  <StarRating rating={selectedReview.rating} />
                  <span>by {selectedReview.author_name || selectedReview.customer_id.slice(0, 8) + '...'}</span>
                  {selectedReview.verified_purchase && (
                    <span className="text-green-600 font-medium">Verified Purchase</span>
                  )}
                </div>
              </div>
            )}

            {moderateMutation.isError && (
              <div className="mt-3 rounded-lg bg-red-50 border border-red-200 p-3">
                <p className="text-sm text-red-700">
                  {moderateMutation.error?.message || 'Failed to moderate review'}
                </p>
              </div>
            )}

            <form onSubmit={handleSubmitModerate} className="mt-4 space-y-4">
              <div>
                <label htmlFor="moderate-reason" className="label-field">
                  Reason
                </label>
                <textarea
                  id="moderate-reason"
                  rows={3}
                  value={reason}
                  onChange={(e) => setReason(e.target.value)}
                  placeholder={
                    moderateAction === 'APPROVED'
                      ? 'Reason for approval (optional)...'
                      : 'Reason for this action...'
                  }
                  className="input-field"
                />
              </div>

              <div className="flex items-center justify-end gap-3 pt-2">
                <button
                  type="button"
                  onClick={handleCloseModerateDialog}
                  className="btn-secondary btn-sm"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  disabled={moderateMutation.isPending}
                  className={cn(
                    'btn-sm',
                    moderateAction === 'APPROVED' && 'btn-primary',
                    moderateAction === 'REJECTED' && 'btn-danger',
                    moderateAction === 'HIDDEN' && 'btn-secondary',
                  )}
                >
                  {moderateMutation.isPending
                    ? 'Processing...'
                    : moderateAction === 'APPROVED'
                      ? 'Approve'
                      : moderateAction === 'REJECTED'
                        ? 'Reject'
                        : 'Hide'}
                </button>
              </div>
            </form>
          </DialogPanel>
        </div>
      </Dialog>
    </div>
  );
}
