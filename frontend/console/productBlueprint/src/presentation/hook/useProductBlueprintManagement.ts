// frontend/console/productBlueprint/src/presentation/hook/useProductBlueprintManagement.ts

import { useMemo, useState, useCallback, useEffect } from "react";
import { useNavigate } from "react-router-dom";

// ★ HTTP Repository から一覧を取得
import { listProductBlueprintsHTTP } from "../../infrastructure/repository/productBlueprintRepositoryHTTP";

// UI 一覧表示用の行モデル
export type UiRow = {
  id: string;
  productName: string;
  brandLabel: string;
  assigneeLabel: string;
  tagLabel: string;
  createdAt: string;
  lastModifiedAt: string;
};

// ★ backend /product-blueprints のレスポンス想定
//   （すべて optional にしておいて UI 用に詰め直す）
type RawProductBlueprintListRow = {
  id?: string;
  productName?: string;
  brandLabel?: string;
  assigneeLabel?: string;
  tagLabel?: string;
  createdAt?: string;
  lastModifiedAt?: string;
};

// "YYYY/MM/DD" → timestamp（ソート用）
const toTs = (yyyyMd: string) => {
  if (!yyyyMd) return 0;
  const [y, m, d] = yyyyMd.split("/").map((v) => parseInt(v, 10));
  return new Date(y, (m || 1) - 1, d || 1).getTime();
};

type SortKey = "createdAt" | "lastModifiedAt" | null;

export interface UseProductBlueprintManagementResult {
  rows: UiRow[];
  brandFilter: string[];
  handleBrandFilterChange: (values: string[]) => void;
  handleSortChange: (
    key: string | null,
    dir: "asc" | "desc" | null
  ) => void;
  handleRowClick: (row: UiRow) => void;
  handleCreate: () => void;
  handleReset: () => void;
}

/**
 * 商品設計一覧画面のロジック
 * - mockdata を廃止し、backend の /product-blueprints を参照
 * - フィルタ・ソート・画面遷移のみ担当
 */
export function useProductBlueprintManagement(): UseProductBlueprintManagementResult {
  const navigate = useNavigate();

  // 一覧データ
  const [allRows, setAllRows] = useState<UiRow[]>([]);

  // フィルタ & ソート状態
  const [brandFilter, setBrandFilter] = useState<string[]>([]);
  const [sortedKey, setSortedKey] = useState<SortKey>(null);
  const [sortedDir, setSortedDir] = useState<"asc" | "desc" | null>(null);

  // ---------------------------
  // 初回ロード: backend から取得
  // ---------------------------
  useEffect(() => {
    (async () => {
      try {
        const list = await listProductBlueprintsHTTP(); // ★ backend API 呼び出し

        // 必要な UI 行構造に整形
        const uiRows: UiRow[] = (list as RawProductBlueprintListRow[]).map(
          (pb) => ({
            id: pb.id ?? "",
            productName: pb.productName ?? "",
            brandLabel: pb.brandLabel ?? "",
            assigneeLabel: pb.assigneeLabel ?? "",
            tagLabel: pb.tagLabel ?? "",
            createdAt: pb.createdAt ?? "",
            lastModifiedAt: pb.lastModifiedAt ?? "",
          }),
        );

        setAllRows(uiRows);
      } catch (err) {
        console.error("[useProductBlueprintManagement] list load failed", err);
        setAllRows([]);
      }
    })();
  }, []);

  // ---------------------------
  // フィルタ・ソート適用
  // ---------------------------
  const rows: UiRow[] = useMemo(() => {
    let work = allRows;

    // ブランド絞り込み
    if (brandFilter.length > 0) {
      work = work.filter((r) => brandFilter.includes(r.brandLabel));
    }

    // ソート適用
    if (sortedKey && sortedDir) {
      work = [...work].sort((a, b) => {
        const av = toTs(a[sortedKey]);
        const bv = toTs(b[sortedKey]);
        return sortedDir === "asc" ? av - bv : bv - av;
      });
    }

    return work;
  }, [allRows, brandFilter, sortedKey, sortedDir]);

  // ---------------------------
  // ハンドラ群
  // ---------------------------
  const handleBrandFilterChange = useCallback((values: string[]) => {
    setBrandFilter(values);
  }, []);

  const handleSortChange = useCallback(
    (key: string | null, dir: "asc" | "desc" | null) => {
      setSortedKey((key as SortKey) ?? null);
      setSortedDir(dir);
    },
    [],
  );

  const handleRowClick = useCallback(
    (row: UiRow) => {
      navigate(`/productBlueprint/detail/${encodeURIComponent(row.id)}`);
    },
    [navigate],
  );

  const handleCreate = useCallback(() => {
    navigate("/productBlueprint/create");
  }, [navigate]);

  const handleReset = useCallback(() => {
    setBrandFilter([]);
    setSortedKey(null);
    setSortedDir(null);
  }, []);

  return {
    rows,
    brandFilter,
    handleBrandFilterChange,
    handleSortChange,
    handleRowClick,
    handleCreate,
    handleReset,
  };
}
