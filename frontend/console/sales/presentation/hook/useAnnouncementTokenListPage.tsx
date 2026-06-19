// frontend/console/sales/src/presentation/hook/useAnnouncementTokenListPage.tsx
import { useCallback, useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";

import {
  buildAnnouncementTokenListNavigateState,
  enrichAnnouncementTokenListRows,
  fetchAnnouncementTokenListRows,
  normalizeAnnouncementTokenListSortKey,
  sortAnnouncementTokenListRows,
  type AnnouncementTokenListSortDir,
  type AnnouncementTokenListSortKey,
} from "../../application/announcement_token_list_service";
import type { SalesRow } from "../../infrastructure/sales_repository_http";
import { useAuth } from "../../../shell/src/auth/presentation/hook/useCurrentMember";

export function useAnnouncementTokenListPage() {
  const navigate = useNavigate();
  const { user, loading, currentMember, loadingMember } = useAuth();

  const [sourceRows, setSourceRows] = useState<SalesRow[]>([]);
  const [sortKey, setSortKey] =
    useState<AnnouncementTokenListSortKey>("tokenName");
  const [sortDir, setSortDir] =
    useState<AnnouncementTokenListSortDir>("asc");
  const [isResetting, setIsResetting] = useState(false);

  const companyId = useMemo(() => {
    return String(currentMember?.companyId ?? user?.companyId ?? "").trim();
  }, [currentMember, user]);

  const isAuthLoading = loading || loadingMember;

  const load = useCallback(async () => {
    if (isAuthLoading) {
      return;
    }

    if (!companyId) {
      setSourceRows([]);
      return;
    }

    try {
      const rows = await fetchAnnouncementTokenListRows(companyId);
      setSourceRows(rows);
    } catch {
      setSourceRows([]);
    }
  }, [companyId, isAuthLoading]);

  useEffect(() => {
    void load();
  }, [load]);

  const rows = useMemo(() => {
    const enrichedRows = enrichAnnouncementTokenListRows(sourceRows);
    return sortAnnouncementTokenListRows(enrichedRows, sortKey, sortDir);
  }, [sourceRows, sortDir, sortKey]);

  const handleChangeSort = useCallback((nextKey: string) => {
    const normalizedKey = normalizeAnnouncementTokenListSortKey(nextKey);

    setSortKey((prevKey) => {
      if (prevKey === normalizedKey) {
        setSortDir((prevDir) => (prevDir === "asc" ? "desc" : "asc"));
        return prevKey;
      }

      setSortDir("asc");
      return normalizedKey;
    });
  }, []);

  const handleReset = useCallback(async () => {
    setIsResetting(true);
    try {
      setSortKey("tokenName");
      setSortDir("asc");
      await load();
    } finally {
      setIsResetting(false);
    }
  }, [load]);

  const handleCreate = useCallback(() => {
    navigate("./create");
  }, [navigate]);

  const handleRowClick = useCallback(
    (tokenBlueprintId: string) => {
      const id = String(tokenBlueprintId ?? "").trim();
      if (!id) return;

      const row = sourceRows.find((x) => x.tokenBlueprintId === id);

      navigate(`./${encodeURIComponent(id)}`, {
        state: buildAnnouncementTokenListNavigateState(row),
      });
    },
    [navigate, sourceRows],
  );

  return {
    rows,
    sortKey,
    sortDir,
    handleChangeSort,
    handleReset,
    handleCreate,
    handleRowClick,
    isResetting,
  };
}

export default useAnnouncementTokenListPage;