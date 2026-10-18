import { api } from './client';
import type {
  WishlistResponse,
  AddWishlistItemRequest,
  MessageResponse,
} from '../types';

export async function getWishlist(): Promise<WishlistResponse> {
  const res = await api.get<WishlistResponse>('/wishlist');
  return res.data;
}

export async function addItem(
  data: AddWishlistItemRequest,
): Promise<WishlistResponse> {
  const res = await api.post<WishlistResponse>('/wishlist/items', data);
  return res.data;
}

export async function removeItem(itemId: string): Promise<MessageResponse> {
  const res = await api.delete<MessageResponse>(`/wishlist/items/${itemId}`);
  return res.data;
}
