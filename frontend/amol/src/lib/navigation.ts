// frontend/amol/src/lib/navigation.ts
import type { User } from "firebase/auth";

export const LANDING_PATH = "/";
export const LISTS_PATH = "/lists";
export const WALLET_PATH = "/wallet";

export function isLoggedIn(user: User | null | undefined): boolean {
  return !!user;
}