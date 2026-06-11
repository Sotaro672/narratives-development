// frontend/console/tokenBlueprint/src/presentation/hook/useTokenBlueprintManagement.tsx

import { useCallback, useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import { useAuth } from "../../../../shell/src/auth/presentation/hook/useCurrentMember";
import type { TokenBlueprint } from "../../domain/entity/tokenBlueprint";
import {
  SortKey,
  SortDir,
  fetchTokenBlueprintsForCompany,
  buildOptionsFromTokenBlueprints,
  filterAndSortTokenBlueprints,
} from "../../application/tokenBlueprintManagementService";

export type UseTokenBlueprintManagementResult = {
  rows: TokenBlueprint[];
  brandOptions: { value: string; label: string }[];
  assigneeOptions: { value: string; label: string }[];
  mintedOptions: { value: string; label: string }[];
  brandFilter: string[];
  assigneeFilter: string[];
  mintedFilter: string[];
  sortKey: SortKey;
  sortDir: SortDir;

  isResetting: boolean;

  handleChangeBrandFilter: (vals: string[]) => void;
  handleChangeAssigneeFilter: (vals: string[]) => void;
  handleChangeMintedFilter: (vals: string[]) => void;
  handleChangeSort: (key: string | null, dir: SortDir) => void;
  handleReset: () => void;
  handleCreate: () => void;
  handleRowClick: (id: string) => void;
};

/**
 * ISO8601 → yyyy/MM/dd HH:mm 形式に整形
 * - 例: 2026/01/24 13:05
 */
function formatDateYYYYMMDDHHmm(iso: string): string {
  if (!iso) return "";
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) {
    return iso;
  }

  const y = d.getFullYear();
  const m = String(d.getMonth() + 1).padStart(2, "0");
  const day = String(d.getDate()).padStart(2, "0");

  const hh = String(d.getHours()).padStart(2, "0");
  const mm = String(d.getMinutes()).padStart(2, "0");

  return `${y}/${m}/${day} ${hh}:${mm}`;
}

export function useTokenBlueprintManagement(): UseTokenBlueprintManagementResult {
  const navigate = useNavigate();
  const { currentMember } = useAuth();

  const [rows, setRows] = useState<TokenBlueprint[]>([]);

  const [brandFilter, setBrandFilter] = useState<string[]>([]);
  const [assigneeFilter, setAssigneeFilter] = useState<string[]>([]);
  const [mintedFilter, setMintedFilter] = useState<string[]>([]);

  const [sortKey, setSortKey] = useState<SortKey>(null);
  const [sortDir, setSortDir] = useState<SortDir>(null);

  const [isResetting, setIsResetting] = useState(false);

  const reload = useCallback(async () => {
    const companyId = currentMember?.companyId;
    if (!companyId) {
      setRows([]);
      return;
    }

    setIsResetting(true);
    try {
      const result = await fetchTokenBlueprintsForCompany(companyId);
      setRows(result);
    } catch {
      setRows([]);
    } finally {
      setIsResetting(false);
    }
  }, [currentMember?.companyId]);

  useEffect(() => {
    void reload();
  }, [reload]);

  const { brandOptions, assigneeOptions } = useMemo(() => {
    const base = buildOptionsFromTokenBlueprints(rows);

    const brandNameById = new Map<string, string>();
    const assigneeNameById = new Map<string, string>();

    rows.forEach((r) => {
      const bid = r.brandId;
      if (bid) {
        const bname = r.brandName ?? "";
        if (bname && !brandNameById.has(bid)) {
          brandNameById.set(bid, bname);
        }
      }

      const aid = r.assigneeId;
      if (aid) {
        const aname = r.assigneeName ?? "";
        if (aname && !assigneeNameById.has(aid)) {
          assigneeNameById.set(aid, aname);
        }
      }
    });

    const brandOptions = base.brandOptions.map((opt) => ({
      ...opt,
      label: brandNameById.get(opt.value) || opt.label || opt.value,
    }));

    const assigneeOptions = base.assigneeOptions.map((opt) => ({
      ...opt,
      label: assigneeNameById.get(opt.value) || opt.label || opt.value,
    }));

    return { brandOptions, assigneeOptions };
  }, [rows]);

  const mintedOptions = useMemo(
    () => [
      { value: "true", label: "true" },
      { value: "false", label: "false" },
    ],
    [],
  );

  const filteredRows: TokenBlueprint[] = useMemo(() => {
    const base = filterAndSortTokenBlueprints(rows, {
      brandFilter,
      assigneeFilter,
      sortKey,
      sortDir,
    });

    if (mintedFilter.length === 0) {
      return base;
    }

    return base.filter((tb) => {
      const mintedValue = String(Boolean(tb.minted));
      return mintedFilter.includes(mintedValue);
    });
  }, [rows, brandFilter, assigneeFilter, mintedFilter, sortKey, sortDir]);

  const displayRows: TokenBlueprint[] = useMemo(() => {
    return filteredRows.map((tb) => ({
      ...tb,
      createdAt: tb.createdAt
        ? formatDateYYYYMMDDHHmm(tb.createdAt)
        : tb.createdAt,
      updatedAt: tb.updatedAt
        ? formatDateYYYYMMDDHHmm(tb.updatedAt)
        : tb.updatedAt,
    }));
  }, [filteredRows]);

  const handleRowClick = useCallback(
    (id: string) => {
      navigate(`/tokenBlueprint/${encodeURIComponent(id)}`);
    },
    [navigate],
  );

  const handleCreate = useCallback(() => {
    navigate("/tokenBlueprint/create");
  }, [navigate]);

  const handleReset = useCallback(() => {
    setBrandFilter([]);
    setAssigneeFilter([]);
    setMintedFilter([]);
    setSortKey(null);
    setSortDir(null);

    void reload();
  }, [reload]);

  const handleChangeBrandFilter = useCallback((vals: string[]) => {
    setBrandFilter(vals);
  }, []);

  const handleChangeAssigneeFilter = useCallback((vals: string[]) => {
    setAssigneeFilter(vals);
  }, []);

  const handleChangeMintedFilter = useCallback((vals: string[]) => {
    setMintedFilter(vals);
  }, []);

  const handleChangeSort = useCallback((key: string | null, dir: SortDir) => {
    setSortKey((key as SortKey) ?? null);
    setSortDir(dir);
  }, []);

  return {
    rows: displayRows,
    brandOptions,
    assigneeOptions,
    mintedOptions,
    brandFilter,
    assigneeFilter,
    mintedFilter,
    sortKey,
    sortDir,
    isResetting,
    handleChangeBrandFilter,
    handleChangeAssigneeFilter,
    handleChangeMintedFilter,
    handleChangeSort,
    handleReset,
    handleCreate,
    handleRowClick,
  };
}