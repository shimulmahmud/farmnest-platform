import { useState } from 'react';
import { Link } from 'react-router-dom';
import {
  Briefcase, MapPin, Clock3, ArrowRight, Check, Search,
  Sprout, Laptop, Heart, TrendingUp,
} from 'lucide-react';

interface JobOpening {
  id: string;
  title: string;
  department: string;
  location: string;
  type: string;
  description: string;
}

const openings: JobOpening[] = [
  {
    id: 'agronomist',
    title: 'Field Agronomist',
    department: 'Agronomy',
    location: 'Dhaka / Remote',
    type: 'Full-time',
    description: 'Provide expert crop advisory to farmers across Bangladesh. Conduct field visits and support product recommendations.',
  },
  {
    id: 'supply-chain',
    title: 'Supply Chain Manager',
    department: 'Operations',
    location: 'Bogura',
    type: 'Full-time',
    description: 'Oversee end-to-end logistics from supplier to farmer. Manage warehouse operations, delivery partnerships, and inventory optimization across Bangladesh.',
  },
  {
    id: 'frontend-dev',
    title: 'Frontend Developer',
    department: 'Engineering',
    location: 'Remote (Bangladesh)',
    type: 'Full-time',
    description: 'Build and maintain the FarmNest web platform using React, TypeScript, and Tailwind CSS. Work closely with design to ship features that improve the farmer experience.',
  },
  {
    id: 'marketing',
    title: 'Marketing & Community Lead',
    department: 'Marketing',
    location: 'Chattogram',
    type: 'Full-time',
    description: 'Drive brand awareness and farmer community engagement. Manage social channels, coordinate on-ground events, and build partnerships with agricultural organizations.',
  },
  {
    id: 'customer-support',
    title: 'Customer Support Specialist',
    department: 'Support',
    location: 'Dhaka',
    type: 'Full-time',
    description: 'Be the voice of FarmNest. Help farmers with orders, product questions, and troubleshooting via phone, WhatsApp, and email. Fluent Bangla required; English is a plus.',
  },
];

const perks = [
  { icon: Heart, title: 'Health & Wellness', desc: 'Comprehensive health insurance for you and your dependents.' },
  { icon: Sprout, title: 'Learning Budget', desc: '৳25,000 annual budget for courses, books, and conferences.' },
  { icon: Clock3, title: 'Flexible Hours', desc: 'Core hours 11am–4pm. The rest is up to you.' },
  { icon: Laptop, title: 'Equipment', desc: 'Choose your own laptop and tools. Remote-friendly setup.' },
  { icon: TrendingUp, title: 'Growth Path', desc: 'Quarterly reviews and a clear promotion framework.' },
  { icon: Briefcase, title: 'Farm Visits', desc: 'Regular field trips to meet the farmers we serve.' },
];

