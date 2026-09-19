// frontend/console/shell/src/pages/tokenBlueprintReviewManagement.tsx

import React, { type KeyboardEvent } from "react";

import { useTokenBlueprintReviewManagement } from "../features/tokenBlueprintReview/presentation/hook/use_tokenBlueprintReviewManagement";
import List, {
  FilterableTableHeader,
  SortableTableHeader,
} from "../layout/List/List";
import type { TokenBlueprintReviewAggregate } from "../shared/types/tokenBlueprintReview";
import { TableCell, TableRow } from "../shared/ui/table";
import { safeDateTimeLabelJa } from "../shared/util/dateJa";

export default function TokenBlueprintReviewManagementPage() {
  const {
    rows,
    brandOptions,
    brandFilter,
    handleChangeBrandFilter,
    sortKey,
    sortDir,
    handleChangeSort,
    handleReset,
    handleRowClick,
    isResetting,
  } = useTokenBlueprintReviewManagement();

  const headers: React.ReactNode[] = [
    "トークン名",
    <FilterableTableHeader
      key="brandName"
      label="ブランド名"
      options={brandOptions}
      selected={brandFilter}
      onChange={handleChangeBrandFilter}
    />,
    "レビュー数",
    "高評価",
    "低評価",
    <SortableTableHeader
      key="createdAt"
      label="作成日時"
      sortKey="createdAt"
      activeKey={sortKey}
      direction={sortDir}
      onChange={handleChangeSort}
    />,
    <SortableTableHeader
      key="updatedAt"
      label="更新日時"
      sortKey="updatedAt"
      activeKey={sortKey}
      direction={sortDir}
      onChange={handleChangeSort}
    />,
  ];

  return (
    <div className="p-0">
      <List
        title="トークン設計レビュー"
        headerCells={headers}
        showResetButton
        isResetting={isResetting}
        onReset={handleReset}
      >
        {rows.map((t: TokenBlueprintReviewAggregate) => (
          <TableRow
            key={t.tokenBlueprintId}
            role="button"
            tabIndex={0}
            className="cursor-pointer"
            onClick={() => handleRowClick(t.tokenBlueprintId)}
            onKeyDown={(e: KeyboardEvent<HTMLTableRowElement>) => {
              if (e.key === "Enter" || e.key === " ") {
                e.preventDefault();
                handleRowClick(t.tokenBlueprintId);
              }
            }}
          >
            <TableCell>{t.tokenBlueprintName ?? "-"}</TableCell>
            <TableCell>{t.brandName ?? "-"}</TableCell>
            <TableCell>{t.topLevelCommentCount}</TableCell>
            <TableCell>{t.likeCount}</TableCell>
            <TableCell>{t.dislikeCount}</TableCell>
            <TableCell>
              {safeDateTimeLabelJa(t.createdAt, t.createdAt || "-")}
            </TableCell>
            <TableCell>
              {safeDateTimeLabelJa(t.updatedAt, t.updatedAt || "-")}
            </TableCell>
          </TableRow>
        ))}
      </List>
    </div>
  );
}