import { useState, useMemo } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  PlusIcon,
  AdjustmentsHorizontalIcon,
  MagnifyingGlassIcon,
} from '@heroicons/react/24/outline';
import { Dialog, DialogPanel, DialogTitle } from '@headlessui/react';
import {
  listWarehouses,
  createWarehouse,
  createStock,
  adjustStock,
  listReservations,
  release,
  consume,
  listMovements,
  listLowStock,
} from '../../../api/inventory';
import { formatDateTime, cn } from '../../../lib/utils';
import DataTable from '../../../components/common/DataTable';
import LoadingSpinner from '../../../components/common/LoadingSpinner';
import ErrorMessage from '../../../components/common/ErrorMessage';
import EmptyState from '../../../components/common/EmptyState';
import Pagination from '../../../components/common/Pagination';
import type {
  StockResponse,
  WarehouseResponse,
  ReservationResponse,
  MovementResponse,
  CreateWarehouseRequest,
  CreateStockRequest,
  AdjustStockRequest,
} from '../../../types';

const TABS = ['Stock', 'Warehouses', 'Reservations', 'Movements', 'Low Stock'] as const;
type Tab = (typeof TABS)[number];

const RESERVATION_STATUSES = ['ALL', 'ACTIVE', 'RELEASED', 'CONSUMED', 'EXPIRED'] as const;
type ReservationStatusFilter = (typeof RESERVATION_STATUSES)[number];

