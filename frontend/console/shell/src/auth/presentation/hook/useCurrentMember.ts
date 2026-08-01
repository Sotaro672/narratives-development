// frontend/console/shell/src/auth/presentation/hook/useCurrentMember.ts

import {
  useAuthContext,
} from "../../application/AuthContext";

/**
 * AuthProviderが保持する認証情報、Member情報、
 * 会社名を参照する。
 *
 * このhook自身ではAPI取得やstate管理を行わない。
 */
export function useAuth() {
  return useAuthContext();
}