import { useMemo, useState } from "react";
import { useNavigate } from "react-router-dom"; // ← 追加
import List, { FilterableTableHeader } from "../../../../shell/src/layout/List/List";
import { Shield } from "lucide-react";
import { ALL_PERMISSIONS, type Permission } from "../../../mockdata";

export default function PermissionList() {
  const navigate = useNavigate(); // ← 追加

  // カテゴリフィルタ
  const [categoryFilter, setCategoryFilter] = useState<string[]>([]);

  const categoryOptions = useMemo(
    () =>
      Array.from(new Set(ALL_PERMISSIONS.map((p) => p.category))).map((v) => ({
        value: v,
        label: v,
      })),
    []
  );

  // フィルタ適用
  const filteredRows = useMemo(() => {
    if (categoryFilter.length === 0) return ALL_PERMISSIONS;
    return ALL_PERMISSIONS.filter((p) => categoryFilter.includes(p.category));
  }, [categoryFilter]);

  // 行クリック時の遷移関数
  const goDetail = (permissionId: string) => {
    navigate(`/permission/${encodeURIComponent(permissionId)}`);
  };

  const headers = [
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

  return (
    <div className="p-0">
      <List
        title="権限管理"
        headerCells={headers}
        showCreateButton={false}
        showResetButton
        onReset={() => {
          setCategoryFilter([]);
          console.log("権限一覧リセット");
        }}
      >
        {filteredRows.map((p: Permission) => (
          <tr
            key={p.name}
            role="button"
            tabIndex={0}
            className="cursor-pointer hover:bg-slate-50 transition-colors"
            onClick={() => goDetail(p.name)} // ← 行クリックで遷移
            onKeyDown={(e) => {
              if (e.key === "Enter" || e.key === " ") {
                e.preventDefault();
                goDetail(p.name);
              }
            }}
          >
            <td>{p.name}</td>
            <td>
              <span className="lp-brand-pill">{p.category}</span>
            </td>
            <td>{p.description}</td>
          </tr>
        ))}
      </List>
    </div>
  );
}
