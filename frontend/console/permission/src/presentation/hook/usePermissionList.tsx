// frontend/console/permission/src/presentation/hook/usePermissionList.tsx
import { useMemo, useState, type ReactNode } from "react";
import { useNavigate } from "react-router-dom";
import { Shield } from "lucide-react";
import { FilterableTableHeader } from "../../../../shell/src/layout/List/List";
import { ALL_PERMISSIONS } from "../../infrastructure/mockdata/mockdata";
import type { Permission } from "../../../../shell/src/shared/types/permission";

type UsePermissionListResult = {
  headers: ReactNode[];
  filteredRows: Permission[];
  goDetail: (permissionId: string) => void;
  handleReset: () => void;
};

export function usePermissionList(): UsePermissionListResult {
  const navigate = useNavigate();

  // カテゴリフィルタ
  const [categoryFilter, setCategoryFilter] = useState<string[]>([]);

  // カテゴリ選択肢
  const categoryOptions = useMemo(
    () =>
      Array.from(new Set(ALL_PERMISSIONS.map((p) => p.category))).map(
        (v): { value: string; label: string } => ({
          value: v,
          label: v,
        }),
      ),
    [],
  );

  // フィルタ適用後の権限リスト
  const filteredRows = useMemo<Permission[]>(() => {
    if (categoryFilter.length === 0) return ALL_PERMISSIONS;
    return ALL_PERMISSIONS.filter((p) =>
      categoryFilter.includes(p.category),
    );
  }, [categoryFilter]);

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
