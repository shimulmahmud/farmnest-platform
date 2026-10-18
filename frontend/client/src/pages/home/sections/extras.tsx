import type { Review } from "@/types";
import { ArrowRight, Quote, Sprout, Star } from "lucide-react";
import { Link } from "react-router-dom";

/*
  Not mounted on the home page since the Grocno-style redesign.
  Kept as-is so they can be dropped back into HomePage if wanted.
*/

/* ─────────────────────────────────────────────
   TESTIMONIALS SECTION
   ───────────────────────────────────────────── */

interface TestimonialsSectionProps {
  reviews: Review[];
}

export function TestimonialsSection({ reviews }: TestimonialsSectionProps) {
  return (
    <section className="py-16 lg:py-20 bg-white">
      <div className="max-w-page mx-auto px-4 sm:px-6 lg:px-8">
        <div className="text-center mb-10">
          <Quote className="w-9 h-9 text-primary/30 mx-auto mb-4" />

          <span className="text-xs font-bold uppercase tracking-widest text-primary">
            Customer feedback
          </span>

          <h2 className="font-headline text-3xl lg:text-4xl font-bold text-gray-900 tracking-tight mt-2">
            What Our Customers Say
          </h2>

          <p className="text-gray-500 mt-3">
            Feedback from people who shop with FarmNest.
          </p>
        </div>

        <div className="grid md:grid-cols-3 gap-6">
          {reviews.map((r) => (
            <article
              key={r.id}
              className="bg-[#fafbf8] rounded-2xl p-6 sm:p-8 border border-gray-100 hover:shadow-md transition-shadow flex flex-col"
            >
              <div className="flex items-center gap-1 text-amber-400 mb-4">
                {[1, 2, 3, 4, 5].map((i) => (
                  <Star
                    key={i}
                    className={`w-4 h-4 ${
                      i <= r.rating
                        ? "fill-amber-400"
                        : "fill-none text-gray-300"
                    }`}
                  />
                ))}
              </div>

              <h3 className="font-headline text-lg font-bold text-gray-900 mb-2">
                {r.title}
              </h3>

              <p className="text-sm text-gray-500 leading-relaxed flex-1 line-clamp-3">
                {r.body}
              </p>

              <div className="flex items-center gap-3 mt-6 pt-4 border-t border-gray-100">
                <img
                  src={r.avatar}
                  alt={r.author}
                  loading="lazy"
                  className="w-10 h-10 rounded-full object-cover"
                />

                <div>
                  <p className="text-sm font-semibold text-gray-900">
                    {r.author}
                  </p>
                  <p className="text-xs text-gray-400">{r.location}</p>
                </div>
              </div>
            </article>
          ))}
        </div>

        <div className="text-center mt-8">
          <Link
            to="/reviews"
            className="inline-flex items-center gap-2 text-primary font-bold text-sm hover:gap-3 transition-all"
          >
            Read All Reviews
            <ArrowRight className="w-4 h-4" />
          </Link>
        </div>
      </div>
    </section>
  );
}

/* ─────────────────────────────────────────────
   CTA BANNER
   ───────────────────────────────────────────── */

export function CtaBanner() {
  return (
    <section className="relative overflow-hidden py-16 lg:py-20 bg-primary text-white">
      <div className="absolute inset-0 opacity-[0.07] pointer-events-none">
        <div className="absolute -top-24 -right-20 w-80 h-80 rounded-full border-[40px] border-white" />
        <div className="absolute -bottom-40 -left-20 w-96 h-96 rounded-full border-[50px] border-white" />
      </div>

      <div className="relative max-w-page mx-auto px-4 sm:px-6 lg:px-8 text-center">
        <span className="inline-flex items-center gap-2 text-xs font-bold uppercase tracking-widest text-white/80">
          <Sprout className="w-4 h-4" />
          Discover FarmNestBD
        </span>

        <h2 className="font-headline text-3xl sm:text-4xl lg:text-5xl font-bold tracking-tight mt-4">
          Bring the Best of the Farm to Your Home
        </h2>

        <p className="text-white/80 mt-4 max-w-xl mx-auto leading-relaxed">
          Explore our agricultural products and discover a convenient way to
          shop with FarmNestBD.
        </p>

        <div className="flex flex-wrap gap-3 justify-center mt-8">
          <Link
            to="/products"
            className="bg-white text-primary px-7 py-3.5 rounded-full font-bold hover:bg-green-50 transition-colors inline-flex items-center gap-2 shadow-sm"
          >
            Start Shopping
            <ArrowRight className="w-4 h-4" />
          </Link>

          <Link
            to="/signin"
            className="border border-white/40 px-7 py-3.5 rounded-full font-bold hover:bg-white/10 transition-colors"
          >
            Create Account
          </Link>
        </div>
      </div>
    </section>
  );
}
