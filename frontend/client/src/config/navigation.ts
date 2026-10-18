export interface NavLink {
  label: string;
  path: string;
  /** Rendered in the sale colour with a bolt icon. */
  highlight?: boolean;
}

export const navLinks: NavLink[] = [
  { label: 'Home', path: '/' },
  { label: 'Products', path: '/products' },
  { label: 'Deals', path: '/offers', highlight: true },
  { label: 'Track Order', path: '/track-order' },
];
