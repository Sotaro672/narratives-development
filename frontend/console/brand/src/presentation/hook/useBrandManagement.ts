//frontend\console\brand\src\presentation\hook\useBrandManagement.ts
import { useMemo, useState, useCallback } from "react";
import {
  ALL_BRANDS,
  toBrandRows,
  type BrandRow,
} from "../../infrastructure/mockdata/mockdata";

// Utility: "YYYY/MM/DD" → timestamp
const toTs = (yyyyMd: string) => {
  const [y, m, d] = yyyyMd.split("/").map((v) => parseInt(v, 10));
  return new Date(y, (m || 1) - 1, d || 1).getTime();
};

export type SortKey = "registeredAt" | null;
export type StatusFilterValue = "active" | "inactive";

export function useBrandManagement() {
  // モックBrand → 一覧表示用Rowへ変換
  const baseRows = useMemo<BrandRow[]>(() => toBrandRows(ALL_BRANDS), []);

  // ────────────────────────────────
  // フィルタ・ソート状態
  // ────────────────────────────────
  const [statusFilter, setStatusFilter] = useState<StatusFilterValue[]>([]);
  const [ownerFilter, setOwnerFilter] = useState<string[]>([]);
  const [activeKey, setActiveKey] = useState<SortKey>("registeredAt");
  const [direction, setDirection] = useState<"asc" | "desc" | null>("desc");

  // ステータス絞り込みオプション
  const statusOptions = useMemo(() => {
    const values = Array.from(
      new Set<StatusFilterValue>(
        baseRows.map((b) => (b.isActive ? "active" : "inactive"))
      )
    );
    return values.map((v) => ({
      value: v,
      label: v === "active" ? "アクティブ" : "停止",
    }));
  }, [baseRows]);

  // 責任者絞り込みオプション
  const ownerOptions = useMemo(() => {
    const values = Array.from(new Set(baseRows.map((b) => b.owner)));
    return values.map((v) => ({
      value: v,
      label: v,
    }));
  }, [baseRows]);

  // ────────────────────────────────
  // データフィルタリング＋ソート
  // ────────────────────────────────
  const rows = useMemo(() => {
    let data = baseRows.filter((b: BrandRow) => {
      const statusValue: StatusFilterValue = b.isActive ? "active" : "inactive";

      const statusOk =
        statusFilter.length === 0 || statusFilter.includes(statusValue);

      const ownerOk =
        ownerFilter.length === 0 || ownerFilter.includes(b.owner);

      return statusOk && ownerOk;
    });

    if (activeKey && direction) {
      data = [...data].sort((a: BrandRow, b: BrandRow) => {
        if (activeKey === "registeredAt") {
          const av = toTs(a.registeredAt);
          const bv = toTs(b.registeredAt);
          return direction === "asc" ? av - bv : bv - av;
        }
        return 0;
      });
    }

    return data;
  }, [baseRows, statusFilter, ownerFilter, activeKey, direction]);

  // バッジクラス
  const statusBadgeClass = (isActive: boolean) =>
    `brand-status-badge ${isActive ? "is-active" : "is-inactive"}`;

  // リセット処理
  const resetFilters = useCallback(() => {
    setStatusFilter([]);
    setOwnerFilter([]);
    setActiveKey("registeredAt");
    setDirection("desc");
    console.log("ブランドリストをリセット");
  }, []);

  return {
    // rows と options
    rows,
    statusOptions,
    ownerOptions,

    // 状態
    statusFilter,
    ownerFilter,
    activeKey,
    direction,

    // Setter
    setStatusFilter,
    setOwnerFilter,
    setActiveKey,
    setDirection,

    // utils
    statusBadgeClass,
    resetFilters,
  };
}
