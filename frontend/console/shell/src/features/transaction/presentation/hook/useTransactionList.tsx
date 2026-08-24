// frontend/console/shell/src/features/transaction/presentation/hook/useTransactionList.tsx

import {
  useCallback,
  useEffect,
  useState,
} from "react";

import {
  listTransactions,
  type TransactionRow,
} from "../../application/transactionService";

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
    "取引履歴の取得に失敗しました。",
  );
}

export function useTransactionList() {
  const [
    transactions,
    setTransactions,
  ] = useState<TransactionRow[]>([]);

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
    totalCount,
    setTotalCount,
  ] = useState(0);

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

        const result =
          await listTransactions({
            page: 1,
            perPage: 100,
          });

        if (cancelled) {
          return;
        }

        setTransactions(
          result.items,
        );

        setTotalCount(
          result.totalCount,
        );
      } catch (
        caughtError: unknown
      ) {
        if (cancelled) {
          return;
        }

        setTransactions([]);

        setTotalCount(0);

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
    transactions,
    loading,
    error,
    totalCount,
    reload,
  };
}