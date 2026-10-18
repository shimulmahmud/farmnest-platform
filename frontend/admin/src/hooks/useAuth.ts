import { useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { useAuthStore } from '../stores/authStore';
import type { User } from '../types';

export function useAuth() {
  const user = useAuthStore((s) => s.user);
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated);
  const isAdmin = useAuthStore((s) => s.isAdmin);
  const clearAuth = useAuthStore((s) => s.clearAuth);
  const updateUser = useAuthStore((s) => s.updateUser);

  return {
    user: user as User | null,
    isAuthenticated,
    isAdmin,
    clearAuth,
    updateUser,
  };
}

export function useRequireAuth() {
  const { isAuthenticated } = useAuth();
  const isInitialized = useAuthStore((s) => s.isInitialized);
  const navigate = useNavigate();

  useEffect(() => {
    // Wait until auth state has been restored from localStorage before redirecting.
    // Without this guard, a direct navigation to a protected route would redirect
    // to /login on the very first render (before initialize() restores the session).
    if (!isInitialized) return;
    if (!isAuthenticated) {
      navigate('/login', { replace: true });
    }
  }, [isAuthenticated, isInitialized, navigate]);

  return { isAuthenticated };
}

export function useRequireAdmin() {
  const { isAuthenticated, isAdmin } = useAuth();
  const isInitialized = useAuthStore((s) => s.isInitialized);
  const navigate = useNavigate();

  useEffect(() => {
    // Wait until auth state has been restored from localStorage before redirecting.
    // Without this guard, a direct navigation to /admin would redirect to /
    // on the very first render (before initialize() restores the admin session).
    if (!isInitialized) return;
    if (!isAuthenticated || !isAdmin) {
      navigate('/login', { replace: true });
    }
  }, [isAuthenticated, isAdmin, isInitialized, navigate]);

  return { isAuthenticated, isAdmin };
}
