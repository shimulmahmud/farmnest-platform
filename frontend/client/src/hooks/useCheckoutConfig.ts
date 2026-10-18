import { useEffect, useState } from 'react';
import { fetchCheckoutConfig, type CheckoutConfig } from '@/api/orders';

let cached: CheckoutConfig | null = null;
let inflight: Promise<CheckoutConfig> | null = null;

/** Shipping fees, free-shipping threshold and payment wallets, straight from the backend. */
export function useCheckoutConfig(): { config: CheckoutConfig | null; error: string | null } {
  const [config, setConfig] = useState<CheckoutConfig | null>(cached);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (cached) return;
    inflight ??= fetchCheckoutConfig();
    inflight
      .then((c) => {
        cached = c;
        setConfig(c);
      })
      .catch((e: unknown) => {
        inflight = null;
        setError(e instanceof Error ? e.message : 'Could not load checkout settings');
      });
  }, []);

  return { config, error };
}
