// frontend/console/productBlueprintReview/src/presentation/pages/productBlueprintReviewManagement.tsx

import List, { FilterableTableHeader } from "../../../../shell/src/layout/List/List";
import { useProductBlueprintReviewManagement } from "../hook/useProductBlueprintReviewManagement";

type Option = { Value: string; Label: string };

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

  // ✅ Name 解決済みのみを候補にする（ID fallback はしない）
  const BrandOptions: Option[] = Array.from(
    new Set(Rows.map((R: any) => String(R.BrandName ?? "")).filter(Boolean)),
  ).map((Name) => ({ Value: String(Name), Label: String(Name) }));

  const AssigneeOptions: Option[] = Array.from(
    new Set(Rows.map((R: any) => String(R.AssigneeName ?? "")).filter(Boolean)),
  ).map((Name) => ({ Value: String(Name), Label: String(Name) }));

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
      options={BrandOptions.map((O) => ({ value: O.Value, label: O.Label }))}
      selected={BrandFilter}
      onChange={HandleBrandFilterChange}
    />,
    <FilterableTableHeader
      key="assignee"
      label="担当者"
      options={AssigneeOptions.map((O) => ({ value: O.Value, label: O.Label }))}
      selected={AssigneeFilter}
      onChange={HandleAssigneeFilterChange}
    />,
  ];

  const ToCount = (V: any): number => {
    const N = typeof V === "number" ? V : Number(V);
    return Number.isFinite(N) ? N : 0;
  };

  // ✅ Name 解決済みのみを表示（ID fallback はしない）
  const Get = (R: any) => ({
    ID: R.ID ?? R.ProductBlueprintID ?? "",
    ProductName: R.ProductName ?? "",
    BrandName: R.BrandName ?? "",
    AssigneeName: R.AssigneeName ?? "",
    Rating1Count: R.Rating1Count ?? 0,
    Rating2Count: R.Rating2Count ?? 0,
    Rating3Count: R.Rating3Count ?? 0,
    Rating4Count: R.Rating4Count ?? 0,
    Rating5Count: R.Rating5Count ?? 0,
  });

  return (
    <List
      title="商品レビュー"
      headerCells={Headers}
      showResetButton
      isResetting={IsResetting}
      onReset={HandleReset}
    >
      {Rows.map((R: any) => {
        const V = Get(R);
        return (
          <tr
            key={String(V.ID)}
            className="cursor-pointer hover:bg-[rgba(0,0,0,0.03)] transition"
            onClick={() => HandleRowClick(R)}
          >
            <td>{V.ProductName}</td>

            {/* 各レートの投稿数（1〜5） */}
            <td>{ToCount(V.Rating1Count)}</td>
            <td>{ToCount(V.Rating2Count)}</td>
            <td>{ToCount(V.Rating3Count)}</td>
            <td>{ToCount(V.Rating4Count)}</td>
            <td>{ToCount(V.Rating5Count)}</td>

            <td>{V.BrandName}</td>
            <td>{V.AssigneeName}</td>
          </tr>
        );
      })}
    </List>
  );
}