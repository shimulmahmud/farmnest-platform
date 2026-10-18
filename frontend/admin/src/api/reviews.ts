import { api } from './client';
import type {
  ReviewResponse,
  CreateReviewRequest,
  UpdateReviewRequest,
  PaginatedResponse,
  MessageResponse,
} from '../types';

export async function listReviews(
  productId: string,
  page = 1,
  perPage = 20,
): Promise<PaginatedResponse<ReviewResponse>> {
  const res = await api.get<PaginatedResponse<ReviewResponse>>(
    `/products/${productId}/reviews`,
    { params: { page, per_page: perPage } },
  );
  return res.data;
}

export async function createReview(
  productId: string,
  data: CreateReviewRequest,
): Promise<ReviewResponse> {
  const res = await api.post<ReviewResponse>(
    `/products/${productId}/reviews`,
    data,
  );
  return res.data;
}

export async function getReview(id: string): Promise<ReviewResponse> {
  const res = await api.get<ReviewResponse>(`/reviews/${id}`);
  return res.data;
}

export async function updateReview(
  id: string,
  data: UpdateReviewRequest,
): Promise<ReviewResponse> {
  const res = await api.patch<ReviewResponse>(`/reviews/${id}`, data);
  return res.data;
}

export async function deleteReview(id: string): Promise<MessageResponse> {
  const res = await api.delete<MessageResponse>(`/reviews/${id}`);
  return res.data;
}
