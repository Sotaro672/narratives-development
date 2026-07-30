// frontend/console/shell/src/features/list/presentation/hook/useListManagement.tsx

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

  // リフレッシュボタン回転用
  isResetting: boolean;
};

export function useListManagement(): UseListManagementResult {
  const navigate = useNavigate();

  const [loading, setLoading] = useState<boolean>(false);
  const [isResetting, setIsResetting] = useState<boolean>(false);
  const [error, setError] = useState<string | null>(null);
  const [vmRowsSource, setVmRowsSource] = useState<ListManagementRowVM[]>([]);

  const reload = useCallback(async () => {
    setLoading(true);
    setIsResetting(true);
    setError(null);

    try {
      const { rows, error } = await loadListManagementRows();
      setVmRowsSource(rows);
      setError(error ?? null);
    } finally {
      setLoading(false);
      setIsResetting(false);
    }
  }, []);

  useEffect(() => {
    void reload();
  }, [reload]);

  const [productFilter, setProductFilter] = useState<string[]>([]);
  const [tokenFilter, setTokenFilter] = useState<string[]>([]);
  const [managerFilter, setManagerFilter] = useState<string[]>([]);
  const [statusFilter, setStatusFilter] = useState<string[]>([]);

  const filters: Filters = useMemo(
    () => ({
      productFilter,
      tokenFilter,
      managerFilter,
      statusFilter,
    }),
    [
      productFilter,
      tokenFilter,
      managerFilter,
      statusFilter,
    ],
  );

  const [activeKey, setActiveKey] = useState<SortKey>(null);
  const [direction, setDirection] = useState<"asc" | "desc" | null>(null);

  const onChangeSort = useCallback(
    (
      key: SortKey,
      dir: "asc" | "desc" | null,
    ) => {
      setActiveKey(key);
      setDirection(dir);
    },
    [],
  );

  const options = useMemo(
    () => buildFilterOptions(vmRowsSource),
    [vmRowsSource],
  );

  const rows = useMemo(() => {
    const filtered = applyFilters(vmRowsSource, filters);

    return applySort(
      filtered,
      activeKey,
      direction,
    );
  }, [
    vmRowsSource,
    filters,
    activeKey,
    direction,
  ]);

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
        sort: {
          activeKey,
          direction,
          onChange: onChangeSort,
        },
      }),
    [
      options,
      filters,
      activeKey,
      direction,
      onChangeSort,
    ],
  );

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

    void reload();
  }, [reload]);

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
    isResetting,
  };
}