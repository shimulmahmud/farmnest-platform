import { Link } from 'react-router-dom';
import { FileText, ArrowRight, Mail } from 'lucide-react';
import { formatPrice } from '@/utils/format';

const sections = [
  {
    title: '1. Acceptance of Terms',
    body: [
      'By accessing or using FarmNest (the "Platform"), you agree to be bound by these Terms of Service. If you do not agree with any part of these terms, please do not use our Platform.',
      'These terms apply to all visitors, users, and others who access or use the Platform.',
    ],
  },
  {
    title: '2. Eligibility',
    body: [
      'You must be at least 18 years old to create an account and place orders on FarmNest.',
      'By registering, you confirm that the information you provide is accurate and complete.',
    ],
  },
  {
    title: '3. Your Account',
    body: [
      'You are responsible for maintaining the confidentiality of your login credentials and for all activities under your account.',
      'Notify us immediately of any unauthorized use of your account or any other security breach.',
      'You agree to provide accurate, current, and complete information during registration and to update such information to keep it accurate.',
    ],
  },
  {
    title: '4. Orders & Payments',
    body: [
      'All orders are subject to availability and confirmation of the order price. We reserve the right to refuse or cancel any order at our discretion.',
      'Prices are listed in Bangladeshi Taka (৳) and are inclusive of applicable taxes unless stated otherwise.',
      'Payment must be completed at checkout. We accept bKash, Nagad, Rocket and cash on delivery.',
      'Once an order is confirmed, you will receive an order ID and tracking details by email and SMS.',
    ],
  },
  {
    title: '5. Shipping & Delivery',
    body: [
      'We strive to deliver orders within 3–7 business days depending on your location. Remote areas may take longer.',
      'Delivery charges, and any free-delivery threshold, are shown at checkout before you place your order.',
      'Delivery times are estimates and not guaranteed. We are not liable for delays caused by courier partners, weather, or events beyond our control.',
      'If an order is lost or damaged in transit, contact us within 48 hours of the expected delivery date for a replacement or refund.',
    ],
  },
  {
    title: '6. Returns & Refunds',
    body: [
      'Seeds and perishable items are non-returnable due to their nature. Please check your order upon delivery.',
      'Non-perishable items (tools, equipment, machinery) can be returned within 7 days of delivery if unused and in original packaging.',
      'To initiate a return, email support@farmnest.in with your order ID and reason for return.',
      'Approved refunds are processed to the original payment method within 5–7 business days.',
      'Machinery returns may be subject to a restocking fee of 10% of the product price.',
    ],
  },
  {
    title: '7. Product Quality & Warranties',
    body: [
      'All products are sourced from verified suppliers and meet quality standards as described on their product pages.',
      'Seeds come with a germination guarantee — if germination falls below the stated rate, we will replace the seeds free of charge.',
      'Machinery and equipment carry manufacturer warranties as specified on individual product pages.',
      'Claims under warranty must be made directly with the manufacturer or through our support team.',
    ],
  },
  {
    title: '8. Acceptable Use',
    body: [
      'You agree not to use the Platform for any unlawful purpose or in violation of any local, state, national, or international law.',
      'You will not attempt to disrupt, reverse engineer, or gain unauthorized access to any part of the Platform or its systems.',
      'You will not post false, misleading, or fraudulent reviews or ratings.',
      'Bulk or automated ordering for resale is not permitted without a partnership agreement.',
    ],
  },
  {
    title: '9. Intellectual Property',
    body: [
      'All content on FarmNest — including logos, text, graphics, product photos, and software — is owned by FarmNest or its licensors and is protected by intellectual property laws.',
      'You may not copy, reproduce, or distribute our content without prior written permission.',
    ],
  },
  {
    title: '10. Limitation of Liability',
    body: [
      'FarmNest is provided "as is" without warranties of any kind. To the fullest extent permitted by law, we disclaim all liability for indirect, incidental, or consequential damages.',
      'Our total liability for any claim arising from your use of the Platform is limited to the amount you paid for the relevant order.',
      'We are not liable for crop outcomes, yield results, or any agricultural decisions made based on information or products from our Platform.',
    ],
  },
  {
    title: '11. Changes to Terms',
    body: [
      'We may update these Terms from time to time. Continued use of the Platform after changes are posted constitutes acceptance of the new terms.',
      'Significant changes will be communicated by email at least 15 days before taking effect.',
    ],
  },
  {
    title: '12. Governing Law',
    body: [
      'These Terms are governed by the laws of Bangladesh. Any disputes shall be subject to the exclusive jurisdiction of the courts in Dhaka.',
    ],
  },
];

export default function TermsPage() {
  return (
    <div className="bg-white">
      <section className="bg-gradient-to-br from-primary via-primary to-primary-dark text-white">
        <div className="max-w-page mx-auto px-4 sm:px-6 lg:px-8 py-14 lg:py-20">
          <div className="flex items-center gap-3">
            <FileText className="w-8 h-8 text-amber-300" />
            <h1 className="font-headline text-3xl lg:text-4xl font-bold tracking-tight">Terms of Service</h1>
          </div>
          <p className="text-white/80 mt-3 max-w-2xl">These terms govern your use of FarmNest. Please read them carefully before placing an order.</p>
          <p className="text-white/50 text-sm mt-4">Last updated: September 5, 2026</p>
        </div>
      </section>

      <div className="max-w-3xl mx-auto px-4 sm:px-6 lg:px-8 py-14">
        <div className="grid lg:grid-cols-[220px_1fr] gap-8">
          <nav className="hidden lg:block">
            <p className="text-xs font-bold uppercase tracking-widest text-gray-400 mb-3">Contents</p>
            <ul className="space-y-2 text-sm">
              {sections.map((s) => (
                <li key={s.title}>
                  <a href={`#${s.title.replace(/[^a-zA-Z0-9]/g, '-').toLowerCase()}`} className="text-gray-500 hover:text-primary transition-colors block">
                    {s.title}
                  </a>
                </li>
              ))}
            </ul>
          </nav>

          <div className="space-y-8">
            {sections.map((s) => (
              <section key={s.title} id={s.title.replace(/[^a-zA-Z0-9]/g, '-').toLowerCase()}>
                <h2 className="font-headline text-xl font-bold text-gray-900 mb-3">{s.title}</h2>
                <ul className="space-y-2.5">
                  {s.body.map((point, i) => (
                    <li key={i} className="text-sm text-gray-600 leading-relaxed flex gap-2.5">
                      <span className="w-1.5 h-1.5 rounded-full bg-primary/40 flex-shrink-0 mt-2" />
                      {point}
                    </li>
                  ))}
                </ul>
              </section>
            ))}

            <div className="bg-primary rounded-2xl p-6 text-white text-center">
              <h2 className="font-headline text-lg font-bold">Questions about these terms?</h2>
              <p className="text-white/80 text-sm mt-2">Our support team is happy to clarify anything for you.</p>
              <a href="mailto:support@farmnest.in" className="inline-flex items-center gap-2 mt-4 bg-white text-primary px-5 py-2.5 rounded-full font-bold text-sm hover:bg-primary-fixed transition-colors">
                <Mail className="w-4 h-4" /> support@farmnest.in
              </a>
            </div>

            <div className="text-center">
              <Link to="/" className="inline-flex items-center gap-2 text-primary font-semibold text-sm hover:gap-3 transition-all">
                Back to Home <ArrowRight className="w-4 h-4" />
              </Link>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
