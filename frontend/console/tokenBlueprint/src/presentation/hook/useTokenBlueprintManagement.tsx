// frontend/console/tokenBlueprint/src/presentation/hook/useTokenBlueprintManagement.tsx

import { useCallback, useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import { useAuth } from "../../../../shell/src/auth/presentation/hook/useCurrentMember";
import type { TokenBlueprint } from "../../../../shell/src/shared/types/tokenBlueprint";
import {
  SortKey,
  SortDir,
  fetchTokenBlueprintsForCompany,
  buildOptionsFromTokenBlueprints,
  filterAndSortTokenBlueprints,
} from "../../application/tokenBlueprintManagementService";

export type UseTokenBlueprintManagementResult = {
  rows: TokenBlueprint[];
  brandOptions: { value: string; label: string }[];
  assigneeOptions: { value: string; label: string }[];
  brandFilter: string[];
  assigneeFilter: string[];
  sortKey: SortKey;
  sortDir: SortDir;

  handleChangeBrandFilter: (vals: string[]) => void;
  handleChangeAssigneeFilter: (vals: string[]) => void;
  handleChangeSort: (key: string | null, dir: SortDir) => void;
  handleReset: () => void;
  handleCreate: () => void;
  handleRowClick: (id: string) => void;
};

/**
 * TokenBlueprint Management ページ用ロジック（Hook）
 * - currentMember.companyId に紐づく TokenBlueprint 一覧を service 経由で取得
 * - フィルタ / ソート / 行クリック / 作成ボタン など UI 以外の要素を集約
 */
export function useTokenBlueprintManagement(): UseTokenBlueprintManagementResult {
  const navigate = useNavigate();
  const { currentMember } = useAuth();

  // 一覧データ
  const [rows, setRows] = useState<TokenBlueprint[]>([]);

  // フィルタ状態（brandId / assigneeId ベース）
  const [brandFilter, setBrandFilter] = useState<string[]>([]);
  const [assigneeFilter, setAssigneeFilter] = useState<string[]>([]);

  // ソート状態
  const [sortKey, setSortKey] = useState<SortKey>(null);
  const [sortDir, setSortDir] = useState<SortDir>(null);

  // ─────────────────────────────
  // データ取得: ListByCompanyID usecase を叩く（service に委譲）
  // ─────────────────────────────
  useEffect(() => {
    const companyId = currentMember?.companyId;
    if (!companyId) return;

    (async () => {
      try {
        const result = await fetchTokenBlueprintsForCompany(companyId);
        setRows(result);
      } catch (e) {
        console.error("[useTokenBlueprintManagement] fetch error:", e);
        setRows([]);
      }
    })();
  }, [currentMember?.companyId]);

  // オプション（brandId / assigneeId から算出）: service に委譲
  const { brandOptions, assigneeOptions } = useMemo(
    () => buildOptionsFromTokenBlueprints(rows),
    [rows],
  );

  // フィルタ + ソート適用後の行: service に委譲
  const filteredRows: TokenBlueprint[] = useMemo(
    () =>
      filterAndSortTokenBlueprints(rows, {
        brandFilter,
        assigneeFilter,
        sortKey,
        sortDir,
      }),
    [rows, brandFilter, assigneeFilter, sortKey, sortDir],
  );

  // 行クリックで詳細へ（id を使用）
  const handleRowClick = useCallback(
    (id: string) => {
      navigate(`/tokenBlueprint/${encodeURIComponent(id)}`);
    },
    [navigate],
  );

  const handleCreate = useCallback(() => {
    navigate("/tokenBlueprint/create");
  }, [navigate]);

  const handleReset = useCallback(() => {
    setBrandFilter([]);
    setAssigneeFilter([]);
    setSortKey(null);
    setSortDir(null);
    console.log("トークン設計一覧リセット");
  }, []);

  const handleChangeBrandFilter = useCallback((vals: string[]) => {
    setBrandFilter(vals);
  }, []);

  const handleChangeAssigneeFilter = useCallback((vals: string[]) => {
    setAssigneeFilter(vals);
  }, []);

  const handleChangeSort = useCallback(
    (key: string | null, dir: SortDir) => {
      setSortKey((key as SortKey) ?? null);
      setSortDir(dir);
    },
    [],
  );

  return {
    rows: filteredRows,
    brandOptions,
    assigneeOptions,
    brandFilter,
    assigneeFilter,
    sortKey,
    sortDir,
    handleChangeBrandFilter,
    handleChangeAssigneeFilter,
    handleChangeSort,
    handleReset,
    handleCreate,
    handleRowClick,
  };
}
