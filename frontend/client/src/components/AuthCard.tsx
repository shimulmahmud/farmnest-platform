import type { ReactNode } from 'react';
import { Link } from 'react-router-dom';
import { ArrowLeft } from 'lucide-react';

/** Centered card used by the sign-in, forgot-password, reset-password and verify-email screens. */
export default function AuthCard({ title, subtitle, children, back = { to: '/', label: 'Back to home' } }: {
  title: string;
  subtitle?: string;
  children: ReactNode;
  back?: { to: string; label: string };
}) {
  return (
    <div className="min-h-screen bg-[#f7f8f6] flex items-center justify-center px-4 py-12">
      <div className="w-full max-w-md">
        <Link to={back.to} className="inline-flex items-center gap-2 text-sm font-semibold text-gray-500 hover:text-primary mb-6">
          <ArrowLeft className="w-4 h-4" /> {back.label}
        </Link>
        <div className="bg-white rounded-2xl border border-gray-100 shadow-sm p-8">
          <div className="flex flex-col items-center mb-7">
            <img src="/farmnest-logo.png" alt="" className="w-12 h-12 rounded-xl object-cover mb-3" />
            <h1 className="font-headline text-2xl font-bold text-primary text-center">{title}</h1>
            {subtitle && <p className="text-sm text-gray-500 mt-1 text-center">{subtitle}</p>}
          </div>
          {children}
        </div>
      </div>
    </div>
  );
}
