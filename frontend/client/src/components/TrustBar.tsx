import { Headphones, PackageCheck, ShieldCheck, Truck, Wallet } from "lucide-react";

const badges = [
  { icon: Truck, title: "Convenient Delivery", desc: "Delivery options at checkout" },
  { icon: ShieldCheck, title: "Quality Focused", desc: "Carefully selected products" },
  { icon: PackageCheck, title: "Easy Ordering", desc: "Shop online with ease" },
  { icon: Wallet, title: "Flexible Payment", desc: "bKash, Nagad, Rocket, cards, COD" },
  { icon: Headphones, title: "Customer Support", desc: "We are here to help" },
];

/** Green highlights bar. Rendered by the footer so it straddles the page/footer edge. */
export default function TrustBar() {
  return (
    <ul
      className="grid grid-cols-1 gap-x-6 gap-y-5 rounded-xl bg-primary px-6 py-5 shadow-level-3 sm:grid-cols-2 lg:grid-cols-5 lg:px-8"
      aria-label="Why shop with FarmNest"
    >
      {badges.map(({ icon: Icon, title, desc }) => (
        <li key={title} className="flex items-center gap-3 text-white">
          <span className="flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-white text-primary">
            <Icon className="h-[18px] w-[18px]" />
          </span>
          <div className="min-w-0">
            <h3 className="font-headline text-[13px] font-semibold leading-tight">{title}</h3>
            <p className="mt-0.5 text-xs leading-snug text-white/80">{desc}</p>
          </div>
        </li>
      ))}
    </ul>
  );
}
