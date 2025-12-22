// frontend/console/list/src/presentation/hook/useListManagement.tsx
import React, { useMemo, useState, useCallback, useEffect } from "react";
import { useNavigate } from "react-router-dom";
import {
  type SortKey,
  type ListManagementRowVM,
  type Filters,
  buildFilterOptions,
  buildHeaders,
  applyFilters,
  applySort,
  loadListManagementRows,
} from "../../application/listManagementService";

export type UseListManagementResult = {
  vm: {
    title: string;
    headers: React.ReactNode[];
    rows: ListManagementRowVM[];
    loading: boolean;
    error: string | null;
  };
  handlers: {
    onReset: () => void;
    onRowClick: (id: string) => void;
    onRowKeyDown: (e: React.KeyboardEvent, id: string) => void;
  };
};

export function useListManagement(): UseListManagementResult {
  const navigate = useNavigate();

  // ── Data state（hook要素のみ） ─────────────────────────────
  const [loading, setLoading] = useState<boolean>(false);
  const [error, setError] = useState<string | null>(null);
  const [vmRowsSource, setVmRowsSource] = useState<ListManagementRowVM[]>([]);

  // 初回ロード
  useEffect(() => {
    let mounted = true;

    (async () => {
      setLoading(true);
      setError(null);

      const { rows, error } = await loadListManagementRows();

      if (!mounted) return;

      setVmRowsSource(rows);
      setError(error ?? null);
      setLoading(false);
    })();

    return () => {
      mounted = false;
    };
  }, []);

  // ── Filter states（5列に合わせて最小化） ──────────────────
  // ✅ titleFilter は型/フィルタ適用のために保持するが、ヘッダーUIには出さない（＝filterable-table-headerを導入しない）
  const [titleFilter, setTitleFilter] = useState<string[]>([]);
  const [productFilter, setProductFilter] = useState<string[]>([]);
  const [tokenFilter, setTokenFilter] = useState<string[]>([]);
  const [managerFilter, setManagerFilter] = useState<string[]>([]);
  const [statusFilter, setStatusFilter] = useState<string[]>([]);

  const filters: Filters = useMemo(
    () => ({ titleFilter, productFilter, tokenFilter, managerFilter, statusFilter }),
    [titleFilter, productFilter, tokenFilter, managerFilter, statusFilter],
  );

  // ── Sort state（必要最低限：id だけ） ─────────────────────
  const [activeKey, setActiveKey] = useState<SortKey>(null);
  const [direction, setDirection] = useState<"asc" | "desc" | null>(null);

  // options
  const options = useMemo(() => buildFilterOptions(vmRowsSource), [vmRowsSource]);

  // ── Build rows (filter → sort) ────────────────────────────
  const rows = useMemo(() => {
    const filtered = applyFilters(vmRowsSource, filters);
    return applySort(filtered, activeKey, direction);
  }, [vmRowsSource, filters, activeKey, direction]);

  // ── Headers ───────────────────────────────────────────────
  const headers: React.ReactNode[] = useMemo(() => {
    // まず service 側のヘッダー（title含む）を生成
    const built = buildHeaders({
      options,
      selected: filters,
      onChange: {
        // NOTE: titleFilter の setter は渡すが、表示上は使わない（後で差し替える）
        setTitleFilter,
        setProductFilter,
        setTokenFilter,
        setManagerFilter,
        setStatusFilter,
      },
    });

    // ✅ タイトル列だけは filterable-table-header を使わず固定の見出しに差し替える
    // buildHeaders が 先頭=タイトル列 の前提（5列構成）で運用
    const plainTitleHeader = <span key="title-header">タイトル</span>;

    if (Array.isArray(built) && built.length > 0) {
      return [plainTitleHeader, ...built.slice(1)];
    }
    return [plainTitleHeader];
  }, [
    options,
    filters,
    setTitleFilter,
    setProductFilter,
    setTokenFilter,
    setManagerFilter,
    setStatusFilter,
  ]);

  // handlers
  const onRowClick = useCallback(
    (id: string) => {
      navigate(`/list/${encodeURIComponent(id)}`);
    },
    [navigate],
  );

  const onRowKeyDown = useCallback(
    (e: React.KeyboardEvent, id: string) => {
      if (e.key === "Enter" || e.key === " ") {
        e.preventDefault();
        navigate(`/list/${encodeURIComponent(id)}`);
      }
    },
    [navigate],
  );

  const onReset = useCallback(() => {
    setTitleFilter([]);
    setProductFilter([]);
    setTokenFilter([]);
    setManagerFilter([]);
    setStatusFilter([]);
    setActiveKey(null);
    setDirection(null);
  }, []);

  return {
    vm: {
      title: "出品管理",
      headers,
      rows,
      loading,
      error,
    },
    handlers: {
      onReset,
      onRowClick,
      onRowKeyDown,
    },
  };
}
