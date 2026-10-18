import { Link } from 'react-router-dom';
import { Shield, ArrowRight, Mail } from 'lucide-react';

const sections = [
  {
    title: '1. Information We Collect',
    body: [
      'Account information: When you create an account, we collect your name, email address, phone number, and delivery address.',
      'Order information: We keep records of your purchases, payment method (processed securely by our payment partners), and delivery preferences.',
      'Browsing data: We use cookies and similar technologies to remember your cart, preferences, and to understand how you use our platform.',
      'Communication: Messages you send us via email, WhatsApp, or our support channels are stored to help us provide better service.',
    ],
  },
  {
    title: '2. How We Use Your Information',
    body: [
      'To process and deliver your orders, including sending you order confirmations and tracking updates.',
      'To provide customer support and agronomist consultations when requested.',
      'To send you seasonal guides, exclusive offers, and product updates (only if you have opted in — you can unsubscribe at any time).',
      'To improve our platform, personalize your experience, and develop new features based on how farmers use FarmNest.',
      'To prevent fraud, verify identity, and comply with legal requirements.',
    ],
  },
  {
    title: '3. What We Never Do',
    body: [
      'We never sell your personal data to third parties.',
      'We never share your contact details with marketing companies.',
      'We never use your farming data to raise prices or discriminate in any way.',
      'We never access your device storage, contacts, or location without explicit permission.',
    ],
  },
  {
    title: '4. Data Sharing',
    body: [
      'Delivery partners: We share your name, address, and phone number with our courier partners (BlueDart, Delhivery) solely to deliver your order.',
      'Payment processors: Payment information is handled entirely by our secure payment gateway. We never store your card or mobile wallet details.',
      'Legal compliance: We may disclose information if required by law or to protect the rights, property, or safety of our users.',
    ],
  },
  {
    title: '5. Data Security',
    body: [
      'All data is transmitted over encrypted HTTPS connections.',
      'Passwords are hashed using industry-standard algorithms — we cannot see your plain-text password.',
      'Access to personal data is restricted to authorized team members on a need-to-know basis.',
      'We conduct regular security audits and promptly notify you of any data breach affecting your information.',
    ],
  },
  {
    title: '6. Your Rights',
    body: [
      'Access: You can request a copy of all personal data we hold about you at any time.',
      'Correction: You can update your account information through your profile or by contacting us.',
      'Deletion: You can request permanent deletion of your account and associated data.',
      'Opt-out: You can unsubscribe from marketing emails with a single click. Transactional emails (order updates) will still be sent.',
      'To exercise any of these rights, email us at privacy@farmnest.in.',
    ],
  },
  {
    title: '7. Cookies',
    body: [
      'Essential cookies: Required for cart functionality, login sessions, and checkout. These cannot be disabled.',
      'Analytics cookies: Help us understand which pages are popular and where we can improve. You can opt out in your browser settings.',
      'We do not use third-party advertising cookies.',
    ],
  },
  {
    title: '8. Children\'s Privacy',
    body: [
      'FarmNest is intended for use by adults. We do not knowingly collect information from children under 18.',
      'If you believe a minor has provided us with personal data, please contact us and we will promptly delete it.',
    ],
  },
  {
    title: '9. Changes to This Policy',
    body: [
      'We may update this policy from time to time. We will notify you by email of any significant changes at least 30 days before they take effect.',
      'The "last updated" date below indicates when the policy was last revised.',
    ],
  },
];

export default function PrivacyPage() {
  return (
    <div className="bg-white">
      <section className="bg-gradient-to-br from-primary via-primary to-primary-dark text-white">
        <div className="max-w-page mx-auto px-4 sm:px-6 lg:px-8 py-14 lg:py-20">
          <div className="flex items-center gap-3">
            <Shield className="w-8 h-8 text-amber-300" />
            <h1 className="font-headline text-3xl lg:text-4xl font-bold tracking-tight">Privacy Policy</h1>
          </div>
          <p className="text-white/80 mt-3 max-w-2xl">Your trust matters to us. This policy explains what data we collect, how we use it, and the controls you have over your information.</p>
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
              <h2 className="font-headline text-lg font-bold">Questions about your privacy?</h2>
              <p className="text-white/80 text-sm mt-2">We're here to help. Reach out to our privacy team anytime.</p>
              <a href="mailto:privacy@farmnest.in" className="inline-flex items-center gap-2 mt-4 bg-white text-primary px-5 py-2.5 rounded-full font-bold text-sm hover:bg-primary-fixed transition-colors">
                <Mail className="w-4 h-4" /> privacy@farmnest.in
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
