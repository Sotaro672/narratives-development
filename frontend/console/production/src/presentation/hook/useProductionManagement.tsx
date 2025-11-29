// frontend/console/production/src/presentation/hook/useProductionManagement.tsx

import React, { useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import {
  FilterableTableHeader,
  SortableTableHeader,
} from "../../../../shell/src/layout/List/List";
import type { ProductionStatus } from "../../../../shell/src/shared/types/production";

import {
  loadProductionRows,
  buildBlueprintOptions,
  buildAssigneeOptions,
  buildStatusOptions,
  buildRowsView,
  type SortKey,
  type ProductionRow,
  type ProductionRowView,
} from "../../application/productionManagementService";

export function useProductionManagement() {
  const navigate = useNavigate();

  // ===== フィルタ状態 =====
  const [blueprintFilter, setBlueprintFilter] = useState<string[]>([]);
  const [assigneeFilter, setAssigneeFilter] = useState<string[]>([]);
  const [statusFilter, setStatusFilter] = useState<ProductionStatus[]>([]);

  // ===== ソート状態 =====
  const [sortKey, setSortKey] = useState<SortKey>(null);
  const [sortDir, setSortDir] = useState<"asc" | "desc" | null>(null);

  // ===== ベース行データ（API から取得した値 + totalQuantity） =====
  const [baseRows, setBaseRows] = useState<ProductionRow[]>([]);

  useEffect(() => {
    let cancelled = false;

    (async () => {
      try {
        const rows = await loadProductionRows();
        if (cancelled) return;
        setBaseRows(rows);
      } catch (e) {
        console.error(
          "[useProductionManagement] failed to load productions:",
          e,
        );
        if (!cancelled) {
          setBaseRows([]);
        }
      }
    })();

    return () => {
      cancelled = true;
    };
  }, []);

  // ===== オプション生成 =====
  const blueprintOptions = useMemo(
    () => buildBlueprintOptions(baseRows),
    [baseRows],
  );

  const assigneeOptions = useMemo(
    () => buildAssigneeOptions(baseRows),
    [baseRows],
  );

  const statusOptions = useMemo(
    () => buildStatusOptions(baseRows),
    [baseRows],
  );

  // ===== フィルタ＋ソート適用 → 表示用行に変換 =====
  const rows: ProductionRowView[] = useMemo(
    () =>
      buildRowsView({
        baseRows,
        blueprintFilter,
        assigneeFilter,
        statusFilter,
        sortKey,
        sortDir,
      }),
    [
      baseRows,
      blueprintFilter,
      assigneeFilter,
      statusFilter,
      sortKey,
      sortDir,
    ],
  );

  // ===== ヘッダー =====
  const headers: React.ReactNode[] = useMemo(
    () => [
      "ID",
      <FilterableTableHeader
        key="blueprint"
        label="商品設計ID"
        options={blueprintOptions}
        selected={blueprintFilter}
        onChange={setBlueprintFilter}
      />,
      <FilterableTableHeader
        key="assignee"
        label="担当者ID"
        options={assigneeOptions}
        selected={assigneeFilter}
        onChange={setAssigneeFilter}
      />,
      <FilterableTableHeader
        key="status"
        label="ステータス"
        options={statusOptions}
        selected={statusFilter as unknown as string[]}
        onChange={(values) => setStatusFilter(values as ProductionStatus[])}
      />,
      <SortableTableHeader
        key="totalQuantity"
        label="総生産数"
        sortKey="totalQuantity"
        activeKey={sortKey}
        direction={sortDir}
        onChange={(key, dir) => {
          setSortKey(key as SortKey);
          setSortDir(dir);
        }}
      />,
      <SortableTableHeader
        key="printedAt"
        label="印刷日"
        sortKey="printedAt"
        activeKey={sortKey}
        direction={sortDir}
        onChange={(key, dir) => {
          setSortKey(key as SortKey);
          setSortDir(dir);
        }}
      />,
      <SortableTableHeader
        key="createdAt"
        label="作成日"
        sortKey="createdAt"
        activeKey={sortKey}
        direction={sortDir}
        onChange={(key, dir) => {
          setSortKey(key as SortKey);
          setSortDir(dir);
        }}
      />,
    ],
    [
      blueprintOptions,
      blueprintFilter,
      assigneeOptions,
      assigneeFilter,
      statusOptions,
      statusFilter,
      sortKey,
      sortDir,
    ],
  );

  // ===== ハンドラ =====
  const handleCreate = () => {
    // 相対パスで ProductionCreate へ
    navigate("create");
  };

  const handleReset = () => {
    setBlueprintFilter([]);
    setAssigneeFilter([]);
    setStatusFilter([]);
    setSortKey(null);
    setSortDir(null);
  };

  const handleRowClick = (id: string) => {
    // 相対パスで詳細へ
    navigate(encodeURIComponent(id));
  };

  return {
    headers,
    rows,
    handleCreate,
    handleReset,
    handleRowClick,
  };
}
