//frontend\admin\shell\src\auth\adminAuth.ts
const ADMIN_EMAIL ="caotailangaogang@gmail.com";
const INITIAL_ADMIN_PASSWORD ="AMOL-Admin-2026#Start!";
const ADMIN_SESSION_KEY ="amol-admin-authenticated";

export function authenticateAdmin(
  email: string,
  password: string,
): boolean {
  const normalizedEmail =
    email.trim().toLowerCase();

  return (
    normalizedEmail === ADMIN_EMAIL &&
    password === INITIAL_ADMIN_PASSWORD
  );
}

export function createAdminSession(): void {
  sessionStorage.setItem(
    ADMIN_SESSION_KEY,
    "true",
  );
}

export function clearAdminSession(): void {
  sessionStorage.removeItem(
    ADMIN_SESSION_KEY,
  );
}

export function hasAdminSession(): boolean {
  return (
    sessionStorage.getItem(
      ADMIN_SESSION_KEY,
    ) === "true"
  );
}