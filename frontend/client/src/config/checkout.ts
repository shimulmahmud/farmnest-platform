/**
 * Checkout settings that live in the browser. Shipping fees, wallet numbers and the free-shipping
 * threshold come from the backend (GET /checkout/config) so the server and the screen never disagree.
 */

/** The cart is kept for 1 hour after its last change. Stock is never reserved by the cart. */
export const CART_TTL_MS = 60 * 60 * 1000;

export type WalletId = 'bkash' | 'nagad' | 'rocket';

/** Look and feel for the wallets the backend offers. */
export const walletStyle: Record<WalletId, { accent: string; desc: string }> = {
  bkash: { accent: 'text-pink-600', desc: 'Send money from your bKash app, then enter the TrxID' },
  nagad: { accent: 'text-orange-600', desc: 'Send money from your Nagad app, then enter the TrxID' },
  rocket: { accent: 'text-purple-600', desc: 'Send money from your Rocket app, then enter the TrxID' },
};
