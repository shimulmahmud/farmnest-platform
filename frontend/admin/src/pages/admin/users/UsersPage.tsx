import { useState, useCallback, useMemo } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  MagnifyingGlassIcon,
  ShieldExclamationIcon,
  NoSymbolIcon,
  CheckCircleIcon,
  XCircleIcon,
} from '@heroicons/react/24/outline';
import { Dialog, DialogPanel, DialogTitle } from '@headlessui/react';
import { listUsers, updateUserRole, updateUserStatus } from '../../../api/admin';
import { formatDate, cn } from '../../../lib/utils';
import DataTable from '../../../components/common/DataTable';
import LoadingSpinner from '../../../components/common/LoadingSpinner';
import ErrorMessage from '../../../components/common/ErrorMessage';
import EmptyState from '../../../components/common/EmptyState';
import Pagination from '../../../components/common/Pagination';
import type { User } from '../../../types';

const ROLES = ['SUPER_ADMIN', 'ADMIN', 'MANAGER', 'SUPPORT', 'CUSTOMER'] as const;
const STATUSES = ['ACTIVE', 'SUSPENDED', 'LOCKED'] as const;


function formatRole(role: string): string {
  return role
    .replace(/_/g, ' ')
    .toLowerCase()
    .replace(/\b\w/g, (c) => c.toUpperCase());
}

function formatUserStatus(status: string): string {
  return status
    .replace(/_/g, ' ')
    .toLowerCase()
    .replace(/\b\w/g, (c) => c.toUpperCase());
}

function getRoleBadgeClass(role: string): string {
  const map: Record<string, string> = {
    SUPER_ADMIN: 'bg-purple-100 text-purple-700',
    ADMIN: 'bg-red-100 text-red-700',
    MANAGER: 'bg-blue-100 text-blue-700',
    SUPPORT: 'bg-yellow-100 text-yellow-700',
    CUSTOMER: 'bg-gray-100 text-gray-700',
  };
  return map[role] || 'bg-gray-100 text-gray-700';
}

function getStatusBadgeClass(status: string): string {
  const map: Record<string, string> = {
    ACTIVE: 'bg-green-100 text-green-700',
    SUSPENDED: 'bg-yellow-100 text-yellow-700',
    LOCKED: 'bg-red-100 text-red-700',
  };
  return map[status] || 'bg-gray-100 text-gray-700';
}

