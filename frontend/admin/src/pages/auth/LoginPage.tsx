import { useState, useEffect, type FormEvent } from 'react';
import { APP_NAME } from '../../config/brand';
import { Link, useNavigate } from 'react-router-dom';
import { useMutation } from '@tanstack/react-query';
import { EnvelopeIcon, LockClosedIcon, EyeIcon, EyeSlashIcon } from '@heroicons/react/24/outline';
import { login } from '../../api/auth';
import { useAuthStore } from '../../stores/authStore';
import { cn } from '../../lib/utils';
import type { User, TokenResponse } from '../../types';

// Dev-mode mock admin user (only used when backend is unavailable)
const DEV_ADMIN: { user: User; tokens: TokenResponse } = {
  user: {
    id: 'dev-super-admin-001',
    email: 'admin@ecommerce.com',
    first_name: 'Super',
    last_name: 'Admin',
    role: 'SUPER_ADMIN',
    status: 'ACTIVE',
    email_verified: true,
    created_at: new Date().toISOString(),
  },
  tokens: {
    access_token: 'dev-access-token-placeholder',
    refresh_token: 'dev-refresh-token-placeholder',
    expires_at: new Date(Date.now() + 24 * 60 * 60 * 1000).toISOString(),
  },
};

function isAdminRole(role: string): boolean {
  const r = role.toUpperCase();
  return r === 'SUPER_ADMIN' || r === 'ADMIN' || r === 'MANAGER';
}

