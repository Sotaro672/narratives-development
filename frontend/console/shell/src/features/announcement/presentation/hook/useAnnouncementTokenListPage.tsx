// frontend/console/shell/src/features/announcement/presentation/hook/useAnnouncementTokenListPage.tsx

import {
  useCallback,
  useEffect,
  useMemo,
  useState,
} from "react";

import {
  useAuthContext,
} from "../../../../auth/application/AuthContext";
import {
  enrichAnnouncementTokenListRows,
  fetchAnnouncementTokenListRows,
  normalizeAnnouncementTokenListSortKey,
  sortAnnouncementTokenListRows,
  type AnnouncementTokenListSortDir,
  type AnnouncementTokenListSortKey,
} from "../../application/announcement_token_list_service";
import type {
  SalesRow,
} from "../../infrastructure/sales_repository_http";

export function useAnnouncementTokenListPage() {
  const {
    loading,
    currentMember,
    loadingMember,
  } = useAuthContext();

  const [
    sourceRows,
    setSourceRows,
  ] = useState<SalesRow[]>([]);

  const [
    sortKey,
    setSortKey,
  ] =
    useState<AnnouncementTokenListSortKey>(
      "tokenName",
    );

  const [
    sortDir,
    setSortDir,
  ] =
    useState<AnnouncementTokenListSortDir>(
      "asc",
    );

  const [
    isResetting,
    setIsResetting,
  ] = useState(false);

  const companyId = useMemo(
    () =>
      String(
        currentMember?.companyId ?? "",
      ).trim(),
    [currentMember?.companyId],
  );

  const isAuthLoading =
    loading || loadingMember;

  const load = useCallback(
    async () => {
      if (isAuthLoading) {
        return;
      }

      if (!companyId) {
        setSourceRows([]);
        return;
      }

      try {
        const rows =
          await fetchAnnouncementTokenListRows();

        setSourceRows(rows);
      } catch {
        setSourceRows([]);
      }
    },
    [
      companyId,
      isAuthLoading,
    ],
  );

  useEffect(() => {
    void load();
  }, [load]);

  const rows = useMemo(() => {
    const enrichedRows =
      enrichAnnouncementTokenListRows(
        sourceRows,
      );

    return sortAnnouncementTokenListRows(
      enrichedRows,
      sortKey,
      sortDir,
    );
  }, [
    sourceRows,
    sortKey,
    sortDir,
  ]);

  const handleChangeSort =
    useCallback(
      (nextKey: string) => {
        const normalizedKey =
          normalizeAnnouncementTokenListSortKey(
            nextKey,
          );

        setSortKey(
          (previousKey) => {
            if (
              previousKey ===
              normalizedKey
            ) {
              setSortDir(
                (previousDirection) =>
                  previousDirection ===
                  "asc"
                    ? "desc"
                    : "asc",
              );

              return previousKey;
            }

            setSortDir("asc");

            return normalizedKey;
          },
        );
      },
      [],
    );

  const handleReset =
    useCallback(async () => {
      setIsResetting(true);

      try {
        setSortKey("tokenName");
        setSortDir("asc");

        await load();
      } finally {
        setIsResetting(false);
      }
    }, [load]);

  return {
    rows,
    sortKey,
    sortDir,
    handleChangeSort,
    handleReset,
    isResetting,
  };
}