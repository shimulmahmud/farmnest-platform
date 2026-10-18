import { useCallback, useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { BadgeCheck, Loader2, Star } from 'lucide-react';
import { ApiError } from '@/api/http';
import { fetchMyDeliveredProductIds, fetchReviews, postReview, type ApiReview } from '@/api/reviews';
import { useAuth } from '@/context/AuthContext';
import type { Product } from '@/types';

function Stars({ value, size = 'h-4 w-4' }: { value: number; size?: string }) {
  return (
    <span className="inline-flex" aria-label={`${value} out of 5 stars`}>
      {[1, 2, 3, 4, 5].map((n) => (
        <Star key={n} className={`${size} ${n <= Math.round(value) ? 'fill-amber-400 text-amber-400' : 'text-gray-300'}`} />
      ))}
    </span>
  );
}

/** Customer reviews for one product. Only customers who received the product can write one; the shop approves it first. */
export default function ProductReviews({ product }: { product: Product }) {
  const { user } = useAuth();
  const [reviews, setReviews] = useState<ApiReview[]>([]);
  const [loading, setLoading] = useState(true);
  const [canReview, setCanReview] = useState(false);

  const [rating, setRating] = useState(0);
  const [title, setTitle] = useState('');
  const [comment, setComment] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState('');
  const [sent, setSent] = useState(false);

  const load = useCallback(() => {
    setLoading(true);
    fetchReviews(product.backendId)
      .then((p) => setReviews(p.items))
      .catch(() => setReviews([]))
      .finally(() => setLoading(false));
  }, [product.backendId]);

  useEffect(load, [load]);

  // Arriving from "Write a review" (#reviews): scroll to this section once it has loaded.
  useEffect(() => {
    if (!loading && window.location.hash === '#reviews') document.getElementById('reviews')?.scrollIntoView({ behavior: 'smooth' });
  }, [loading]);

  // Does this signed-in customer have a delivered order with this product?
  useEffect(() => {
    setCanReview(false);
    if (!user) return;
    fetchMyDeliveredProductIds()
      .then((ids) => setCanReview(ids.has(product.backendId)))
      .catch(() => setCanReview(false));
  }, [user, product.backendId]);

  const average = reviews.length ? reviews.reduce((n, r) => n + r.rating, 0) / reviews.length : 0;

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    if (rating < 1) return setError('Please choose a star rating.');
    if (!title.trim()) return setError('Please add a short title.');
    setSubmitting(true);
    try {
      await postReview(product.backendId, { rating, title: title.trim(), comment: comment.trim() });
      setSent(true);
      setRating(0);
      setTitle('');
      setComment('');
    } catch (err) {
      setError(
        err instanceof ApiError && err.status === 409
          ? 'You have already reviewed this product.'
          : err instanceof ApiError && err.status === 403
            ? 'Only customers who have received this product can review it.'
            : err instanceof Error ? err.message : 'Could not send your review.',
      );
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <section id="reviews" className="mt-16" aria-labelledby="reviews-heading">
      <h2 id="reviews-heading" className="font-headline text-2xl font-bold mb-6">Customer reviews</h2>

      <div className="grid gap-10 lg:grid-cols-[280px_1fr]">
        <div>
          {reviews.length > 0 ? (
            <div className="rounded-xl border border-gray-100 p-5">
              <p className="font-headline text-4xl font-bold">{average.toFixed(1)}</p>
              <Stars value={average} size="h-5 w-5" />
              <p className="mt-1 text-sm text-gray-500">{reviews.length} review{reviews.length === 1 ? '' : 's'}</p>
            </div>
          ) : (
            <p className="text-sm text-gray-500">{loading ? 'Loading reviews…' : 'No reviews yet.'}</p>
          )}

          <div className="mt-5 rounded-xl bg-gray-50 p-5 text-sm">
            {sent ? (
              <p className="text-emerald-700">Thank you! Your review will appear after it has been approved.</p>
            ) : !user ? (
              <p className="text-gray-600"><Link to="/signin" state={{ from: `/products/${product.id}` }} className="font-semibold text-primary hover:underline">Sign in</Link> to review products you have bought.</p>
            ) : !canReview ? (
              <p className="text-gray-600">You can review this product once your order has been delivered.</p>
            ) : (
              <form onSubmit={submit} className="space-y-3">
                <p className="font-semibold text-gray-800">Write a review</p>
                <div className="flex gap-1" role="radiogroup" aria-label="Your rating">
                  {[1, 2, 3, 4, 5].map((n) => (
                    <button key={n} type="button" role="radio" aria-checked={rating === n} onClick={() => setRating(n)} aria-label={`${n} star${n > 1 ? 's' : ''}`}>
                      <Star className={`h-7 w-7 ${n <= rating ? 'fill-amber-400 text-amber-400' : 'text-gray-300'}`} />
                    </button>
                  ))}
                </div>
                <input value={title} onChange={(e) => setTitle(e.target.value)} placeholder="Title" maxLength={100} className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary" />
                <textarea value={comment} onChange={(e) => setComment(e.target.value)} placeholder="What did you think? (optional)" rows={4} maxLength={2000} className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary" />
                {error && <p className="text-xs text-red-600">{error}</p>}
                <button type="submit" disabled={submitting} className="flex items-center gap-2 rounded-lg bg-primary px-5 py-2 font-semibold text-white hover:bg-primary-hover disabled:opacity-60">
                  {submitting && <Loader2 className="h-4 w-4 animate-spin" />} Submit review
                </button>
              </form>
            )}
          </div>
        </div>

        <ul className="space-y-6">
          {reviews.map((r) => (
            <li key={r.id} className="border-b border-gray-100 pb-6">
              <div className="flex flex-wrap items-center gap-2">
                <Stars value={r.rating} />
                <strong className="text-sm">{r.title}</strong>
              </div>
              {r.comment && <p className="mt-2 text-sm leading-relaxed text-gray-700 whitespace-pre-line">{r.comment}</p>}
              <p className="mt-2 flex flex-wrap items-center gap-2 text-xs text-gray-500">
                <span>{r.author_name || 'Customer'}</span>
                <span>· {new Date(r.created_at).toLocaleDateString('en-GB', { dateStyle: 'medium' })}</span>
                {r.verified_purchase && <span className="inline-flex items-center gap-1 text-emerald-700"><BadgeCheck className="h-3.5 w-3.5" /> Verified purchase</span>}
              </p>
            </li>
          ))}
        </ul>
      </div>
    </section>
  );
}
