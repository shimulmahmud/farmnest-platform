import { Link } from 'react-router-dom';
import { Banknote, Facebook, Headphones, Instagram, Twitter, Youtube } from 'lucide-react';
import type { ReactNode } from 'react';
import { footerSections, contactInfo, socialIcons } from '@/config/footer';
import { useCatalog } from '@/context/CatalogContext';

/** Payment options shown in the footer. */
const paymentMethods = [
  { id: 'bkash', label: 'bKash' },
  { id: 'nagad', label: 'Nagad' },
  { id: 'rocket', label: 'Rocket' },
  { id: 'cod', label: 'Cash on Delivery' },
];
import { NewsletterForm } from '@/components/ui';
import TrustBar from '@/components/TrustBar';

const socialIconMap = { Facebook, Twitter, Instagram, Youtube };

/** Stylised wordmarks on white chips (swap in the official logo SVGs when you have them). */
const paymentMarks: Record<string, ReactNode> = {
  bkash: <span className="text-[13px] font-extrabold tracking-tight text-[#E2136E]">bKash</span>,
  nagad: <span className="text-[13px] font-extrabold tracking-tight text-[#F26B1D]">Nagad</span>,
  rocket: <span className="text-[13px] font-extrabold tracking-tight text-[#8C3494]">Rocket</span>,
  visa: <span className="text-sm font-black italic tracking-tight text-[#1A1F71]">VISA</span>,
  mastercard: (
    <span className="relative block h-5 w-8" aria-hidden="true">
      <span className="absolute left-0 top-0 h-5 w-5 rounded-full bg-[#EB001B]" />
      <span className="absolute left-3 top-0 h-5 w-5 rounded-full bg-[#F79E1B] opacity-90 mix-blend-multiply" />
    </span>
  ),
  cod: (
    <>
      <Banknote className="h-4 w-4 text-green-700" aria-hidden="true" />
      <span className="text-[11px] font-bold text-gray-700">COD</span>
    </>
  ),
};

const linkClass = 'text-[13px] text-white/60 transition-colors hover:text-white';

export default function Footer() {
  const { categories } = useCatalog();
  return (
    <footer className="mt-14">
      {/* The highlights bar straddles the page edge: the dark background only covers its lower half. */}
      <div className="relative">
        <div className="absolute inset-x-0 bottom-0 h-1/2 bg-[#101010]" aria-hidden="true" />
        <div className="relative mx-auto max-w-page px-4 sm:px-6 lg:px-8">
          <TrustBar />
        </div>
      </div>

      <div className="bg-[#101010] text-white/60">
        <div className="mx-auto max-w-page px-4 pb-10 pt-10 sm:px-6 lg:px-8">
          <div className="grid grid-cols-2 gap-x-8 gap-y-10 md:grid-cols-3 lg:grid-cols-[1.25fr_repeat(3,1fr)_1.5fr]">
            <div className="col-span-2 md:col-span-3 lg:col-span-1">
              <div className="flex items-center gap-2.5">
                <Headphones className="h-7 w-7 text-emerald-400" strokeWidth={1.5} aria-hidden="true" />
                <div className="leading-tight">
                  <a
                    href={`tel:${contactInfo.phone.replace(/\s/g, '')}`}
                    className="block font-headline text-base font-bold text-emerald-400 hover:underline"
                  >
                    {contactInfo.phone}
                  </a>
                  <span className="text-[11px] text-white/50">Customer support</span>
                </div>
              </div>

              <div className="mt-5 space-y-1.5 text-[13px] leading-relaxed">
                <p>{contactInfo.address}</p>
                <p>
                  <a href={`mailto:${contactInfo.email}`} className="transition-colors hover:text-white">
                    {contactInfo.email}
                  </a>
                </p>
              </div>

              <div className="mt-5 flex items-center gap-2">
                {socialIcons.map((name, i) => {
                  const Icon = socialIconMap[name];
                  return (
                    <a
                      key={name}
                      href="#"
                      aria-label={name}
                      className={`flex h-7 w-7 items-center justify-center rounded-full border transition-colors ${
                        i === 0
                          ? 'border-primary bg-primary text-white'
                          : 'border-white/15 text-white/70 hover:border-primary hover:bg-primary hover:text-white'
                      }`}
                    >
                      <Icon className="h-3.5 w-3.5" />
                    </a>
                  );
                })}
              </div>
            </div>

            {footerSections.map((rawSection) => {
              // The Shop column lists the shop's real categories (they come from the admin panel).
              const section = rawSection.title === 'Shop'
                ? { ...rawSection, links: [rawSection.links[0], ...categories.slice(0, 5).map((c) => ({ label: c.label, path: `/products?category=${c.slug}` }))] }
                : rawSection;
              return (
              <nav key={section.title} aria-label={section.title}>
                <h4 className="mb-4 font-headline text-sm font-semibold text-white">{section.title}</h4>
                <ul className="space-y-2.5">
                  {section.links.map((link) => (
                    <li key={link.label}>
                      <Link to={link.path} className={linkClass}>
                        {link.label}
                      </Link>
                    </li>
                  ))}
                </ul>
              </nav>
              );
            })}

            <div className="col-span-2 md:col-span-3 lg:col-span-1">
              <h4 className="mb-4 font-headline text-sm font-semibold text-white">Subscribe</h4>
              <p className="mb-4 text-[13px] leading-relaxed">
                Stay updated with farming tips, exclusive deals and product updates.
              </p>
              <NewsletterForm variant="pill" />
            </div>
          </div>
        </div>

        <div className="border-t border-white/10">
          <div className="mx-auto flex max-w-page flex-col gap-4 px-4 py-5 sm:flex-row sm:items-center sm:justify-between sm:px-6 lg:px-8">
            <p className="text-xs text-white/50">
              Copyright &copy; 2026 <span className="text-emerald-400">FarmNest</span>. All Rights Reserved
            </p>
            <ul className="flex flex-wrap items-center gap-2" aria-label="Accepted payment methods">
              {paymentMethods.map((method) => (
                <li
                  key={method.id}
                  className="flex h-8 min-w-[52px] items-center justify-center gap-1 rounded-md bg-white px-2.5"
                >
                  {paymentMarks[method.id] ?? <span className="text-[11px] font-bold text-gray-700">{method.label}</span>}
                  <span className="sr-only">{method.label}</span>
                </li>
              ))}
            </ul>
          </div>
        </div>
      </div>
    </footer>
  );
}
