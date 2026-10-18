import { ChevronRight } from "lucide-react";
import type { ReactNode } from "react";
import { Link } from "react-router-dom";

interface PageHeaderProps {
  /** Short label used in the breadcrumb. */
  crumb: string;
  title: string;
  description?: string;
  /** Optional element aligned to the right (e.g. a call-to-action button). */
  action?: ReactNode;
}

export default function PageHeader({ crumb, title, description, action }: PageHeaderProps) {
  return (
    <section className="border-b border-outline-variant/60 bg-primary-fixed/40">
      <div className="max-w-page mx-auto flex flex-wrap items-end justify-between gap-5 px-4 py-8 sm:px-6 lg:px-8 lg:py-10">
        <div>
          <nav aria-label="Breadcrumb" className="flex items-center gap-1.5 text-xs text-on-surface-variant">
            <Link to="/" className="transition-colors hover:text-primary">Home</Link>
            <ChevronRight className="h-3 w-3" aria-hidden="true" />
            <span aria-current="page" className="font-semibold text-primary">{crumb}</span>
          </nav>
          <h1 className="mt-3 font-headline text-3xl font-bold tracking-tight text-on-surface sm:text-4xl">{title}</h1>
          {description && <p className="mt-2 max-w-2xl text-sm leading-relaxed text-on-surface-variant sm:text-base">{description}</p>}
        </div>
        {action}
      </div>
    </section>
  );
}
