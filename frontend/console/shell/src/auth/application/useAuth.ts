// frontend/console/shell/src/auth/application/useAuth.ts
import { useAuthContext } from "./AuthContext";

export function useAuth() {
  return useAuthContext();
}
