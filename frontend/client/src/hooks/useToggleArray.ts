import { useMemo, useState } from 'react';

export function useToggleArray(initial: string[] = []) {
  const [items, setItems] = useState<string[]>(initial);

  const toggle = (value: string) => {
    setItems((prev) =>
      prev.includes(value) ? prev.filter((v) => v !== value) : [...prev, value]
    );
  };

  const clear = () => setItems([]);

  return { items, toggle, clear, setItems };
}

export function useToggleArrayWithReset(initial: string[] = []) {
  const { items, toggle, setItems } = useToggleArray(initial);
  const reset = () => setItems(initial);
  return { items, toggle, reset, setItems };
}
