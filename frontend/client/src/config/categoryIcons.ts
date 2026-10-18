import {
  Sprout, Leaf, Shovel, Bug, Droplets, Tractor,
} from 'lucide-react';
import type { LucideIcon } from 'lucide-react';

export const categoryIcons: Record<string, LucideIcon> = {
  seeds: Sprout,
  fertilizers: Leaf,
  tools: Shovel,
  pesticides: Bug,
  irrigation: Droplets,
  machinery: Tractor,
};
