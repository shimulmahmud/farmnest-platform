import { Headphones, PackageCheck, ShieldCheck, Truck } from "lucide-react";

const badges = [
  { icon: Truck, title: "Convenient Delivery", desc: "Delivery options at checkout" },
  { icon: ShieldCheck, title: "Quality Focused", desc: "Carefully selected products" },
  { icon: PackageCheck, title: "Easy Ordering", desc: "Shop online with ease" },
  { icon: Headphones, title: "Customer Support", desc: "We are here to help" },
];

export default function TrustBar() {
  return (
    <section className="max-w-page mx-auto px-4 pb-10 pt-6 sm:px-6 lg:px-8" aria-label="Why shop with FarmNest">
      <ul className="grid grid-cols-1 gap-5 rounded-2xl bg-primary px-6 py-6 sm:grid-cols-2 lg:grid-cols-4 lg:px-8">
        {badges.map(({ icon: Icon, title, desc }) => (
          <li key={title} className="flex items-center gap-3 text-white">
            <span className="flex h-11 w-11 shrink-0 items-center justify-center rounded-full bg-white/15">
              <Icon className="h-5 w-5" />
            </span>
            <div>
              <h3 className="font-headline text-sm font-bold">{title}</h3>
              <p className="mt-0.5 text-xs text-white/80">{desc}</p>
            </div>
          </li>
        ))}
      </ul>
    </section>
  );
}
