// frontend/console/sales/src/presentation/hook/useSalesManagement.tsx
import { useCallback, useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";

import {
  buildSalesManagementNavigateState,
  enrichSalesManagementRows,
  fetchSalesManagementRows,
  normalizeSalesManagementSortKey,
  sortSalesManagementRows,
  type SalesManagementSortDir,
  type SalesManagementSortKey,
} from "../../application/sales_management_service";
import type { SalesRow } from "../../infrastructure/sales_repository_http";
import { useAuth } from "../../../shell/src/auth/presentation/hook/useCurrentMember";

export function useSalesManagement() {
  const navigate = useNavigate();
  const { user, loading, currentMember, loadingMember } = useAuth();

  const [sourceRows, setSourceRows] = useState<SalesRow[]>([]);
  const [sortKey, setSortKey] =
    useState<SalesManagementSortKey>("tokenName");
  const [sortDir, setSortDir] = useState<SalesManagementSortDir>("asc");
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
      const rows = await fetchSalesManagementRows(companyId);
      setSourceRows(rows);
    } catch {
      setSourceRows([]);
    }
  }, [companyId, isAuthLoading]);

  useEffect(() => {
    void load();
  }, [load]);

  const rows = useMemo(() => {
    const enrichedRows = enrichSalesManagementRows(sourceRows);
    return sortSalesManagementRows(enrichedRows, sortKey, sortDir);
  }, [sourceRows, sortDir, sortKey]);

  const handleChangeSort = useCallback((nextKey: string) => {
    const normalizedKey = normalizeSalesManagementSortKey(nextKey);

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
        state: buildSalesManagementNavigateState(row),
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

export default useSalesManagement;