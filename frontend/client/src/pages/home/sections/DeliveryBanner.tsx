import { useCheckoutConfig } from "@/hooks/useCheckoutConfig";
import { heroImages } from "@/data/images";
import { formatPrice } from "@/utils/format";
import { Link } from "react-router-dom";

export default function DeliveryBanner() {
  const { config } = useCheckoutConfig();
  const standard = config?.shipping_methods.find((m) => m.id === "standard");
  if (!config || !standard) return null;

  return (
    <section className="max-w-page mx-auto px-4 py-6 sm:px-6 lg:px-8">
      <div className="relative overflow-hidden rounded-2xl bg-primary-fixed-dim">
        <img
          src={heroImages.farmAerial}
          alt=""
          loading="lazy"
          className="fade-right absolute inset-y-0 left-0 hidden h-full w-1/4 object-cover md:block"
        />
        <img
          src={heroImages.featuredSeed}
          alt=""
          loading="lazy"
          className="fade-left absolute inset-y-0 right-0 hidden h-full w-1/4 object-cover md:block"
        />

        <div className="relative z-10 mx-auto flex max-w-xl flex-col items-center px-6 py-12 text-center sm:py-14">
          <h2 className="font-headline text-3xl font-bold tracking-tight text-primary-dark sm:text-4xl">
            Home Delivery
          </h2>
          <p className="mt-3 text-sm text-on-primary-fixed-variant sm:text-base">
            {config.free_shipping_threshold > 0 && <>Free on orders above {formatPrice(config.free_shipping_threshold)}. </>}
            {standard.label} arrives in {standard.detail.toLowerCase()}.
          </p>
          <Link
            to="/products"
            className="mt-6 rounded-full bg-on-background px-7 py-3 text-sm font-bold text-white transition-colors hover:bg-primary-dark"
          >
            Shop Now
          </Link>
        </div>
      </div>
    </section>
  );
}
