import { useEffect, useRef, useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { PaperAirplaneIcon, MagnifyingGlassIcon } from '@heroicons/react/24/outline';
import {
  getSupportSummary, getSupportTicket, listSupportTickets, replySupportTicket, setSupportStatus,
  type TicketStatus,
} from '../../../api/support';
import { cn } from '../../../lib/utils';
import LoadingSpinner from '../../../components/common/LoadingSpinner';

const FILTERS: { id: string; label: string }[] = [
  { id: 'OPEN', label: 'Needs reply' },
  { id: 'ANSWERED', label: 'Answered' },
  { id: 'CLOSED', label: 'Closed' },
  { id: '', label: 'All' },
];

const when = (iso: string) =>
  new Date(iso).toLocaleString('en-GB', { dateStyle: 'medium', timeStyle: 'short' });

const statusBadge = (s: TicketStatus) =>
  s === 'OPEN' ? 'badge-yellow' : s === 'ANSWERED' ? 'badge-green' : 'badge-gray';

export default function SupportPage() {
  const queryClient = useQueryClient();
  const [searchParams, setSearchParams] = useSearchParams();
  const selected = searchParams.get('ticket') ?? '';
  const [filter, setFilter] = useState('OPEN');
  const [search, setSearch] = useState('');
  const [reply, setReply] = useState('');
  const [error, setError] = useState('');
  const bottom = useRef<HTMLDivElement>(null);
  const lastCount = useRef(0);

  const summaryQuery = useQuery({ queryKey: ['admin', 'support-summary'], queryFn: getSupportSummary, refetchInterval: 20_000 });
  const listQuery = useQuery({
    queryKey: ['admin', 'support-list', filter, search],
    queryFn: () => listSupportTickets(1, { status: filter || undefined, q: search.trim() || undefined }),
    refetchInterval: 20_000,
  });
  const ticketQuery = useQuery({
    queryKey: ['admin', 'support-ticket', selected],
    queryFn: () => getSupportTicket(selected),
    enabled: Boolean(selected),
    refetchInterval: 10_000,
  });

  const refresh = () => queryClient.invalidateQueries({ queryKey: ['admin'] });

  const replyMutation = useMutation({
    mutationFn: () => replySupportTicket(selected, reply.trim()),
    onSuccess: () => { setReply(''); setError(''); refresh(); },
    onError: (e: unknown) => setError(e instanceof Error ? e.message : 'Could not send the reply'),
  });
  const statusMutation = useMutation({
    mutationFn: (status: TicketStatus) => setSupportStatus(selected, status),
    onSuccess: refresh,
    onError: (e: unknown) => setError(e instanceof Error ? e.message : 'Could not change the status'),
  });

  const ticket = ticketQuery.data;
  useEffect(() => {
    const n = ticket?.messages.length ?? 0;
    if (n > lastCount.current) bottom.current?.scrollIntoView({ behavior: 'smooth' });
    lastCount.current = n;
  }, [ticket?.messages.length, ticket?.id]);

  const list = listQuery.data?.data ?? [];
  const s = summaryQuery.data;
  const count = (id: string) => (id === 'OPEN' ? s?.open : id === 'ANSWERED' ? s?.answered : id === 'CLOSED' ? s?.closed : undefined);

  const open = (id: string) => {
    const next = new URLSearchParams(searchParams);
    next.set('ticket', id);
    setSearchParams(next);
    lastCount.current = 0;
    setError('');
  };

  return (
    <div className="space-y-4">
      <div>
        <h1 className="text-2xl font-bold text-gray-900">Support messages</h1>
        <p className="mt-1 text-sm text-gray-500">Answer customer questions. Replies are shown to the customer and emailed to them.</p>
      </div>

      <div className="grid gap-4 lg:grid-cols-[360px_1fr] items-start">
        {/* List */}
        <div className="card p-0 overflow-hidden">
          <div className="border-b border-gray-100 p-3 space-y-3">
            <div className="flex flex-wrap gap-1.5">
              {FILTERS.map((f) => (
                <button
                  key={f.id || 'all'}
                  onClick={() => setFilter(f.id)}
                  className={cn('rounded-lg px-2.5 py-1 text-xs font-medium', filter === f.id ? 'bg-primary-600 text-white' : 'border border-gray-300 text-gray-600 hover:bg-gray-50')}
                >
                  {f.label}{count(f.id) !== undefined ? ` (${count(f.id)})` : ''}
                </button>
              ))}
            </div>
            <div className="relative">
              <MagnifyingGlassIcon className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-gray-400" />
              <input value={search} onChange={(e) => setSearch(e.target.value)} placeholder="Search name, email, order, subject" className="input-field pl-9" />
            </div>
          </div>

          <ul className="max-h-[68vh] divide-y divide-gray-100 overflow-y-auto">
            {listQuery.isLoading && <li className="p-6"><LoadingSpinner /></li>}
            {!listQuery.isLoading && list.length === 0 && <li className="p-6 text-center text-sm text-gray-500">No conversations here.</li>}
            {list.map((t) => (
              <li key={t.id}>
                <button onClick={() => open(t.id)} className={cn('w-full p-4 text-left hover:bg-gray-50', selected === t.id && 'bg-primary-50')}>
                  <div className="flex items-center justify-between gap-2">
                    <p className={cn('truncate text-sm', t.staff_unread ? 'font-bold text-gray-900' : 'font-medium text-gray-800')}>
                      {t.staff_unread && <span className="mr-1.5 inline-block h-2 w-2 rounded-full bg-red-500 align-middle" />}
                      {t.subject}
                    </p>
                    <span className={statusBadge(t.status)}>{t.status === 'OPEN' ? 'Needs reply' : t.status === 'ANSWERED' ? 'Answered' : 'Closed'}</span>
                  </div>
                  <p className="mt-0.5 truncate text-xs text-gray-500">{t.customer_name} · {t.customer_email}</p>
                  {t.last_message && (
                    <p className="mt-1 truncate text-xs text-gray-400">{t.last_message.sender_role === 'STAFF' ? 'You: ' : ''}{t.last_message.body}</p>
                  )}
                  <p className="mt-1 text-[11px] text-gray-400">{t.number}{t.order_number ? ` · ${t.order_number}` : ''} · {when(t.last_message_at)}</p>
                </button>
              </li>
            ))}
          </ul>
        </div>

        {/* Conversation */}
        <div className="card p-0 overflow-hidden">
          {!selected ? (
            <div className="p-16 text-center text-sm text-gray-500">Pick a conversation on the left.</div>
          ) : ticketQuery.isLoading ? (
            <div className="p-10"><LoadingSpinner /></div>
          ) : !ticket ? (
            <div className="p-10 text-center text-sm text-red-600">Could not load this conversation.</div>
          ) : (
            <>
              <header className="flex flex-wrap items-start justify-between gap-3 border-b border-gray-100 p-4">
                <div className="min-w-0">
                  <h2 className="truncate text-lg font-semibold text-gray-900">{ticket.subject}</h2>
                  <p className="text-xs text-gray-500">
                    {ticket.customer_name} · <a href={`mailto:${ticket.customer_email}`} className="text-primary-600 hover:underline">{ticket.customer_email}</a>
                    {' · '}{ticket.number}{ticket.order_number ? ` · Order ${ticket.order_number}` : ''} · {ticket.category.toLowerCase()}
                  </p>
                </div>
                <div className="flex items-center gap-2">
                  <span className={statusBadge(ticket.status)}>{ticket.status === 'OPEN' ? 'Needs reply' : ticket.status === 'ANSWERED' ? 'Answered' : 'Closed'}</span>
                  {ticket.status === 'CLOSED' ? (
                    <button onClick={() => statusMutation.mutate('OPEN')} className="btn-secondary text-xs">Reopen</button>
                  ) : (
                    <button onClick={() => statusMutation.mutate('CLOSED')} className="btn-secondary text-xs">Close</button>
                  )}
                </div>
              </header>

              <div className="max-h-[52vh] space-y-3 overflow-y-auto bg-gray-50 p-4">
                {ticket.messages.map((m) => {
                  const staff = m.sender_role === 'STAFF';
                  return (
                    <div key={m.id} className={cn('flex', staff ? 'justify-end' : 'justify-start')}>
                      <div className={cn('max-w-[80%] rounded-2xl px-4 py-2.5 text-sm', staff ? 'bg-primary-600 text-white rounded-br-sm' : 'bg-white border border-gray-200 text-gray-800 rounded-bl-sm')}>
                        <p className={cn('mb-0.5 text-xs font-semibold', staff ? 'text-white/80' : 'text-gray-500')}>{m.sender_name}</p>
                        <p className="whitespace-pre-wrap break-words">{m.body}</p>
                        <p className={cn('mt-1 text-[10px]', staff ? 'text-white/70' : 'text-gray-400')}>{when(m.created_at)}</p>
                      </div>
                    </div>
                  );
                })}
                <div ref={bottom} />
              </div>

              <form
                className="border-t border-gray-100 p-4"
                onSubmit={(e) => { e.preventDefault(); if (reply.trim()) replyMutation.mutate(); }}
              >
                {error && <p className="mb-2 text-xs text-red-600">{error}</p>}
                <div className="flex gap-3">
                  <textarea
                    value={reply}
                    onChange={(e) => setReply(e.target.value)}
                    onKeyDown={(e) => { if ((e.ctrlKey || e.metaKey) && e.key === 'Enter' && reply.trim()) replyMutation.mutate(); }}
                    rows={3}
                    maxLength={2000}
                    placeholder="Write your reply… (Ctrl+Enter to send)"
                    className="input-field flex-1 resize-none"
                  />
                  <button type="submit" disabled={replyMutation.isPending || !reply.trim()} className="btn-primary self-end">
                    <PaperAirplaneIcon className="mr-1 h-4 w-4" /> {replyMutation.isPending ? 'Sending…' : 'Send'}
                  </button>
                </div>
              </form>
            </>
          )}
        </div>
      </div>
    </div>
  );
}
