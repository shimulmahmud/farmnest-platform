import { useState, useCallback, Fragment } from 'react';
import { useQuery } from '@tanstack/react-query';
import { MagnifyingGlassIcon, ChevronDownIcon, ChevronUpIcon, ClipboardDocumentListIcon } from '@heroicons/react/24/outline';
import { listAuditLogs } from '../../../api/admin';
import { formatDateTime, truncate } from '../../../lib/utils';
import LoadingSpinner from '../../../components/common/LoadingSpinner';
import ErrorMessage from '../../../components/common/ErrorMessage';
import EmptyState from '../../../components/common/EmptyState';
import Pagination from '../../../components/common/Pagination';
import type { AuditEntryResponse } from '../../../types';

const RESOURCE_TYPES = [
  'ALL',
  'USER',
  'PRODUCT',
  'CATEGORY',
  'ORDER',
  'SHIPMENT',
  'REVIEW',
  'CAMPAIGN',
  'PROMOTION',
  'SEGMENT',
  'INVENTORY',
] as const;

type ResourceTypeFilter = (typeof RESOURCE_TYPES)[number];

function JsonBlock({ label, data }: { label: string; data: unknown }) {
  if (data === undefined || data === null) return null;
  const formatted = JSON.stringify(data, null, 2);
  if (!formatted || formatted === '{}' || formatted === '[]') return null;
  return (
    <div className="mt-1">
      <span className="text-xs font-semibold text-gray-500 uppercase">{label}</span>
      <pre className="mt-1 rounded-lg bg-gray-50 border border-gray-200 p-3 text-xs font-mono text-gray-700 overflow-x-auto max-h-48 overflow-y-auto whitespace-pre-wrap break-all">
        {formatted}
      </pre>
    </div>
  );
}

