// frontend/console/brand/src/presentation/hook/useBrandManagement.ts
import { useMemo, useState, useCallback, useEffect } from "react";
import { useAuth } from "../../../../shell/src/auth/presentation/hook/useCurrentMember";

import type { BrandRow as BrandRowBase } from "../../application/brandService";
import { listBrands } from "../../application/brandService";

// ★ 共通型（SortOrder など）を導入
import type { SortOrder } from "../../../../shell/src/shared/types/common/common";

// ★ memberID → 「姓 名」を解決するフックをここで利用
import { useMemberList } from "../../../../member/src/presentation/hooks/useMemberList";

export type SortKey = "registeredAt" | "updatedAt" | null;
export type StatusFilterValue = "active" | "inactive";

// ★ BrandRow をローカルで拡張して updatedAt を必須にする
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
  const companyId = (currentMember?.companyId ?? "").trim();

  const [baseRows, setBaseRows] = useState<BrandRow[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<Error | null>(null);

  const [statusFilter, setStatusFilter] = useState<StatusFilterValue[]>([]);

  // ★ managerId フィルタ
  const [managerFilter, setManagerFilter] = useState<string[]>([]);

  const [activeKey, setActiveKey] = useState<SortKey>("registeredAt");
  const [direction, setDirection] = useState<SortOrder | null>("desc");

  // ★ リロード用キー（Refreshボタン押下で再読み込みさせる）
  const [reloadKey, setReloadKey] = useState(0);

  // ★ member 用フックから ID → 氏名変換関数をここで取得（Promise<string> を返す想定）
  const { getNameLastFirstByID } = useMemberList();

  // ★ 責任者フィルタ用オプション（ID→名前を解決したものを state で持つ）
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
        setError(null);

        if (!companyId) {
          setBaseRows([]);
          setManagerOptions([]);
          return;
        }

        const rawRows = await listBrands(companyId);

        // ★ updatedAt を必須プロパティとして付与
        const rows: BrandRow[] = (
          rawRows as (BrandRowBase & { updatedAt?: string })[]
        ).map((b) => {
          const rawUpdated = (b.updatedAt ?? "").trim();
          // backend から updatedAt が来ていればそれを使い、なければ registeredAt をフォールバック
          const safeUpdated =
            rawUpdated !== "" ? rawUpdated : (b as any).registeredAt ?? "";

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
        if (!cancelled) setLoading(false);
      }
    };

    void load();
    return () => {
      cancelled = true;
    };
  }, [companyId, reloadKey]);

  // ★ baseRows から managerId 一覧を取り出して「責任者名付きオプション」に変換
  useEffect(() => {
    let cancelled = false;

    const buildManagerOptions = async () => {
      const ids = Array.from(
        new Set(
          baseRows
            .map((b) => (b.managerId ?? "").trim())
            .filter((v) => v !== ""),
        ),
      );

      const opts: ManagerOption[] = [];

      for (const id of ids) {
        try {
          const name = (await getNameLastFirstByID(id)) || id;
          opts.push({ value: id, label: name });
        } catch (e) {
          // eslint-disable-next-line no-console
          console.error("[useBrandManagement] resolve manager name error:", e);
          opts.push({ value: id, label: id });
        }
      }

      if (!cancelled) {
        setManagerOptions(opts);
      }
    };

    void buildManagerOptions();

    return () => {
      cancelled = true;
    };
  }, [baseRows, getNameLastFirstByID]);

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

  // managerOptions は上の useEffect で state 管理しているので useMemo は不要

  // フィルタ＋ソート
  const rows = useMemo(() => {
    let data = baseRows.filter((b) => {
      const statusValue: StatusFilterValue = b.isActive ? "active" : "inactive";

      const statusOk =
        statusFilter.length === 0 || statusFilter.includes(statusValue);

      const managerValue = (b.managerId ?? "").trim();
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

  // ▼ Refreshボタン用：フィルタとソートを初期化し、一覧も再取得
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
    managerOptions, // ★ managerName 付きオプション

    loading,
    error,

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

    // ★ ページ側でそのまま使えるように公開（Promise<string>）
    getNameLastFirstByID,
  };
}
