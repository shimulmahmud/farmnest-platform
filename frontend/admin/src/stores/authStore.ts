import { create } from 'zustand';
import type { User, TokenResponse } from '../types';

const ACCESS_TOKEN_KEY = 'ecommerce_access_token';
const REFRESH_TOKEN_KEY = 'ecommerce_refresh_token';
const USER_KEY = 'ecommerce_user';

interface AuthState {
  user: User | null;
  accessToken: string | null;
  refreshToken: string | null;
  isAuthenticated: boolean;
  isAdmin: boolean;
  isInitialized: boolean;
  setAuth: (user: User, tokens: TokenResponse) => void;
  clearAuth: () => void;
  updateUser: (user: User) => void;
  initialize: () => void;
}

function isAdminRole(role: string): boolean {
  const r = role.toUpperCase();
  return r === 'SUPER_ADMIN' || r === 'ADMIN' || r === 'MANAGER';
}

export const useAuthStore = create<AuthState>((set) => ({
  user: null,
  accessToken: null,
  refreshToken: null,
  isAuthenticated: false,
  isAdmin: false,
  isInitialized: false,

  setAuth: (user: User, tokens: TokenResponse) => {
    localStorage.setItem(ACCESS_TOKEN_KEY, tokens.access_token);
    localStorage.setItem(REFRESH_TOKEN_KEY, tokens.refresh_token);
    localStorage.setItem(USER_KEY, JSON.stringify({ ...user, expires_at: tokens.expires_at }));
    set({
      user,
      accessToken: tokens.access_token,
      refreshToken: tokens.refresh_token,
      isAuthenticated: true,
      isAdmin: isAdminRole(user.role),
    });
  },

  clearAuth: () => {
    localStorage.removeItem(ACCESS_TOKEN_KEY);
    localStorage.removeItem(REFRESH_TOKEN_KEY);
    localStorage.removeItem(USER_KEY);
    set({
      user: null,
      accessToken: null,
      refreshToken: null,
      isAuthenticated: false,
      isAdmin: false,
    });
  },

  updateUser: (user: User) => {
    const currentExpiresAt = localStorage.getItem(USER_KEY);
    let expiresAt: string | undefined;
    if (currentExpiresAt) {
      try {
        const parsed = JSON.parse(currentExpiresAt);
        expiresAt = parsed.expires_at;
      } catch {
        // ignore parse error
      }
    }
    localStorage.setItem(USER_KEY, JSON.stringify({ ...user, expires_at: expiresAt }));
    set({
      user,
      isAuthenticated: true,
      isAdmin: isAdminRole(user.role),
    });
  },

  initialize: () => {
    const accessToken = localStorage.getItem(ACCESS_TOKEN_KEY);
    const refreshToken = localStorage.getItem(REFRESH_TOKEN_KEY);
    const userJson = localStorage.getItem(USER_KEY);

    if (!accessToken || !userJson) {
      localStorage.removeItem(ACCESS_TOKEN_KEY);
      localStorage.removeItem(REFRESH_TOKEN_KEY);
      localStorage.removeItem(USER_KEY);
      set({ user: null, accessToken: null, refreshToken: null, isAuthenticated: false, isAdmin: false, isInitialized: true });
      return;
    }

    try {
      const parsed = JSON.parse(userJson);
      const expiresAt = parsed.expires_at;

      if (expiresAt && Date.now() > new Date(expiresAt).getTime()) {
        localStorage.removeItem(ACCESS_TOKEN_KEY);
        localStorage.removeItem(REFRESH_TOKEN_KEY);
        localStorage.removeItem(USER_KEY);
        set({ user: null, accessToken: null, refreshToken: null, isAuthenticated: false, isAdmin: false, isInitialized: true });
        return;
      }

      const { expires_at: _, ...user } = parsed;
      set({
        user: user as User,
        accessToken,
        refreshToken,
        isAuthenticated: true,
        isAdmin: isAdminRole((user as User).role),
        isInitialized: true,
      });
    } catch {
      localStorage.removeItem(ACCESS_TOKEN_KEY);
      localStorage.removeItem(REFRESH_TOKEN_KEY);
      localStorage.removeItem(USER_KEY);
      set({ user: null, accessToken: null, refreshToken: null, isAuthenticated: false, isAdmin: false, isInitialized: true });
    }
  },
}));

export function getAccessToken(): string | null {
  return useAuthStore.getState().accessToken;
}
