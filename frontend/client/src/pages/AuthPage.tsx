import { useState } from 'react';
import { Link, useLocation, useNavigate } from 'react-router-dom';
import { AlertCircle, ArrowLeft, CheckCircle2, Loader2, Lock, Mail, MailCheck, User } from 'lucide-react';
import { useAuth } from '@/context/AuthContext';

type Tab = 'signin' | 'signup';
type Step = 'form' | 'verify';

const inputClass =
  'w-full pl-10 pr-4 py-2.5 border border-gray-300 rounded-lg text-sm focus:outline-none focus:border-primary focus:ring-1 focus:ring-primary transition-colors';

export default function AuthPage({ initialTab = 'signin' }: { initialTab?: Tab }) {
  const { signIn, signUp, verifyEmail, resendVerification } = useAuth();
  const navigate = useNavigate();
  const location = useLocation();
  // Where to go after signing in (e.g. back to checkout).
  const from = (location.state as { from?: string } | null)?.from ?? '/';

  const [tab, setTab] = useState<Tab>(initialTab);
  const [step, setStep] = useState<Step>('form');
  const [firstName, setFirstName] = useState('');
  const [lastName, setLastName] = useState('');
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [code, setCode] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [notice, setNotice] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  const switchTab = (t: Tab) => {
    setTab(t);
    setStep('form');
    setError(null);
    setNotice(null);
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    setNotice(null);

    if (!email.trim() || !password) {
      setError('Please enter your email and password.');
      return;
    }
    if (tab === 'signup') {
      if (!firstName.trim() || !lastName.trim()) {
        setError('Please enter your first and last name.');
        return;
      }
      if (password.length < 8) {
        setError('Password must be at least 8 characters.');
        return;
      }
    }

    setLoading(true);
    if (tab === 'signin') {
      const result = await signIn(email.trim(), password);
      setLoading(false);
      if (result.error) {
        if (result.needsVerification) {
          setStep('verify');
          setNotice('Your email is not verified yet. Enter the code we emailed you.');
        } else {
          setError(result.error);
        }
        return;
      }
      navigate(from, { replace: true });
      return;
    }

    const result = await signUp({ email: email.trim(), password, firstName: firstName.trim(), lastName: lastName.trim() });
    setLoading(false);
    if (result.error) {
      setError(result.error);
      return;
    }
    setStep('verify');
    setNotice('Account created. We emailed you a confirmation link. Click it, or paste the code below.');
  };

  const handleVerify = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    if (!code.trim()) {
      setError('Please paste the verification code from your email.');
      return;
    }
    setLoading(true);
    const result = await verifyEmail(code);
    if (result.error) {
      setLoading(false);
      setError(result.error);
      return;
    }
    // Verified: sign in right away.
    const signed = await signIn(email.trim(), password);
    setLoading(false);
    if (signed.error) {
      setStep('form');
      setTab('signin');
      setNotice('Email verified. Please sign in.');
      return;
    }
    navigate(from, { replace: true });
  };

  return (
    <div className="min-h-screen bg-[#f7f8f6] flex items-center justify-center px-4 py-12">
      <div className="w-full max-w-md">
        <Link to="/" className="inline-flex items-center gap-2 text-sm font-semibold text-gray-500 hover:text-primary mb-6">
          <ArrowLeft className="w-4 h-4" /> Back to home
        </Link>

        <div className="bg-white rounded-2xl border border-gray-100 shadow-sm p-8">
          <div className="flex flex-col items-center mb-7">
            <img src="/farmnest-logo.png" alt="FarmNest" className="w-12 h-12 rounded-xl object-cover mb-3" />
            <h1 className="font-headline text-2xl font-bold text-primary">
              {step === 'verify' ? 'Verify your email' : tab === 'signin' ? 'Welcome back' : 'Create your account'}
            </h1>
            <p className="text-sm text-gray-500 mt-1 text-center">
              {step === 'verify'
                ? `We sent a code to ${email || 'your email'}`
                : tab === 'signin' ? 'Sign in to continue' : 'Join in seconds'}
            </p>
          </div>

          {notice && (
            <div className="flex items-start gap-2 bg-emerald-50 border border-emerald-200 text-emerald-800 text-sm rounded-lg px-4 py-3 mb-5">
              <CheckCircle2 className="w-4 h-4 mt-0.5 flex-shrink-0" />
              <span>{notice}</span>
            </div>
          )}
          {error && (
            <div className="flex items-start gap-2 bg-red-50 border border-red-200 text-red-700 text-sm rounded-lg px-4 py-3 mb-5">
              <AlertCircle className="w-4 h-4 mt-0.5 flex-shrink-0" />
              <span>{error}</span>
            </div>
          )}

          {step === 'verify' ? (
            <form onSubmit={handleVerify} className="space-y-5">
              <div>
                <label className="block text-sm font-semibold text-gray-700 mb-2">Verification code</label>
                <div className="relative">
                  <MailCheck className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
                  <input value={code} onChange={(e) => setCode(e.target.value)} placeholder="Paste the code from the email" className={inputClass} autoFocus />
                </div>
                {import.meta.env.DEV && (
                  <p className="mt-2 text-xs text-gray-400">Local development: open Mailpit (usually http://localhost:8025) to read the email.</p>
                )}
              </div>
              <button type="submit" disabled={loading} className="w-full bg-primary text-white font-semibold py-3 rounded-lg hover:bg-primary-hover transition-colors disabled:opacity-60 flex items-center justify-center gap-2">
                {loading && <Loader2 className="w-4 h-4 animate-spin" />} Verify &amp; continue
              </button>
              <button
                type="button"
                disabled={loading || !email.trim()}
                onClick={async () => {
                  setError(null);
                  const r = await resendVerification(email.trim());
                  if (r.error) setError(r.error);
                  else setNotice('We sent you a new confirmation email.');
                }}
                className="w-full text-sm font-semibold text-primary hover:underline disabled:opacity-50"
              >
                Send the email again
              </button>
              <button type="button" onClick={() => setStep('form')} className="w-full text-sm font-semibold text-gray-500 hover:text-primary">
                Back
              </button>
            </form>
          ) : (
            <>
              <div className="flex bg-gray-100 rounded-lg p-1 mb-6">
                {(['signin', 'signup'] as Tab[]).map((t) => (
                  <button
                    key={t}
                    type="button"
                    onClick={() => switchTab(t)}
                    className={`flex-1 py-2 rounded-md text-sm font-semibold transition-colors ${tab === t ? 'bg-white text-primary shadow-sm' : 'text-gray-500 hover:text-gray-700'}`}
                  >
                    {t === 'signin' ? 'Sign In' : 'Sign Up'}
                  </button>
                ))}
              </div>

              <form onSubmit={handleSubmit} className="space-y-5">
                {tab === 'signup' && (
                  <div className="grid grid-cols-2 gap-3">
                    <div>
                      <label className="block text-sm font-semibold text-gray-700 mb-2">First name</label>
                      <div className="relative">
                        <User className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
                        <input value={firstName} onChange={(e) => setFirstName(e.target.value)} className={inputClass} autoComplete="given-name" />
                      </div>
                    </div>
                    <div>
                      <label className="block text-sm font-semibold text-gray-700 mb-2">Last name</label>
                      <input value={lastName} onChange={(e) => setLastName(e.target.value)} className="w-full px-4 py-2.5 border border-gray-300 rounded-lg text-sm focus:outline-none focus:border-primary focus:ring-1 focus:ring-primary" autoComplete="family-name" />
                    </div>
                  </div>
                )}

                <div>
                  <label className="block text-sm font-semibold text-gray-700 mb-2">Email</label>
                  <div className="relative">
                    <Mail className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
                    <input type="email" value={email} onChange={(e) => setEmail(e.target.value)} placeholder="you@example.com" className={inputClass} autoComplete="email" />
                  </div>
                </div>

                <div>
                  <label className="block text-sm font-semibold text-gray-700 mb-2">Password</label>
                  <div className="relative">
                    <Lock className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
                    <input
                      type="password"
                      value={password}
                      onChange={(e) => setPassword(e.target.value)}
                      placeholder={tab === 'signup' ? 'At least 8 characters' : 'Your password'}
                      className={inputClass}
                      autoComplete={tab === 'signin' ? 'current-password' : 'new-password'}
                    />
                  </div>
                </div>

                {tab === 'signin' && (
                  <div className="-mt-2 text-right">
                    <Link to="/forgot-password" className="text-sm font-semibold text-primary hover:underline">Forgot password?</Link>
                  </div>
                )}

                <button type="submit" disabled={loading} className="w-full bg-primary text-white font-semibold py-3 rounded-lg hover:bg-primary-hover transition-colors disabled:opacity-60 flex items-center justify-center gap-2">
                  {loading && <Loader2 className="w-4 h-4 animate-spin" />}
                  {tab === 'signin' ? 'Sign In' : 'Create Account'}
                </button>
              </form>

              <p className="text-center text-sm text-gray-500 mt-6">
                {tab === 'signin' ? (
                  <>Don&apos;t have an account? <button onClick={() => switchTab('signup')} className="text-primary font-semibold hover:underline">Sign up</button></>
                ) : (
                  <>Already have an account? <button onClick={() => switchTab('signin')} className="text-primary font-semibold hover:underline">Sign in</button></>
                )}
              </p>
            </>
          )}
        </div>
      </div>
    </div>
  );
}
