export const SITE_NAME = 'FarmNest';

export const DEFAULT_DESCRIPTION =
  'Buy quality seeds, fertilizers, farming tools and smart-farm equipment online in Bangladesh. Pay with bKash, Nagad, Rocket or cash on delivery.';

export interface RouteMeta {
  title: string;
  description: string;
  /** Keep out of search results (private or transactional pages). */
  noindex?: boolean;
}

/** Title/description for every fixed route. Product pages build theirs from the product. */
export const routeMeta: Record<string, RouteMeta> = {
  '/': {
    title: 'FarmNest | Seeds, Fertilizers & Farming Supplies in Bangladesh',
    description: DEFAULT_DESCRIPTION,
  },
  '/products': {
    title: 'Shop Farming Products',
    description: 'Browse seeds, fertilizers, tools, pesticides, irrigation kits and machinery. Filter by category and price, with delivery across Bangladesh.',
  },
  '/offers': {
    title: 'Flash Sale',
    description: 'Limited-time offers on farming products. Discounts are applied automatically at the prices shown.',
  },
  '/reviews': {
    title: 'Customer Reviews',
    description: 'Read verified reviews from farmers about FarmNest seeds, fertilizers, tools and irrigation equipment.',
  },
  '/technology': {
    title: 'Smart Farm Technology',
    description: 'IoT sensors, drip irrigation and automation for precision farming in Bangladesh.',
  },
  '/about': {
    title: 'About Us',
    description: "Learn how FarmNest is helping Bangladesh's farmers buy better inputs and grow more.",
  },
  '/careers': {
    title: 'Careers',
    description: 'Join the FarmNest team and help transform agriculture in Bangladesh.',
  },
  '/partners': {
    title: 'Become a Partner',
    description: 'Partner with FarmNest to reach farmers across Bangladesh.',
  },
  '/privacy': {
    title: 'Privacy Policy',
    description: 'How FarmNest collects, uses and protects your personal information.',
  },
  '/terms': {
    title: 'Terms of Service',
    description: 'The terms that apply when you shop with FarmNest, covering orders, payment, delivery, returns and your responsibilities.',
  },
  '/wishlist': { title: 'My Wishlist', description: DEFAULT_DESCRIPTION, noindex: true },
  '/track-order': { title: 'Track Your Order', description: DEFAULT_DESCRIPTION, noindex: true },
  '/checkout': { title: 'Checkout', description: DEFAULT_DESCRIPTION, noindex: true },
  '/signin': { title: 'Sign In', description: DEFAULT_DESCRIPTION, noindex: true },
  '/signup': { title: 'Create Account', description: DEFAULT_DESCRIPTION, noindex: true },
};

export const formatTitle = (title: string) => (title.includes(SITE_NAME) ? title : `${title} | ${SITE_NAME}`);