export default function AuditLogsPage() {
  const [page, setPage] = useState(1);
  const [actionFilter, setActionFilter] = useState('');
  const [resourceTypeFilter, setResourceTypeFilter] = useState<ResourceTypeFilter>('ALL');
  const [expandedRow, setExpandedRow] = useState<string | null>(null);

  const logsQuery = useQuery({
    queryKey: ['admin', 'audit-logs', page, actionFilter, resourceTypeFilter],
    queryFn: () =>
      listAuditLogs(
        page,
        20,
        resourceTypeFilter === 'ALL' ? undefined : resourceTypeFilter,
        actionFilter.trim() || undefined,
      ),
  });

  const logs = logsQuery.data?.data ?? [];
  const pagination = logsQuery.data?.pagination;

  const handleResourceTypeFilter = useCallback((type: ResourceTypeFilter) => {
    setResourceTypeFilter(type);
    setPage(1);
  }, []);

  const handleActionSearch = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
    setActionFilter(e.target.value);
    setPage(1);
  }, []);

  const toggleExpand = useCallback((id: string) => {
    setExpandedRow((prev) => (prev === id ? null : id));
  }, []);

  if (logsQuery.isLoading) return <LoadingSpinner />;

  if (logsQuery.isError) {
    return (
      <ErrorMessage
        message={logsQuery.error?.message || 'Failed to load audit logs'}
        onRetry={() => logsQuery.refetch()}
      />
    );
  }

  const columns = [
    {
      key: 'expand',
      header: '',
      render: (row: AuditEntryResponse) => (
        <button
          onClick={(e) => {
            e.stopPropagation();
            toggleExpand(row.id);
          }}
          className="p-1 rounded hover:bg-gray-100 transition-colors"
          aria-label="Toggle details"
        >
          {expandedRow === row.id ? (
            <ChevronUpIcon className="h-4 w-4 text-gray-500" />
          ) : (
            <ChevronDownIcon className="h-4 w-4 text-gray-500" />
          )}
        </button>
      ),
    },
    {
      key: 'created_at',
      header: 'Time',
      render: (row: AuditEntryResponse) => (
        <span className="text-sm text-gray-600 whitespace-nowrap">{formatDateTime(row.created_at)}</span>
      ),
    },
    {
      key: 'actor_id',
      header: 'Actor',
      render: (row: AuditEntryResponse) => (
        <span className="text-sm font-mono text-gray-700">
          {truncate(row.actor_id, 12)}
        </span>
      ),
    },
    {
      key: 'action',
      header: 'Action',
      render: (row: AuditEntryResponse) => {
        const actionColor = (() => {
          if (row.action.startsWith('CREATE')) return 'text-green-700 bg-green-50';
          if (row.action.startsWith('UPDATE')) return 'text-blue-700 bg-blue-50';
          if (row.action.startsWith('DELETE')) return 'text-red-700 bg-red-50';
          return 'text-gray-700 bg-gray-50';
        })();
        return (
          <span className={`inline-flex rounded px-2 py-0.5 text-xs font-medium ${actionColor}`}>
            {row.action}
          </span>
        );
      },
    },
    {
      key: 'resource_type',
      header: 'Resource Type',
      render: (row: AuditEntryResponse) => (
        <span className="inline-flex rounded-full bg-gray-100 px-2.5 py-0.5 text-xs font-medium text-gray-700">
          {row.resource_type}
        </span>
      ),
    },
    {
      key: 'resource_id',
      header: 'Resource ID',
      render: (row: AuditEntryResponse) => (
        <span className="text-sm font-mono text-gray-600">
          {truncate(row.resource_id, 12)}
        </span>
      ),
    },
    {
      key: 'ip_address',
      header: 'IP Address',
      render: (row: AuditEntryResponse) => (
        <span className="text-sm text-gray-500 font-mono">{row.ip_address}</span>
      ),
    },
    {
      key: 'request_id',
      header: 'Request ID',
      render: (row: AuditEntryResponse) => (
        <span className="text-xs font-mono text-gray-400">
          {row.request_id ? truncate(row.request_id, 12) : '—'}
        </span>
      ),
    },
  ];

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-gray-900">Audit Logs</h1>
        <p className="mt-1 text-sm text-gray-500">
          Read-only log of all administrative actions
        </p>
      </div>

      {/* Filters */}
      <div className="flex flex-col sm:flex-row gap-4">
        <div className="relative max-w-xs flex-1">
          <MagnifyingGlassIcon className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-gray-400" />
          <input
            type="text"
            placeholder="Filter by action..."
            value={actionFilter}
            onChange={handleActionSearch}
            className="input-field pl-10"
          />
        </div>
        <div>
          <select
            value={resourceTypeFilter}
            onChange={(e) =>
              handleResourceTypeFilter(e.target.value as ResourceTypeFilter)
            }
            className="input-field w-full sm:w-auto"
          >
            {RESOURCE_TYPES.map((rt) => (
              <option key={rt} value={rt}>
                {rt === 'ALL' ? 'All Resource Types' : rt}
              </option>
            ))}
          </select>
        </div>
      </div>

      {logs.length === 0 ? (
        <EmptyState
          icon={<ClipboardDocumentListIcon className="h-10 w-10 text-gray-400" />}
          title="No audit logs found"
          description="No logs match the current filter."
        />
      ) : (
        <>
          {/* Table with Expandable Details */}
          <div className="overflow-x-auto rounded-lg border border-gray-200">
            <table className="min-w-full divide-y divide-gray-200">
              <thead className="bg-gray-50">
                <tr>
                  {columns.map((col) => (
                    <th
                      key={col.key}
                      scope="col"
                      className="px-4 py-3 text-left text-xs font-semibold text-gray-600 uppercase tracking-wider"
                    >
                      {col.header}
                    </th>
                  ))}
                </tr>
              </thead>
              <tbody className="bg-white divide-y divide-gray-100">
                {logs.map((row) => {
                  const isExpanded = expandedRow === row.id;
                  return (
                    <Fragment key={row.id}>
                      <tr
                        onClick={() => toggleExpand(row.id)}
                        className="cursor-pointer hover:bg-gray-50 transition-colors"
                      >
                        {columns.map((col) => (
                          <td
                            key={col.key}
                            className="px-4 py-3 text-sm text-gray-700 whitespace-nowrap"
                          >
                            {col.render(row)}
                          </td>
                        ))}
                      </tr>
                      {/* Expanded Detail Row */}
                      {isExpanded && (
                        <tr key={`${row.id}-detail`}>
                          <td colSpan={columns.length} className="px-4 py-4 bg-gray-50">
                            <div className="space-y-3">
                              <JsonBlock label="Old Value" data={row.old_value} />
                              <JsonBlock label="New Value" data={row.new_value} />
                              <JsonBlock label="Details" data={row.details} />
                              {(row.old_value === undefined &&
                                row.new_value === undefined &&
                                row.details === undefined) && (
                                <p className="text-sm text-gray-400 italic">
                                  No additional details available for this entry.
                                </p>
                              )}
                            </div>
                          </td>
                        </tr>
                      )}
                    </Fragment>
                  );
                })}
              </tbody>
            </table>
          </div>

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
    </div>
  );
}
