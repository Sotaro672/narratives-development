// frontend/console/permission/src/presentation/hook/usePermissionList.tsx
import { useEffect, useMemo, useState, type ReactNode } from "react";
import { useNavigate } from "react-router-dom";
import { Shield } from "lucide-react";
import { FilterableTableHeader } from "../../../../shell/src/layout/List/List";
import type { Permission } from "../../../../shell/src/shared/types/permission";
import { PermissionRepositoryHTTP } from "../../infrastructure/http/permissionRepositoryHTTP";

type UsePermissionListResult = {
  headers: ReactNode[];
  filteredRows: Permission[];
  goDetail: (permissionId: string) => void;
  handleReset: () => void;
};

// HTTP 経由で backend (/permissions) にアクセスするリポジトリ
const permissionRepo = new PermissionRepositoryHTTP();

export function usePermissionList(): UsePermissionListResult {
  const navigate = useNavigate();

  // backend から取得した権限一覧
  const [rows, setRows] = useState<Permission[]>([]);

  // カテゴリフィルタ
  const [categoryFilter, setCategoryFilter] = useState<string[]>([]);

  // 初回マウント時に backend から一覧取得
  useEffect(() => {
    (async () => {
      try {
        const result = await permissionRepo.list(); // /permissions へ GET
        setRows(result.items);
      } catch (e) {
        console.error("[usePermissionList] failed to load permissions", e);
        setRows([]);
      }
    })();
  }, []);

  // カテゴリ選択肢（取得済み rows からユニークカテゴリを生成）
  const categoryOptions = useMemo(
    () =>
      Array.from(new Set(rows.map((p) => p.category))).map(
        (v): { value: string; label: string } => ({
          value: v,
          label: v,
        }),
      ),
    [rows],
  );

  // フィルタ適用後の権限リスト（クライアント側でカテゴリ絞り込み）
  const filteredRows = useMemo<Permission[]>(() => {
    if (categoryFilter.length === 0) return rows;
    return rows.filter((p) => categoryFilter.includes(p.category));
  }, [rows, categoryFilter]);

  // 行クリック時の遷移関数（idベースで詳細へ）
  const goDetail = (permissionId: string) => {
    navigate(`/permission/${encodeURIComponent(permissionId)}`);
  };

  // リセットボタン押下時
  const handleReset = () => {
    setCategoryFilter([]);
    console.log("権限一覧リセット");
  };

  // テーブルヘッダー（ReactNode 配列）
  const headers: ReactNode[] = [
    <>
      <span className="inline-flex items-center gap-2">
        <Shield size={16} />
        <span>権限名</span>
      </span>
    </>,
    // カテゴリ（Filterable）
    <FilterableTableHeader
      key="category"
      label="カテゴリ"
      options={categoryOptions}
      selected={categoryFilter}
      onChange={setCategoryFilter}
    />,
    "説明",
  ];

  return {
    headers,
    filteredRows,
    goDetail,
    handleReset,
  };
}
