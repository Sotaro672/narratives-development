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
    isResetting,
    setIsResetting,
  ] = useState(false);

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

          const result =
            await listTransactions({
              page: 1,
              perPage: 100,
            });

          setTransactions(
            result.items,
          );

          setTotalCount(
            result.totalCount,
          );
        } catch (
          caughtError: unknown
        ) {
          setTransactions([]);

          setTotalCount(0);

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
    transactions,
    loading,
    isResetting,
    error,
    totalCount,
    reload,
  };
}