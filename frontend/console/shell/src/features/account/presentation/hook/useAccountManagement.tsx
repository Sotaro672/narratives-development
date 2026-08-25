// frontend/console/shell/src/features/account/presentation/hook/useAccountManagement.tsx

import {
  useCallback,
  useEffect,
  useState,
} from "react";

import type {
  AccountRow,
} from "../../application/accountService";
import {
  listAccounts,
} from "../../application/accountService";

function normalizeError(
  error: unknown,
): Error {
  if (error instanceof Error) {
    return error;
  }

  if (
    typeof error === "string" &&
    error
  ) {
    return new Error(
      error,
    );
  }

  return new Error(
    "口座情報の取得に失敗しました。",
  );
}

export function useAccountManagement() {
  const [
    accounts,
    setAccounts,
  ] = useState<AccountRow[]>([]);

  const [
    loading,
    setLoading,
  ] = useState(true);

  const [
    isResetting,
    setIsResetting,
  ] = useState(false);

  const [
    error,
    setError,
  ] = useState<Error | null>(
    null,
  );

  const load =
    useCallback(
      async (
        resetting: boolean,
      ) => {
        if (resetting) {
          setIsResetting(true);
        } else {
          setLoading(true);
        }

        try {
          setError(null);

          const rows =
            await listAccounts();

          setAccounts(
            rows,
          );
        } catch (
          caughtError: unknown
        ) {
          setAccounts([]);

          setError(
            normalizeError(
              caughtError,
            ),
          );
        } finally {
          if (resetting) {
            setIsResetting(false);
          } else {
            setLoading(false);
          }
        }
      },
      [],
    );

  useEffect(() => {
    void load(false);
  }, [
    load,
  ]);

  const reload =
    useCallback(() => {
      void load(true);
    }, [
      load,
    ]);

  return {
    accounts,
    loading,
    isResetting,
    error,
    reload,
  };
}