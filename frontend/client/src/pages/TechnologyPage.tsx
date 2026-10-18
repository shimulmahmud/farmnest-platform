import { useState } from 'react';
import { ArrowRight, Droplets, Leaf, Map, Radio, Sprout, TimerReset, Check, Loader2 } from 'lucide-react';
import { Link } from 'react-router-dom';
import { heroImages } from '@/data/images';
import { Modal } from '@/components/ui';

export default function TechnologyPage() {
  const [consultOpen, setConsultOpen] = useState(false);

  return (
    <div className="bg-white text-gray-900">
      <section className="relative overflow-hidden min-h-[560px] flex items-center">
        <img src={heroImages.farmAerial} alt="Precision farm fields" className="absolute inset-0 w-full h-full object-cover opacity-30" />
        <div className="absolute inset-0 bg-gradient-to-r from-white via-white/95 to-white/20" />
        <div className="relative max-w-7xl mx-auto w-full px-6 lg:px-10 py-20">
          <div className="max-w-xl">
            <span className="inline-flex items-center gap-2 rounded-full bg-gray-100 border border-gray-200 px-3 py-1 text-[11px] font-bold uppercase tracking-widest text-primary">
              <span className="w-1.5 h-1.5 rounded-full bg-primary" /> Smart Farm Initiative
            </span>
            <h1 className="font-headline text-5xl md:text-6xl font-bold leading-[1.05] mt-6">
              Data-Driven Yields,<br /><span className="text-primary">Sustainable Future.</span>
            </h1>
            <p className="text-gray-600 leading-relaxed mt-6 max-w-lg">
              Transform your agricultural operations with FarmNest. Our integrated IoT ecosystems provide real-time precision insights, optimizing resource use while maximizing harvest potential for enterprise-scale farms.
            </p>
            <div className="flex flex-wrap gap-3 mt-8">
              <button
                onClick={() => setConsultOpen(true)}
                className="inline-flex items-center gap-2 bg-primary text-white px-5 py-3 font-semibold hover:bg-primary-hover transition-colors"
              >
                Request Free Consultation <ArrowRight className="w-4 h-4" />
              </button>
              <a href="#technology" className="inline-flex items-center gap-2 border border-gray-300 px-5 py-3 font-semibold hover:border-primary hover:text-primary transition-colors">
                Explore Technology
              </a>
            </div>
          </div>
          <div className="hidden lg:block absolute right-10 top-1/2 -translate-y-1/2 w-80 bg-white/90 backdrop-blur-md rounded-2xl border border-white shadow-xl p-7">
            <div className="flex justify-between items-center pb-5 border-b border-gray-200">
              <h2 className="font-headline text-xl font-semibold">System Health</h2>
              <span className="w-10 h-10 rounded-full bg-primary/10 text-primary flex items-center justify-center">
                <Leaf className="w-5 h-5" />
              </span>
            </div>
            <div className="pt-5">
              <div className="flex justify-between text-xs font-semibold mb-2">
                <span>Soil Moisture Index</span>
                <span className="bg-primary/10 text-primary px-2 py-1 rounded">Optimal</span>
              </div>
              <div className="h-2 bg-gray-200 rounded-full overflow-hidden">
                <div className="w-[84%] h-full bg-primary rounded-full" />
              </div>
            </div>
            <div className="grid grid-cols-2 gap-4 mt-8">
              <div className="rounded-xl bg-gray-50 p-4">
                <span className="text-[10px] uppercase tracking-wider text-gray-500">Active Sensors</span>
                <strong className="block text-2xl mt-1">1,248</strong>
                <small className="text-primary">↗ +12%</small>
              </div>
              <div className="rounded-xl bg-gray-50 p-4">
                <span className="text-[10px] uppercase tracking-wider text-gray-500">Resource Efficiency</span>
                <strong className="block text-2xl mt-1">94.2%</strong>
                <small className="text-primary">Target met</small>
              </div>
            </div>
          </div>
        </div>
      </section>

      <section id="technology" className="bg-white py-20 px-6 lg:px-10 bg-[linear-gradient(rgba(19,77,29,.04)_1px,transparent_1px),linear-gradient(90deg,rgba(19,77,29,.04)_1px,transparent_1px)] bg-[size:28px_28px]">
        <div className="max-w-7xl mx-auto">
          <div className="text-center max-w-2xl mx-auto">
            <h2 className="font-headline text-3xl md:text-4xl font-bold">Precision from Seed to Silo</h2>
            <p className="text-gray-600 mt-4 leading-relaxed">Our integrated IoT architecture creates a continuous feedback loop, turning raw environmental data into actionable agronomic intelligence.</p>
          </div>
          <div className="grid lg:grid-cols-3 gap-5 mt-14">
            <div className="lg:col-span-2 rounded-2xl border border-gray-100 bg-white p-8 grid md:grid-cols-2 gap-8 items-center shadow-sm">
              <div>
                <span className="inline-flex p-3 rounded-lg bg-primary/10 text-primary"><Map /></span>
                <h3 className="font-headline text-2xl font-semibold mt-5">Macro-Level Geospatial Mapping</h3>
                <p className="text-gray-600 leading-relaxed mt-3">Integrate drone imagery and satellite topographical data to identify micro-climates and variation across thousands of acres before planting begins.</p>
                <button onClick={() => setConsultOpen(true)} className="inline-flex items-center gap-2 text-primary text-sm font-bold mt-8 hover:gap-3 transition-all">
                  Learn how it works <ArrowRight className="w-4 h-4" />
                </button>
              </div>
              <img src={heroImages.smartTech} alt="Farm technology dashboard" className="rounded-xl h-64 w-full object-cover" />
            </div>
            <div className="rounded-2xl border border-gray-100 bg-white p-8 shadow-sm">
              <span className="inline-flex p-3 rounded-lg bg-blue-50 text-blue-700"><Droplets /></span>
              <h3 className="font-headline text-2xl font-semibold mt-5">Automated Hydration</h3>
              <p className="text-gray-600 leading-relaxed mt-3">Smart valve integration responds to real-time subterranean moisture sensors, preventing over-watering and nutrient runoff.</p>
              <div className="mt-12 pt-5 border-t border-gray-100 flex justify-between items-center">
                <span className="text-sm text-gray-500">Water Saved (Avg)</span>
                <strong className="text-2xl text-blue-700">32%</strong>
              </div>
            </div>
            <div className="rounded-2xl border border-gray-100 bg-white p-8 shadow-sm">
              <h3 className="font-headline text-2xl font-semibold">Soil Biome Analysis <Sprout className="inline w-5 h-5 text-amber-700" /></h3>
              <p className="text-gray-600 leading-relaxed mt-4">Continuous NPK monitoring ensures fertilizer is applied exactly when and where the soil demands it.</p>
              <div className="h-2 bg-gray-200 rounded-full mt-10"><div className="w-3/4 h-full bg-amber-800 rounded-full" /></div>
            </div>
            <div className="lg:col-span-2 rounded-2xl border border-gray-100 bg-white p-8 grid md:grid-cols-3 gap-6 items-center shadow-sm text-center">
              <div><Radio className="mx-auto text-primary" /><strong className="block text-2xl mt-3">24/7</strong><span className="text-xs uppercase tracking-widest text-gray-500">Uptime Monitoring</span></div>
              <div><TimerReset className="mx-auto text-primary" /><strong className="block text-2xl mt-3">&lt; 1ms</strong><span className="text-xs uppercase tracking-widest text-gray-500">Data Latency</span></div>
              <div><Sprout className="mx-auto text-primary" /><strong className="block text-2xl mt-3">10k+</strong><span className="text-xs uppercase tracking-widest text-gray-500">Node Capacity</span></div>
            </div>
          </div>
        </div>
      </section>

      {consultOpen && <ConsultationModal onClose={() => setConsultOpen(false)} />}
    </div>
  );
}

