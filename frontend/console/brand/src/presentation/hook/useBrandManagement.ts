// frontend/console/brand/src/presentation/hook/useBrandManagement.ts
import { useMemo, useState, useCallback, useEffect } from "react";
import { useAuth } from "../../../../shell/src/auth/presentation/hook/useCurrentMember";

import type { BrandRow as BrandRowBase } from "../../application/brandService";
import { listBrands } from "../../application/brandService";

// 共通型（SortOrder など）
import type { SortOrder } from "../../../../shell/src/shared/types/common/common";

export type SortKey = "registeredAt" | "updatedAt" | null;
export type StatusFilterValue = "active" | "inactive";

// BrandRow をローカルで拡張して updatedAt を必須にする
export type BrandRow = BrandRowBase & {
  updatedAt: string;
};

// フィルタ用オプション型（FilterableTableHeader と互換）
type ManagerOption = {
  value: string;
  label: string;
};

const toTs = (yyyyMd: string) => {
  if (!yyyyMd) return 0;
  const [y, m, d] = yyyyMd.split("/").map((v) => parseInt(v, 10));
  return new Date(y, (m || 1) - 1, d || 1).getTime();
};

export function useBrandManagement() {
  const { currentMember } = useAuth();
  const companyId = currentMember?.companyId ?? "";

  const [baseRows, setBaseRows] = useState<BrandRow[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<Error | null>(null);

  // リフレッシュボタン回転用（List の isResetting に渡す）
  const [isResetting, setIsResetting] = useState(false);

  const [statusFilter, setStatusFilter] = useState<StatusFilterValue[]>([]);

  // managerId フィルタ
  const [managerFilter, setManagerFilter] = useState<string[]>([]);

  const [activeKey, setActiveKey] = useState<SortKey>("registeredAt");
  const [direction, setDirection] = useState<SortOrder | null>("desc");

  // リロード用キー（Refreshボタン押下で再読み込みさせる）
  const [reloadKey, setReloadKey] = useState(0);

  // 責任者フィルタ用オプション（backend の memberName を label にする）
  const [managerOptions, setManagerOptions] = useState<ManagerOption[]>([]);

  // ステータスバッジ className（現状は使っていなくても残しておく）
  const statusBadgeClass = (isActive: boolean) =>
    `brand-status-badge ${isActive ? "is-active" : "is-inactive"}`;

  // データ読み込み
  useEffect(() => {
    let cancelled = false;

    const load = async () => {
      try {
        setLoading(true);
        setIsResetting(true);
        setError(null);

        if (!companyId) {
          setBaseRows([]);
          setManagerOptions([]);
          return;
        }

        const rawRows = await listBrands(companyId);

        // updatedAt を必須プロパティとして付与（加工は最小：空なら registeredAt を使う）
        const rows: BrandRow[] = (
          rawRows as (BrandRowBase & { updatedAt?: string })[]
        ).map((b) => {
          const rawUpdated = b.updatedAt ?? "";
          const safeUpdated = rawUpdated !== "" ? rawUpdated : b.registeredAt ?? "";

          return {
            ...b,
            updatedAt: safeUpdated,
          };
        });

        if (!cancelled) {
          setBaseRows(rows);
        }
      } catch (e: any) {
        if (!cancelled) {
          const err = e instanceof Error ? e : new Error(String(e));
          setError(err);
          setBaseRows([]);
          setManagerOptions([]);
        }
      } finally {
        if (!cancelled) {
          setLoading(false);
          setIsResetting(false);
        }
      }
    };

    void load();
    return () => {
      cancelled = true;
    };
  }, [companyId, reloadKey]);

  // baseRows から「責任者名付きオプション」を構築（memberName をそのまま label に）
  useEffect(() => {
    const seen = new Set<string>();
    const opts: ManagerOption[] = [];

    for (const b of baseRows) {
      const id = b.managerId ?? "";
      if (!id) continue;
      if (seen.has(id)) continue;
      seen.add(id);

      // backend の memberName（= managerName）を優先。無ければ id を表示。
      const label = b.memberName ?? "";
      opts.push({ value: id, label: label !== "" ? label : id });
    }

    setManagerOptions(opts);
  }, [baseRows]);

  // ステータスフィルタ
  const statusOptions = useMemo(() => {
    const values = Array.from(
      new Set<StatusFilterValue>(
        baseRows.map((b) => (b.isActive ? "active" : "inactive")),
      ),
    );
    return values.map((v) => ({
      value: v,
      label: v === "active" ? "アクティブ" : "停止",
    }));
  }, [baseRows]);

  // フィルタ＋ソート
  const rows = useMemo(() => {
    let data = baseRows.filter((b) => {
      const statusValue: StatusFilterValue = b.isActive ? "active" : "inactive";

      const statusOk =
        statusFilter.length === 0 || statusFilter.includes(statusValue);

      const managerValue = b.managerId ?? "";
      const managerOk =
        managerFilter.length === 0 ||
        (managerValue !== "" && managerFilter.includes(managerValue));

      return statusOk && managerOk;
    });

    if (activeKey && direction) {
      data = [...data].sort((a, b) => {
        if (activeKey === "registeredAt") {
          const av = toTs(a.registeredAt);
          const bv = toTs(b.registeredAt);
          return direction === "asc" ? av - bv : bv - av;
        }
        if (activeKey === "updatedAt") {
          const av = toTs(a.updatedAt);
          const bv = toTs(b.updatedAt);
          return direction === "asc" ? av - bv : bv - av;
        }
        return 0;
      });
    }
    return data;
  }, [baseRows, statusFilter, managerFilter, activeKey, direction]);

  // Refreshボタン用：フィルタとソートを初期化し、一覧も再取得
  const resetFilters = useCallback(() => {
    setStatusFilter([]);
    setManagerFilter([]);
    setActiveKey("registeredAt");
    setDirection("desc");
    setReloadKey((k) => k + 1);
  }, []);

  return {
    rows,
    statusOptions,
    managerOptions,

    loading,
    error,

    isResetting,

    statusFilter,
    managerFilter,
    activeKey,
    direction,

    setStatusFilter,
    setManagerFilter,
    setActiveKey,
    setDirection,

    statusBadgeClass,
    resetFilters,
  };
}