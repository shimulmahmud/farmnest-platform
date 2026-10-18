import { createContext, useCallback, useContext, useEffect, useMemo, useState, type ReactNode } from 'react';
import * as authApi from '@/api/auth';
import { tokens } from '@/api/http';

interface AuthContextValue {
  user: authApi.AuthUser | null;
  loading: boolean;
  /** Returns an error message, or null on success. */
  signIn: (email: string, password: string) => Promise<{ error: string | null; needsVerification?: boolean }>;
  /** Creates the account; the customer must then verify their email. */
  signUp: (input: { email: string; password: string; firstName: string; lastName: string }) => Promise<{ error: string | null }>;
  verifyEmail: (token: string) => Promise<{ error: string | null }>;
  resendVerification: (email: string) => Promise<{ error: string | null }>;
  signOut: () => Promise<void>;
}

const AuthContext = createContext<AuthContextValue | undefined>(undefined);

const messageOf = (e: unknown) => (e instanceof Error ? e.message : 'Something went wrong');

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<authApi.AuthUser | null>(null);
  const [loading, setLoading] = useState(Boolean(tokens.access));

  // Restore the session on page load.
  useEffect(() => {
    if (!tokens.access) return;
    authApi
      .fetchMe()
      .then(setUser)
      .catch(() => tokens.clear())
      .finally(() => setLoading(false));
  }, []);

  // The HTTP layer announces when a refresh failed.
  useEffect(() => {
    const onSignedOut = () => setUser(null);
    window.addEventListener('farmnest:signed-out', onSignedOut);
    return () => window.removeEventListener('farmnest:signed-out', onSignedOut);
  }, []);

  const signIn = useCallback<AuthContextValue['signIn']>(async (email, password) => {
    try {
      setUser(await authApi.login(email, password));
      return { error: null };
    } catch (e) {
      const code = (e as { code?: string }).code;
      return { error: messageOf(e), needsVerification: code === 'EMAIL_NOT_VERIFIED' };
    }
  }, []);

  const signUp = useCallback<AuthContextValue['signUp']>(async ({ email, password, firstName, lastName }) => {
    try {
      await authApi.register({ email, password, first_name: firstName, last_name: lastName });
      return { error: null };
    } catch (e) {
      return { error: messageOf(e) };
    }
  }, []);

  const verifyEmail = useCallback<AuthContextValue['verifyEmail']>(async (token) => {
    try {
      await authApi.verifyEmail(token.trim());
      return { error: null };
    } catch (e) {
      return { error: messageOf(e) };
    }
  }, []);

  const resendVerification = useCallback<AuthContextValue['resendVerification']>(async (email) => {
    try {
      await authApi.resendVerification(email);
      return { error: null };
    } catch (e) {
      return { error: messageOf(e) };
    }
  }, []);

  const signOut = useCallback(async () => {
    try {
      await authApi.logout();
    } catch {
      /* tokens are cleared regardless */
    }
    setUser(null);
  }, []);

  const value = useMemo(
    () => ({ user, loading, signIn, signUp, verifyEmail, resendVerification, signOut }),
    [user, loading, signIn, signUp, verifyEmail, resendVerification, signOut],
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error('useAuth must be used inside AuthProvider');
  return ctx;
}
