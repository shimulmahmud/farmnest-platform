import { api } from './client';
import type {
  DashboardCounts,
  User,
  CustomerResponse,
  AuditEntryResponse,
  ReviewModerationResponse,
  UpdateUserRoleRequest,
  UpdateUserStatusRequest,
  ModerateReviewRequest,
  PaginatedResponse,
  MessageResponse,
} from '../types';

// ============================================================================
// Dashboard
// ============================================================================

export async function getDashboardCounts(): Promise<DashboardCounts> {
  const res = await api.get<DashboardCounts>('/admin/dashboard/counts');
  return res.data;
}

// ============================================================================
// Users
// ============================================================================

export async function listUsers(
  page = 1,
  perPage = 20,
): Promise<PaginatedResponse<User>> {
  const res = await api.get<PaginatedResponse<User>>('/admin/users', {
    params: { page, per_page: perPage },
  });
  return res.data;
}

export async function getUser(id: string): Promise<User> {
  const res = await api.get<User>(`/admin/users/${id}`);
  return res.data;
}

export async function updateUserRole(
  id: string,
  data: UpdateUserRoleRequest,
): Promise<User> {
  const res = await api.patch<User>(`/admin/users/${id}/role`, data);
  return res.data;
}

export async function updateUserStatus(
  id: string,
  data: UpdateUserStatusRequest,
): Promise<User> {
  const res = await api.patch<User>(`/admin/users/${id}/status`, data);
  return res.data;
}

// ============================================================================
// Customers
// ============================================================================

export async function listCustomers(
  page = 1,
  perPage = 20,
): Promise<PaginatedResponse<CustomerResponse>> {
  const res = await api.get<PaginatedResponse<CustomerResponse>>(
    '/admin/customers',
    { params: { page, per_page: perPage } },
  );
  return res.data;
}

export async function getCustomer(id: string): Promise<CustomerResponse> {
  const res = await api.get<CustomerResponse>(`/admin/customers/${id}`);
  return res.data;
}

// ============================================================================
// Audit Logs
// ============================================================================

export async function listAuditLogs(
  page = 1,
  perPage = 20,
  resourceType?: string,
  action?: string,
): Promise<PaginatedResponse<AuditEntryResponse>> {
  const params: Record<string, string | number | undefined> = {
    page,
    per_page: perPage,
  };
  if (resourceType !== undefined) params.resource_type = resourceType;
  if (action !== undefined) params.action = action;

  const res = await api.get<PaginatedResponse<AuditEntryResponse>>(
    '/admin/audit-logs',
    { params },
  );
  return res.data;
}

// ============================================================================
// Review Moderation
// ============================================================================

export async function listReviewsModeration(
  page = 1,
  perPage = 20,
  status?: string,
): Promise<PaginatedResponse<ReviewModerationResponse>> {
  const params: Record<string, string | number | undefined> = {
    page,
    per_page: perPage,
  };
  if (status !== undefined) params.status = status;

  const res = await api.get<PaginatedResponse<ReviewModerationResponse>>(
    '/admin/reviews',
    { params },
  );
  return res.data;
}

export async function moderateReview(
  id: string,
  data: ModerateReviewRequest,
): Promise<ReviewModerationResponse> {
  const res = await api.patch<ReviewModerationResponse>(
    `/admin/reviews/${id}/moderate`,
    data,
  );
  return res.data;
}

export async function deleteModerationReview(
  id: string,
): Promise<MessageResponse> {
  const res = await api.delete<MessageResponse>(`/admin/reviews/${id}`);
  return res.data;
}