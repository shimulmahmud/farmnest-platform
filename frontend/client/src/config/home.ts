/** Pastel backdrops, handed out to categories in order (works for any category names). */
const tones = ['bg-rose-100', 'bg-emerald-100', 'bg-amber-100', 'bg-orange-100', 'bg-sky-100', 'bg-lime-100'];
const lightTones = ['bg-rose-50', 'bg-amber-50', 'bg-emerald-50', 'bg-sky-50', 'bg-orange-50', 'bg-lime-50'];

export const toneAt = (index: number): string => tones[index % tones.length];
export const lightToneAt = (index: number): string => lightTones[index % lightTones.length];

export const BEST_SELLER_LIMIT = 6;
export const FEATURED_LIMIT = 8;
export const SHELF_MAX_PRODUCTS = 5;
/** How many category promo cards / shelves the home page shows. */
export const PROMO_COUNT = 3;
export const SHELF_COUNT = 3;
