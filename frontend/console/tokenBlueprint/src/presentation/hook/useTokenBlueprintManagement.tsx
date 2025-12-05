// frontend/console/tokenBlueprint/src/presentation/hook/useTokenBlueprintManagement.tsx

import { useCallback, useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import { TOKEN_BLUEPRINTS } from "../../infrastructure/mockdata/tokenBlueprint_mockdata";
import type { TokenBlueprint } from "../../../../shell/src/shared/types/tokenBlueprint";

/** ISO8601 → timestamp（不正値は 0 扱い） */
const toTs = (iso: string): number => {
  if (!iso) return 0;
  const t = Date.parse(iso);
  return Number.isNaN(t) ? 0 : t;
};

type SortKey = "createdAt" | null;
type SortDir = "asc" | "desc" | null;

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
 * TokenBlueprint Management ページ用ロジック
 * - フィルタ / ソート / 行クリック / 作成ボタン など UI 以外の要素を集約
 */
export function useTokenBlueprintManagement(): UseTokenBlueprintManagementResult {
  const navigate = useNavigate();

  // フィルタ状態（brandId / assigneeId ベース）
  const [brandFilter, setBrandFilter] = useState<string[]>([]);
  const [assigneeFilter, setAssigneeFilter] = useState<string[]>([]);

  // ソート状態
  const [sortKey, setSortKey] = useState<SortKey>(null);
  const [sortDir, setSortDir] = useState<SortDir>(null);

  // オプション（brandId / assigneeId から算出）
  const brandOptions = useMemo(
    () =>
      Array.from(new Set(TOKEN_BLUEPRINTS.map((r) => r.brandId))).map(
        (v) => ({
          value: v,
          label: v,
        }),
      ),
    [],
  );

  const assigneeOptions = useMemo(
    () =>
      Array.from(new Set(TOKEN_BLUEPRINTS.map((r) => r.assigneeId))).map(
        (v) => ({
          value: v,
          label: v,
        }),
      ),
    [],
  );

  // フィルタ + ソート適用後の行
  const rows: TokenBlueprint[] = useMemo(() => {
    let data = TOKEN_BLUEPRINTS.filter(
      (r) =>
        (brandFilter.length === 0 || brandFilter.includes(r.brandId)) &&
        (assigneeFilter.length === 0 ||
          assigneeFilter.includes(r.assigneeId)),
    );

    if (sortKey && sortDir) {
      data = [...data].sort((a, b) => {
        const av = toTs(a[sortKey]);
        const bv = toTs(b[sortKey]);
        return sortDir === "asc" ? av - bv : bv - av;
      });
    }

    return data;
  }, [brandFilter, assigneeFilter, sortKey, sortDir]);

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
    rows,
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
