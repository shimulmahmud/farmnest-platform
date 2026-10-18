import { useState } from 'react';
import { Link } from 'react-router-dom';
import {
  Handshake, Check, ArrowRight, Store, Truck, Users, TrendingUp,
  ShieldCheck, Mail, Phone, User, Building,
} from 'lucide-react';

const benefits = [
  { icon: Store, title: 'Nationwide Reach', desc: 'Tap into our network of 12,000+ farmers across 28 states without setting up your own storefront.' },
  { icon: Truck, title: 'Logistics Handled', desc: 'We handle storage, packaging, and delivery. You focus on producing quality goods.' },
  { icon: TrendingUp, title: 'Fair Margins', desc: 'Keep up to 80% of the selling price. Our transparent fee structure means no hidden cuts.' },
  { icon: ShieldCheck, title: 'Quality Certification', desc: 'Get verified with our quality badge. Certified partners see 3x more orders on average.' },
  { icon: Users, title: 'Dedicated Manager', desc: 'Every partner gets a personal account manager to help with listings, pricing, and growth.' },
  { icon: Handshake, title: 'Long-term Growth', desc: 'Top-performing partners get featured placement and co-marketing opportunities.' },
];

const steps = [
  { title: 'Submit Application', desc: 'Fill out the form below with your business details and product categories.' },
  { title: 'Verification Call', desc: 'Our team reviews your application and schedules a 20-minute onboarding call.' },
  { title: 'Quality Check', desc: 'Send us product samples for quality testing and certification.' },
  { title: 'Go Live', desc: 'List your products and start receiving orders within 48 hours of approval.' },
];

