// frontend/console/list/src/presentation/hook/useListManagement.tsx

import React, { useMemo, useState, useCallback, useEffect } from "react";
import { useNavigate } from "react-router-dom";

import type { ListStatus } from "../../../../shell/src/shared/types/list";

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

  // ── Filter states（4列に合わせて最小化） ──────────────────
  const [productFilter, setProductFilter] = useState<string[]>([]);
  const [tokenFilter, setTokenFilter] = useState<string[]>([]);
  const [managerFilter, setManagerFilter] = useState<string[]>([]);
  const [statusFilter, setStatusFilter] = useState<string[]>([]); // holds ListStatus as string

  const filters: Filters = useMemo(
    () => ({ productFilter, tokenFilter, managerFilter, statusFilter }),
    [productFilter, tokenFilter, managerFilter, statusFilter],
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

  // ── Headers（serviceへ移譲） ──────────────────────────────
  const headers: React.ReactNode[] = useMemo(
    () =>
      buildHeaders({
        options,
        selected: filters,
        onChange: {
          setProductFilter,
          setTokenFilter,
          setManagerFilter,
          setStatusFilter,
        },
      }),
    [options, filters, setProductFilter, setTokenFilter, setManagerFilter, setStatusFilter],
  );

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