export default function UsersPage() {
  const queryClient = useQueryClient();
  const [page, setPage] = useState(1);
  const [search, setSearch] = useState('');

  const [roleDialogOpen, setRoleDialogOpen] = useState(false);
  const [statusDialogOpen, setStatusDialogOpen] = useState(false);
  const [selectedUser, setSelectedUser] = useState<User | null>(null);
  const [newRole, setNewRole] = useState('');
  const [newStatus, setNewStatus] = useState('');

  const usersQuery = useQuery({
    queryKey: ['admin', 'users', page],
    queryFn: () => listUsers(page, 20),
  });

  const roleMutation = useMutation({
    mutationFn: ({ id, role }: { id: string; role: string }) =>
      updateUserRole(id, { role }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin', 'users'] });
      setRoleDialogOpen(false);
      setSelectedUser(null);
      setNewRole('');
    },
  });

  const statusMutation = useMutation({
    mutationFn: ({ id, status }: { id: string; status: string }) =>
      updateUserStatus(id, { status }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin', 'users'] });
      setStatusDialogOpen(false);
      setSelectedUser(null);
      setNewStatus('');
    },
  });

  const users = usersQuery.data?.data ?? [];
  const pagination = usersQuery.data?.pagination;

  const filteredUsers = useMemo(() => {
    if (!search.trim()) return users;
    const q = search.toLowerCase();
    return users.filter(
      (u) =>
        u.email.toLowerCase().includes(q) ||
        `${u.first_name} ${u.last_name}`.toLowerCase().includes(q) ||
        u.role.toLowerCase().includes(q),
    );
  }, [users, search]);

  const handleOpenRoleDialog = useCallback((user: User) => {
    setSelectedUser(user);
    setNewRole(user.role);
    setRoleDialogOpen(true);
  }, []);

  const handleCloseRoleDialog = useCallback(() => {
    setRoleDialogOpen(false);
    setSelectedUser(null);
    setNewRole('');
  }, []);

  const handleSubmitRole = useCallback(
    (e: React.FormEvent) => {
      e.preventDefault();
      if (!selectedUser || !newRole) return;
      roleMutation.mutate({ id: selectedUser.id, role: newRole });
    },
    [selectedUser, newRole, roleMutation],
  );

  const handleOpenStatusDialog = useCallback((user: User) => {
    setSelectedUser(user);
    setNewStatus(user.status);
    setStatusDialogOpen(true);
  }, []);

  const handleCloseStatusDialog = useCallback(() => {
    setStatusDialogOpen(false);
    setSelectedUser(null);
    setNewStatus('');
  }, []);

  const handleSubmitStatus = useCallback(
    (e: React.FormEvent) => {
      e.preventDefault();
      if (!selectedUser || !newStatus) return;
      statusMutation.mutate({ id: selectedUser.id, status: newStatus });
    },
    [selectedUser, newStatus, statusMutation],
  );

  if (usersQuery.isLoading) return <LoadingSpinner />;

  if (usersQuery.isError) {
    return (
      <ErrorMessage
        message={usersQuery.error?.message || 'Failed to load users'}
        onRetry={() => usersQuery.refetch()}
      />
    );
  }

  const columns = [
    {
      key: 'email',
      header: 'Email',
      render: (row: User) => (
        <span className="font-medium text-gray-900">{row.email}</span>
      ),
    },
    {
      key: 'name',
      header: 'Name',
      render: (row: User) => (
        <span>{row.first_name} {row.last_name}</span>
      ),
    },
    {
      key: 'role',
      header: 'Role',
      render: (row: User) => (
        <span
          className={cn(
            'inline-flex rounded-full px-2.5 py-0.5 text-xs font-medium',
            getRoleBadgeClass(row.role),
          )}
        >
          {formatRole(row.role)}
        </span>
      ),
    },
    {
      key: 'status',
      header: 'Status',
      render: (row: User) => (
        <span
          className={cn(
            'inline-flex rounded-full px-2.5 py-0.5 text-xs font-medium',
            getStatusBadgeClass(row.status),
          )}
        >
          {formatUserStatus(row.status)}
        </span>
      ),
    },
    {
      key: 'email_verified',
      header: 'Verified',
      render: (row: User) =>
        row.email_verified ? (
          <CheckCircleIcon className="h-5 w-5 text-green-500" />
        ) : (
          <XCircleIcon className="h-5 w-5 text-gray-300" />
        ),
    },
    {
      key: 'created_at',
      header: 'Created',
      render: (row: User) => (
        <span className="text-gray-500 text-sm">{formatDate(row.created_at)}</span>
      ),
    },
    {
      key: 'actions',
      header: 'Actions',
      render: (row: User) => (
        <div className="flex items-center gap-2">
          <button
            onClick={(e) => {
              e.stopPropagation();
              handleOpenRoleDialog(row);
            }}
            className="inline-flex items-center gap-1 rounded-lg px-2.5 py-1.5 text-xs font-medium text-primary-600 hover:bg-primary-50 transition-colors"
          >
            <ShieldExclamationIcon className="h-3.5 w-3.5" />
            Role
          </button>
          <button
            onClick={(e) => {
              e.stopPropagation();
              handleOpenStatusDialog(row);
            }}
            className="inline-flex items-center gap-1 rounded-lg px-2.5 py-1.5 text-xs font-medium text-yellow-600 hover:bg-yellow-50 transition-colors"
          >
            <NoSymbolIcon className="h-3.5 w-3.5" />
            Status
          </button>
        </div>
      ),
    },
  ];

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-gray-900">Users</h1>
        <p className="mt-1 text-sm text-gray-500">
          Manage user accounts, roles, and statuses
        </p>
      </div>

      <div className="relative max-w-sm">
        <MagnifyingGlassIcon className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-gray-400" />
        <input
          type="text"
          placeholder="Search by email, name, or role..."
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          className="input-field pl-10"
        />
      </div>

      {filteredUsers.length === 0 ? (
        <EmptyState
          icon={<NoSymbolIcon className="h-10 w-10 text-gray-400" />}
          title={search ? 'No users match your search' : 'No users found'}
          description={
            search
              ? 'Try adjusting your search query.'
              : 'There are no users in the system yet.'
          }
        />
      ) : (
        <>
          <DataTable
            columns={columns}
            data={filteredUsers as unknown as Record<string, unknown>[]}
            emptyMessage="No users found."
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

      {/* Change Role Dialog */}
      <Dialog open={roleDialogOpen} onClose={handleCloseRoleDialog} className="relative z-50">
        <div className="fixed inset-0 bg-black/30" aria-hidden="true" />
        <div className="fixed inset-0 flex items-center justify-center p-4">
          <DialogPanel className="w-full max-w-md rounded-2xl bg-white p-6 shadow-xl">
            <DialogTitle className="text-lg font-semibold text-gray-900">
              Change User Role
            </DialogTitle>
            {selectedUser && (
              <div className="mt-3 text-sm text-gray-500">
                User:{' '}
                <span className="font-medium text-gray-900">{selectedUser.email}</span>
                {' — '}Current Role:{' '}
                <span className={cn('inline-flex rounded-full px-2 py-0.5 text-xs font-medium', getRoleBadgeClass(selectedUser.role))}>
                  {formatRole(selectedUser.role)}
                </span>
              </div>
            )}
            {roleMutation.isError && (
              <div className="mt-3 rounded-lg bg-red-50 border border-red-200 p-3">
                <p className="text-sm text-red-700">
                  {roleMutation.error?.message || 'Failed to update role'}
                </p>
              </div>
            )}
            <form onSubmit={handleSubmitRole} className="mt-4 space-y-4">
              <div>
                <label htmlFor="role-select" className="label-field">
                  New Role <span className="text-red-500">*</span>
                </label>
                <select
                  id="role-select"
                  required
                  value={newRole}
                  onChange={(e) => setNewRole(e.target.value)}
                  className="input-field"
                >
                  {ROLES.map((r) => (
                    <option key={r} value={r}>
                      {formatRole(r)}
                    </option>
                  ))}
                </select>
              </div>
              <div className="flex items-center justify-end gap-3 pt-2">
                <button
                  type="button"
                  onClick={handleCloseRoleDialog}
                  className="btn-secondary btn-sm"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  disabled={roleMutation.isPending || newRole === selectedUser?.role}
                  className="btn-primary btn-sm"
                >
                  {roleMutation.isPending ? 'Saving...' : 'Update Role'}
                </button>
              </div>
            </form>
          </DialogPanel>
        </div>
      </Dialog>

      {/* Change Status Dialog */}
      <Dialog open={statusDialogOpen} onClose={handleCloseStatusDialog} className="relative z-50">
        <div className="fixed inset-0 bg-black/30" aria-hidden="true" />
        <div className="fixed inset-0 flex items-center justify-center p-4">
          <DialogPanel className="w-full max-w-md rounded-2xl bg-white p-6 shadow-xl">
            <DialogTitle className="text-lg font-semibold text-gray-900">
              Change User Status
            </DialogTitle>
            {selectedUser && (
              <div className="mt-3 text-sm text-gray-500">
                User:{' '}
                <span className="font-medium text-gray-900">{selectedUser.email}</span>
                {' — '}Current Status:{' '}
                <span className={cn('inline-flex rounded-full px-2 py-0.5 text-xs font-medium', getStatusBadgeClass(selectedUser.status))}>
                  {formatUserStatus(selectedUser.status)}
                </span>
              </div>
            )}
            {statusMutation.isError && (
              <div className="mt-3 rounded-lg bg-red-50 border border-red-200 p-3">
                <p className="text-sm text-red-700">
                  {statusMutation.error?.message || 'Failed to update status'}
                </p>
              </div>
            )}
            <form onSubmit={handleSubmitStatus} className="mt-4 space-y-4">
              <div>
                <label htmlFor="status-select" className="label-field">
                  New Status <span className="text-red-500">*</span>
                </label>
                <select
                  id="status-select"
                  required
                  value={newStatus}
                  onChange={(e) => setNewStatus(e.target.value)}
                  className="input-field"
                >
                  {STATUSES.map((s) => (
                    <option key={s} value={s}>
                      {formatUserStatus(s)}
                    </option>
                  ))}
                </select>
              </div>
              <div className="flex items-center justify-end gap-3 pt-2">
                <button
                  type="button"
                  onClick={handleCloseStatusDialog}
                  className="btn-secondary btn-sm"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  disabled={statusMutation.isPending || newStatus === selectedUser?.status}
                  className="btn-primary btn-sm"
                >
                  {statusMutation.isPending ? 'Saving...' : 'Update Status'}
                </button>
              </div>
            </form>
          </DialogPanel>
        </div>
      </Dialog>
    </div>
  );
}
