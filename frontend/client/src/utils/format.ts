/**
 * Prices are in Bangladeshi taka (BDT). Bangladesh groups digits in lakh/crore (1,00,000),
 * which `en-IN` grouping matches, so it is used for the number and the ৳ sign is added by hand.
 */
const numberFormat = new Intl.NumberFormat('en-IN', { maximumFractionDigits: 0 });

export const formatPrice = (price: number): string => `৳${numberFormat.format(price)}`;
