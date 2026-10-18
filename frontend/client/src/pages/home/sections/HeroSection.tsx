import { useCatalog } from "@/context/CatalogContext";
import { formatPrice } from "@/utils/format";
import { ArrowRight, ChevronLeft, ChevronRight } from "lucide-react";
import { useCallback, useEffect, useState } from "react";
import { Link } from "react-router-dom";

const buildSlides = (startingPrice: string, firstCategory: string) => [
  {
    badge: "100% Organic & Fresh",
    heading: "Fresh Healthy Agricultural Products",
    subheading:
      "Explore quality agricultural products, from seeds and fresh produce to grains and everyday farm essentials.",
    cta: "Shop Now",
    ctaLink: "/products",
    secondaryCta: "View Offers",
    secondaryLink: "/offers",
    image:
      "https://images.pexels.com/photos/32908660/pexels-photo-32908660.jpeg?auto=compress&cs=tinysrgb&w=1000&h=700&fit=crop",
  },
  {
    badge: "From Farmers Field to Your Kitchen",
    heading: "Premium Seeds & Fresh Produce",
    subheading:
      `Starting from ${startingPrice} — discover handpicked products, delivered to your door across Bangladesh.`,
    cta: "Shop Now",
    ctaLink: firstCategory ? `/products?category=${firstCategory}` : "/products",
    secondaryCta: "Explore Categories",
    secondaryLink: "/#categories",
    image:
      "https://images.pexels.com/photos/7891849/pexels-photo-7891849.jpeg?auto=compress&cs=tinysrgb&w=1000&h=700&fit=crop",
  },
  {
    badge: "Smart Farm Technology",
    heading: "Modern Solutions for Modern Farms",
    subheading:
      "Explore IoT sensors, drip irrigation kits, and smart farming tools to maximize your yield and efficiency.",
    cta: "Explore Tech",
    ctaLink: "/technology",
    secondaryCta: "Learn More",
    secondaryLink: "/about",
    image:
      "https://images.pexels.com/photos/34182311/pexels-photo-34182311.jpeg?auto=compress&cs=tinysrgb&w=1000&h=700&fit=crop",
  },
];

const AUTOPLAY_MS = 6000;

export default function HeroSection() {
  const { products, categories } = useCatalog();
  const startingPrice = products.length ? formatPrice(Math.min(...products.map((p) => p.price))) : "a low price";
  const heroSlides = buildSlides(startingPrice, categories[0]?.slug ?? "");
  const [current, setCurrent] = useState(0);
  const [paused, setPaused] = useState(false);
  const total = heroSlides.length;

  const next = useCallback(() => setCurrent((c) => (c + 1) % total), [total]);
  const prev = useCallback(() => setCurrent((c) => (c - 1 + total) % total), [total]);

  useEffect(() => {
    if (paused || window.matchMedia("(prefers-reduced-motion: reduce)").matches) return;
    const timer = setInterval(next, AUTOPLAY_MS);
    return () => clearInterval(timer);
  }, [next, paused]);

  const slide = heroSlides[current];

  return (
    <section className="max-w-page mx-auto px-4 pt-5 sm:px-6 lg:px-8" aria-roledescription="carousel" aria-label="Featured">
      <div
        className="relative min-h-[340px] overflow-hidden rounded-2xl bg-gradient-to-r from-primary-dark via-primary-container to-primary sm:min-h-[400px] lg:min-h-[440px]"
        onMouseEnter={() => setPaused(true)}
        onMouseLeave={() => setPaused(false)}
        onFocus={() => setPaused(true)}
        onBlur={() => setPaused(false)}
      >
        {heroSlides.map((s, i) => (
          <img
            key={s.image}
            src={s.image}
            alt=""
            aria-hidden="true"
            className={`fade-left absolute inset-y-0 right-0 h-full w-full object-cover opacity-0 transition-opacity duration-700 sm:w-3/5 ${
              i === current ? "!opacity-40 sm:!opacity-100" : ""
            }`}
          />
        ))}

        <div
          key={current}
          className="relative z-10 flex min-h-[340px] max-w-xl flex-col items-start justify-center px-6 py-10 animate-fade-in sm:min-h-[400px] sm:px-10 lg:min-h-[440px] lg:px-14"
        >
          <span className="rounded bg-accent px-2.5 py-1 text-xs font-bold text-on-secondary-fixed">{slide.badge}</span>

          <h1 className="mt-5 font-headline text-3xl font-semibold leading-[1.1] tracking-tight text-white sm:text-4xl lg:text-5xl">
            {slide.heading}
          </h1>

          <p className="mt-4 max-w-md text-sm leading-relaxed text-white/85 sm:text-base">{slide.subheading}</p>

          <div className="mt-7 flex flex-wrap gap-3">
            <Link
              to={slide.ctaLink}
              className="inline-flex items-center gap-2 rounded-full bg-accent px-6 py-3 text-sm font-bold text-on-secondary-fixed transition-colors hover:bg-accent-hover"
            >
              {slide.cta}
              <ArrowRight className="h-4 w-4" />
            </Link>
            <Link
              to={slide.secondaryLink}
              className="rounded-full border border-white/40 px-6 py-3 text-sm font-bold text-white transition-colors hover:bg-white/10"
            >
              {slide.secondaryCta}
            </Link>
          </div>
        </div>

        {/* Slide dots (vertical, right edge) */}
        <div className="absolute right-4 top-1/2 z-20 flex -translate-y-1/2 flex-col gap-2">
          {heroSlides.map((s, i) => (
            <button
              key={s.heading}
              onClick={() => setCurrent(i)}
              className={`w-2 rounded-full transition-all duration-300 ${
                i === current ? "h-6 bg-white" : "h-2 bg-white/45 hover:bg-white/70"
              }`}
              aria-label={`Go to slide ${i + 1}`}
              aria-current={i === current}
            />
          ))}
        </div>

        <div className="absolute bottom-4 right-4 z-20 hidden gap-2 sm:flex">
          <button
            onClick={prev}
            className="flex h-9 w-9 items-center justify-center rounded-full bg-white/15 text-white backdrop-blur-sm transition-colors hover:bg-white/30"
            aria-label="Previous slide"
          >
            <ChevronLeft className="h-4 w-4" />
          </button>
          <button
            onClick={next}
            className="flex h-9 w-9 items-center justify-center rounded-full bg-white/15 text-white backdrop-blur-sm transition-colors hover:bg-white/30"
            aria-label="Next slide"
          >
            <ChevronRight className="h-4 w-4" />
          </button>
        </div>
      </div>
    </section>
  );
}
