// frontend/console/shell/src/auth/application/AuthContext.tsx

import * as React from "react";

import {
  onAuthStateChanged,
  type User,
} from "firebase/auth";

import {
  doc,
  getDoc,
} from "firebase/firestore";

import {
  auth,
  db,
} from "../infrastructure/config/firebaseClient";

import {
  getCompanyNameById,
} from "./companyService";

import {
  fetchCurrentMember,
} from "./memberService";

import type {
  Auth,
} from "../../shared/types/auth";

import type {
  MemberDTO,
} from "../../shared/types/member";

type AuthContextValue = {
  // Firebase Authenticationとusersドキュメント
  user: Auth | null;
  loading: boolean;

  // Backendのmembers/me
  currentMember: MemberDTO | null;
  loadingMember: boolean;
  memberError: string | null;

  // Backendのcompanies/{companyId}
  companyName: string | null;
};

const initialAuthContextValue:
  AuthContextValue = {
    user: null,
    loading: true,

    currentMember: null,
    loadingMember: false,
    memberError: null,

    companyName: null,
  };

const AuthContext =
  React.createContext<
    AuthContextValue | undefined
  >(
    undefined,
  );

function mapFirebaseUserBase(
  user: User,
): Omit<
  Auth,
  | "companyId"
  | "permissions"
  | "assignedBrands"
> {
  return {
    uid: user.uid,
    email: user.email ?? null,
    displayName: user.displayName ?? null,
  };
}

async function loadAuthUser(
  firebaseUser: User,
): Promise<Auth> {
  const base =
    mapFirebaseUserBase(
      firebaseUser,
    );

  try {
    const userRef =
      doc(
        db,
        "users",
        firebaseUser.uid,
      );

    const snapshot =
      await getDoc(
        userRef,
      );

    const data =
      snapshot.exists()
        ? (
            snapshot.data() as Record<
              string,
              unknown
            >
          )
        : {};

    const companyId =
      typeof data.companyId ===
      "string"
        ? data.companyId.trim() ||
          null
        : null;

    const permissions =
      Array.isArray(
        data.permissions,
      )
        ? data.permissions
            .filter(
              (
                value,
              ): value is string =>
                typeof value ===
                "string",
            )
            .map(
              (value) =>
                value.trim(),
            )
            .filter(
              (value) =>
                value.length > 0,
            )
        : [];

    const assignedBrands =
      Array.isArray(
        data.assignedBrands,
      )
        ? data.assignedBrands
            .filter(
              (
                value,
              ): value is string =>
                typeof value ===
                "string",
            )
            .map(
              (value) =>
                value.trim(),
            )
            .filter(
              (value) =>
                value.length > 0,
            )
        : [];

    return {
      ...base,
      companyId,
      permissions,
      assignedBrands,
    };
  } catch (
    error: unknown
  ) {
    console.error(
      "[AuthContext] failed to load user profile:",
      error,
    );

    return {
      ...base,
      companyId: null,
      permissions: [],
      assignedBrands: [],
    };
  }
}

function sleep(
  milliseconds: number,
): Promise<void> {
  return new Promise(
    (resolve) => {
      setTimeout(
        resolve,
        milliseconds,
      );
    },
  );
}

/**
 * 新規登録直後はBackend側のmember作成が
 * 完了していない場合があるため、短時間だけ再試行する。
 *
 * MemberDTOへの変換はmemberServiceが担当する。
 */
async function fetchCurrentMemberWithRetry(
  retries: number,
  retryDelayMs: number,
): Promise<MemberDTO | null> {
  for (
    let attempt = 0;
    attempt <= retries;
    attempt += 1
  ) {
    try {
      const member =
        await fetchCurrentMember();

      if (
        member?.id &&
        member.companyId
      ) {
        return member;
      }

      if (
        attempt < retries
      ) {
        await sleep(
          retryDelayMs *
            (attempt + 1),
        );

        continue;
      }

      return member;
    } catch (
      error: unknown
    ) {
      if (
        attempt < retries
      ) {
        await sleep(
          retryDelayMs *
            (attempt + 1),
        );

        continue;
      }

      throw error;
    }
  }

  return null;
}

