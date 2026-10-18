import { get, post, tokens } from './http';

export interface AuthUser {
  id: string;
  email: string;
  first_name: string;
  last_name: string;
  role: string;
  status: string;
  email_verified: boolean;
}

interface AuthResult {
  user: AuthUser;
  tokens: { access_token: string; refresh_token: string; expires_at: string };
}

export async function register(input: { email: string; password: string; first_name: string; last_name: string }) {
  return post<AuthResult>('/auth/register', input, false);
}

export async function login(email: string, password: string): Promise<AuthUser> {
  const res = await post<AuthResult>('/auth/login', { email, password }, false);
  tokens.set(res.tokens.access_token, res.tokens.refresh_token);
  return res.user;
}

export async function verifyEmail(token: string) {
  return post<{ message: string }>('/auth/verify-email', { token }, false);
}

export async function resendVerification(email: string) {
  return post<{ message: string }>('/auth/resend-verification', { email }, false);
}

export async function forgotPassword(email: string) {
  // Always succeeds, so nobody can find out which emails have accounts.
  return post<{ message: string }>('/auth/forgot-password', { email }, false);
}

export async function resetPassword(token: string, newPassword: string) {
  return post<{ message: string }>('/auth/reset-password', { token, new_password: newPassword }, false);
}

export async function logout() {
  const refresh = tokens.refresh;
  try {
    if (refresh) await post('/auth/logout', { refresh_token: refresh });
  } finally {
    tokens.clear();
  }
}

export async function fetchMe(): Promise<AuthUser> {
  return get<AuthUser>('/me', undefined, true);
}
