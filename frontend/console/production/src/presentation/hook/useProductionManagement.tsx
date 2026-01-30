// frontend/console/production/src/presentation/hook/useProductionManagement.tsx

import React, { useEffect, useMemo, useState, useCallback } from "react";
import { useNavigate } from "react-router-dom";
import {
  FilterableTableHeader,
  SortableTableHeader,
} from "../../../../shell/src/layout/List/List";

import {
  loadProductionRows,
  buildRowsView,
  type SortKey,
  type ProductionRow,
  type ProductionRowView,
} from "../../application/productionManagementService";

function extractBackendJsonErrorMessage(e: unknown): string {
  // productionManagementService 側の throw が `...（500 ）\n{"error":"..."}` のような形式になることがあるので、JSON 部分を拾う
  const raw = (e as any)?.message ?? String(e ?? "");
  const m = raw.match(/\{[\s\S]*\}$/);
  if (!m) return raw;

  try {
    const obj = JSON.parse(m[0]);
    const msg = typeof obj?.error === "string" ? obj.error : "";
    return msg || raw;
  } catch {
    return raw;
  }
}

function isInvalidCompanyIDError(e: unknown): boolean {
  const msg = extractBackendJsonErrorMessage(e);
  return msg.includes("invalid companyId") || msg.includes("invalid companyID");
}

/**
 * printed(boolean) -> 日本語ラベル
 * - true: 印刷済
 * - false: 印刷前
 */
function formatPrintedJa(printed: boolean): string {
  return printed ? "印刷済" : "印刷前";
}

export function useProductionManagement() {
  const navigate = useNavigate();

  // ===== フィルタ状態 =====
  const [blueprintFilter, setBlueprintFilter] = useState<string[]>([]);
  const [brandFilter, setBrandFilter] = useState<string[]>([]);
  const [assigneeFilter, setAssigneeFilter] = useState<string[]>([]);
  const [printedFilter, setPrintedFilter] = useState<boolean[]>([]);

  // ===== ソート状態 =====
  const [sortKey, setSortKey] = useState<SortKey>(null);
  const [sortDir, setSortDir] = useState<"asc" | "desc" | null>(null);

  // ===== ベース行データ（API から取得した値 + totalQuantity） =====
  const [baseRows, setBaseRows] = useState<ProductionRow[]>([]);

  // ===== ローディング / エラー =====
  const [loading, setLoading] = useState(false);
  const [loadError, setLoadError] = useState<string | null>(null);

  const reload = useCallback(async () => {
    setLoading(true);
    setLoadError(null);

    try {
      const rows = await loadProductionRows();
      setBaseRows(rows ?? []);
    } catch (e) {
      // ★ companyId 無しのユーザーが /productions を叩くと backend が 500 で弾く（方針どおり）
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
    }
  }, []);

  useEffect(() => {
    let cancelled = false;

    (async () => {
      if (cancelled) return;
      await reload();
    })();

    return () => {
      cancelled = true;
    };
  }, [reload]);

  // ===== オプション生成 =====
  // プロダクト名フィルタ: value は productBlueprintId, label は productName（なければ ID）
  const blueprintOptions = useMemo(() => {
    const map = new Map<string, string>();

    for (const row of baseRows) {
      const id = row.productBlueprintId;
      if (!id) continue;
      if (!map.has(id)) {
        const name = (row as any).productName ?? "";
        const label = name || id;
        map.set(id, label);
      }
    }

    return Array.from(map.entries()).map(([value, label]) => ({
      value,
      label,
    }));
  }, [baseRows]);

  // ブランドフィルタ: value / label ともに brandName
  const brandOptions = useMemo(() => {
    const map = new Map<string, string>();

    for (const row of baseRows) {
      const name = ((row as any).brandName ?? "") as string;
      const trimmed = name.trim();
      if (!trimmed) continue;
      if (!map.has(trimmed)) {
        map.set(trimmed, trimmed);
      }
    }

    return Array.from(map.entries()).map(([value, label]) => ({
      value,
      label,
    }));
  }, [baseRows]);

  // 担当者フィルタ: value は assigneeId, label は assigneeName（なければ ID）
  const assigneeOptions = useMemo(() => {
    const map = new Map<string, string>();

    for (const row of baseRows) {
      const id = (row.assigneeId ?? "").trim();
      if (!id) continue;
      if (!map.has(id)) {
        const name = (row as any).assigneeName ?? "";
        const label = (name as string).trim() || id;
        map.set(id, label);
      }
    }

    return Array.from(map.entries()).map(([value, label]) => ({
      value,
      label,
    }));
  }, [baseRows]);

  // 印刷状態フィルタ（true/false を固定で提供）
  const printedOptions = useMemo(
    () => [
      { value: true as any, label: "印刷済" },
      { value: false as any, label: "印刷前" },
    ],
    [],
  );

  // ===== フィルタ＋ソート適用 → 表示用行に変換 =====
  const allRowsView: ProductionRowView[] = useMemo(
    () =>
      buildRowsView({
        baseRows,
        blueprintFilter,
        assigneeFilter,
        printedFilter,
        sortKey,
        sortDir,
      }),
    [baseRows, blueprintFilter, assigneeFilter, printedFilter, sortKey, sortDir],
  );

  // ブランドフィルタは View 行に対して適用
  const rows: ProductionRowView[] = useMemo(() => {
    if (brandFilter.length === 0) return allRowsView;

    return allRowsView.filter((r) =>
      brandFilter.includes((r.brandName ?? "").trim()),
    );
  }, [allRowsView, brandFilter]);

  // ===== ヘッダー =====
  const headers: React.ReactNode[] = useMemo(
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
        selected={printedFilter as unknown as string[]}
        onChange={(values) => setPrintedFilter(values as unknown as boolean[])}
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

  // ===== ハンドラ =====
  const handleCreate = () => {
    // 相対パスで ProductionCreate へ
    navigate("create");
  };

  const handleReset = () => {
    setBlueprintFilter([]);
    setBrandFilter([]);
    setAssigneeFilter([]);
    setPrintedFilter([]);
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

    loading,
    loadError,
    reload,

    handleCreate,
    handleReset,
    handleRowClick,
  };
}