export default function CareersPage() {
  const [query, setQuery] = useState('');
  const [dept, setDept] = useState('All');

  const departments = ['All', ...Array.from(new Set(openings.map((o) => o.department)))];

  const filtered = openings.filter((o) => {
    const matchesQuery = o.title.toLowerCase().includes(query.toLowerCase()) || o.description.toLowerCase().includes(query.toLowerCase());
    const matchesDept = dept === 'All' || o.department === dept;
    return matchesQuery && matchesDept;
  });

  return (
    <div className="bg-white">
      <section className="bg-gradient-to-br from-primary via-primary to-primary-dark text-white">
        <div className="max-w-page mx-auto px-4 sm:px-6 lg:px-8 py-16 lg:py-24 text-center">
          <span className="inline-flex items-center gap-2 bg-white/15 backdrop-blur rounded-full px-4 py-1.5 text-xs font-bold uppercase tracking-widest">
            <Briefcase className="w-4 h-4" /> Careers
          </span>
          <h1 className="font-headline text-4xl lg:text-5xl font-bold mt-6 tracking-tight">Grow With Us</h1>
          <p className="text-white/85 text-lg mt-5 max-w-2xl mx-auto leading-relaxed">
            We're on a mission to transform agriculture in Bangladesh. Join a team that's passionate about technology, sustainability, and making a real difference for farmers.
          </p>
        </div>
      </section>

      <section className="max-w-page mx-auto px-4 sm:px-6 lg:px-8 py-14">
        <div className="grid lg:grid-cols-3 gap-8">
          <div className="lg:col-span-1">
            <h2 className="font-headline text-2xl font-bold text-gray-900">Why FarmNest?</h2>
            <p className="text-gray-600 mt-3 text-sm leading-relaxed">
              We're a small team that moves fast and cares deeply. Every role here has direct impact on the farmers we serve.
            </p>
          </div>
          <div className="lg:col-span-2 grid sm:grid-cols-2 lg:grid-cols-3 gap-4">
            {perks.map((perk) => (
              <div key={perk.title} className="bg-white rounded-xl border border-gray-100 p-5 shadow-sm">
                <div className="w-10 h-10 rounded-lg bg-primary/10 flex items-center justify-center mb-3">
                  <perk.icon className="w-5 h-5 text-primary" />
                </div>
                <h3 className="font-headline text-sm font-bold text-gray-900">{perk.title}</h3>
                <p className="text-xs text-gray-500 mt-1.5 leading-relaxed">{perk.desc}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      <section className="bg-white border-y border-gray-100">
        <div className="max-w-page mx-auto px-4 sm:px-6 lg:px-8 py-14">
          <div className="flex flex-col sm:flex-row sm:items-end justify-between gap-4 mb-8">
            <div>
              <h2 className="font-headline text-3xl font-bold text-gray-900 tracking-tight">Open Positions</h2>
              <p className="text-gray-500 mt-2">{filtered.length} role{filtered.length !== 1 ? 's' : ''} currently available</p>
            </div>
          </div>

          <div className="flex flex-col sm:flex-row gap-3 mb-6">
            <div className="relative flex-1">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
              <input
                type="text"
                value={query}
                onChange={(e) => setQuery(e.target.value)}
                placeholder="Search roles..."
                className="w-full pl-10 pr-4 py-2.5 bg-gray-50 border border-gray-200 rounded-lg text-sm focus:outline-none focus:border-primary focus:bg-white focus:ring-1 focus:ring-primary/20 transition-all"
              />
            </div>
            <div className="flex gap-2 flex-wrap">
              {departments.map((d) => (
                <button
                  key={d}
                  onClick={() => setDept(d)}
                  className={`px-4 py-2 rounded-lg text-sm font-medium transition-colors ${
                    dept === d ? 'bg-primary text-white' : 'bg-gray-50 text-gray-600 hover:bg-gray-100'
                  }`}
                >
                  {d}
                </button>
              ))}
            </div>
          </div>

          <div className="space-y-3">
            {filtered.length === 0 ? (
              <div className="text-center py-16">
                <p className="text-gray-400 text-sm">No roles match your search right now. Check back soon or send your resume to careers@farmnest.in</p>
              </div>
            ) : (
              filtered.map((job) => (
                <div key={job.id} className="bg-gray-50 rounded-xl border border-gray-100 p-5 sm:p-6 hover:border-primary/20 hover:shadow-sm transition-all">
                  <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
                    <div className="flex-1">
                      <div className="flex flex-wrap items-center gap-2 mb-2">
                        <span className="text-xs font-bold uppercase tracking-wide text-primary bg-primary/10 px-2 py-0.5 rounded">{job.department}</span>
                        <span className="inline-flex items-center gap-1 text-xs text-gray-500"><MapPin className="w-3 h-3" /> {job.location}</span>
                        <span className="inline-flex items-center gap-1 text-xs text-gray-500"><Clock3 className="w-3 h-3" /> {job.type}</span>
                      </div>
                      <h3 className="font-headline text-lg font-bold text-gray-900">{job.title}</h3>
                      <p className="text-sm text-gray-500 mt-2 leading-relaxed">{job.description}</p>
                    </div>
                    <a
                      href={`mailto:careers@farmnest.in?subject=Application: ${job.title}`}
                      className="flex-shrink-0 bg-primary text-white px-5 py-2.5 rounded-lg text-sm font-semibold hover:bg-primary-hover transition-colors inline-flex items-center gap-2"
                    >
                      Apply <ArrowRight className="w-4 h-4" />
                    </a>
                  </div>
                </div>
              ))
            )}
          </div>
        </div>
      </section>

      <section className="max-w-page mx-auto px-4 sm:px-6 lg:px-8 py-14">
        <div className="bg-primary rounded-2xl p-8 sm:p-12 text-center text-white">
          <h2 className="font-headline text-2xl lg:text-3xl font-bold">Don't see the right fit?</h2>
          <p className="text-white/80 mt-3 max-w-lg mx-auto">Send us your resume and tell us how you'd like to contribute. We're always looking for great people.</p>
          <a href="mailto:careers@farmnest.in" className="inline-flex items-center gap-2 mt-6 bg-white text-primary px-6 py-3 rounded-full font-bold hover:bg-primary-fixed transition-colors">
            careers@farmnest.in <ArrowRight className="w-4 h-4" />
          </a>
        </div>
      </section>
    </div>
  );
}
