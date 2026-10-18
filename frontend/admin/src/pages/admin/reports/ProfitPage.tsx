import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { getProfitReport, getProductProfit, getOrderProfit } from '../../../api/reports';
import { formatCurrency, formatDateTime } from '../../../lib/utils';
import LoadingSpinner from '../../../components/common/LoadingSpinner';
import ErrorMessage from '../../../components/common/ErrorMessage';
import StatusBadge from '../../../components/common/StatusBadge';

const ymd = (d: Date) => {
  const p = (n: number) => String(n).padStart(2, '0');
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())}`;
};

const today = new Date();
const monthStart = new Date(today.getFullYear(), today.getMonth(), 1);

const BASES = [
  { id: 'SHIPPED,DELIVERED', label: 'Shipped + delivered' },
  { id: 'DELIVERED', label: 'Delivered only' },
];

function Stat({ label, value, hint, tone }: { label: string; value: string; hint?: string; tone?: 'good' | 'bad' }) {
  return (
    <div className="card">
      <p className="text-xs font-medium uppercase text-gray-500">{label}</p>
      <p className={`mt-1 text-2xl font-bold ${tone === 'bad' ? 'text-red-600' : tone === 'good' ? 'text-emerald-600' : 'text-gray-900'}`}>{value}</p>
      {hint && <p className="mt-1 text-xs text-gray-500">{hint}</p>}
    </div>
  );
}

export default function ProfitPage() {
  const [from, setFrom] = useState(ymd(monthStart));
  const [to, setTo] = useState(ymd(today));
  const [basis, setBasis] = useState(BASES[0].id);
  const [groupBy, setGroupBy] = useState<'day' | 'month'>('day');
  const [tab, setTab] = useState<'products' | 'worst' | 'orders'>('products');

  const range = { from, to, statuses: basis };

  const summaryQuery = useQuery({
    queryKey: ['admin', 'profit', from, to, basis, groupBy],
    queryFn: () => getProfitReport(range, groupBy),
  });
  const productsQuery = useQuery({
    queryKey: ['admin', 'profit-products', from, to, basis, tab],
    queryFn: () => getProductProfit(range, tab === 'worst' ? 'worst' : 'best', 20),
    enabled: tab !== 'orders',
  });
  const ordersQuery = useQuery({
    queryKey: ['admin', 'profit-orders', from, to, basis],
    queryFn: () => getOrderProfit(range, 1, 50),
    enabled: tab === 'orders',
  });

  const s = summaryQuery.data?.summary;
  const trend = summaryQuery.data?.trend ?? [];
  const maxRevenue = Math.max(1, ...trend.map((t) => t.revenue));

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-gray-900">Profit</h1>
        <p className="mt-1 text-sm text-gray-500">
          Gross profit = sales − what the goods cost you. Net profit also adds shipping charged and subtracts courier charges.
        </p>
      </div>

      <div className="card flex flex-wrap items-end gap-4">
        <div>
          <label className="label-field">From</label>
          <input type="date" value={from} onChange={(e) => setFrom(e.target.value)} className="input-field" />
        </div>
        <div>
          <label className="label-field">To</label>
          <input type="date" value={to} onChange={(e) => setTo(e.target.value)} className="input-field" />
        </div>
        <div>
          <label className="label-field">Count orders that are</label>
          <select value={basis} onChange={(e) => setBasis(e.target.value)} className="input-field">
            {BASES.map((b) => <option key={b.id} value={b.id}>{b.label}</option>)}
          </select>
        </div>
        <div>
          <label className="label-field">Trend by</label>
          <select value={groupBy} onChange={(e) => setGroupBy(e.target.value as 'day' | 'month')} className="input-field">
            <option value="day">Day</option>
            <option value="month">Month</option>
          </select>
        </div>
      </div>

      {summaryQuery.isLoading && <LoadingSpinner />}
      {summaryQuery.isError && <ErrorMessage message={summaryQuery.error?.message || 'Failed to load report'} onRetry={() => summaryQuery.refetch()} />}

      {s && (
        <>
          {s.items_missing_cost > 0 && (
            <div className="rounded-lg bg-amber-50 border border-amber-200 p-3 text-sm text-amber-800">
              {s.items_missing_cost} order item(s) have no recorded cost (older orders), so profit is overstated for them.
            </div>
          )}
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
            <Stat label="Orders" value={String(s.orders)} hint={`${s.units_sold} units sold`} />
            <Stat label="Sales (items)" value={formatCurrency(s.items_revenue)} />
            <Stat label="Coupon discounts" value={formatCurrency(s.discounts)} hint="given to customers" />
            <Stat label="Cost of goods" value={formatCurrency(s.cost_of_goods)} />
            <Stat label="Gross profit" value={formatCurrency(s.gross_profit)} hint={`${s.margin_percent}% margin`} tone={s.gross_profit >= 0 ? 'good' : 'bad'} />
            <Stat label="Shipping charged" value={formatCurrency(s.shipping_charged)} />
            <Stat label="Courier charges" value={formatCurrency(s.courier_cost)} />
            <Stat label="Net profit" value={formatCurrency(s.net_profit)} hint="before gateway fees, ads, packaging" tone={s.net_profit >= 0 ? 'good' : 'bad'} />
          </div>

          <div className="card">
            <h2 className="text-lg font-semibold text-gray-900 mb-4">Trend</h2>
            {trend.length === 0 ? (
              <p className="text-sm text-gray-500">No orders in this period.</p>
            ) : (
              <div className="space-y-2">
                {trend.map((t) => (
                  <div key={t.period} className="grid grid-cols-12 items-center gap-2 text-xs">
                    <span className="col-span-2 text-gray-600">{t.period}</span>
                    <div className="col-span-6 h-4 rounded bg-gray-100 overflow-hidden">
                      <div className="h-full bg-primary-500" style={{ width: `${(t.revenue / maxRevenue) * 100}%` }} />
                    </div>
                    <span className="col-span-2 text-right text-gray-600">{formatCurrency(t.revenue)}</span>
                    <span className={`col-span-2 text-right font-medium ${t.gross_profit >= 0 ? 'text-emerald-600' : 'text-red-600'}`}>{formatCurrency(t.gross_profit)}</span>
                  </div>
                ))}
                <p className="text-xs text-gray-400 pt-1">Bar = sales, right column = gross profit</p>
              </div>
            )}
          </div>
        </>
      )}

      <div className="card space-y-4">
        <div className="flex gap-2">
          {[
            { id: 'products', label: 'Best products' },
            { id: 'worst', label: 'Lowest-profit products' },
            { id: 'orders', label: 'Each order' },
          ].map((t) => (
            <button
              key={t.id}
              onClick={() => setTab(t.id as typeof tab)}
              className={`rounded-lg px-3 py-1.5 text-xs font-medium ${tab === t.id ? 'bg-primary-600 text-white' : 'bg-white text-gray-600 border border-gray-300'}`}
            >
              {t.label}
            </button>
          ))}
        </div>

        {tab !== 'orders' ? (
          productsQuery.isLoading ? <LoadingSpinner /> : (productsQuery.data ?? []).length === 0 ? (
            <p className="text-sm text-gray-500">No sales in this period.</p>
          ) : (
            <table className="w-full text-sm">
              <thead>
                <tr className="text-left text-xs uppercase text-gray-500 border-b">
                  <th className="py-2">Product</th><th className="py-2 text-right">Units</th><th className="py-2 text-right">Sales</th><th className="py-2 text-right">Cost</th><th className="py-2 text-right">Profit</th><th className="py-2 text-right">Margin</th>
                </tr>
              </thead>
              <tbody>
                {productsQuery.data!.map((p) => (
                  <tr key={p.sku} className="border-b border-gray-100">
                    <td className="py-2"><p className="font-medium text-gray-900">{p.name}</p><p className="text-xs text-gray-500 font-mono">{p.sku}</p></td>
                    <td className="py-2 text-right">{p.units_sold}</td>
                    <td className="py-2 text-right">{formatCurrency(p.revenue)}</td>
                    <td className="py-2 text-right">{formatCurrency(p.cost_of_goods)}</td>
                    <td className={`py-2 text-right font-medium ${p.gross_profit >= 0 ? 'text-emerald-600' : 'text-red-600'}`}>{formatCurrency(p.gross_profit)}</td>
                    <td className="py-2 text-right">{p.margin_percent}%</td>
                  </tr>
                ))}
              </tbody>
            </table>
          )
        ) : ordersQuery.isLoading ? <LoadingSpinner /> : (ordersQuery.data?.data ?? []).length === 0 ? (
          <p className="text-sm text-gray-500">No orders in this period.</p>
        ) : (
          <table className="w-full text-sm">
            <thead>
              <tr className="text-left text-xs uppercase text-gray-500 border-b">
                <th className="py-2">Order</th><th className="py-2">Status</th><th className="py-2 text-right">Sales</th><th className="py-2 text-right">Cost</th><th className="py-2 text-right">Gross</th><th className="py-2 text-right">Net</th>
              </tr>
            </thead>
            <tbody>
              {ordersQuery.data!.data.map((o) => (
                <tr key={o.order_id} className="border-b border-gray-100">
                  <td className="py-2"><p className="font-medium text-gray-900">{o.order_number}</p><p className="text-xs text-gray-500">{formatDateTime(o.created_at)}</p></td>
                  <td className="py-2"><StatusBadge status={o.status} type="order" /></td>
                  <td className="py-2 text-right">{formatCurrency(o.items_revenue)}</td>
                  <td className="py-2 text-right">{formatCurrency(o.cost_of_goods)}</td>
                  <td className={`py-2 text-right ${o.gross_profit >= 0 ? 'text-emerald-600' : 'text-red-600'}`}>{formatCurrency(o.gross_profit)}</td>
                  <td className={`py-2 text-right font-medium ${o.net_profit >= 0 ? 'text-emerald-600' : 'text-red-600'}`}>{formatCurrency(o.net_profit)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </div>
  );
}
