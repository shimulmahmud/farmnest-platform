import { Minus, Plus } from 'lucide-react';

interface QuantitySelectorProps {
  quantity: number;
  onDecrease: () => void;
  onIncrease: () => void;
  size?: 'sm' | 'md';
}

export default function QuantitySelector({ quantity, onDecrease, onIncrease, size = 'md' }: QuantitySelectorProps) {
  const btnClass = size === 'sm' ? 'p-1.5' : 'p-3';
  const iconClass = size === 'sm' ? 'w-3.5 h-3.5' : 'w-4 h-4';
  const textClass = size === 'sm' ? 'w-8 text-sm' : 'w-12';

  return (
    <div className="flex items-center border border-gray-200 rounded-md">
      <button onClick={onDecrease} className={`${btnClass} text-gray-600 hover:text-primary`} aria-label="Decrease quantity">
        <Minus className={iconClass} />
      </button>
      <span className={`${textClass} text-center font-semibold`}>{quantity}</span>
      <button onClick={onIncrease} className={`${btnClass} text-gray-600 hover:text-primary`} aria-label="Increase quantity">
        <Plus className={iconClass} />
      </button>
    </div>
  );
}