export default function PartnersPage() {
  const [submitted, setSubmitted] = useState(false);
  const [form, setForm] = useState({ name: '', business: '', email: '', phone: '', category: '', message: '' });

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    setSubmitted(true);
  };

  const update = (key: string, value: string) => setForm((f) => ({ ...f, [key]: value }));

  return (
    <div className="bg-white">
      <section className="bg-gradient-to-br from-primary via-primary to-primary-dark text-white">
        <div className="max-w-page mx-auto px-4 sm:px-6 lg:px-8 py-16 lg:py-24 text-center">
          <span className="inline-flex items-center gap-2 bg-white/15 backdrop-blur rounded-full px-4 py-1.5 text-xs font-bold uppercase tracking-widest">
            <Handshake className="w-4 h-4" /> Partnerships
          </span>
          <h1 className="font-headline text-4xl lg:text-5xl font-bold mt-6 tracking-tight">Partner With Us</h1>
          <p className="text-white/85 text-lg mt-5 max-w-2xl mx-auto leading-relaxed">
            Are you a supplier, grower, or manufacturer of quality agricultural products? Join FarmNest and reach farmers across Bangladesh.
          </p>
        </div>
      </section>

      <section className="max-w-page mx-auto px-4 sm:px-6 lg:px-8 py-16">
        <div className="text-center mb-12">
          <h2 className="font-headline text-3xl lg:text-4xl font-bold text-gray-900 tracking-tight">Why Partner With FarmNest?</h2>
          <p className="text-gray-500 mt-3 max-w-xl mx-auto">We make it easy to grow your business while doing what you do best.</p>
        </div>
        <div className="grid sm:grid-cols-2 lg:grid-cols-3 gap-6">
          {benefits.map((b) => (
            <div key={b.title} className="bg-white rounded-2xl border border-gray-100 p-6 shadow-sm">
              <div className="w-12 h-12 rounded-xl bg-primary/10 flex items-center justify-center mb-4">
                <b.icon className="w-6 h-6 text-primary" />
              </div>
              <h3 className="font-headline text-lg font-bold text-gray-900">{b.title}</h3>
              <p className="text-sm text-gray-500 mt-2 leading-relaxed">{b.desc}</p>
            </div>
          ))}
        </div>
      </section>

      <section className="bg-white border-y border-gray-100">
        <div className="max-w-page mx-auto px-4 sm:px-6 lg:px-8 py-16">
          <div className="text-center mb-12">
            <h2 className="font-headline text-3xl lg:text-4xl font-bold text-gray-900 tracking-tight">How It Works</h2>
            <p className="text-gray-500 mt-3">From application to your first order in four simple steps.</p>
          </div>
          <div className="grid sm:grid-cols-2 lg:grid-cols-4 gap-6">
            {steps.map((step, i) => (
              <div key={step.title} className="relative">
                <div className="bg-gray-50 rounded-2xl border border-gray-100 p-6 h-full">
                  <div className="w-10 h-10 rounded-full bg-primary text-white flex items-center justify-center font-headline font-bold text-lg mb-4">
                    {i + 1}
                  </div>
                  <h3 className="font-headline text-base font-bold text-gray-900">{step.title}</h3>
                  <p className="text-sm text-gray-500 mt-2 leading-relaxed">{step.desc}</p>
                </div>
                {i < steps.length - 1 && (
                  <ArrowRight className="hidden lg:block absolute top-1/2 -right-4 w-6 h-6 text-gray-300 -translate-y-1/2 z-10" />
                )}
              </div>
            ))}
          </div>
        </div>
      </section>

      <section className="max-w-2xl mx-auto px-4 sm:px-6 lg:px-8 py-16">
        <div className="text-center mb-8">
          <h2 className="font-headline text-3xl font-bold text-gray-900 tracking-tight">Apply to Become a Partner</h2>
          <p className="text-gray-500 mt-3">Tell us about your business and we'll get back to you within 2 business days.</p>
        </div>

        {submitted ? (
          <div className="bg-white rounded-2xl border border-gray-100 shadow-sm p-8 text-center">
            <div className="w-14 h-14 rounded-full bg-green-100 text-green-600 flex items-center justify-center mx-auto mb-4">
              <Check className="w-7 h-7" />
            </div>
            <h3 className="font-headline text-xl font-bold text-gray-900">Application Received!</h3>
            <p className="text-gray-500 mt-2 text-sm leading-relaxed">
              Thank you for your interest, {form.name || 'partner'}. Our partnerships team will review your application and reach out to <strong className="text-gray-700">{form.email || 'you'}</strong> within 2 business days.
            </p>
            <Link to="/" className="inline-flex items-center gap-2 mt-6 bg-primary text-white px-6 py-3 rounded-lg font-semibold hover:bg-primary-hover transition-colors">
              Back to Home <ArrowRight className="w-4 h-4" />
            </Link>
          </div>
        ) : (
          <form onSubmit={handleSubmit} className="bg-white rounded-2xl border border-gray-100 shadow-sm p-6 sm:p-8 space-y-4">
            <div className="grid sm:grid-cols-2 gap-4">
              <div>
                <label className="block text-sm font-semibold text-gray-700 mb-1.5">Your Name</label>
                <div className="relative">
                  <User className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
                  <input required value={form.name} onChange={(e) => update('name', e.target.value)} placeholder="Full name" className="w-full pl-10 pr-4 py-3 bg-gray-50 border border-gray-200 rounded-lg text-sm focus:outline-none focus:border-primary focus:bg-white focus:ring-1 focus:ring-primary/20 transition-all" />
                </div>
              </div>
              <div>
                <label className="block text-sm font-semibold text-gray-700 mb-1.5">Business Name</label>
                <div className="relative">
                  <Building className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
                  <input required value={form.business} onChange={(e) => update('business', e.target.value)} placeholder="Company or farm name" className="w-full pl-10 pr-4 py-3 bg-gray-50 border border-gray-200 rounded-lg text-sm focus:outline-none focus:border-primary focus:bg-white focus:ring-1 focus:ring-primary/20 transition-all" />
                </div>
              </div>
              <div>
                <label className="block text-sm font-semibold text-gray-700 mb-1.5">Email</label>
                <div className="relative">
                  <Mail className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
                  <input required type="email" value={form.email} onChange={(e) => update('email', e.target.value)} placeholder="you@business.com" className="w-full pl-10 pr-4 py-3 bg-gray-50 border border-gray-200 rounded-lg text-sm focus:outline-none focus:border-primary focus:bg-white focus:ring-1 focus:ring-primary/20 transition-all" />
                </div>
              </div>
              <div>
                <label className="block text-sm font-semibold text-gray-700 mb-1.5">Phone</label>
                <div className="relative">
                  <Phone className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
                  <input required value={form.phone} onChange={(e) => update('phone', e.target.value)} placeholder="+880 ..." className="w-full pl-10 pr-4 py-3 bg-gray-50 border border-gray-200 rounded-lg text-sm focus:outline-none focus:border-primary focus:bg-white focus:ring-1 focus:ring-primary/20 transition-all" />
                </div>
              </div>
            </div>
            <div>
              <label className="block text-sm font-semibold text-gray-700 mb-1.5">Product Category</label>
              <select required value={form.category} onChange={(e) => update('category', e.target.value)} className="w-full px-4 py-3 bg-gray-50 border border-gray-200 rounded-lg text-sm focus:outline-none focus:border-primary focus:bg-white focus:ring-1 focus:ring-primary/20 transition-all">
                <option value="">Select a category</option>
                <option>Seeds</option>
                <option>Fertilizers</option>
                <option>Tools & Equipment</option>
                <option>Pesticides</option>
                <option>Irrigation Systems</option>
                <option>Machinery</option>
                <option>Other</option>
              </select>
            </div>
            <div>
              <label className="block text-sm font-semibold text-gray-700 mb-1.5">Tell us about your products</label>
              <textarea value={form.message} onChange={(e) => update('message', e.target.value)} rows={4} placeholder="What do you produce or supply? What makes your products special?" className="w-full px-4 py-3 bg-gray-50 border border-gray-200 rounded-lg text-sm focus:outline-none focus:border-primary focus:bg-white focus:ring-1 focus:ring-primary/20 transition-all resize-none" />
            </div>
            <button type="submit" className="w-full bg-primary text-white py-3 rounded-lg font-semibold hover:bg-primary-hover transition-colors flex items-center justify-center gap-2">
              Submit Application <ArrowRight className="w-4 h-4" />
            </button>
            <p className="text-xs text-gray-400 text-center">By submitting, you agree to our <Link to="/terms" className="underline hover:text-gray-600">Terms of Service</Link> and <Link to="/privacy" className="underline hover:text-gray-600">Privacy Policy</Link>.</p>
          </form>
        )}
      </section>
    </div>
  );
}
