// frontend/console/shell/src/pages/productBlueprintReviewManagement.tsx

import type { KeyboardEvent } from "react";

import { useProductBlueprintReviewManagement } from "../features/productBlueprintReview/presentation/hook/useProductBlueprintReviewManagement";
import List, {
  FilterableTableHeader,
} from "../layout/List/List";
import { TableCell, TableRow } from "../shared/ui/table";

import "../styles/productBlueprintReview.css";

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
        <TableRow
          key={Row.ID || Row.ProductBlueprintID}
          className="pbrm-row"
          role="button"
          tabIndex={0}
          onClick={() => HandleRowClick(Row)}
          onKeyDown={(event: KeyboardEvent<HTMLTableRowElement>) => {
            if (event.key === "Enter" || event.key === " ") {
              event.preventDefault();
              HandleRowClick(Row);
            }
          }}
        >
          <TableCell>{Row.ProductName}</TableCell>
          <TableCell>{Row.Rating1Count}</TableCell>
          <TableCell>{Row.Rating2Count}</TableCell>
          <TableCell>{Row.Rating3Count}</TableCell>
          <TableCell>{Row.Rating4Count}</TableCell>
          <TableCell>{Row.Rating5Count}</TableCell>
          <TableCell>{Row.BrandName}</TableCell>
          <TableCell>{Row.AssigneeName}</TableCell>
        </TableRow>
      ))}
    </List>
  );
}