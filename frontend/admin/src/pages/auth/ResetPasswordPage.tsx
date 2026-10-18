import { useState, type FormEvent, useEffect } from 'react';
import { APP_NAME } from '../../config/brand';
import { Link, useSearchParams } from 'react-router-dom';
import { useMutation } from '@tanstack/react-query';
import {
  LockClosedIcon,
  EyeIcon,
  EyeSlashIcon,
  CheckCircleIcon,
  ExclamationTriangleIcon,
} from '@heroicons/react/24/outline';
import { resetPassword } from '../../api/auth';
import { cn } from '../../lib/utils';

export default function ResetPasswordPage() {
  const [searchParams] = useSearchParams();
  const token = searchParams.get('token') || '';

  const [form, setForm] = useState({
    new_password: '',
    confirm_password: '',
  });
  const [showPassword, setShowPassword] = useState(false);
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});
  const [isSuccess, setIsSuccess] = useState(false);

  // Redirect if no token is present
  useEffect(() => {
    if (!token) {
      setFieldErrors({
        general: 'Invalid or missing reset token. Please request a new password reset link.',
      });
    }
  }, [token]);

  const resetMutation = useMutation({
    mutationFn: resetPassword,
    onSuccess: () => {
      setIsSuccess(true);
    },
    onError: (error: unknown) => {
      const err = error as {
        response?: { data?: { message?: string; error?: string } };
      };
      const message =
        err?.response?.data?.message ||
        err?.response?.data?.error ||
        'Reset failed. The token may have expired. Please request a new password reset link.';
      setFieldErrors({ general: message });
    },
  });

  const handleSubmit = (e: FormEvent) => {
    e.preventDefault();
    setFieldErrors({});

    const errors: Record<string, string> = {};
    if (!form.new_password) {
      errors.new_password = 'Please enter a new password';
    } else if (form.new_password.length < 8) {
      errors.new_password = 'Password must be at least 8 characters';
    }
    if (!form.confirm_password) {
      errors.confirm_password = 'Please confirm your new password';
    } else if (form.new_password !== form.confirm_password) {
      errors.confirm_password = 'Passwords do not match';
    }

    if (Object.keys(errors).length > 0) {
      setFieldErrors(errors);
      return;
    }

    resetMutation.mutate({
      token,
      new_password: form.new_password,
    });
  };

  return (
    <div className="min-h-screen flex flex-col items-center justify-center bg-gray-50 px-4 py-12">
      <div className="w-full max-w-md">
        {/* Logo / Branding */}
        <div className="text-center mb-8">
          <Link to="/" className="inline-flex items-center gap-2.5">
            <div className="w-9 h-9 bg-primary-600 rounded-lg flex items-center justify-center">
              <svg className="h-5 w-5 text-white" fill="none" viewBox="0 0 24 24" strokeWidth={2} stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" d="M2.25 3h1.386c.51 0 .955.343 1.087.835l.383 1.437M7.5 14.25a3 3 0 00-3 3h15.75m-12.75-3h11.218c1.121-2.3 2.1-4.684 2.924-7.138a60.114 60.114 0 00-16.536-1.84M7.5 14.25L5.106 5.272M6 20.25a.75.75 0 11-1.5 0 .75.75 0 011.5 0zm12.75 0a.75.75 0 11-1.5 0 .75.75 0 011.5 0z" />
              </svg>
            </div>
            <span className="text-xl font-bold text-gray-900 tracking-tight">{APP_NAME}</span>
          </Link>
        </div>

        <div className="card">
          {isSuccess ? (
            /* ============ SUCCESS STATE ============ */
            <div className="text-center py-4">
              <div className="mx-auto flex items-center justify-center h-14 w-14 rounded-full bg-green-100 mb-5">
                <CheckCircleIcon className="h-8 w-8 text-green-600" />
              </div>
              <h1 className="text-2xl font-bold text-gray-900 mb-2">
                Password Reset Successful
              </h1>
              <p className="text-sm text-gray-500 leading-relaxed">
                Your password has been reset successfully. You can now sign in with your new password.
              </p>
              <div className="mt-8">
                <Link to="/login" className="btn-primary w-full text-center">
                  Go to Sign In
                </Link>
              </div>
            </div>
          ) : (
            /* ============ FORM STATE ============ */
            <>
              <h1 className="text-2xl font-bold text-gray-900 text-center mb-2">
                Reset Password
              </h1>
              <p className="text-sm text-gray-500 text-center mb-8">
                Enter your new password
              </p>

              {/* Token warning */}
              {!token && (
                <div className="mb-6 rounded-lg bg-amber-50 border border-amber-200 p-3">
                  <div className="flex items-start gap-2">
                    <ExclamationTriangleIcon className="h-5 w-5 text-amber-500 flex-shrink-0 mt-0.5" />
                    <p className="text-sm text-amber-800">
                      No valid reset token detected. Please access this page from the link in your email, or request a new password reset.
                    </p>
                  </div>
                </div>
              )}

              {/* General error */}
              {fieldErrors.general && (
                <div className="mb-6 rounded-lg bg-red-50 border border-red-200 p-3">
                  <p className="text-sm text-red-700">
                    {fieldErrors.general}
                  </p>
                </div>
              )}

              <form onSubmit={handleSubmit} className="space-y-5" noValidate>
                {/* New Password */}
                <div>
                  <label htmlFor="new_password" className="label-field">
                    New Password
                  </label>
                  <div className="relative">
                    <div className="pointer-events-none absolute inset-y-0 left-0 flex items-center pl-3">
                      <LockClosedIcon className="h-4 w-4 text-gray-400" />
                    </div>
                    <input
                      id="new_password"
                      type={showPassword ? 'text' : 'password'}
                      autoComplete="new-password"
                      value={form.new_password}
                      onChange={(e) =>
                        setForm((prev) => ({
                          ...prev,
                          new_password: e.target.value,
                        }))
                      }
                      className={cn(
                        'input-field pl-10 pr-10',
                        fieldErrors.new_password &&
                          'border-red-500 focus:border-red-500 focus:ring-red-500',
                      )}
                      placeholder="Minimum 8 characters"
                      disabled={!token}
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
                  {fieldErrors.new_password && (
                    <p className="mt-1 text-xs text-red-600">
                      {fieldErrors.new_password}
                    </p>
                  )}
                </div>

                {/* Confirm Password */}
                <div>
                  <label htmlFor="confirm_password" className="label-field">
                    Confirm New Password
                  </label>
                  <div className="relative">
                    <div className="pointer-events-none absolute inset-y-0 left-0 flex items-center pl-3">
                      <LockClosedIcon className="h-4 w-4 text-gray-400" />
                    </div>
                    <input
                      id="confirm_password"
                      type={showPassword ? 'text' : 'password'}
                      autoComplete="new-password"
                      value={form.confirm_password}
                      onChange={(e) =>
                        setForm((prev) => ({
                          ...prev,
                          confirm_password: e.target.value,
                        }))
                      }
                      className={cn(
                        'input-field pl-10',
                        fieldErrors.confirm_password &&
                          'border-red-500 focus:border-red-500 focus:ring-red-500',
                      )}
                      placeholder="Re-enter new password"
                      disabled={!token}
                    />
                  </div>
                  {fieldErrors.confirm_password && (
                    <p className="mt-1 text-xs text-red-600">
                      {fieldErrors.confirm_password}
                    </p>
                  )}
                </div>

                {/* Submit */}
                <button
                  type="submit"
                  disabled={resetMutation.isPending || !token}
                  className="btn-primary w-full py-3"
                >
                  {resetMutation.isPending ? (
                    <span className="inline-flex items-center gap-2">
                      <svg
                        className="animate-spin h-4 w-4"
                        xmlns="http://www.w3.org/2000/svg"
                        fill="none"
                        viewBox="0 0 24 24"
                      >
                        <circle
                          className="opacity-25"
                          cx="12"
                          cy="12"
                          r="10"
                          stroke="currentColor"
                          strokeWidth="4"
                        />
                        <path
                          className="opacity-75"
                          fill="currentColor"
                          d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
                        />
                      </svg>
                      Resetting...
                    </span>
                  ) : (
                    'Reset Password'
                  )}
                </button>
              </form>

              <p className="mt-6 text-center text-sm text-gray-500">
                <Link
                  to="/login"
                  className="font-semibold text-primary-600 hover:text-primary-700 transition-colors"
                >
                  Back to Sign In
                </Link>
              </p>
            </>
          )}
        </div>

        {/* Bottom text */}
        <p className="mt-8 text-center text-xs text-gray-400">
          &copy; {new Date().getFullYear()} {APP_NAME}. All rights reserved.
        </p>
      </div>
    </div>
  );
}
