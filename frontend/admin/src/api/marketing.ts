import { api } from './client';
import type {
  CampaignResponse,
  CreateCampaignRequest,
  UpdateCampaignRequest,
  SegmentResponse,
  CreateSegmentRequest,
  UpdateSegmentRequest,
  PromotionResponse,
  CreatePromotionRequest,
  UpdatePromotionRequest,
  PaginatedResponse,
  MessageResponse,
} from '../types';

// ============================================================================
// Campaigns
// ============================================================================

export async function listCampaigns(
  page = 1,
  perPage = 20,
): Promise<PaginatedResponse<CampaignResponse>> {
  const res = await api.get<PaginatedResponse<CampaignResponse>>(
    '/admin/marketing/campaigns',
    { params: { page, per_page: perPage } },
  );
  return res.data;
}

export async function getCampaign(id: string): Promise<CampaignResponse> {
  const res = await api.get<CampaignResponse>(`/admin/marketing/campaigns/${id}`);
  return res.data;
}

export async function createCampaign(
  data: CreateCampaignRequest,
): Promise<CampaignResponse> {
  const res = await api.post<CampaignResponse>('/admin/marketing/campaigns', data);
  return res.data;
}

export async function updateCampaign(
  id: string,
  data: UpdateCampaignRequest,
): Promise<CampaignResponse> {
  const res = await api.put<CampaignResponse>(
    `/admin/marketing/campaigns/${id}`,
    data,
  );
  return res.data;
}

export async function transitionCampaignStatus(
  id: string,
  data: { status: string },
): Promise<CampaignResponse> {
  const res = await api.patch<CampaignResponse>(
    `/admin/marketing/campaigns/${id}/status`,
    data,
  );
  return res.data;
}

// ============================================================================
// Segments
// ============================================================================

export async function listSegments(
  page = 1,
  perPage = 20,
): Promise<PaginatedResponse<SegmentResponse>> {
  const res = await api.get<PaginatedResponse<SegmentResponse>>(
    '/admin/marketing/segments',
    { params: { page, per_page: perPage } },
  );
  return res.data;
}

export async function getSegment(id: string): Promise<SegmentResponse> {
  const res = await api.get<SegmentResponse>(`/admin/marketing/segments/${id}`);
  return res.data;
}

export async function createSegment(
  data: CreateSegmentRequest,
): Promise<SegmentResponse> {
  const res = await api.post<SegmentResponse>('/admin/marketing/segments', data);
  return res.data;
}

export async function updateSegment(
  id: string,
  data: UpdateSegmentRequest,
): Promise<SegmentResponse> {
  const res = await api.put<SegmentResponse>(
    `/admin/marketing/segments/${id}`,
    data,
  );
  return res.data;
}

export async function deleteSegment(id: string): Promise<MessageResponse> {
  const res = await api.delete<MessageResponse>(`/admin/marketing/segments/${id}`);
  return res.data;
}

// ============================================================================
// Promotions
// ============================================================================

export async function listPromotions(
  page = 1,
  perPage = 20,
): Promise<PaginatedResponse<PromotionResponse>> {
  const res = await api.get<PaginatedResponse<PromotionResponse>>(
    '/admin/marketing/promotions',
    { params: { page, per_page: perPage } },
  );
  return res.data;
}

export async function getPromotion(id: string): Promise<PromotionResponse> {
  const res = await api.get<PromotionResponse>(`/admin/marketing/promotions/${id}`);
  return res.data;
}

export async function createPromotion(
  data: CreatePromotionRequest,
): Promise<PromotionResponse> {
  const res = await api.post<PromotionResponse>(
    '/admin/marketing/promotions',
    data,
  );
  return res.data;
}

export async function updatePromotion(
  id: string,
  data: UpdatePromotionRequest,
): Promise<PromotionResponse> {
  const res = await api.put<PromotionResponse>(
    `/admin/marketing/promotions/${id}`,
    data,
  );
  return res.data;
}

export async function transitionPromotionStatus(
  id: string,
  data: { status: string },
): Promise<PromotionResponse> {
  const res = await api.patch<PromotionResponse>(
    `/admin/marketing/promotions/${id}/status`,
    data,
  );
  return res.data;
}

