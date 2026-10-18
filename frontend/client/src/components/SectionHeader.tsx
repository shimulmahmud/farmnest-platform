import { Link } from "react-router-dom";

interface SectionHeaderProps {
  title: string;
  to?: string;
  linkLabel?: string;
  id?: string;
}

export default function SectionHeader({ title, to, linkLabel = "View all", id }: SectionHeaderProps) {
  return (
    <div className="mb-5 flex items-center justify-between gap-4">
      <h2 id={id} className="font-headline text-xl font-bold tracking-tight text-on-surface sm:text-2xl">
        {title}
      </h2>
      {to && (
        <Link
          to={to}
          className="shrink-0 text-xs font-semibold text-on-surface-variant underline underline-offset-4 transition-colors hover:text-primary sm:text-sm"
        >
          {linkLabel}
        </Link>
      )}
    </div>
  );
}
