import { getPage, post } from './http';

export interface ApiReview {
  id: string;
  product_id: string;
  rating: number;
  title: string;
  comment: string;
  author_name: string;
  verified_purchase: boolean;
  created_at: string;
}

export const fetchReviews = (productBackendId: string, page = 1) =>
  getPage<ApiReview>(`/products/${productBackendId}/reviews`, { page, per_page: 20 });

export const postReview = (productBackendId: string, input: { rating: number; title: string; comment: string }) =>
  post<ApiReview>(`/products/${productBackendId}/reviews`, input);

export const fetchMyDeliveredProductIds = async (): Promise<Set<string>> => {
  // Which products this customer has actually received (they may review only those).
  const ids = new Set<string>();
  const page = await getPage<{ status: string; items: { product_id: string }[] }>('/orders', { page: 1, per_page: 100 }, true);
  for (const o of page.items) {
    if (o.status === 'DELIVERED') o.items.forEach((i) => ids.add(i.product_id));
  }
  return ids;
};