function ConsultationModal({ onClose }: { onClose: () => void }) {
  const [name, setName] = useState('');
  const [email, setEmail] = useState('');
  const [phone, setPhone] = useState('');
  const [farmSize, setFarmSize] = useState('');
  const [message, setMessage] = useState('');
  const [error, setError] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const [submitted, setSubmitted] = useState(false);

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!name.trim() || !email.trim() || !phone.trim()) {
      setError('Please fill in your name, email, and phone number.');
      return;
    }
    if (!email.includes('@')) {
      setError('Please enter a valid email address.');
      return;
    }
    setError('');
    setSubmitting(true);
    setTimeout(() => {
      setSubmitting(false);
      setSubmitted(true);
    }, 1200);
  };

  return (
    <Modal open onClose={onClose}>
      {submitted ? (
        <div className="p-10 text-center">
          <div className="mx-auto mb-5 w-14 h-14 rounded-full bg-primary/10 text-primary flex items-center justify-center">
            <Check className="w-7 h-7" />
          </div>
          <h2 className="font-headline text-2xl font-bold text-primary mb-3">Request Received!</h2>
          <p className="text-gray-600 leading-relaxed mb-6">
            Thank you for your interest. Our smart farming team will contact you within 24 hours to schedule your free consultation.
          </p>
          <button onClick={onClose} className="bg-primary text-white px-6 py-3 rounded-lg font-semibold hover:bg-primary-hover transition-colors">
            Close
          </button>
        </div>
      ) : (
        <>
          <div className="p-6 border-b border-gray-100">
            <h2 className="font-headline text-xl font-bold text-on-surface">Free Consultation</h2>
            <p className="text-sm text-gray-500 mt-0.5">Our agronomist team will reach out to you.</p>
          </div>
          <form onSubmit={handleSubmit} className="p-6 space-y-5">
            {error && (
              <div className="bg-red-50 border border-red-200 text-red-700 text-sm rounded-lg px-4 py-3">
                {error}
              </div>
            )}
            <label className="block text-sm font-semibold text-gray-700">
              Full Name
              <input type="text" value={name} onChange={(e) => setName(e.target.value)} placeholder="John Doe" className="mt-2 w-full border border-gray-300 rounded-lg px-3 py-2.5 text-sm font-normal focus:border-primary focus:outline-none" />
            </label>
            <div className="grid sm:grid-cols-2 gap-4">
              <label className="block text-sm font-semibold text-gray-700">
                Email
                <input type="email" value={email} onChange={(e) => setEmail(e.target.value)} placeholder="you@example.com" className="mt-2 w-full border border-gray-300 rounded-lg px-3 py-2.5 text-sm font-normal focus:border-primary focus:outline-none" />
              </label>
              <label className="block text-sm font-semibold text-gray-700">
                Phone
                <input type="tel" value={phone} onChange={(e) => setPhone(e.target.value)} placeholder="01XXXXXXXXX" className="mt-2 w-full border border-gray-300 rounded-lg px-3 py-2.5 text-sm font-normal focus:border-primary focus:outline-none" />
              </label>
            </div>
            <label className="block text-sm font-semibold text-gray-700">
              Farm Size (acres)
              <input type="text" value={farmSize} onChange={(e) => setFarmSize(e.target.value)} placeholder="e.g. 50 acres" className="mt-2 w-full border border-gray-300 rounded-lg px-3 py-2.5 text-sm font-normal focus:border-primary focus:outline-none" />
            </label>
            <label className="block text-sm font-semibold text-gray-700">
              Tell us about your needs
              <textarea value={message} onChange={(e) => setMessage(e.target.value)} placeholder="What challenges are you looking to solve?" rows={3} className="mt-2 w-full border border-gray-300 rounded-lg px-3 py-2.5 text-sm font-normal focus:border-primary focus:outline-none resize-none" />
            </label>
            <button type="submit" disabled={submitting} className="w-full bg-primary text-white font-semibold py-3 rounded-lg hover:bg-primary-hover transition-colors disabled:opacity-60 flex items-center justify-center gap-2">
              {submitting && <Loader2 className="w-4 h-4 animate-spin" />}
              {submitting ? 'Submitting...' : 'Request Consultation'}
            </button>
          </form>
        </>
      )}
    </Modal>
  );
}
