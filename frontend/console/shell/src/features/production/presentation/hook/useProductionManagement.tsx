// frontend/console/shell/src/features/production/presentation/hook/useProductionManagement.tsx

import React, { useCallback, useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import {
  FilterableTableHeader,
  SortableTableHeader,
} from "../../../../layout/List/List";
import type {
  ProductionListRow,
  ProductionListRowView,
  ProductionSortDirection,
  ProductionSortKey,
} from "../../../../shared/types/production";
import {
  buildRowsView,
  loadProductionRows,
} from "../../application/productionManagementService";

function extractBackendJsonErrorMessage(e: unknown): string {
  const raw = e instanceof Error ? e.message : String(e ?? "");
  const match = raw.match(/\{[\s\S]*\}$/);

  if (!match) return raw;

  try {
    const parsed: unknown = JSON.parse(match[0]);

    if (
      typeof parsed === "object" &&
      parsed !== null &&
      "error" in parsed &&
      typeof parsed.error === "string"
    ) {
      return parsed.error || raw;
    }

    return raw;
  } catch {
    return raw;
  }
}

function isInvalidCompanyIDError(e: unknown): boolean {
  const message = extractBackendJsonErrorMessage(e);
  return (
    message.includes("invalid companyId") ||
    message.includes("invalid companyID")
  );
}

export function useProductionManagement() {
  const navigate = useNavigate();

  const [blueprintFilter, setBlueprintFilter] = useState<string[]>([]);
  const [brandFilter, setBrandFilter] = useState<string[]>([]);
  const [assigneeFilter, setAssigneeFilter] = useState<string[]>([]);
  const [printedFilter, setPrintedFilter] = useState<boolean[]>([]);

  const [sortKey, setSortKey] = useState<ProductionSortKey>(null);
  const [sortDir, setSortDir] = useState<ProductionSortDirection>(null);

  const [baseRows, setBaseRows] = useState<ProductionListRow[]>([]);
  const [loading, setLoading] = useState(false);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [isResetting, setIsResetting] = useState(false);

  const reload = useCallback(async () => {
    setLoading(true);
    setIsResetting(true);
    setLoadError(null);

    try {
      const loadedRows = await loadProductionRows();
      setBaseRows(loadedRows);
    } catch (e) {
      if (isInvalidCompanyIDError(e)) {
        setLoadError(
          "会社情報（companyId）が未設定のため、生産計画一覧を表示できません。先に会社を作成（または招待を受諾）してください。",
        );
      } else {
        setLoadError("生産計画一覧の取得に失敗しました。");
      }

      setBaseRows([]);
    } finally {
      setLoading(false);
      setIsResetting(false);
    }
  }, []);

  useEffect(() => {
    let cancelled = false;

    void (async () => {
      if (cancelled) return;
      await reload();
    })();

    return () => {
      cancelled = true;
    };
  }, [reload]);

  const blueprintOptions = useMemo(() => {
    const map = new Map<string, string>();

    for (const row of baseRows) {
      const id = row.productBlueprintId;
      if (!id || map.has(id)) continue;

      map.set(id, row.productName || id);
    }

    return Array.from(map.entries()).map(([value, label]) => ({
      value,
      label,
    }));
  }, [baseRows]);

  const brandOptions = useMemo(() => {
    const map = new Map<string, string>();

    for (const row of baseRows) {
      const name = row.brandName.trim();
      if (!name || map.has(name)) continue;

      map.set(name, name);
    }

    return Array.from(map.entries()).map(([value, label]) => ({
      value,
      label,
    }));
  }, [baseRows]);

  const assigneeOptions = useMemo(() => {
    const map = new Map<string, string>();

    for (const row of baseRows) {
      const id = row.assigneeId.trim();
      if (!id || map.has(id)) continue;

      map.set(id, row.assigneeName.trim() || id);
    }

    return Array.from(map.entries()).map(([value, label]) => ({
      value,
      label,
    }));
  }, [baseRows]);

  const printedOptions = useMemo(
    () => [
      { value: "true", label: "印刷済" },
      { value: "false", label: "印刷前" },
    ],
    [],
  );

  const allRowsView = useMemo<ProductionListRowView[]>(
    () =>
      buildRowsView({
        baseRows,
        blueprintFilter,
        assigneeFilter,
        printedFilter,
        sortKey,
        sortDir,
      }),
    [
      baseRows,
      blueprintFilter,
      assigneeFilter,
      printedFilter,
      sortKey,
      sortDir,
    ],
  );

  const rows = useMemo<ProductionListRowView[]>(() => {
    if (brandFilter.length === 0) return allRowsView;

    return allRowsView.filter((row) =>
      brandFilter.includes(row.brandName.trim()),
    );
  }, [allRowsView, brandFilter]);

  const headers = useMemo<React.ReactNode[]>(
    () => [
      <FilterableTableHeader
        key="blueprint"
        label="プロダクト名"
        options={blueprintOptions}
        selected={blueprintFilter}
        onChange={setBlueprintFilter}
      />,
      <FilterableTableHeader
        key="brand"
        label="ブランド"
        options={brandOptions}
        selected={brandFilter}
        onChange={setBrandFilter}
      />,
      <FilterableTableHeader
        key="assignee"
        label="担当者"
        options={assigneeOptions}
        selected={assigneeFilter}
        onChange={setAssigneeFilter}
      />,
      <FilterableTableHeader
        key="printed"
        label="印刷状態"
        options={printedOptions}
        selected={printedFilter.map(String)}
        onChange={(values) =>
          setPrintedFilter(values.map((value) => value === "true"))
        }
      />,
      <SortableTableHeader
        key="totalQuantity"
        label="総生産数"
        sortKey="totalQuantity"
        activeKey={sortKey}
        direction={sortDir}
        onChange={(key, direction) => {
          setSortKey(key as ProductionSortKey);
          setSortDir(direction);
        }}
      />,
      <SortableTableHeader
        key="printedAt"
        label="印刷日"
        sortKey="printedAt"
        activeKey={sortKey}
        direction={sortDir}
        onChange={(key, direction) => {
          setSortKey(key as ProductionSortKey);
          setSortDir(direction);
        }}
      />,
      <SortableTableHeader
        key="createdAt"
        label="作成日"
        sortKey="createdAt"
        activeKey={sortKey}
        direction={sortDir}
        onChange={(key, direction) => {
          setSortKey(key as ProductionSortKey);
          setSortDir(direction);
        }}
      />,
    ],
    [
      blueprintOptions,
      blueprintFilter,
      brandOptions,
      brandFilter,
      assigneeOptions,
      assigneeFilter,
      printedOptions,
      printedFilter,
      sortKey,
      sortDir,
    ],
  );

  const handleCreate = useCallback(() => {
    navigate("create");
  }, [navigate]);

  const handleReset = useCallback(() => {
    setBlueprintFilter([]);
    setBrandFilter([]);
    setAssigneeFilter([]);
    setPrintedFilter([]);
    setSortKey(null);
    setSortDir(null);
    void reload();
  }, [reload]);

  const handleRowClick = useCallback(
    (id: string) => {
      navigate(encodeURIComponent(id));
    },
    [navigate],
  );

  return {
    headers,
    rows,
    loading,
    isResetting,
    loadError,
    reload,
    handleCreate,
    handleReset,
    handleRowClick,
  };
}

export default useProductionManagement;