export const AuthProvider:
  React.FC<{
    children:
      React.ReactNode;
  }> = ({
    children,
  }) => {
    const [
      state,
      setState,
    ] =
      React.useState<AuthContextValue>(
        initialAuthContextValue,
      );

    React.useEffect(
      () => {
        let active =
          true;

        let requestSequence =
          0;

        const unsubscribe =
          onAuthStateChanged(
            auth,
            (
              firebaseUser,
            ) => {
              const currentRequest =
                ++requestSequence;

              async function resolveAuthState() {
                if (!active) {
                  return;
                }

                if (
                  !firebaseUser
                ) {
                  setState({
                    user: null,
                    loading: false,

                    currentMember:
                      null,
                    loadingMember:
                      false,
                    memberError:
                      null,

                    companyName:
                      null,
                  });

                  return;
                }

                setState({
                  user: null,
                  loading: true,

                  currentMember:
                    null,
                  loadingMember:
                    true,
                  memberError:
                    null,

                  companyName:
                    null,
                });

                const authUser =
                  await loadAuthUser(
                    firebaseUser,
                  );

                if (
                  !active ||
                  currentRequest !==
                    requestSequence
                ) {
                  return;
                }

                // Firebaseの認証情報はMember取得より先に公開する
                setState(
                  (
                    current,
                  ) => ({
                    ...current,
                    user:
                      authUser,
                    loading:
                      false,
                  }),
                );

                let currentMember:
                  | MemberDTO
                  | null =
                  null;

                let memberError:
                  | string
                  | null =
                  null;

                try {
                  const member =
                    await fetchCurrentMemberWithRetry(
                      8,
                      250,
                    );

                  if (
                    member?.id &&
                    member.companyId
                  ) {
                    currentMember =
                      member;
                  } else {
                    memberError =
                      "ログインユーザーの会社情報を確認できませんでした。";
                  }
                } catch (
                  error: unknown
                ) {
                  memberError =
                    error instanceof
                    Error
                      ? error.message
                      : "failed to fetch member";
                }

                if (
                  !active ||
                  currentRequest !==
                    requestSequence
                ) {
                  return;
                }

                setState(
                  (
                    current,
                  ) => ({
                    ...current,
                    currentMember,
                    loadingMember:
                      false,
                    memberError,
                  }),
                );

                const effectiveCompanyId =
                  currentMember
                    ?.companyId
                    .trim() ||
                  authUser
                    .companyId
                    ?.trim() ||
                  "";

                if (
                  !effectiveCompanyId
                ) {
                  setState(
                    (
                      current,
                    ) => ({
                      ...current,
                      companyName:
                        null,
                    }),
                  );

                  return;
                }

                try {
                  const companyName =
                    await getCompanyNameById(
                      effectiveCompanyId,
                    );

                  if (
                    !active ||
                    currentRequest !==
                      requestSequence
                  ) {
                    return;
                  }

                  setState(
                    (
                      current,
                    ) => ({
                      ...current,
                      companyName,
                    }),
                  );
                } catch {
                  if (
                    !active ||
                    currentRequest !==
                      requestSequence
                  ) {
                    return;
                  }

                  setState(
                    (
                      current,
                    ) => ({
                      ...current,
                      companyName:
                        null,
                    }),
                  );
                }
              }

              void resolveAuthState();
            },
          );

        return () => {
          active =
            false;

          requestSequence +=
            1;

          unsubscribe();
        };
      },
      [],
    );

    const contextValue =
      React.useMemo(
        () => state,
        [state],
      );

    return (
      <AuthContext.Provider
        value={
          contextValue
        }
      >
        {children}
      </AuthContext.Provider>
    );
  };

export function useAuthContext():
  AuthContextValue {
  const context =
    React.useContext(
      AuthContext,
    );

  if (!context) {
    throw new Error(
      "useAuthContext must be used within AuthProvider",
    );
  }

  return context;
}