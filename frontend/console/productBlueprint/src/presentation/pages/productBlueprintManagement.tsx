import List, {
  FilterableTableHeader,
  SortableTableHeader,
} from "../../../../shell/src/layout/List/List";
import { useProductBlueprintManagement } from "../hook/useProductBlueprintManagement";
import { useNavigate } from "react-router-dom";

export default function ProductBlueprintManagement() {
  const navigate = useNavigate();

  const {
    rows,
    brandFilter,
    assigneeFilter,
    tagFilter,
    handleBrandFilterChange,
    handleAssigneeFilterChange,
    handleTagFilterChange,
    handleSortChange,
    handleRowClick,
    handleCreate,
    handleReset,
  } = useProductBlueprintManagement();

  // -----------------------------
  // ゴミ箱ボタン押下 → 削除済み一覧へ遷移
  // -----------------------------
  const handleTrash = () => {
    navigate("/productBlueprint/deleted");
  };

  // rows からオプションを動的生成
  const brandOptions = Array.from(
    new Set(rows.map((r) => r.brandName).filter(Boolean)),
  ).map((name) => ({ value: name, label: name }));

  const assigneeOptions = Array.from(
    new Set(rows.map((r) => r.assigneeName).filter(Boolean)),
  ).map((name) => ({ value: name, label: name }));

  const tagOptions = Array.from(
    new Set(rows.map((r) => r.productIdTag).filter(Boolean)),
  ).map((tag) => ({ value: tag, label: tag }));

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
    <FilterableTableHeader
      key="tag"
      label="タグ種別"
      options={tagOptions}
      selected={tagFilter}
      onChange={handleTagFilterChange}
    />,
    <SortableTableHeader
      key="createdAt"
      label="作成日"
      sortKey="createdAt"
      activeKey={null}
      direction={null}
      onChange={handleSortChange}
    />,
    <SortableTableHeader
      key="updatedAt"
      label="最終更新日"
      sortKey="updatedAt"
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
      // ★ ゴミ箱ボタン（削除済み一覧へ）
      showTrashButton
      onTrash={handleTrash}
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
          <td>{r.productIdTag}</td>
          <td>{r.createdAt}</td>
          <td>{r.updatedAt}</td>
        </tr>
      ))}
    </List>
  );
}
