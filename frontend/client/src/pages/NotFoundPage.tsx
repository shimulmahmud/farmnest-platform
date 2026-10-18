import { Link } from 'react-router-dom';
import { Sprout } from 'lucide-react';

export default function NotFoundPage() {
  return (
    <div className="min-h-[60vh] flex flex-col items-center justify-center text-center px-4">
      <div className="w-14 h-14 rounded-lg bg-primary flex items-center justify-center mb-5">
        <Sprout className="w-7 h-7 text-white" />
      </div>
      <h1 className="font-headline text-5xl font-bold text-primary">404</h1>
      <p className="text-gray-600 mt-3 max-w-sm">We couldn't find the page you were looking for. It may have been moved or no longer exists.</p>
      <Link to="/" className="mt-7 bg-primary text-white px-6 py-3 rounded-lg font-semibold hover:bg-primary-hover transition-colors">Back to Home</Link>
    </div>
  );
}
