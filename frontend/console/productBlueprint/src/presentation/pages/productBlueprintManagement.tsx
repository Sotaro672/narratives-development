// frontend/productBlueprint/src/presentation/pages/productBlueprintManagement.tsx

import List, {
  FilterableTableHeader,
  SortableTableHeader,
} from "../../../../shell/src/layout/List/List";
import { useProductBlueprintManagement } from "../hook/useProductBlueprintManagement";

export default function ProductBlueprintManagement() {
  const {
    rows,
    brandFilter,
    handleBrandFilterChange,
    handleSortChange,
    handleRowClick,
    handleCreate,
    handleReset,
  } = useProductBlueprintManagement();

  // ヘッダー定義（UI / スタイル側にのみ責務を残す）
  const headers = [
    "プロダクト",
    <FilterableTableHeader
      key="brand"
      label="ブランド"
      options={[
        { value: "LUMINA Fashion", label: "LUMINA Fashion" },
        { value: "NEXUS Street", label: "NEXUS Street" },
      ]}
      selected={brandFilter}
      onChange={handleBrandFilterChange}
    />,
    "担当者",
    "タグ種別",
    <SortableTableHeader
      key="createdAt"
      label="作成日"
      sortKey="createdAt"
      activeKey={null} // activeKey / direction は hook 内で管理＆判定させるため null を渡す
      direction={null}
      onChange={handleSortChange}
    />,
    <SortableTableHeader
      key="lastModifiedAt"
      label="最終更新日"
      sortKey="lastModifiedAt"
      activeKey={null}
      direction={null}
      onChange={handleSortChange}
    />,
  ];

  return (
    <List
      title="商品設計"
      headerCells={headers}
      showCreateButton
      createLabel="商品設計を作成"
      onCreate={handleCreate}
      showResetButton
      onReset={handleReset}
    >
      {rows.map((r) => (
        <tr
          key={r.id}
          className="cursor-pointer hover:bg-[rgba(0,0,0,0.03)] transition"
          onClick={() => handleRowClick(r)}
        >
          <td>{r.productName}</td>
          <td>
            <span className="lp-brand-pill">{r.brandLabel}</span>
          </td>
          <td>{r.assigneeLabel}</td>
          <td>{r.tagLabel}</td>
          <td>{r.createdAt}</td>
          <td>{r.lastModifiedAt}</td>
        </tr>
      ))}
    </List>
  );
}
