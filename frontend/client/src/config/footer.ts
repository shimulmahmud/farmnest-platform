export interface FooterLink {
  label: string;
  path: string;
}

export interface FooterSection {
  title: string;
  links: FooterLink[];
}

export const footerSections: FooterSection[] = [
  {
    title: 'Shop',
    links: [
      { label: 'All Products', path: '/products' },
      { label: 'Seeds', path: '/products?category=seeds' },
      { label: 'Fertilizers', path: '/products?category=fertilizers' },
      { label: 'Tools', path: '/products?category=tools' },
      { label: 'Machinery', path: '/products?category=machinery' },
    ],
  },
  {
    title: 'Resources',
    links: [
      { label: 'Categories', path: '/#categories' },
      { label: 'Track Order', path: '/track-order' },
      { label: 'Help & Support', path: '/support' },
    ],
  },
  {
    title: 'Company',
    links: [
      { label: 'About Us', path: '/about' },
      { label: 'Careers', path: '/careers' },
      { label: 'Partner With Us', path: '/partners' },
      { label: 'Privacy Policy', path: '/privacy' },
      { label: 'Terms of Service', path: '/terms' },
    ],
  },
];

export const contactInfo = {
  email: 'support@farmnest.in',
  phone: '+880 1XXX-XXXXXX',
  address: 'Dhaka, Bangladesh',
};

export const socialIcons = ['Facebook', 'Twitter', 'Instagram', 'Youtube'] as const;
