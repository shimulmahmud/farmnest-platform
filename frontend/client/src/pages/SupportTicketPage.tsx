import { useCallback, useEffect, useRef, useState } from 'react';
import { Link, Navigate, useLocation, useParams } from 'react-router-dom';
import { AlertCircle, ArrowLeft, Loader2, Send } from 'lucide-react';
import { useAuth } from '@/context/AuthContext';
import { closeTicket, getTicket, replyTicket, type Ticket } from '@/api/support';
import { ApiError } from '@/api/http';
import { statusStyle, statusText } from '@/components/supportStatus';

export default function SupportTicketPage() {
  const { id = '' } = useParams();
  const { user, loading: authLoading } = useAuth();
  const location = useLocation();

  const [ticket, setTicket] = useState<Ticket | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [reply, setReply] = useState('');
  const [sending, setSending] = useState(false);
  const [sendError, setSendError] = useState('');
  const bottom = useRef<HTMLDivElement>(null);
  const lastCount = useRef(0);

  const load = useCallback(() => {
    getTicket(id)
      .then((t) => { setTicket(t); setError(''); })
      .catch((e: unknown) => setError(e instanceof ApiError && e.status === 404 ? 'We could not find this conversation.' : e instanceof Error ? e.message : 'Could not load the conversation'))
      .finally(() => setLoading(false));
  }, [id]);

  useEffect(() => {
    if (!user) return;
    load();
    const timer = window.setInterval(load, 15_000); // new replies appear without reloading
    return () => window.clearInterval(timer);
  }, [user, load]);

  // Scroll to the newest message when one arrives.
  useEffect(() => {
    const n = ticket?.messages.length ?? 0;
    if (n > lastCount.current) bottom.current?.scrollIntoView({ behavior: lastCount.current === 0 ? 'auto' : 'smooth' });
    lastCount.current = n;
  }, [ticket?.messages.length]);

  if (authLoading) return <div className="min-h-[60vh] flex items-center justify-center text-gray-500" aria-busy="true">Loading…</div>;
  if (!user) return <Navigate to="/signin" replace state={{ from: location.pathname }} />;

  const send = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!reply.trim()) return;
    setSendError('');
    setSending(true);
    try {
      setTicket(await replyTicket(id, reply.trim()));
      setReply('');
    } catch (err) {
      setSendError(err instanceof Error ? err.message : 'Could not send your message.');
    } finally {
      setSending(false);
    }
  };

  const close = async () => {
    try {
      setTicket(await closeTicket(id));
    } catch (err) {
      setSendError(err instanceof Error ? err.message : 'Could not close the conversation.');
    }
  };

  return (
    <div className="bg-[#f7f8f6] min-h-[70vh] py-8">
      <div className="max-w-3xl mx-auto px-4 sm:px-6">
        <Link to="/support" className="inline-flex items-center gap-2 text-sm font-semibold text-gray-600 hover:text-primary mb-5">
          <ArrowLeft className="w-4 h-4" /> All messages
        </Link>

        {loading && <p className="flex items-center gap-2 text-gray-500"><Loader2 className="w-4 h-4 animate-spin" /> Loading…</p>}
        {error && !ticket && (
          <div className="flex items-start gap-2 rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700">
            <AlertCircle className="w-4 h-4 mt-0.5 flex-shrink-0" /><span>{error}</span>
          </div>
        )}

        {ticket && (
          <div className="bg-white rounded-2xl border border-gray-100 shadow-sm">
            <header className="flex flex-wrap items-start justify-between gap-3 border-b border-gray-100 p-5">
              <div className="min-w-0">
                <h1 className="font-headline text-xl font-bold text-gray-900">{ticket.subject}</h1>
                <p className="mt-1 text-xs text-gray-500">{ticket.number}{ticket.order_number ? ` · Order ${ticket.order_number}` : ''}</p>
              </div>
              <div className="flex items-center gap-3">
                <span className={`rounded-full border px-3 py-1 text-xs font-semibold ${statusStyle(ticket.status)}`}>{statusText(ticket.status)}</span>
                {ticket.status !== 'CLOSED' && (
                  <button onClick={close} className="text-xs font-semibold text-gray-500 hover:text-red-600">Close</button>
                )}
              </div>
            </header>

            <div className="max-h-[55vh] space-y-4 overflow-y-auto p-5" aria-live="polite">
              {ticket.messages.map((m) => {
                const mine = m.sender_role === 'CUSTOMER';
                return (
                  <div key={m.id} className={`flex ${mine ? 'justify-end' : 'justify-start'}`}>
                    <div className={`max-w-[85%] rounded-2xl px-4 py-3 text-sm ${mine ? 'bg-primary text-white rounded-br-sm' : 'bg-gray-100 text-gray-800 rounded-bl-sm'}`}>
                      {!mine && <p className="mb-1 text-xs font-semibold text-gray-500">{m.sender_name}</p>}
                      <p className="whitespace-pre-wrap break-words">{m.body}</p>
                      <p className={`mt-1 text-[10px] ${mine ? 'text-white/70' : 'text-gray-400'}`}>
                        {new Date(m.created_at).toLocaleString('en-GB', { dateStyle: 'medium', timeStyle: 'short' })}
                      </p>
                    </div>
                  </div>
                );
              })}
              <div ref={bottom} />
            </div>

            <form onSubmit={send} className="border-t border-gray-100 p-4">
              {ticket.status === 'CLOSED' && (
                <p className="mb-2 text-xs text-gray-500">This conversation is closed. Writing again will reopen it.</p>
              )}
              {sendError && <p className="mb-2 text-xs text-red-600">{sendError}</p>}
              <div className="flex gap-3">
                <textarea
                  value={reply}
                  onChange={(e) => setReply(e.target.value)}
                  onKeyDown={(e) => { if ((e.ctrlKey || e.metaKey) && e.key === 'Enter') send(e); }}
                  rows={2}
                  maxLength={2000}
                  placeholder="Write a reply…"
                  aria-label="Your reply"
                  className="min-w-0 flex-1 resize-none rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
                />
                <button type="submit" disabled={sending || !reply.trim()} className="inline-flex items-center gap-2 self-end rounded-lg bg-primary px-5 py-2.5 text-sm font-semibold text-white hover:bg-primary-hover disabled:opacity-50">
                  {sending ? <Loader2 className="w-4 h-4 animate-spin" /> : <Send className="w-4 h-4" />} Send
                </button>
              </div>
            </form>
          </div>
        )}
      </div>
    </div>
  );
}
