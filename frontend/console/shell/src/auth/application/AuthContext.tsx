// frontend/console/shell/src/auth/application/AuthContext.tsx

import * as React from "react";
import {
  onAuthStateChanged,
  type User,
} from "firebase/auth";

import { auth } from "../infrastructure/config/firebaseClient";
import { getCompanyNameById } from "./companyService";
import { fetchCurrentMember } from "./memberService";

import type { Auth } from "../../shared/types/auth";
import type { MemberDTO } from "../../shared/types/member";

type AuthContextValue = {
  // Firebase Authentication
  user: Auth | null;
  loading: boolean;

  // Backend GET /members/me
  currentMember: MemberDTO | null;
  loadingMember: boolean;
  memberError: string | null;

  // Backend GET /companies/{companyId}
  companyName: string | null;
};

const initialAuthContextValue: AuthContextValue = {
  user: null,
  loading: true,
  currentMember: null,
  loadingMember: false,
  memberError: null,
  companyName: null,
};

const AuthContext = React.createContext<AuthContextValue | undefined>(undefined);

function toAuthUser(firebaseUser: User): Auth {
  return {
    uid: firebaseUser.uid,
    email: firebaseUser.email ?? null,
    displayName: firebaseUser.displayName ?? null,
    companyId: null,
    permissions: [],
    assignedBrands: [],
  };
}

function sleep(milliseconds: number): Promise<void> {
  return new Promise((resolve) => {
    setTimeout(resolve, milliseconds);
  });
}

/**
 * 新規登録直後はBackend側のmember作成が完了していない場合があるため、
 * 短時間だけGET /members/meを再試行する。
 */
async function fetchCurrentMemberWithRetry(
  retries: number,
  retryDelayMs: number,
): Promise<MemberDTO | null> {
  for (let attempt = 0; attempt <= retries; attempt += 1) {
    try {
      const member = await fetchCurrentMember();

      if (member?.id && member.companyId) {
        return member;
      }

      if (attempt < retries) {
        await sleep(retryDelayMs * (attempt + 1));
        continue;
      }

      return member;
    } catch (error: unknown) {
      if (attempt < retries) {
        await sleep(retryDelayMs * (attempt + 1));
        continue;
      }

      throw error;
    }
  }

  return null;
}

export const AuthProvider: React.FC<{
  children: React.ReactNode;
}> = ({ children }) => {
  const [state, setState] = React.useState<AuthContextValue>(
    initialAuthContextValue,
  );

  React.useEffect(() => {
    let active = true;
    let requestSequence = 0;

    const unsubscribe = onAuthStateChanged(auth, (firebaseUser) => {
      const currentRequest = ++requestSequence;

      async function resolveAuthState() {
        if (!active) {
          return;
        }

        if (!firebaseUser) {
          setState({
            user: null,
            loading: false,
            currentMember: null,
            loadingMember: false,
            memberError: null,
            companyName: null,
          });
          return;
        }

        const authUser = toAuthUser(firebaseUser);

        setState({
          user: authUser,
          loading: false,
          currentMember: null,
          loadingMember: true,
          memberError: null,
          companyName: null,
        });

        let currentMember: MemberDTO | null = null;
        let memberError: string | null = null;

        try {
          const member = await fetchCurrentMemberWithRetry(8, 250);

          if (member?.id && member.companyId) {
            currentMember = member;
          } else {
            memberError =
              "ログインユーザーの会社情報を確認できませんでした。";
          }
        } catch (error: unknown) {
          memberError =
            error instanceof Error
              ? error.message
              : "failed to fetch member";
        }

        if (!active || currentRequest !== requestSequence) {
          return;
        }

        const resolvedUser: Auth = currentMember
          ? {
              ...authUser,
              companyId: currentMember.companyId,
              permissions: currentMember.permissions,
              assignedBrands: currentMember.assignedBrands ?? [],
            }
          : authUser;

        setState((current) => ({
          ...current,
          user: resolvedUser,
          currentMember,
          loadingMember: false,
          memberError,
        }));

        if (!currentMember?.companyId) {
          return;
        }

        try {
          const companyName = await getCompanyNameById(
            currentMember.companyId,
          );

          if (!active || currentRequest !== requestSequence) {
            return;
          }

          setState((current) => ({
            ...current,
            companyName,
          }));
        } catch {
          if (!active || currentRequest !== requestSequence) {
            return;
          }

          setState((current) => ({
            ...current,
            companyName: null,
          }));
        }
      }

      void resolveAuthState();
    });

    return () => {
      active = false;
      requestSequence += 1;
      unsubscribe();
    };
  }, []);

  const contextValue = React.useMemo(() => state, [state]);

  return (
    <AuthContext.Provider value={contextValue}>
      {children}
    </AuthContext.Provider>
  );
};

export function useAuthContext(): AuthContextValue {
  const context = React.useContext(AuthContext);

  if (!context) {
    throw new Error(
      "useAuthContext must be used within AuthProvider",
    );
  }

  return context;
}