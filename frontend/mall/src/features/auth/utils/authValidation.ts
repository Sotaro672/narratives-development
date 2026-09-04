// frontend/src/features/auth/utils/authValidation.ts

export function normalizeEmail(email: string): string {
  return email.trim();
}

export function isEmailValid(email: string): boolean {
  return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email.trim());
}

export function isPasswordValid(password: string): boolean {
  return password.length >= 6;
}

export function isPasswordMatch(
  password: string,
  passwordConfirmation: string
): boolean {
  return password === passwordConfirmation;
}