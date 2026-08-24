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
    error,
    setError,
  ] = useState<Error | null>(
    null,
  );

  const [
    reloadKey,
    setReloadKey,
  ] = useState(0);

  useEffect(() => {
    let cancelled = false;

    const load = async () => {
      try {
        setLoading(true);
        setError(null);

        const rows =
          await listAccounts();

        if (cancelled) {
          return;
        }

        setAccounts(
          rows,
        );
      } catch (
        caughtError: unknown
      ) {
        if (cancelled) {
          return;
        }

        setAccounts([]);

        setError(
          normalizeError(
            caughtError,
          ),
        );
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    };

    void load();

    return () => {
      cancelled = true;
    };
  }, [
    reloadKey,
  ]);

  const reload =
    useCallback(() => {
      setReloadKey(
        (current) =>
          current + 1,
      );
    }, []);

  return {
    accounts,
    loading,
    error,
    reload,
  };
}