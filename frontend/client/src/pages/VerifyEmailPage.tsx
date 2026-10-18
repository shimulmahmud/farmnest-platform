import { useEffect, useRef, useState } from 'react';
import { Link, useSearchParams } from 'react-router-dom';
import { CheckCircle2, Loader2, XCircle } from 'lucide-react';
import AuthCard from '@/components/AuthCard';
import { verifyEmail } from '@/api/auth';

/** Opened from the button in the verification email: confirms the address automatically. */
export default function VerifyEmailPage() {
  const [params] = useSearchParams();
  const token = params.get('token') ?? '';
  const [state, setState] = useState<'working' | 'ok' | 'failed'>(token ? 'working' : 'failed');
  const [message, setMessage] = useState(token ? '' : 'This link is incomplete.');
  const started = useRef(false); // React StrictMode runs effects twice in development; a token works only once.

  useEffect(() => {
    if (!token || started.current) return;
    started.current = true;
    verifyEmail(token)
      .then(() => setState('ok'))
      .catch((e: unknown) => {
        setState('failed');
        setMessage(e instanceof Error ? e.message : 'Verification failed.');
      });
  }, [token]);

  return (
    <AuthCard title={state === 'ok' ? 'Email confirmed' : state === 'failed' ? 'Could not confirm your email' : 'Confirming your email…'}>
      <div className="text-center space-y-4">
        {state === 'working' && <Loader2 className="mx-auto w-10 h-10 animate-spin text-primary" />}
        {state === 'ok' && (
          <>
            <CheckCircle2 className="mx-auto w-10 h-10 text-emerald-600" />
            <p className="text-sm text-gray-700">Thank you! Your account is active. You can sign in now.</p>
            <Link to="/signin" className="inline-block rounded-lg bg-primary px-6 py-2.5 text-sm font-semibold text-white hover:bg-primary-hover">Sign in</Link>
          </>
        )}
        {state === 'failed' && (
          <>
            <XCircle className="mx-auto w-10 h-10 text-red-500" />
            <p className="text-sm text-gray-700">{message || 'The link is not valid or has expired.'}</p>
            <p className="text-xs text-gray-500">If you already confirmed your email, just sign in. Otherwise sign in again to get a new verification email.</p>
            <Link to="/signin" className="inline-block rounded-lg bg-primary px-6 py-2.5 text-sm font-semibold text-white hover:bg-primary-hover">Go to sign in</Link>
          </>
        )}
      </div>
    </AuthCard>
  );
}