export default function InventoryPage() {
  const queryClient = useQueryClient();
  const [activeTab, setActiveTab] = useState<Tab>('Stock');

  // --- Stock Tab ---
  const [createStockOpen, setCreateStockOpen] = useState(false);
  const [adjustStockOpen, setAdjustStockOpen] = useState(false);
  const stockSearch = '';
  const [stockPage, setStockPage] = useState(1);

  // Create stock form
  const [csForm, setCsForm] = useState<CreateStockRequest>({
    sku: '',
    warehouse_id: '',
    total_stock: 0,
    low_stock_threshold: 5,
  });

  // Adjust stock form
  const [asForm, setAsForm] = useState<AdjustStockRequest>({
    sku: '',
    warehouse_id: '',
    quantity: 0,
    reason: '',
  });

  // --- Warehouse Tab ---
  const [warehouseOpen, setWarehouseOpen] = useState(false);
  const [whForm, setWhForm] = useState<CreateWarehouseRequest>({
    name: '',
    code: '',
    address: { line1: '', city: '', state: '', postal_code: '', country: '' },
  });

  // --- Reservations Tab ---
  const [resFilter, setResFilter] = useState<ReservationStatusFilter>('ALL');
  const [resPage, setResPage] = useState(1);

  // --- Movements Tab ---
  const [movPage, setMovPage] = useState(1);
  const [movSkuSearch, setMovSkuSearch] = useState('');

  // --- Low Stock Tab ---
  const [lowPage, setLowPage] = useState(1);

  // ===== Queries =====
  const warehousesQuery = useQuery({
    queryKey: ['warehouses'],
    queryFn: listWarehouses,
  });

  const warehouses = useMemo(
    () => warehousesQuery.data?.data ?? [],
    [warehousesQuery.data],
  );

  const warehouseMap = useMemo(() => {
    const map = new Map<string, string>();
    for (const wh of warehouses) {
      map.set(wh.id, `${wh.name} (${wh.code})`);
    }
    return map;
  }, [warehouses]);

  // Stock query
  const stockQuery = useQuery({
    queryKey: ['admin', 'stock', stockPage, stockSearch],
    queryFn: () =>
      listLowStock(stockPage, 20), // Re-use low-stock list as stock listing
    enabled: activeTab === 'Stock',
  });

  // Low stock query
  const lowStockQuery = useQuery({
    queryKey: ['admin', 'low-stock', lowPage],
    queryFn: () => listLowStock(lowPage, 20),
    enabled: activeTab === 'Low Stock',
  });

  // Reservations query
  const reservationsQuery = useQuery({
    queryKey: ['admin', 'reservations', resPage, resFilter],
    queryFn: () =>
      listReservations(
        undefined,
        undefined,
        resFilter === 'ALL' ? undefined : resFilter,
        resPage,
        20,
      ),
    enabled: activeTab === 'Reservations',
  });

  // Movements query
  const movementsQuery = useQuery({
    queryKey: ['admin', 'movements', movPage, movSkuSearch],
    queryFn: () =>
      listMovements(
        movSkuSearch || undefined,
        undefined,
        movPage,
        20,
      ),
    enabled: activeTab === 'Movements',
  });

  // ===== Mutations =====
  const createStockMutation = useMutation({
    mutationFn: createStock,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin', 'stock'] });
      queryClient.invalidateQueries({ queryKey: ['admin', 'low-stock'] });
      setCreateStockOpen(false);
      setCsForm({ sku: '', warehouse_id: '', total_stock: 0, low_stock_threshold: 5 });
    },
  });

  const adjustStockMutation = useMutation({
    mutationFn: adjustStock,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin', 'stock'] });
      queryClient.invalidateQueries({ queryKey: ['admin', 'low-stock'] });
      queryClient.invalidateQueries({ queryKey: ['admin', 'movements'] });
      setAdjustStockOpen(false);
      setAsForm({ sku: '', warehouse_id: '', quantity: 0, reason: '' });
    },
  });

  const createWarehouseMutation = useMutation({
    mutationFn: createWarehouse,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['warehouses'] });
      setWarehouseOpen(false);
      setWhForm({
        name: '',
        code: '',
        address: { line1: '', city: '', state: '', postal_code: '', country: '' },
      });
    },
  });

  const releaseMutation = useMutation({
    mutationFn: release,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin', 'reservations'] });
      queryClient.invalidateQueries({ queryKey: ['admin', 'stock'] });
    },
  });

  const consumeMutation = useMutation({
    mutationFn: consume,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin', 'reservations'] });
      queryClient.invalidateQueries({ queryKey: ['admin', 'stock'] });
    },
  });

  // ===== Render Helpers =====
  const renderTabContent = () => {
    switch (activeTab) {
      case 'Stock':
        return renderStockTab();
      case 'Warehouses':
        return renderWarehousesTab();
      case 'Reservations':
        return renderReservationsTab();
      case 'Movements':
        return renderMovementsTab();
      case 'Low Stock':
        return renderLowStockTab();
      default:
        return null;
    }
  };

  const renderStockTab = () => {
    if (stockQuery.isLoading) return <LoadingSpinner />;
    if (stockQuery.isError) {
      return (
        <ErrorMessage
          message={stockQuery.error?.message || 'Failed to load stock'}
          onRetry={() => stockQuery.refetch()}
        />
      );
    }

    const stockItems = stockQuery.data?.data ?? [];
    const pag = stockQuery.data?.pagination;

    if (stockItems.length === 0) {
      return (
        <EmptyState
          title="No stock items"
          description="Create stock entries to start tracking inventory."
          action={
            <button
              onClick={() => setCreateStockOpen(true)}
              className="btn-primary inline-flex items-center gap-2"
            >
              <PlusIcon className="h-4 w-4" />
              Create Stock
            </button>
          }
        />
      );
    }

    const columns = [
      { key: 'sku', header: 'SKU' },
      {
        key: 'warehouse_id',
        header: 'Warehouse',
        render: (row: StockResponse) => (
          <span className="text-gray-600">{warehouseMap.get(row.warehouse_id) || row.warehouse_id}</span>
        ),
      },
      { key: 'total_stock', header: 'Total Stock' },
      { key: 'reserved_stock', header: 'Reserved' },
      { key: 'available_stock', header: 'Available' },
      { key: 'low_stock_threshold', header: 'Threshold' },
      {
        key: 'is_low_stock',
        header: 'Low Stock',
        render: (row: StockResponse) => (
          <span className={row.is_low_stock ? 'badge-red' : 'badge-green'}>
            {row.is_low_stock ? 'Yes' : 'No'}
          </span>
        ),
      },
      {
        key: 'updated_at',
        header: 'Updated',
        render: (row: StockResponse) => (
          <span className="text-gray-500 text-sm">{formatDateTime(row.updated_at)}</span>
        ),
      },
    ];

    return (
      <>
        <div className="flex flex-wrap gap-3 mb-4">
          <button
            onClick={() => setCreateStockOpen(true)}
            className="btn-primary inline-flex items-center gap-2"
          >
            <PlusIcon className="h-4 w-4" />
            Create Stock
          </button>
          <button
            onClick={() => setAdjustStockOpen(true)}
            className="btn-secondary inline-flex items-center gap-2"
          >
            <AdjustmentsHorizontalIcon className="h-4 w-4" />
            Adjust Stock
          </button>
        </div>
        <DataTable columns={columns} data={stockItems as unknown as Record<string, unknown>[]} />
        {pag && (
          <div className="mt-4">
            <Pagination page={pag.page} totalPages={pag.total_pages} onPageChange={setStockPage} />
          </div>
        )}
      </>
    );
  };

  const renderWarehousesTab = () => {
    if (warehousesQuery.isLoading) return <LoadingSpinner />;
    if (warehousesQuery.isError) {
      return (
        <ErrorMessage
          message={warehousesQuery.error?.message || 'Failed to load warehouses'}
          onRetry={() => warehousesQuery.refetch()}
        />
      );
    }

    if (warehouses.length === 0) {
      return (
        <EmptyState
          title="No warehouses"
          description="Create your first warehouse to start managing inventory."
          action={
            <button
              onClick={() => setWarehouseOpen(true)}
              className="btn-primary inline-flex items-center gap-2"
            >
              <PlusIcon className="h-4 w-4" />
              New Warehouse
            </button>
          }
        />
      );
    }

    const columns = [
      {
        key: 'name',
        header: 'Name',
        render: (row: WarehouseResponse) => (
          <div>
            <p className="font-medium text-gray-900">{row.name}</p>
            <p className="text-xs text-gray-500">{row.code}</p>
          </div>
        ),
      },
      {
        key: 'address',
        header: 'Address',
        render: (row: WarehouseResponse) => (
          <span className="text-gray-600 text-sm">
            {row.address.line1}, {row.address.city}, {row.address.state} {row.address.postal_code}
          </span>
        ),
      },
      {
        key: 'is_active',
        header: 'Status',
        render: (row: WarehouseResponse) => (
          <span className={row.is_active ? 'badge-green' : 'badge-red'}>
            {row.is_active ? 'Active' : 'Inactive'}
          </span>
        ),
      },
      {
        key: 'created_at',
        header: 'Created',
        render: (row: WarehouseResponse) => (
          <span className="text-gray-500 text-sm">{formatDateTime(row.created_at)}</span>
        ),
      },
    ];

    return (
      <>
        <div className="flex gap-3 mb-4">
          <button
            onClick={() => setWarehouseOpen(true)}
            className="btn-primary inline-flex items-center gap-2"
          >
            <PlusIcon className="h-4 w-4" />
            New Warehouse
          </button>
        </div>
        <DataTable columns={columns} data={warehouses as unknown as Record<string, unknown>[]} />
      </>
    );
  };

  const renderReservationsTab = () => {
    if (reservationsQuery.isLoading) return <LoadingSpinner />;
    if (reservationsQuery.isError) {
      return (
        <ErrorMessage
          message={reservationsQuery.error?.message || 'Failed to load reservations'}
          onRetry={() => reservationsQuery.refetch()}
        />
      );
    }

    const reservations = reservationsQuery.data?.data ?? [];
    const pag = reservationsQuery.data?.pagination;

    const columns = [
      { key: 'id', header: 'ID', render: (row: ReservationResponse) => <span className="text-gray-500 text-xs font-mono">{row.id.slice(0, 8)}...</span> },
      { key: 'sku', header: 'SKU', render: (row: ReservationResponse) => <span className="font-medium">{row.sku}</span> },
      {
        key: 'warehouse_id',
        header: 'Warehouse',
        render: (row: ReservationResponse) => (
          <span className="text-gray-600">{warehouseMap.get(row.warehouse_id) || row.warehouse_id}</span>
        ),
      },
      { key: 'quantity', header: 'Qty' },
      { key: 'reference', header: 'Reference', render: (row: ReservationResponse) => <span className="text-gray-600 text-sm">{row.reference}</span> },
      {
        key: 'status',
        header: 'Status',
        render: (row: ReservationResponse) => (
          <span className={cn(
            row.status === 'ACTIVE' && 'badge-yellow',
            row.status === 'RELEASED' && 'badge-blue',
            row.status === 'CONSUMED' && 'badge-green',
            row.status === 'EXPIRED' && 'badge-gray',
          )}>
            {row.status.charAt(0) + row.status.slice(1).toLowerCase()}
          </span>
        ),
      },
      {
        key: 'expires_at',
        header: 'Expires',
        render: (row: ReservationResponse) => (
          <span className="text-gray-500 text-sm">{row.expires_at ? formatDateTime(row.expires_at) : '—'}</span>
        ),
      },
      {
        key: 'actions',
        header: 'Actions',
        render: (row: ReservationResponse) => (
          <div className="flex items-center gap-2">
            {row.status === 'ACTIVE' && (
              <>
                <button
                  onClick={(e) => {
                    e.stopPropagation();
                    releaseMutation.mutate(row.id);
                  }}
                  disabled={releaseMutation.isPending}
                  className="inline-flex items-center gap-1 rounded-lg px-2.5 py-1.5 text-xs font-medium text-primary-600 hover:bg-primary-50 transition-colors disabled:opacity-50"
                >
                  Release
                </button>
                <button
                  onClick={(e) => {
                    e.stopPropagation();
                    consumeMutation.mutate(row.id);
                  }}
                  disabled={consumeMutation.isPending}
                  className="inline-flex items-center gap-1 rounded-lg px-2.5 py-1.5 text-xs font-medium text-green-600 hover:bg-green-50 transition-colors disabled:opacity-50"
                >
                  Consume
                </button>
              </>
            )}
            <span className="text-xs text-gray-400">—</span>
          </div>
        ),
      },
    ];

    return (
      <>
        <div className="flex flex-wrap gap-2 mb-4">
          {RESERVATION_STATUSES.map((s) => (
            <button
              key={s}
              onClick={() => { setResFilter(s); setResPage(1); }}
              className={cn(
                'rounded-lg px-3 py-1.5 text-xs font-medium transition-colors',
                resFilter === s
                  ? 'bg-primary-600 text-white'
                  : 'bg-white text-gray-600 border border-gray-300 hover:bg-gray-50',
              )}
            >
              {s === 'ALL' ? 'All' : s.charAt(0) + s.slice(1).toLowerCase()}
            </button>
          ))}
        </div>
        {reservations.length === 0 ? (
          <EmptyState title="No reservations" description="No reservations match the current filter." />
        ) : (
          <>
            <DataTable columns={columns} data={reservations as unknown as Record<string, unknown>[]} />
            {pag && (
              <div className="mt-4">
                <Pagination page={pag.page} totalPages={pag.total_pages} onPageChange={setResPage} />
              </div>
            )}
          </>
        )}
      </>
    );
  };

  const renderMovementsTab = () => {
    if (movementsQuery.isLoading) return <LoadingSpinner />;
    if (movementsQuery.isError) {
      return (
        <ErrorMessage
          message={movementsQuery.error?.message || 'Failed to load movements'}
          onRetry={() => movementsQuery.refetch()}
        />
      );
    }

    const movements = movementsQuery.data?.data ?? [];
    const pag = movementsQuery.data?.pagination;

    const columns = [
      { key: 'id', header: 'ID', render: (row: MovementResponse) => <span className="text-gray-500 text-xs font-mono">{row.id.slice(0, 8)}...</span> },
      { key: 'sku', header: 'SKU', render: (row: MovementResponse) => <span className="font-medium">{row.sku}</span> },
      {
        key: 'warehouse_id',
        header: 'Warehouse',
        render: (row: MovementResponse) => (
          <span className="text-gray-600">{warehouseMap.get(row.warehouse_id) || row.warehouse_id}</span>
        ),
      },
      {
        key: 'type',
        header: 'Type',
        render: (row: MovementResponse) => (
          <span className={cn(
            row.type === 'IN' && 'badge-green',
            row.type === 'OUT' && 'badge-red',
            row.type === 'ADJUSTMENT' && 'badge-yellow',
            row.type === 'RESERVATION' && 'badge-blue',
            (!['IN', 'OUT', 'ADJUSTMENT', 'RESERVATION'].includes(row.type)) && 'badge-gray',
          )}>
            {row.type}
          </span>
        ),
      },
      {
        key: 'quantity',
        header: 'Qty',
        render: (row: MovementResponse) => (
          <span className={row.quantity >= 0 ? 'text-green-700' : 'text-red-700'}>
            {row.quantity >= 0 ? '+' : ''}{row.quantity}
          </span>
        ),
      },
      { key: 'reference', header: 'Reference', render: (row: MovementResponse) => <span className="text-gray-600 text-sm">{row.reference || '—'}</span> },
      { key: 'reason', header: 'Reason', render: (row: MovementResponse) => <span className="text-gray-600 text-sm">{row.reason || '—'}</span> },
      {
        key: 'created_at',
        header: 'Date',
        render: (row: MovementResponse) => (
          <span className="text-gray-500 text-sm">{formatDateTime(row.created_at)}</span>
        ),
      },
    ];

    return (
      <>
        <div className="flex gap-3 mb-4">
          <div className="relative flex-1 max-w-xs">
            <MagnifyingGlassIcon className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-gray-400" />
            <input
              type="text"
              value={movSkuSearch}
              onChange={(e) => { setMovSkuSearch(e.target.value); setMovPage(1); }}
              placeholder="Filter by SKU..."
              className="input-field pl-10"
            />
          </div>
        </div>
        {movements.length === 0 ? (
          <EmptyState title="No movements" description="No stock movement records found." />
        ) : (
          <>
            <DataTable columns={columns} data={movements as unknown as Record<string, unknown>[]} />
            {pag && (
              <div className="mt-4">
                <Pagination page={pag.page} totalPages={pag.total_pages} onPageChange={setMovPage} />
              </div>
            )}
          </>
        )}
      </>
    );
  };

  const renderLowStockTab = () => {
    if (lowStockQuery.isLoading) return <LoadingSpinner />;
    if (lowStockQuery.isError) {
      return (
        <ErrorMessage
          message={lowStockQuery.error?.message || 'Failed to load low stock items'}
          onRetry={() => lowStockQuery.refetch()}
        />
      );
    }

    const items = lowStockQuery.data?.data ?? [];
    const pag = lowStockQuery.data?.pagination;

    const columns = [
      { key: 'sku', header: 'SKU', render: (row: StockResponse) => <span className="font-medium">{row.sku}</span> },
      {
        key: 'warehouse_id',
        header: 'Warehouse',
        render: (row: StockResponse) => (
          <span className="text-gray-600">{warehouseMap.get(row.warehouse_id) || row.warehouse_id}</span>
        ),
      },
      {
        key: 'available_stock',
        header: 'Available',
        render: (row: StockResponse) => (
          <span className="text-red-700 font-medium">{row.available_stock}</span>
        ),
      },
      { key: 'reserved_stock', header: 'Reserved' },
      { key: 'total_stock', header: 'Total' },
      {
        key: 'low_stock_threshold',
        header: 'Threshold',
        render: (row: StockResponse) => <span className="text-gray-600">{row.low_stock_threshold}</span>,
      },
      {
        key: 'updated_at',
        header: 'Updated',
        render: (row: StockResponse) => (
          <span className="text-gray-500 text-sm">{formatDateTime(row.updated_at)}</span>
        ),
      },
    ];

    return (
      <>
        {items.length === 0 ? (
          <EmptyState title="All stocked up" description="No items are currently below their low-stock threshold." />
        ) : (
          <>
            <DataTable columns={columns} data={items as unknown as Record<string, unknown>[]} />
            {pag && (
              <div className="mt-4">
                <Pagination page={pag.page} totalPages={pag.total_pages} onPageChange={setLowPage} />
              </div>
            )}
          </>
        )}
      </>
    );
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div>
        <h1 className="text-2xl font-bold text-gray-900">Inventory</h1>
        <p className="mt-1 text-sm text-gray-500">
          Manage stock levels, warehouses, reservations, and movements
        </p>
      </div>

      {/* Tabs */}
      <div className="flex flex-wrap gap-1 border-b border-gray-200">
        {TABS.map((tab) => (
          <button
            key={tab}
            onClick={() => setActiveTab(tab)}
            className={cn(
              'px-4 py-2.5 text-sm font-medium border-b-2 -mb-px transition-colors',
              activeTab === tab
                ? 'border-primary-600 text-primary-600'
                : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300',
            )}
          >
            {tab}
          </button>
        ))}
      </div>

      {/* Tab Content */}
      <div className="card">{renderTabContent()}</div>

      {/* ===== Create Stock Dialog ===== */}
      <Dialog open={createStockOpen} onClose={() => setCreateStockOpen(false)} className="relative z-50">
        <div className="fixed inset-0 bg-black/30" aria-hidden="true" />
        <div className="fixed inset-0 flex items-center justify-center p-4">
          <DialogPanel className="w-full max-w-lg rounded-2xl bg-white p-6 shadow-xl">
            <DialogTitle className="text-lg font-semibold text-gray-900">Create Stock Entry</DialogTitle>
            {createStockMutation.isError && (
              <div className="mt-3 rounded-lg bg-red-50 border border-red-200 p-3">
                <p className="text-sm text-red-700">{createStockMutation.error?.message || 'Failed to create stock'}</p>
              </div>
            )}
            <form
              onSubmit={(e) => {
                e.preventDefault();
                createStockMutation.mutate(csForm);
              }}
              className="mt-4 space-y-4"
            >
              <div>
                <label htmlFor="cs-sku" className="label-field">SKU <span className="text-red-500">*</span></label>
                <input id="cs-sku" type="text" required value={csForm.sku} onChange={(e) => setCsForm((p) => ({ ...p, sku: e.target.value }))} placeholder="SKU-001" className="input-field" />
              </div>
              <div>
                <label htmlFor="cs-warehouse" className="label-field">Warehouse <span className="text-red-500">*</span></label>
                <select id="cs-warehouse" required value={csForm.warehouse_id} onChange={(e) => setCsForm((p) => ({ ...p, warehouse_id: e.target.value }))} className="input-field">
                  <option value="">Select warehouse</option>
                  {warehouses.map((wh) => (
                    <option key={wh.id} value={wh.id}>{wh.name} ({wh.code})</option>
                  ))}
                </select>
              </div>
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label htmlFor="cs-total" className="label-field">Total Stock <span className="text-red-500">*</span></label>
                  <input id="cs-total" type="number" required min="0" value={csForm.total_stock} onChange={(e) => setCsForm((p) => ({ ...p, total_stock: Number(e.target.value) }))} className="input-field" />
                </div>
                <div>
                  <label htmlFor="cs-threshold" className="label-field">Low Stock Threshold</label>
                  <input id="cs-threshold" type="number" min="0" value={csForm.low_stock_threshold ?? 5} onChange={(e) => setCsForm((p) => ({ ...p, low_stock_threshold: Number(e.target.value) || undefined }))} className="input-field" />
                </div>
              </div>
              <div className="flex justify-end gap-3 pt-2">
                <button type="button" onClick={() => setCreateStockOpen(false)} className="btn-secondary">Cancel</button>
                <button type="submit" disabled={createStockMutation.isPending} className="btn-primary">{createStockMutation.isPending ? 'Creating...' : 'Create'}</button>
              </div>
            </form>
          </DialogPanel>
        </div>
      </Dialog>

      {/* ===== Adjust Stock Dialog ===== */}
      <Dialog open={adjustStockOpen} onClose={() => setAdjustStockOpen(false)} className="relative z-50">
        <div className="fixed inset-0 bg-black/30" aria-hidden="true" />
        <div className="fixed inset-0 flex items-center justify-center p-4">
          <DialogPanel className="w-full max-w-lg rounded-2xl bg-white p-6 shadow-xl">
            <DialogTitle className="text-lg font-semibold text-gray-900">Adjust Stock</DialogTitle>
            {adjustStockMutation.isError && (
              <div className="mt-3 rounded-lg bg-red-50 border border-red-200 p-3">
                <p className="text-sm text-red-700">{adjustStockMutation.error?.message || 'Failed to adjust stock'}</p>
              </div>
            )}
            <form
              onSubmit={(e) => {
                e.preventDefault();
                adjustStockMutation.mutate(asForm);
              }}
              className="mt-4 space-y-4"
            >
              <div>
                <label htmlFor="as-sku" className="label-field">SKU <span className="text-red-500">*</span></label>
                <input id="as-sku" type="text" required value={asForm.sku} onChange={(e) => setAsForm((p) => ({ ...p, sku: e.target.value }))} placeholder="SKU-001" className="input-field" />
              </div>
              <div>
                <label htmlFor="as-warehouse" className="label-field">Warehouse <span className="text-red-500">*</span></label>
                <select id="as-warehouse" required value={asForm.warehouse_id} onChange={(e) => setAsForm((p) => ({ ...p, warehouse_id: e.target.value }))} className="input-field">
                  <option value="">Select warehouse</option>
                  {warehouses.map((wh) => (
                    <option key={wh.id} value={wh.id}>{wh.name} ({wh.code})</option>
                  ))}
                </select>
              </div>
              <div>
                <label htmlFor="as-qty" className="label-field">Quantity <span className="text-red-500">*</span></label>
                <input id="as-qty" type="number" required value={asForm.quantity} onChange={(e) => setAsForm((p) => ({ ...p, quantity: Number(e.target.value) }))} placeholder="Use negative to reduce" className="input-field" />
                <p className="mt-1 text-xs text-gray-400">Use a positive number to add stock, negative to remove.</p>
              </div>
              <div>
                <label htmlFor="as-reason" className="label-field">Reason <span className="text-red-500">*</span></label>
                <textarea id="as-reason" required rows={2} value={asForm.reason} onChange={(e) => setAsForm((p) => ({ ...p, reason: e.target.value }))} placeholder="Reason for adjustment" className="input-field" />
              </div>
              <div className="flex justify-end gap-3 pt-2">
                <button type="button" onClick={() => setAdjustStockOpen(false)} className="btn-secondary">Cancel</button>
                <button type="submit" disabled={adjustStockMutation.isPending} className="btn-primary">{adjustStockMutation.isPending ? 'Adjusting...' : 'Adjust'}</button>
              </div>
            </form>
          </DialogPanel>
        </div>
      </Dialog>

      {/* ===== Create Warehouse Dialog ===== */}
      <Dialog open={warehouseOpen} onClose={() => setWarehouseOpen(false)} className="relative z-50">
        <div className="fixed inset-0 bg-black/30" aria-hidden="true" />
        <div className="fixed inset-0 flex items-center justify-center p-4">
          <DialogPanel className="w-full max-w-lg rounded-2xl bg-white p-6 shadow-xl max-h-[90vh] overflow-y-auto">
            <DialogTitle className="text-lg font-semibold text-gray-900">New Warehouse</DialogTitle>
            {createWarehouseMutation.isError && (
              <div className="mt-3 rounded-lg bg-red-50 border border-red-200 p-3">
                <p className="text-sm text-red-700">{createWarehouseMutation.error?.message || 'Failed to create warehouse'}</p>
              </div>
            )}
            <form
              onSubmit={(e) => {
                e.preventDefault();
                createWarehouseMutation.mutate(whForm);
              }}
              className="mt-4 space-y-4"
            >
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label htmlFor="wh-name" className="label-field">Name <span className="text-red-500">*</span></label>
                  <input id="wh-name" type="text" required value={whForm.name} onChange={(e) => setWhForm((p) => ({ ...p, name: e.target.value }))} placeholder="Warehouse name" className="input-field" />
                </div>
                <div>
                  <label htmlFor="wh-code" className="label-field">Code <span className="text-red-500">*</span></label>
                  <input id="wh-code" type="text" required value={whForm.code} onChange={(e) => setWhForm((p) => ({ ...p, code: e.target.value }))} placeholder="WH-001" className="input-field" />
                </div>
              </div>
              <div className="space-y-3">
                <p className="label-field mb-0">Address</p>
                <input type="text" required value={whForm.address.line1} onChange={(e) => setWhForm((p) => ({ ...p, address: { ...p.address, line1: e.target.value } }))} placeholder="Street address" className="input-field" />
                <div className="grid grid-cols-2 gap-4">
                  <input type="text" required value={whForm.address.city} onChange={(e) => setWhForm((p) => ({ ...p, address: { ...p.address, city: e.target.value } }))} placeholder="City" className="input-field" />
                  <input type="text" required value={whForm.address.state} onChange={(e) => setWhForm((p) => ({ ...p, address: { ...p.address, state: e.target.value } }))} placeholder="State" className="input-field" />
                </div>
                <div className="grid grid-cols-2 gap-4">
                  <input type="text" required value={whForm.address.postal_code} onChange={(e) => setWhForm((p) => ({ ...p, address: { ...p.address, postal_code: e.target.value } }))} placeholder="Postal code" className="input-field" />
                  <input type="text" required value={whForm.address.country} onChange={(e) => setWhForm((p) => ({ ...p, address: { ...p.address, country: e.target.value } }))} placeholder="Country" className="input-field" />
                </div>
              </div>
              <div className="flex justify-end gap-3 pt-2">
                <button type="button" onClick={() => setWarehouseOpen(false)} className="btn-secondary">Cancel</button>
                <button type="submit" disabled={createWarehouseMutation.isPending} className="btn-primary">{createWarehouseMutation.isPending ? 'Creating...' : 'Create'}</button>
              </div>
            </form>
          </DialogPanel>
        </div>
      </Dialog>
    </div>
  );
}
