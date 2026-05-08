// frontend/console/productBlueprint/src/presentation/pages/productBlueprintManagement.tsx
import List, {
  FilterableTableHeader,
  SortableTableHeader,
} from "../../../../shell/src/layout/List/List";
import { useProductBlueprintManagement } from "../hook/useProductBlueprintManagement";

export default function ProductBlueprintManagement() {
  const {
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
    isResetting,
  } = useProductBlueprintManagement();

  // rows からオプションを動的生成
  const brandOptions = Array.from(
    new Set(rows.map((r) => r.brandName).filter(Boolean)),
  ).map((name) => ({ value: name, label: name }));

  const assigneeOptions = Array.from(
    new Set(rows.map((r) => r.assigneeName).filter(Boolean)),
  ).map((name) => ({ value: name, label: name }));

  // printed のフィルタ選択肢（固定）
  const printedOptions = [
    { value: "未印刷", label: "未印刷" },
    { value: "印刷済み", label: "印刷済み" },
  ];

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
      key="printed"
      label="印刷"
      options={printedOptions}
      selected={printedFilter}
      onChange={handlePrintedFilterChange}
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
      isResetting={isResetting}
      onReset={handleReset}
    >
      {rows.map((r) => (
        <tr
          key={r.id}
          className="cursor-pointer hover:bg-[rgba(0,0,0,0.03)] transition"
          onClick={() => handleRowClick(r)}
        >
          <td>{r.productName}</td>
          <td>{r.brandName}</td>
          <td>{r.assigneeName}</td>
          <td>{r.printed ? "印刷済み" : "未印刷"}</td>
          <td>{r.createdAt}</td>
          <td>{r.updatedAt}</td>
        </tr>
      ))}
    </List>
  );
}