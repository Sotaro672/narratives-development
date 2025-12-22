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
  // ✅ titleFilter は型/フィルタ適用のために保持するが、ヘッダーUIには出さない
  const [titleFilter, setTitleFilter] = useState<string[]>([]);
  const [productFilter, setProductFilter] = useState<string[]>([]);
  const [tokenFilter, setTokenFilter] = useState<string[]>([]);
  const [managerFilter, setManagerFilter] = useState<string[]>([]);
  const [statusFilter, setStatusFilter] = useState<string[]>([]);

  const filters: Filters = useMemo(
    () => ({ titleFilter, productFilter, tokenFilter, managerFilter, statusFilter }),
    [titleFilter, productFilter, tokenFilter, managerFilter, statusFilter],
  );

  // ── Sort state（id / createdAt） ──────────────────────────
  const [activeKey, setActiveKey] = useState<SortKey>(null);
  const [direction, setDirection] = useState<"asc" | "desc" | null>(null);

  // sort handler（service の buildHeaders が要求する形に合わせる）
  const onChangeSort = useCallback((key: SortKey, dir: "asc" | "desc" | null) => {
    setActiveKey(key);
    setDirection(dir);
  }, []);

  // options
  const options = useMemo(() => buildFilterOptions(vmRowsSource), [vmRowsSource]);

  // ── Build rows (filter → sort) ────────────────────────────
  const rows = useMemo(() => {
    const filtered = applyFilters(vmRowsSource, filters);
    return applySort(filtered, activeKey, direction);
  }, [vmRowsSource, filters, activeKey, direction]);

  // ── Headers ───────────────────────────────────────────────
  const headers: React.ReactNode[] = useMemo(() => {
    // service 側のヘッダー（status の右隣に createdAt を含む）を生成
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
      sort: {
        activeKey,
        direction,
        onChange: onChangeSort,
      },
    });

    // ✅ タイトル列だけは filterable-table-header を使わず固定の見出しに差し替える
    const plainTitleHeader = <span key="title-header">タイトル</span>;

    if (Array.isArray(built) && built.length > 0) {
      // built は [title, product, token, manager, status, createdAt] の想定
      return [plainTitleHeader, ...built.slice(1)];
    }
    return [plainTitleHeader];
  }, [
    options,
    filters,
    activeKey,
    direction,
    onChangeSort,
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
