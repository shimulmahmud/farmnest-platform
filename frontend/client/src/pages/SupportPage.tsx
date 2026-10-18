import { useCallback, useEffect, useState } from 'react';
import { Link, Navigate, useLocation, useNavigate, useSearchParams } from 'react-router-dom';
import { AlertCircle, Headset, Loader2, MessageSquarePlus, ChevronRight } from 'lucide-react';
import { useAuth } from '@/context/AuthContext';
import { CATEGORIES, createTicket, listMyTickets, type TicketSummary } from '@/api/support';
import { statusStyle, statusText } from '@/components/supportStatus';

const field = 'w-full rounded-lg border border-gray-300 px-3 py-2.5 text-sm focus:outline-none focus:border-primary focus:ring-1 focus:ring-primary';

export default function SupportPage() {
  const { user, loading: authLoading } = useAuth();
  const location = useLocation();
  const navigate = useNavigate();
  const [params] = useSearchParams();

  const [tickets, setTickets] = useState<TicketSummary[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  // New message form. "?order=ORD-..." (from My orders) opens it with the order filled in.
  const orderFromUrl = params.get('order') ?? '';
  const [showForm, setShowForm] = useState(Boolean(orderFromUrl));
  const [subject, setSubject] = useState(orderFromUrl ? `Question about order ${orderFromUrl}` : '');
  const [category, setCategory] = useState(orderFromUrl ? 'ORDER' : 'OTHER');
  const [orderNumber, setOrderNumber] = useState(orderFromUrl);
  const [message, setMessage] = useState('');
  const [sending, setSending] = useState(false);
  const [formError, setFormError] = useState('');

  const load = useCallback(() => {
    listMyTickets()
      .then((p) => setTickets(p.items))
      .catch((e: unknown) => setError(e instanceof Error ? e.message : 'Could not load your messages'))
      .finally(() => setLoading(false));
  }, []);

  useEffect(() => {
    if (!user) return;
    load();
    const timer = window.setInterval(load, 30_000); // pick up replies without reloading
    return () => window.clearInterval(timer);
  }, [user, load]);

  if (authLoading) return <div className="min-h-[60vh] flex items-center justify-center text-gray-500" aria-busy="true">Loading…</div>;
  if (!user) return <Navigate to="/signin" replace state={{ from: location.pathname + location.search }} />;

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    setFormError('');
    if (subject.trim().length < 3) return setFormError('Please add a short subject.');
    if (!message.trim()) return setFormError('Please write your message.');
    setSending(true);
    try {
      const t = await createTicket({ subject: subject.trim(), category, order_number: orderNumber.trim() || undefined, message: message.trim() });
      navigate(`/support/${t.id}`);
    } catch (err) {
      setFormError(err instanceof Error ? err.message : 'Could not send your message.');
    } finally {
      setSending(false);
    }
  };

  return (
    <div className="bg-[#f7f8f6] min-h-[70vh] py-10">
      <div className="max-w-3xl mx-auto px-4 sm:px-6">
        <div className="flex flex-wrap items-center justify-between gap-3 mb-6">
          <div>
            <h1 className="font-headline text-3xl font-bold text-primary flex items-center gap-3"><Headset className="w-7 h-7" /> Help &amp; support</h1>
            <p className="text-sm text-gray-600 mt-1">Message our team. We reply here and by email.</p>
          </div>
          {!showForm && (
            <button onClick={() => setShowForm(true)} className="inline-flex items-center gap-2 rounded-lg bg-primary px-5 py-2.5 text-sm font-semibold text-white hover:bg-primary-hover">
              <MessageSquarePlus className="w-4 h-4" /> New message
            </button>
          )}
        </div>

        {showForm && (
          <form onSubmit={submit} className="bg-white rounded-2xl border border-gray-100 p-6 shadow-sm space-y-4 mb-8">
            <h2 className="font-headline text-lg font-bold">Send us a message</h2>
            {formError && (
              <div className="flex items-start gap-2 rounded-lg border border-red-200 bg-red-50 px-3 py-2.5 text-sm text-red-700">
                <AlertCircle className="w-4 h-4 mt-0.5 flex-shrink-0" /><span>{formError}</span>
              </div>
            )}
            <div className="grid gap-4 sm:grid-cols-2">
              <div>
                <label className="mb-1.5 block text-sm font-semibold text-gray-700">What is it about?</label>
                <select value={category} onChange={(e) => setCategory(e.target.value)} className={field}>
                  {CATEGORIES.map((c) => <option key={c.id} value={c.id}>{c.label}</option>)}
                </select>
              </div>
              <div>
                <label className="mb-1.5 block text-sm font-semibold text-gray-700">Order number (optional)</label>
                <input value={orderNumber} onChange={(e) => setOrderNumber(e.target.value)} placeholder="ORD-20261011-ABC123" className={field} />
              </div>
            </div>
            <div>
              <label className="mb-1.5 block text-sm font-semibold text-gray-700">Subject</label>
              <input value={subject} onChange={(e) => setSubject(e.target.value)} maxLength={120} className={field} />
            </div>
            <div>
              <label className="mb-1.5 block text-sm font-semibold text-gray-700">Message</label>
              <textarea value={message} onChange={(e) => setMessage(e.target.value)} rows={5} maxLength={2000} className={field} placeholder="Tell us what happened and how we can help." />
              <p className="mt-1 text-right text-xs text-gray-400">{message.length}/2000</p>
            </div>
            <div className="flex justify-end gap-3">
              <button type="button" onClick={() => setShowForm(false)} className="rounded-lg border border-gray-300 px-5 py-2.5 text-sm font-semibold text-gray-700 hover:bg-gray-50">Cancel</button>
              <button type="submit" disabled={sending} className="inline-flex items-center gap-2 rounded-lg bg-primary px-6 py-2.5 text-sm font-semibold text-white hover:bg-primary-hover disabled:opacity-60">
                {sending && <Loader2 className="w-4 h-4 animate-spin" />} Send message
              </button>
            </div>
          </form>
        )}

        {loading && <p className="flex items-center gap-2 text-gray-500"><Loader2 className="w-4 h-4 animate-spin" /> Loading…</p>}
        {error && <p className="text-sm text-red-600">{error}</p>}
        {!loading && !error && tickets.length === 0 && !showForm && (
          <div className="text-center bg-white rounded-2xl border border-gray-100 p-10">
            <p className="font-semibold text-gray-700">You have no messages yet.</p>
            <p className="text-sm text-gray-500 mt-1">Need help with an order or a product? Send us a message.</p>
          </div>
        )}

        <ul className="space-y-3">
          {tickets.map((t) => (
            <li key={t.id}>
              <Link to={`/support/${t.id}`} className="flex items-center gap-4 bg-white rounded-2xl border border-gray-100 p-5 shadow-sm hover:border-primary/40 transition-colors">
                <div className="min-w-0 flex-1">
                  <div className="flex flex-wrap items-center gap-2">
                    <p className={`truncate ${t.customer_unread ? 'font-bold text-gray-900' : 'font-semibold text-gray-800'}`}>{t.subject}</p>
                    {t.customer_unread && <span className="rounded-full bg-primary px-2 py-0.5 text-[10px] font-bold text-white">New reply</span>}
                  </div>
                  {t.last_message && (
                    <p className="mt-1 truncate text-sm text-gray-500">{t.last_message.sender_role === 'STAFF' ? 'Support: ' : 'You: '}{t.last_message.body}</p>
                  )}
                  <p className="mt-1 text-xs text-gray-400">
                    {t.number}{t.order_number ? ` · ${t.order_number}` : ''} · {new Date(t.last_message_at).toLocaleString('en-GB', { dateStyle: 'medium', timeStyle: 'short' })}
                  </p>
                </div>
                <span className={`rounded-full border px-3 py-1 text-xs font-semibold ${statusStyle(t.status)}`}>{statusText(t.status)}</span>
                <ChevronRight className="w-4 h-4 text-gray-300" />
              </Link>
            </li>
          ))}
        </ul>
      </div>
    </div>
  );
}
