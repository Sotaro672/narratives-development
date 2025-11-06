// frontend/permission/src/pages/permissionList.tsx
import { useMemo, useState } from "react";
import List, { FilterableTableHeader } from "../../../shell/src/layout/List/List";
import { Shield } from "lucide-react";

type Permission = {
  name: string;
  category: string;
  description: string;
};

const ALL_PERMISSIONS: Permission[] = [
  { name: "wallet.view", category: "wallet", description: "ウォレット情報の閲覧" },
  { name: "wallet.edit", category: "wallet", description: "ウォレット設定の編集" },
  { name: "inquiry.view", category: "inquiry", description: "問い合わせの閲覧" },
  { name: "inquiry.manage", category: "inquiry", description: "問い合わせ対応・管理" },
  { name: "organization.admin", category: "organization", description: "組織の完全な管理権限" },
  { name: "brand.create", category: "brand", description: "ブランドの作成" },
  { name: "brand.edit", category: "brand", description: "ブランド情報の編集" },
  { name: "brand.delete", category: "brand", description: "ブランドの削除" },
  { name: "token.create", category: "token", description: "トークンの作成" },
  { name: "token.manage", category: "token", description: "トークンの管理・配布" },
  { name: "listing.view", category: "listing", description: "出品情報の閲覧" },
  { name: "order.manage", category: "order", description: "注文の管理" },
];

export default function PermissionList() {
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
        {filteredRows.map((p) => (
          <tr key={p.name}>
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