export default function LoginPage() {
  const navigate = useNavigate();
  const setAuth = useAuthStore((s) => s.setAuth);
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated);
  const isAdmin = useAuthStore((s) => s.isAdmin);

  // Safety net: if user is already authenticated (e.g., restored from localStorage),
  // redirect them away from the login page immediately.
  useEffect(() => {
    if (isAuthenticated) {
      if (isAdmin) {
        navigate('/admin', { replace: true });
      } else {
        navigate('/login', { replace: true });
      }
    }
  }, [isAuthenticated, isAdmin, navigate]);

  const [form, setForm] = useState({
    email: '',
    password: '',
  });
  const [showPassword, setShowPassword] = useState(false);
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});

  const loginMutation = useMutation({
    mutationFn: login,
    onSuccess: (data) => {
      setAuth(data.user, data.tokens);
      // Navigate directly via setTimeout to escape React 19 batching.
      // This is the primary redirect mechanism after a fresh login.
      setTimeout(() => {
        if (isAdminRole(data.user.role)) {
          navigate('/admin', { replace: true });
        } else {
          // Non-admin users have no access to this panel
          setFieldErrors({ general: 'Access denied. Admin access required.' });
        }
      }, 10);
    },
    onError: (error: unknown) => {
      const err = error as { response?: { data?: { error?: { message?: string; code?: string }; message?: string } } };
      const message =
        err?.response?.data?.error?.message ||
        err?.response?.data?.message ||
        'Login failed. Please check your email and password.';
      setFieldErrors({ general: message });
    },
  });

  const handleDevLogin = () => {
    setAuth(DEV_ADMIN.user, DEV_ADMIN.tokens);
    setTimeout(() => navigate('/admin', { replace: true }), 10);
  };

  const handleSubmit = (e: FormEvent) => {
    e.preventDefault();
    setFieldErrors({});
    const errors: Record<string, string> = {};
    if (!form.email.trim()) {
      errors.email = 'Please enter your email address';
    } else if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(form.email)) {
      errors.email = 'Please enter a valid email address';
    }
    if (!form.password) {
      errors.password = 'Please enter your password';
    }

    if (Object.keys(errors).length > 0) {
      setFieldErrors(errors);
      return;
    }

    loginMutation.mutate({
      email: form.email.trim(),
      password: form.password,
    });
  };

  return (
    <div className="min-h-screen flex flex-col items-center justify-center bg-surface-50 px-4 py-12">
      <div className="w-full max-w-md">
        {/* Logo / Branding */}
        <div className="text-center mb-8">
          <Link to="/admin" className="inline-flex items-center gap-2.5">
            <div className="w-9 h-9 bg-primary-600 rounded-xl flex items-center justify-center">
              <svg className="h-5 w-5 text-white" fill="none" viewBox="0 0 24 24" strokeWidth={2} stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" d="M2.25 3h1.386c.51 0 .955.343 1.087.835l.383 1.437M7.5 14.25a3 3 0 00-3 3h15.75m-12.75-3h11.218c1.121-2.3 2.1-4.684 2.924-7.138a60.114 60.114 0 00-16.536-1.84M7.5 14.25L5.106 5.272M6 20.25a.75.75 0 11-1.5 0 .75.75 0 011.5 0zm12.75 0a.75.75 0 11-1.5 0 .75.75 0 011.5 0z" />
              </svg>
            </div>
            <span className="text-xl font-bold text-gray-900 tracking-tight">{APP_NAME}</span>
          </Link>
        </div>

        {/* Card */}
        <div className="card shadow-soft">
          <h1 className="text-2xl font-bold text-gray-900 text-center mb-2">
            Welcome Back
          </h1>
          <p className="text-sm text-gray-500 text-center mb-8">
            Sign in to your account to continue
          </p>

          {/* General error */}
          {fieldErrors.general && (
            <div className="mb-6 rounded-xl bg-red-50 border border-red-200 p-3">
              <p className="text-sm text-red-700">{fieldErrors.general}</p>
            </div>
          )}

          <form onSubmit={handleSubmit} className="space-y-5" noValidate>
            {/* Email */}
            <div>
              <label htmlFor="email" className="label-field">
                Email Address
              </label>
              <div className="relative">
                <div className="pointer-events-none absolute inset-y-0 left-0 flex items-center pl-3">
                  <EnvelopeIcon className="h-4 w-4 text-gray-400" />
                </div>
                <input
                  id="email"
                  type="email"
                  autoComplete="email"
                  value={form.email}
                  onChange={(e) =>
                    setForm((prev) => ({ ...prev, email: e.target.value }))
                  }
                  className={cn(
                    'input-field pl-10',
                    fieldErrors.email && 'border-red-500 focus:border-red-500 focus:ring-red-500',
                  )}
                  placeholder="you@example.com"
                />
              </div>
              {fieldErrors.email && (
                <p className="mt-1 text-xs text-red-600">{fieldErrors.email}</p>
              )}
            </div>

            {/* Password */}
            <div>
              <div className="flex items-center justify-between">
                <label htmlFor="password" className="label-field">
                  Password
                </label>
                <Link
                  to="/forgot-password"
                  className="text-xs font-medium text-primary-600 hover:text-primary-700 transition-colors"
                >
                  Forgot password?
                </Link>
              </div>
              <div className="relative">
                <div className="pointer-events-none absolute inset-y-0 left-0 flex items-center pl-3">
                  <LockClosedIcon className="h-4 w-4 text-gray-400" />
                </div>
                <input
                  id="password"
                  type={showPassword ? 'text' : 'password'}
                  autoComplete="current-password"
                  value={form.password}
                  onChange={(e) =>
                    setForm((prev) => ({ ...prev, password: e.target.value }))
                  }
                  className={cn(
                    'input-field pl-10 pr-10',
                    fieldErrors.password && 'border-red-500 focus:border-red-500 focus:ring-red-500',
                  )}
                  placeholder="Enter your password"
                />
                <button
                  type="button"
                  onClick={() => setShowPassword(!showPassword)}
                  className="absolute inset-y-0 right-0 flex items-center pr-3 text-gray-400 hover:text-gray-600 transition-colors"
                  tabIndex={-1}
                  aria-label={showPassword ? 'Hide password' : 'Show password'}
                >
                  {showPassword ? (
                    <EyeSlashIcon className="h-4 w-4" />
                  ) : (
                    <EyeIcon className="h-4 w-4" />
                  )}
                </button>
              </div>
              {fieldErrors.password && (
                <p className="mt-1 text-xs text-red-600">{fieldErrors.password}</p>
              )}
            </div>

            {/* Dev-mode bypass */}
            {import.meta.env.DEV && (
              <div className="relative">
                <div className="absolute inset-0 flex items-center">
                  <div className="w-full border-t border-gray-200" />
                </div>
                <div className="relative flex justify-center text-xs">
                  <span className="bg-white px-3 text-gray-400">or</span>
                </div>
              </div>
            )}
            {import.meta.env.DEV && (
              <button
                type="button"
                onClick={handleDevLogin}
                className="btn-secondary w-full py-3"
              >
                Dev Login (No Backend)
              </button>
            )}

            {/* Submit */}
            <button
              type="submit"
              disabled={loginMutation.isPending}
              className="btn-primary w-full py-3"
            >
              {loginMutation.isPending ? (
                <span className="inline-flex items-center gap-2">
                  <svg className="animate-spin h-4 w-4" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
                    <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                    <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
                  </svg>
                  Signing in...
                </span>
              ) : (
                'Sign In'
              )}
            </button>
          </form>


        </div>

        {/* Bottom text */}
        <p className="mt-8 text-center text-xs text-gray-400">
          &copy; {new Date().getFullYear()} {APP_NAME}. All rights reserved.
        </p>
      </div>
    </div>
  );
}
