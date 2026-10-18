interface StatusBadgeProps {
  status: string;
  type?: 'order' | 'payment' | 'shipment' | 'review' | 'default';
}

const orderStatusMap: Record<string, string> = {
  PENDING: 'badge-yellow',
  PAYMENT_PENDING: 'badge-yellow',
  CONFIRMED: 'badge-blue',
  PAID: 'badge-blue',
  REFUND_PENDING: 'badge-red',
  PROCESSING: 'badge-blue',
  SHIPPED: 'badge-blue',
  DELIVERED: 'badge-green',
  CANCELLED: 'badge-red',
  REFUNDED: 'badge-red',
};

const paymentStatusMap: Record<string, string> = {
  PENDING: 'badge-yellow',
  AUTHORIZED: 'badge-blue',
  PAID: 'badge-green',
  FAILED: 'badge-red',
  REFUNDED: 'badge-red',
};

const shipmentStatusMap: Record<string, string> = {
  PENDING: 'badge-yellow',
  PROCESSING: 'badge-blue',
  SHIPPED: 'badge-blue',
  DELIVERED: 'badge-green',
  CANCELLED: 'badge-red',
};

const reviewStatusMap: Record<string, string> = {
  PENDING: 'badge-yellow',
  APPROVED: 'badge-green',
  REJECTED: 'badge-red',
  HIDDEN: 'badge-gray',
};

function getBadgeClass(type: string, status: string): string {
  const upperStatus = status.toUpperCase();

  switch (type) {
    case 'order':
      return orderStatusMap[upperStatus] || 'badge-gray';
    case 'payment':
      return paymentStatusMap[upperStatus] || 'badge-gray';
    case 'shipment':
      return shipmentStatusMap[upperStatus] || 'badge-gray';
    case 'review':
      return reviewStatusMap[upperStatus] || 'badge-gray';
    default:
      return 'badge-gray';
  }
}

export default function StatusBadge({ status, type = 'default' }: StatusBadgeProps) {
  const badgeClass = getBadgeClass(type, status);
  const displayStatus = status
    .replace(/_/g, ' ')
    .toLowerCase()
    .replace(/\b\w/g, (c) => c.toUpperCase());

  return <span className={badgeClass}>{displayStatus}</span>;
}
