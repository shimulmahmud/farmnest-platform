import { useState } from 'react';
import { ArrowRight, Check } from 'lucide-react';

interface NewsletterFormProps {
  variant?: 'light' | 'dark' | 'pill';
  className?: string;
}

export default function NewsletterForm({ variant = 'dark', className = '' }: NewsletterFormProps) {
  const [email, setEmail] = useState('');
  const [subscribed, setSubscribed] = useState(false);
  const [error, setError] = useState('');

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!email.trim() || !email.includes('@')) {
      setError('Please enter a valid email.');
      return;
    }
    setError('');
    setSubscribed(true);
  };

  if (subscribed) {
    return (
      <div className={`rounded-lg p-4 text-center text-sm font-semibold ${variant !== 'light' ? 'bg-white/15 text-white' : 'bg-green-50 text-green-700'} ${className}`}>
        <span className="flex items-center justify-center gap-2">
          <Check className="w-4 h-4" /> Subscribed! Watch your inbox for updates.
        </span>
      </div>
    );
  }

  const isDark = variant === 'dark';

  if (variant === 'pill') {
    return (
      <form onSubmit={handleSubmit} className={className} noValidate>
        <div className="relative">
          <input
            type="email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            placeholder="Email address"
            aria-label="Email address"
            className="w-full rounded-full border border-transparent bg-white/[0.08] py-3 pl-5 pr-14 text-[13px] text-white placeholder-white/45 transition-colors focus:border-white/40 focus:outline-none"
          />
          <button
            type="submit"
            aria-label="Subscribe"
            className="absolute right-1.5 top-1/2 flex h-8 w-8 -translate-y-1/2 items-center justify-center rounded-full bg-primary text-white transition-colors hover:bg-primary-hover focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-white"
          >
            <ArrowRight className="h-4 w-4" />
          </button>
        </div>
        {error && <p className="mt-2 text-xs text-white/70">{error}</p>}
      </form>
    );
  }

  return (
    <form onSubmit={handleSubmit} className={`space-y-3 ${className}`}>
      <input
        type="email"
        value={email}
        onChange={(e) => setEmail(e.target.value)}
        placeholder="your@email.com"
        className={`w-full rounded-lg px-4 py-3 text-sm focus:outline-none transition-colors ${
          isDark
            ? 'bg-white/10 border border-white/30 text-white placeholder-white/50 focus:border-white/70'
            : 'border border-gray-300 focus:border-primary'
        }`}
      />
      {error && (
        <p className={`text-xs ${isDark ? 'text-white/70' : 'text-red-600'}`}>{error}</p>
      )}
      <button
        type="submit"
        className={`w-full py-3 rounded-lg text-sm font-bold transition-colors flex items-center justify-center gap-2 ${
          isDark
            ? 'bg-white text-primary hover:bg-primary-fixed'
            : 'bg-primary text-white hover:bg-primary-hover'
        }`}
      >
        Subscribe <ArrowRight className="w-4 h-4" />
      </button>
    </form>
  );
}
