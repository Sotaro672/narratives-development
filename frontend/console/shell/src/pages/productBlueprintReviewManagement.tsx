// frontend/console/shell/src/pages/productBlueprintReviewManagement.tsx

import List, {
  FilterableTableHeader,
} from "../layout/List/List";

import { useProductBlueprintReviewManagement } from "../features/productBlueprintReview/presentation/hook/useProductBlueprintReviewManagement";

type FilterOption = {
  value: string;
  label: string;
};

function BuildFilterOptions(
  Values: string[],
): FilterOption[] {
  return Array.from(
    new Set(Values.filter(Boolean)),
  ).map((Value) => ({
    value: Value,
    label: Value,
  }));
}

export default function ProductBlueprintReviewManagement() {
  const {
    Rows,
    BrandFilter,
    AssigneeFilter,
    HandleBrandFilterChange,
    HandleAssigneeFilterChange,
    HandleRowClick,
    HandleReset,
    IsResetting,
  } = useProductBlueprintReviewManagement();

  const BrandOptions = BuildFilterOptions(
    Rows.map((Row) => Row.BrandName),
  );

  const AssigneeOptions = BuildFilterOptions(
    Rows.map((Row) => Row.AssigneeName),
  );

  const Headers = [
    "商品名",
    "★1",
    "★2",
    "★3",
    "★4",
    "★5",
    <FilterableTableHeader
      key="brand"
      label="ブランド"
      options={BrandOptions}
      selected={BrandFilter}
      onChange={HandleBrandFilterChange}
    />,
    <FilterableTableHeader
      key="assignee"
      label="担当者"
      options={AssigneeOptions}
      selected={AssigneeFilter}
      onChange={HandleAssigneeFilterChange}
    />,
  ];

  return (
    <List
      title="商品レビュー"
      headerCells={Headers}
      showResetButton
      isResetting={IsResetting}
      onReset={HandleReset}
    >
      {Rows.map((Row) => (
        <tr
          key={Row.ID || Row.ProductBlueprintID}
          className="cursor-pointer transition hover:bg-[rgba(0,0,0,0.03)]"
          onClick={() => HandleRowClick(Row)}
        >
          <td>{Row.ProductName}</td>

          <td>{Row.Rating1Count}</td>
          <td>{Row.Rating2Count}</td>
          <td>{Row.Rating3Count}</td>
          <td>{Row.Rating4Count}</td>
          <td>{Row.Rating5Count}</td>

          <td>{Row.BrandName}</td>
          <td>{Row.AssigneeName}</td>
        </tr>
      ))}
    </List>
  );
}