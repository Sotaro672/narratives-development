// frontend/console/sales/src/presentation/hook/useSalesManagement.tsx
import { useCallback, useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";

import {
  listSalesByCompanyId,
  type SalesRow,
} from "../../infrastructure/repository_http";
import { useAuth } from "../../../shell/src/auth/presentation/hook/useCurrentMember";

type SalesManagementRow = SalesRow & {
  issueCount: number;
};

type SortKey = "tokenName" | "brandName" | "issueCount";
type SortDir = "asc" | "desc";

function compareStrings(a: string, b: string): number {
  return a.localeCompare(b, "ja");
}

function compareNumbers(a: number, b: number): number {
  return a - b;
}

function sortRows(
  rows: SalesManagementRow[],
  sortKey: SortKey,
  sortDir: SortDir,
): SalesManagementRow[] {
  const next = [...rows];

  next.sort((a, b) => {
    let result = 0;

    switch (sortKey) {
      case "tokenName":
        result = compareStrings(a.tokenName ?? "", b.tokenName ?? "");
        break;
      case "brandName":
        result = compareStrings(a.brandName ?? "", b.brandName ?? "");
        break;
      case "issueCount":
        result = compareNumbers(a.issueCount ?? 0, b.issueCount ?? 0);
        break;
      default:
        result = 0;
        break;
    }

    return sortDir === "asc" ? result : -result;
  });

  return next;
}

export function useSalesManagement() {
  const navigate = useNavigate();
  const { user, loading, currentMember, loadingMember } = useAuth();

  const [sourceRows, setSourceRows] = useState<SalesRow[]>([]);
  const [sortKey, setSortKey] = useState<SortKey>("tokenName");
  const [sortDir, setSortDir] = useState<SortDir>("asc");
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
      const result = await listSalesByCompanyId(companyId);
      setSourceRows(Array.isArray(result.rows) ? result.rows : []);
    } catch {
      setSourceRows([]);
    }
  }, [companyId, isAuthLoading]);

  useEffect(() => {
    void load();
  }, [load]);

  const rows = useMemo(() => {
    const enrichedRows: SalesManagementRow[] = sourceRows.map((row) => ({
      ...row,
      issueCount: Array.isArray(row.mintAddresses) ? row.mintAddresses.length : 0,
    }));

    return sortRows(enrichedRows, sortKey, sortDir);
  }, [sourceRows, sortDir, sortKey]);

  const handleChangeSort = useCallback((nextKey: string) => {
    const normalizedKey: SortKey =
      nextKey === "tokenName"
        ? "tokenName"
        : nextKey === "brandName"
          ? "brandName"
          : "issueCount";

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

  const handleRowClick = useCallback(
    (tokenBlueprintId: string) => {
      const row = sourceRows.find((x) => x.tokenBlueprintId === tokenBlueprintId);
      const id = String(tokenBlueprintId ?? "").trim();

      if (!id) return;

      navigate(`./${encodeURIComponent(id)}`, {
        state: {
          tokenName: row?.tokenName ?? "",
          brandId: row?.brandId ?? "",
          brandName: row?.brandName ?? "",
          mintAddresses: row?.mintAddresses ?? [],
          modelIds: row?.modelIds ?? [],
          productBlueprints: row?.productBlueprints ?? [],
          owners: row?.owners ?? [],
        },
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
    handleRowClick,
    isResetting,
  };
}

export default useSalesManagement;