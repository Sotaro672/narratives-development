// frontend/admin/shell/src/auth/application/adminAuth.ts

const ADMIN_PASSWORD = "AMOL-Admin-2026#Start!";
const ADMIN_SESSION_KEY = "amol-admin-authenticated";

function isBrowser(): boolean {
  return typeof window !== "undefined";
}

export function isAdminAuthenticated(): boolean {
  if (!isBrowser()) {
    return false;
  }

  return window.sessionStorage.getItem(ADMIN_SESSION_KEY) === "true";
}

export async function signInAdmin(password: string): Promise<void> {
  if (password !== ADMIN_PASSWORD) {
    throw new Error("invalid_admin_password");
  }

  if (!isBrowser()) {
    throw new Error("admin_auth_storage_unavailable");
  }

  window.sessionStorage.setItem(ADMIN_SESSION_KEY, "true");
}

export async function signOutAdmin(): Promise<void> {
  if (!isBrowser()) {
    return;
  }

  window.sessionStorage.removeItem(ADMIN_SESSION_KEY);
}

export function observeAdminAuth(
  callback: (authenticated: boolean) => void,
): () => void {
  callback(isAdminAuthenticated());

  return () => {};
}