import { Link } from 'react-router-dom';
import {
  Sprout, Heart, Target, Leaf, Users, TrendingUp, Award,
  ShieldCheck, Truck, ArrowRight, Quote,
} from 'lucide-react';

export default function AboutPage() {
  return (
    <div className="bg-white">
      <section className="bg-gradient-to-br from-primary via-primary to-primary-dark text-white">
        <div className="max-w-page mx-auto px-4 sm:px-6 lg:px-8 py-16 lg:py-24 text-center">
          <span className="inline-flex items-center gap-2 bg-white/15 backdrop-blur rounded-full px-4 py-1.5 text-xs font-bold uppercase tracking-widest">
            <Sprout className="w-4 h-4" /> Our Story
          </span>
          <h1 className="font-headline text-4xl lg:text-5xl font-bold mt-6 tracking-tight">Empowering Bangladesh's Farmers</h1>
          <p className="text-white/85 text-lg mt-5 max-w-2xl mx-auto leading-relaxed">
            FarmNest was founded with a simple mission: to give every farmer access to quality products, fair prices, and expert guidance — all in one place.
          </p>
        </div>
      </section>

      <section className="max-w-page mx-auto px-4 sm:px-6 lg:px-8 py-16">
        <div className="grid lg:grid-cols-2 gap-12 items-center">
          <div>
            <h2 className="font-headline text-3xl font-bold text-gray-900 tracking-tight">From a small idea to a growing platform</h2>
            <p className="text-gray-600 mt-5 leading-relaxed">
              In 2026, we noticed a gap in the agricultural supply chain. Farmers across Bangladesh were spending hours travelling to nearby towns just to find quality seeds, fertilizers, and equipment — often at inflated prices with no guarantee of authenticity.
            </p>
            <p className="text-gray-600 mt-4 leading-relaxed">
              We built FarmNest to bridge that gap. By partnering directly with certified suppliers and growers, we cut out the middlemen and pass the savings on to the people who matter most — the farmers. Today, we serve thousands of farmers across the country with doorstep delivery, expert agronomist support, and a 100% quality guarantee.
            </p>
            <div className="grid grid-cols-2 gap-4 mt-8">
              <div className="bg-white rounded-xl border border-gray-100 p-5 shadow-sm">
                <p className="font-headline text-3xl font-bold text-primary">12,000+</p>
                <p className="text-sm text-gray-500 mt-1">Farmers served</p>
              </div>
              <div className="bg-white rounded-xl border border-gray-100 p-5 shadow-sm">
                <p className="font-headline text-3xl font-bold text-primary">500+</p>
                <p className="text-sm text-gray-500 mt-1">Products available</p>
              </div>
              <div className="bg-white rounded-xl border border-gray-100 p-5 shadow-sm">
                <p className="font-headline text-3xl font-bold text-primary">28</p>
                <p className="text-sm text-gray-500 mt-1">States covered</p>
              </div>
              <div className="bg-white rounded-xl border border-gray-100 p-5 shadow-sm">
                <p className="font-headline text-3xl font-bold text-primary">98%</p>
                <p className="text-sm text-gray-500 mt-1">Satisfaction rate</p>
              </div>
            </div>
          </div>
          <div className="relative rounded-2xl overflow-hidden shadow-lg h-80 lg:h-96">
            <img
              src="https://images.pexels.com/photos/2933243/pexels-photo-2933243.jpeg?auto=compress&cs=tinysrgb&w=900&h=700&fit=crop"
              alt="Farmer in field"
              className="w-full h-full object-cover"
            />
          </div>
        </div>
      </section>

      <section className="bg-white border-y border-gray-100">
        <div className="max-w-page mx-auto px-4 sm:px-6 lg:px-8 py-16">
          <div className="text-center mb-12">
            <h2 className="font-headline text-3xl lg:text-4xl font-bold text-gray-900 tracking-tight">Our Values</h2>
            <p className="text-gray-500 mt-3 max-w-xl mx-auto">The principles that guide everything we do.</p>
          </div>
          <div className="grid sm:grid-cols-2 lg:grid-cols-3 gap-6">
            {[
              { icon: Heart, title: 'Farmer-First', desc: 'Every decision starts with what benefits the farmer. Fair prices, honest advice, and genuine support.' },
              { icon: ShieldCheck, title: 'Quality Guaranteed', desc: 'Every product is sourced from certified suppliers and tested for authenticity before it reaches you.' },
              { icon: Leaf, title: 'Sustainable Future', desc: 'We promote eco-friendly, organic, and climate-resilient products that protect the land for generations.' },
              { icon: Users, title: 'Community Driven', desc: 'We listen to farmers, share knowledge, and build tools that solve real problems on the ground.' },
              { icon: TrendingUp, title: 'Fair Pricing', desc: 'By eliminating middlemen, we ensure farmers get the best prices without compromising on quality.' },
              { icon: Award, title: 'Expert Guidance', desc: 'Our agronomists are always one call away — free consultation with every order above ৳5,000.' },
            ].map((v) => (
              <div key={v.title} className="bg-gray-50 rounded-2xl p-6 border border-gray-100">
                <div className="w-12 h-12 rounded-xl bg-primary/10 flex items-center justify-center mb-4">
                  <v.icon className="w-6 h-6 text-primary" />
                </div>
                <h3 className="font-headline text-lg font-bold text-gray-900">{v.title}</h3>
                <p className="text-sm text-gray-500 mt-2 leading-relaxed">{v.desc}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      <section className="max-w-page mx-auto px-4 sm:px-6 lg:px-8 py-16">
        <div className="text-center mb-12">
          <Quote className="w-10 h-10 text-primary/30 mx-auto mb-4" />
          <h2 className="font-headline text-3xl lg:text-4xl font-bold text-gray-900 tracking-tight">Meet the People Behind FarmNest</h2>
        </div>
        <div className="grid sm:grid-cols-2 lg:grid-cols-4 gap-6">
          {[
            { name: 'Arjun Reddy', role: 'Founder & CEO', img: 'https://images.pexels.com/photos/2379004/pexels-photo-2379004.jpeg?auto=compress&cs=tinysrgb&w=400&h=400&fit=crop' },
            { name: 'Priya Sharma', role: 'Head of Operations', img: 'https://images.pexels.com/photos/3727464/pexels-photo-3727464.jpeg?auto=compress&cs=tinysrgb&w=400&h=400&fit=crop' },
            { name: 'Vikram Patel', role: 'Chief Agronomist', img: 'https://images.pexels.com/photos/3777943/pexels-photo-3777943.jpeg?auto=compress&cs=tinysrgb&w=400&h=400&fit=crop' },
            { name: 'Sneha Iyer', role: 'Head of Partnerships', img: 'https://images.pexels.com/photos/3779448/pexels-photo-3779448.jpeg?auto=compress&cs=tinysrgb&w=400&h=400&fit=crop' },
          ].map((person) => (
            <div key={person.name} className="bg-white rounded-2xl border border-gray-100 shadow-sm overflow-hidden text-center group">
              <div className="aspect-square overflow-hidden bg-gray-50">
                <img src={person.img} alt={person.name} className="w-full h-full object-cover group-hover:scale-105 transition-transform duration-500" />
              </div>
              <div className="p-5">
                <h3 className="font-headline text-base font-bold text-gray-900">{person.name}</h3>
                <p className="text-sm text-primary font-medium mt-1">{person.role}</p>
              </div>
            </div>
          ))}
        </div>
      </section>

      <section className="bg-primary text-white">
        <div className="max-w-page mx-auto px-4 sm:px-6 lg:px-8 py-14 text-center">
          <h2 className="font-headline text-3xl lg:text-4xl font-bold tracking-tight">Join the FarmNest family</h2>
          <p className="text-white/80 mt-4 max-w-xl mx-auto">Thousands of farmers already trust us for their daily needs. Start your journey today.</p>
          <div className="flex flex-wrap gap-3 justify-center mt-8">
            <Link to="/products" className="bg-white text-primary px-7 py-3.5 rounded-full font-bold hover:bg-primary-fixed transition-colors inline-flex items-center gap-2">
              Start Shopping <ArrowRight className="w-4 h-4" />
            </Link>
            <Link to="/partners" className="border border-white/40 px-7 py-3.5 rounded-full font-bold hover:bg-white/10 transition-colors">
              Partner With Us
            </Link>
          </div>
        </div>
      </section>
    </div>
  );
}
