// frontend/console/productBlueprint/src/presentation/hook/useProductBlueprintManagement.ts

import { useMemo, useState, useCallback, useEffect } from "react";
import { useNavigate } from "react-router-dom";

import {
  fetchProductBlueprintManagementRows,
  filterAndSortProductBlueprintRows,
  type UiRow,
  type ProductBlueprintSortKey,
  type SortDirection,
} from "../../application/productBlueprintManagementService";

import { safeDateTimeLabelJa } from "../../../../shell/src/shared/util/dateJa";

export interface UseProductBlueprintManagementResult {
  rows: UiRow[];

  // フィルタ状態
  brandFilter: string[];
  assigneeFilter: string[];
  printedFilter: string[];

  // フィルタ変更ハンドラ
  handleBrandFilterChange: (values: string[]) => void;
  handleAssigneeFilterChange: (values: string[]) => void;
  handlePrintedFilterChange: (values: string[]) => void;

  // ソート変更ハンドラ
  handleSortChange: (key: string | null, dir: "asc" | "desc" | null) => void;

  // 行クリック & 画面操作
  handleRowClick: (row: UiRow) => void;
  handleCreate: () => void;
  handleReset: () => void;

  // ゴミ箱ボタン押下時のハンドラ（削除一覧ページへ遷移）
  handleTrash: () => void;
}

/**
 * dateJa.ts を使って安全に整形し、表示は "yyyy/MM/dd HH:mm" に揃える。
 * - dateJa は "yyyy/MM/dd HH:mm:ss" を返すため、UI表示では秒を落とす
 * - parse できない場合は dateJa が生文字返しする（既存互換）
 */
function formatDateTimeYYYYMMDDHHmm(v: string | null | undefined): string {
  const label = safeDateTimeLabelJa(v, "");
  if (!label) return "";

  // "yyyy/MM/dd HH:mm:ss" -> "yyyy/MM/dd HH:mm"
  const m = label.match(/^(\d{4}\/\d{2}\/\d{2} \d{2}:\d{2})(?::\d{2})?$/);
  if (m) return m[1];

  // 想定外フォーマットはそのまま返す（dateJa の方針に合わせる）
  return label;
}

/**
 * 商品設計一覧画面のロジック
 * - backend の /product-blueprints を参照
 * - フィルタ・ソート・画面遷移のみ担当
 */
export function useProductBlueprintManagement(): UseProductBlueprintManagementResult {
  const navigate = useNavigate();

  // 一覧データ
  const [allRows, setAllRows] = useState<UiRow[]>([]);

  // フィルタ & ソート状態
  const [brandFilter, setBrandFilter] = useState<string[]>([]);
  const [assigneeFilter, setAssigneeFilter] = useState<string[]>([]);
  const [printedFilter, setPrintedFilter] = useState<string[]>([]);
  const [sortedKey, setSortedKey] = useState<ProductBlueprintSortKey>(null);
  const [sortedDir, setSortedDir] = useState<SortDirection>(null);

  // ---------------------------
  // 一覧取得処理（初回 & リフレッシュ共通）
  // ---------------------------
  const load = useCallback(async () => {
    try {
      const uiRows = await fetchProductBlueprintManagementRows();
      setAllRows(uiRows);
    } catch {
      setAllRows([]);
    }
  }, []);

  // ---------------------------
  // 初回ロード: backend から取得
  // ---------------------------
  useEffect(() => {
    void load();
  }, [load]);

  // ---------------------------
  // フィルタ・ソート適用
  // ---------------------------
  const filteredSortedRows: UiRow[] = useMemo(
    () =>
      filterAndSortProductBlueprintRows({
        allRows,
        brandFilter,
        assigneeFilter,
        printedFilter,
        sortedKey,
        sortedDir,
      }),
    [allRows, brandFilter, assigneeFilter, printedFilter, sortedKey, sortedDir],
  );

  // 表示用に createdAt / updatedAt を yyyy/MM/dd HH:mm に整形して返す
  // - UiRow のキーを上書きするだけ（型を増やさない）
  const rows: UiRow[] = useMemo(() => {
    return filteredSortedRows.map((r) => ({
      ...r,
      createdAt: formatDateTimeYYYYMMDDHHmm(r.createdAt),
      updatedAt: formatDateTimeYYYYMMDDHHmm(r.updatedAt),
    }));
  }, [filteredSortedRows]);

  // ---------------------------
  // ハンドラ群
  // ---------------------------
  const handleBrandFilterChange = useCallback((values: string[]) => {
    setBrandFilter(values);
  }, []);

  const handleAssigneeFilterChange = useCallback((values: string[]) => {
    setAssigneeFilter(values);
  }, []);

  const handlePrintedFilterChange = useCallback((values: string[]) => {
    setPrintedFilter(values);
  }, []);

  const handleSortChange = useCallback(
    (key: string | null, dir: "asc" | "desc" | null) => {
      setSortedKey((key as ProductBlueprintSortKey) ?? null);
      setSortedDir(dir as SortDirection);
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
    // フィルタ・ソート状態をリセット
    setBrandFilter([]);
    setAssigneeFilter([]);
    setPrintedFilter([]);
    setSortedKey(null);
    setSortedDir(null);

    // 一覧を再取得（リフレッシュ）
    void load();
  }, [load]);

  // ゴミ箱ボタン押下 → 削除済み一覧ページへ遷移
  const handleTrash = useCallback(() => {
    navigate("/productBlueprint/deleted");
  }, [navigate]);

  return {
    rows,
    brandFilter,
    assigneeFilter,
    printedFilter,
    handleBrandFilterChange,
    handleAssigneeFilterChange,
    handlePrintedFilterChange,
    handleSortChange,
    handleRowClick,
    handleCreate,
    handleReset,
    handleTrash,
  };
}
