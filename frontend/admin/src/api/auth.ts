import { api } from './client';
import type {
  AuthResponse,
  LoginRequest,
  RegisterRequest,
  ForgotPasswordRequest,
  ResetPasswordRequest,
  MessageResponse,
  TokenResponse,
} from '../types';

export async function login(data: LoginRequest): Promise<AuthResponse> {
  const res = await api.post<AuthResponse>('/auth/login', data);
  return res.data;
}

export async function register(data: RegisterRequest): Promise<AuthResponse> {
  const res = await api.post<AuthResponse>('/auth/register', data);
  return res.data;
}

export async function refresh(refreshToken: string): Promise<TokenResponse> {
  const res = await api.post<TokenResponse>('/auth/refresh', {
    refresh_token: refreshToken,
  });
  return res.data;
}

export async function logout(): Promise<MessageResponse> {
  const res = await api.post<MessageResponse>('/auth/logout');
  return res.data;
}

export async function verifyEmail(token: string): Promise<MessageResponse> {
  const res = await api.post<MessageResponse>('/auth/verify-email', {
    token,
  });
  return res.data;
}

export async function forgotPassword(
  data: ForgotPasswordRequest,
): Promise<MessageResponse> {
  const res = await api.post<MessageResponse>('/auth/forgot-password', data);
  return res.data;
}

export async function resetPassword(
  data: ResetPasswordRequest,
): Promise<MessageResponse> {
  const res = await api.post<MessageResponse>('/auth/reset-password', data);
  return res.data;
}
