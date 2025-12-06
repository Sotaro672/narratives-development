// frontend/console/tokenOperation/src/presentation/hook/useTokenOperation.tsx

import { useCallback, useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import type {
  TokenOperationExtended,
} from "../../../../shell/src/shared/types/tokenOperation";
import { useAuth } from "../../../../shell/src/auth/presentation/hook/useCurrentMember";
import {
  SortKey,
  SortDir,
  FilterOption,
  fetchTokenOperationsForCompany,
  buildOptionsFromTokenOperations,
  filterAndSortTokenOperations,
} from "../../application/tokenOperationService";

export type UseTokenOperationReturn = {
  brandFilter: string[];
  assigneeFilter: string[];
  brandOptions: FilterOption[];
  assigneeOptions: FilterOption[];
  activeKey: SortKey;
  direction: SortDir;
  rows: TokenOperationExtended[];
  handleSortChange: (key: string | null, dir: SortDir) => void;
  handleBrandFilterChange: (values: string[]) => void;
  handleAssigneeFilterChange: (values: string[]) => void;
  handleReset: () => void;
  goDetail: (operationId: string) => void;
};

export function useTokenOperation(): UseTokenOperationReturn {
  const navigate = useNavigate();
  const { currentMember } = useAuth();

  // ── 一覧データ（backend から取得する運用トークン一覧） ───────────────
  const [allRows, setAllRows] = useState<TokenOperationExtended[]>([]);

  useEffect(() => {
    const companyId = currentMember?.companyId?.trim();
    if (!companyId) return;

    (async () => {
      try {
        const list = await fetchTokenOperationsForCompany(companyId);
        setAllRows(list);
      } catch (e) {
        console.error("[useTokenOperation] fetch error:", e);
        setAllRows([]);
      }
    })();
  }, [currentMember?.companyId]);

  // ── Filter state（ブランド・担当者） ─────────────────────────────
  const [brandFilter, setBrandFilter] = useState<string[]>([]);
  const [assigneeFilter, setAssigneeFilter] = useState<string[]>([]);

  // ── Sort state ─────────────────────────────────────────────
  const [activeKey, setActiveKey] = useState<SortKey>(null);
  const [direction, setDirection] = useState<SortDir>(null);

  // ── Filter options（取得データから動的に生成） ─────────────────────
  const { brandOptions, assigneeOptions } = useMemo(
    () => buildOptionsFromTokenOperations(allRows),
    [allRows],
  );

  // ── Build rows (filter → sort) ────────────────────────────
  const rows = useMemo(
    () =>
      filterAndSortTokenOperations(allRows, {
        brandFilter,
        assigneeFilter,
        sortKey: activeKey,
        sortDir: direction,
      }),
    [allRows, brandFilter, assigneeFilter, activeKey, direction],
  );

  // ソート変更
  const handleSortChange = useCallback(
    (key: string | null, dir: SortDir) => {
      setActiveKey((key as SortKey) ?? null);
      setDirection(dir);
    },
    [],
  );

  // フィルタ変更
  const handleBrandFilterChange = useCallback((values: string[]) => {
    setBrandFilter(values);
  }, []);

  const handleAssigneeFilterChange = useCallback((values: string[]) => {
    setAssigneeFilter(values);
  }, []);

  // リセット
  const handleReset = useCallback(() => {
    setBrandFilter([]);
    setAssigneeFilter([]);
    setActiveKey(null);
    setDirection(null);
  }, []);

  // 詳細ページへ遷移（クリックした行の ID を使用）
  const goDetail = useCallback(
    (operationId: string) => {
      navigate(`/operation/${encodeURIComponent(operationId)}`);
    },
    [navigate],
  );

  return {
    brandFilter,
    assigneeFilter,
    brandOptions,
    assigneeOptions,
    activeKey,
    direction,
    rows,
    handleSortChange,
    handleBrandFilterChange,
    handleAssigneeFilterChange,
    handleReset,
    goDetail,
  };
}
