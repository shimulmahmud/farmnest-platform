export interface Category {
  id: string;
  slug: string;
  label: string;
  icon: string;
  count: number;
  description: string;
  image?: string;
}

/** One sellable option of a product (size, pack, colour...). Prices are whole BDT. */
export interface ProductVariant {
  id: string;
  sku: string;
  label: string;
  options: Record<string, string>;
  price: number;
  originalPrice: number | null;
  inStock: boolean;
  weightG: number;
  image?: string;
}

export interface Product {
  /** URL-safe slug. Used in routes, wishlist and cart keys. */
  id: string;
  /** Backend product id (needed to place orders). */
  backendId: string;
  name: string;
  category: string;
  categorySlug: string;
  /** Price of the headline (cheapest in-stock) variant. */
  price: number;
  /** Old price of the headline variant, when it is on sale. */
  originalPrice: number | null;
  unit: string;
  rating: number;
  reviews: number;
  badge?: string;
  image: string;
  images: string[];
  description: string;
  shortDescription: string;
  inStock: boolean;
  brand: string;
  certifications: string[];
  specs: Record<string, string>;
  tags: string[];
  variants: ProductVariant[];
  /** True when the buyer has to pick between several variants. */
  hasChoices: boolean;
}

export interface Review {
  id: string;
  productId: string;
  author: string;
  avatar: string;
  location: string;
  rating: number;
  date: string;
  title: string;
  body: string;
  helpful: number;
  verified: boolean;
}
