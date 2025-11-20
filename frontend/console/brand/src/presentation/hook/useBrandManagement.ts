// frontend/console/brand/src/presentation/hook/useBrandManagement.ts
import { useMemo, useState, useCallback, useEffect } from "react";
import { useAuth } from "../../../../shell/src/auth/presentation/hook/useCurrentMember";

import type { BrandRow } from "../../application/brandService";
import { listBrands } from "../../application/brandService";

export type SortKey = "registeredAt" | null;
export type StatusFilterValue = "active" | "inactive";

const toTs = (yyyyMd: string) => {
  const [y, m, d] = yyyyMd.split("/").map((v) => parseInt(v, 10));
  return new Date(y, (m || 1) - 1, d || 1).getTime();
};

export function useBrandManagement() {
  const { currentMember } = useAuth();
  const companyId = (currentMember?.companyId ?? "").trim();

  const [baseRows, setBaseRows] = useState<BrandRow[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<Error | null>(null);

  const [statusFilter, setStatusFilter] = useState<StatusFilterValue[]>([]);
  const [ownerFilter, setOwnerFilter] = useState<string[]>([]);
  const [activeKey, setActiveKey] = useState<SortKey>("registeredAt");
  const [direction, setDirection] = useState<"asc" | "desc" | null>("desc");

  // ステータスバッジ className
  const statusBadgeClass = (isActive: boolean) =>
    `brand-status-badge ${isActive ? "is-active" : "is-inactive"}`;

  // データ読み込み
  useEffect(() => {
    let cancelled = false;

    const load = async () => {
      try {
        setLoading(true);
        setError(null);

        if (!companyId) {
          setBaseRows([]);
          return;
        }

        const rows = await listBrands(companyId);
        if (!cancelled) setBaseRows(rows);
      } catch (e: any) {
        if (!cancelled) {
          setError(e instanceof Error ? e : new Error(String(e)));
          setBaseRows([]);
        }
      } finally {
        if (!cancelled) setLoading(false);
      }
    };

    void load();
    return () => {
      cancelled = true;
    };
  }, [companyId]);

  // ステータスフィルタ
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

  // owner は廃止 → ownerOptions は managerId ベース
  const ownerOptions = useMemo(() => {
    const ids = new Set(
      baseRows.map((b) => (b.managerId ?? "").trim()).filter(Boolean)
    );
    return Array.from(ids).map((id) => ({
      value: id,
      label: id, // 表示名もIDのみ
    }));
  }, [baseRows]);

  // フィルタ＋ソート
  const rows = useMemo(() => {
    let data = baseRows.filter((b) => {
      const statusValue: StatusFilterValue = b.isActive ? "active" : "inactive";

      const statusOk =
        statusFilter.length === 0 || statusFilter.includes(statusValue);

      const managerValue = (b.managerId ?? "").trim();
      const ownerOk =
        ownerFilter.length === 0 ||
        (managerValue !== "" && ownerFilter.includes(managerValue));

      return statusOk && ownerOk;
    });

    if (activeKey && direction) {
      data = [...data].sort((a, b) => {
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

  const resetFilters = useCallback(() => {
    setStatusFilter([]);
    setOwnerFilter([]);
    setActiveKey("registeredAt");
    setDirection("desc");
  }, []);

  return {
    rows,
    statusOptions,
    ownerOptions,

    loading,
    error,

    statusFilter,
    ownerFilter,
    activeKey,
    direction,

    setStatusFilter,
    setOwnerFilter,
    setActiveKey,
    setDirection,

    statusBadgeClass,
    resetFilters,
  };
}
