import { Link } from 'react-router-dom';
import { Minus, Plus, ShoppingBag, Trash2, X } from 'lucide-react';
import { useCart } from '@/context/CartContext';
import { formatPrice } from '@/utils/format';
import { QuantitySelector } from '@/components/ui';

export default function CartDrawer({ open, onClose }: { open: boolean; onClose: () => void }) {
  const { lines, subtotal, updateQuantity, removeItem } = useCart();

  return (
    <>
      <div
        className={`fixed inset-0 z-[60] bg-black/40 transition-opacity duration-300 ${open ? 'opacity-100' : 'opacity-0 pointer-events-none'}`}
        onClick={onClose}
      />
      <aside
        className={`fixed top-0 right-0 z-[70] h-full w-full max-w-md bg-white flex flex-col transition-all duration-300 ${open ? 'translate-x-0 shadow-2xl' : 'translate-x-full shadow-none'}`}
      >
        <header className="flex items-center justify-between px-5 h-16 border-b border-gray-100">
          <h2 className="font-headline text-xl font-bold flex items-center gap-2">
            <ShoppingBag className="w-5 h-5 text-primary" /> Your Cart
          </h2>
          <button onClick={onClose} className="p-2 text-gray-500 hover:text-gray-900" aria-label="Close cart">
            <X className="w-5 h-5" />
          </button>
        </header>

        {lines.length === 0 ? (
          <div className="flex-1 flex flex-col items-center justify-center text-center px-6">
            <ShoppingBag className="w-12 h-12 text-gray-300 mb-4" />
            <p className="font-headline text-lg font-semibold text-gray-700">Your cart is empty</p>
            <p className="text-sm text-gray-500 mt-1">Add some farm essentials to get started.</p>
            <Link to="/products" onClick={onClose} className="mt-6 bg-primary text-white px-5 py-2.5 rounded-lg font-semibold hover:bg-primary-hover transition-colors">
              Browse Products
            </Link>
          </div>
        ) : (
          <>
            <div className="flex-1 overflow-y-auto px-5 py-4 space-y-4">
              {lines.map(({ product, variant, quantity }) => (
                <div key={variant.id} className="flex gap-3">
                  <img src={variant.image || product.image} alt={product.name} className="w-20 h-20 rounded-lg object-cover flex-shrink-0" />
                  <div className="flex-1 min-w-0">
                    <p className="font-semibold text-sm line-clamp-2">{product.name}</p>
                    <p className="text-xs text-gray-500 mt-0.5">{product.hasChoices ? variant.label : product.unit}</p>
                    <div className="flex items-center justify-between mt-2">
                      <QuantitySelector
                        quantity={quantity}
                        onDecrease={() => updateQuantity(variant.id, quantity - 1)}
                        onIncrease={() => updateQuantity(variant.id, quantity + 1)}
                        size="sm"
                      />
                      <strong className="text-sm text-primary">{formatPrice(variant.price * quantity)}</strong>
                    </div>
                  </div>
                  <button onClick={() => removeItem(variant.id)} className="p-1.5 text-gray-400 hover:text-red-600 self-start" aria-label="Remove item">
                    <Trash2 className="w-4 h-4" />
                  </button>
                </div>
              ))}
            </div>

            <footer className="border-t border-gray-100 px-5 py-5 space-y-4">
              <div className="flex justify-between text-sm">
                <span className="text-gray-600">Subtotal</span>
                <strong className="font-headline text-lg text-primary">{formatPrice(subtotal)}</strong>
              </div>
              <p className="text-xs text-gray-500">Delivery charge is added at checkout.</p>
              <Link
                to="/checkout"
                onClick={onClose}
                className="block w-full text-center bg-primary text-white py-3 rounded-lg font-semibold hover:bg-primary-hover transition-colors"
              >
                Proceed to Checkout
              </Link>
              <button onClick={onClose} className="block w-full text-center text-sm font-semibold text-gray-600 hover:text-primary">
                Continue Shopping
              </button>
            </footer>
          </>
        )}
      </aside>
    </>
  );
}
