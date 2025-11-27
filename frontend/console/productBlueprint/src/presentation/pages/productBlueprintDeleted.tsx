// frontend/console/productBlueprint/src/presentation/pages/productBlueprintDeleted.tsx

import { useNavigate } from "react-router-dom";
import List, {
  FilterableTableHeader,
  SortableTableHeader,
} from "../../../../shell/src/layout/List/List";
import { useProductBlueprintDeleted } from "../hook/useProductBlueprintDeleted";

/**
 * 論理削除済み ProductBlueprint 一覧ページ
 * - ヘッダー構成: プロダクト / ブランド / 担当者 / 削除日 / 期限日
 * - ブランド / 担当者: FilterableTableHeader
 * - 削除日 / 期限日: SortableTableHeader
 */
export default function ProductBlueprintDeleted() {
  const navigate = useNavigate();

  const {
    rows,
    brandFilter,
    assigneeFilter,
    handleBrandFilterChange,
    handleAssigneeFilterChange,
    handleSortChange,
    handleRowClick,
    handleReset,
  } = useProductBlueprintDeleted();

  // キャンセルボタン（×）押下時: 通常の一覧に戻る
  const handleCancel = () => {
    navigate("/productBlueprint");
  };

  // rows からオプションを動的生成
  const brandOptions = Array.from(
    new Set(rows.map((r) => r.brandName).filter(Boolean)),
  ).map((name) => ({ value: name, label: name }));

  const assigneeOptions = Array.from(
    new Set(rows.map((r) => r.assigneeName).filter(Boolean)),
  ).map((name) => ({ value: name, label: name }));

  const headers = [
    "プロダクト",
    <FilterableTableHeader
      key="brand"
      label="ブランド"
      options={brandOptions}
      selected={brandFilter}
      onChange={handleBrandFilterChange}
    />,
    <FilterableTableHeader
      key="assignee"
      label="担当者"
      options={assigneeOptions}
      selected={assigneeFilter}
      onChange={handleAssigneeFilterChange}
    />,
    <SortableTableHeader
      key="deletedAt"
      label="削除日"
      sortKey="deletedAt"
      activeKey={null}
      direction={null}
      onChange={handleSortChange}
    />,
    <SortableTableHeader
      key="expireAt"
      label="期限日"
      sortKey="expireAt"
      activeKey={null}
      direction={null}
      onChange={handleSortChange}
    />,
  ];

  return (
    <List
      title="削除済み商品設計"
      headerCells={headers}
      // 削除済み一覧なので作成ボタンは表示しない（デフォルト false のまま）
      showResetButton
      onReset={handleReset}
      // ★ キャンセルボタン（×）をリフレッシュボタンの右隣りに表示
      showCancelButton
      onCancel={handleCancel}
    >
      {rows.map((r) => (
        <tr
          key={r.id}
          className="cursor-pointer hover:bg-[rgba(0,0,0,0.03)] transition"
          onClick={() => handleRowClick(r)}
        >
          <td>{r.productName}</td>
          <td>
            <span className="lp-brand-pill">{r.brandName}</span>
          </td>
          <td>{r.assigneeName}</td>
          <td>{r.deletedAt}</td>
          <td>{r.expireAt}</td>
        </tr>
      ))}
    </List>
  );
}
