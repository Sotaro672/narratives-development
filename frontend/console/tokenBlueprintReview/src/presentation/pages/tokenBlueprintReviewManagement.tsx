// frontend/console/tokenBlueprintReview/src/presentation/pages/tokenBlueprintReviewManagement.tsx

import React from "react";
import List, {
  SortableTableHeader,
  FilterableTableHeader,
} from "../../../../shell/src/layout/List/List";
import { safeDateTimeLabelJa } from "../../../../shell/src/shared/util/dateJa";
import { useTokenBlueprintReviewManagement } from "../hook/use_tokenBlueprintReviewManagement";
import type { TokenBlueprintReviewAggregate } from "../../domain/entity";

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
          <tr
            key={t.tokenBlueprintId}
            role="button"
            tabIndex={0}
            className="cursor-pointer hover:bg-slate-50 transition-colors"
            onClick={() => handleRowClick(t.tokenBlueprintId)}
            onKeyDown={(e) => {
              if (e.key === "Enter" || e.key === " ") {
                e.preventDefault();
                handleRowClick(t.tokenBlueprintId);
              }
            }}
          >
            <td>{t.tokenBlueprintName ?? "-"}</td>
            <td>{t.brandName ?? "-"}</td>
            <td>{t.topLevelCommentCount}</td>
            <td>{t.likeCount}</td>
            <td>{t.dislikeCount}</td>
            <td>{safeDateTimeLabelJa(t.createdAt, t.createdAt || "-")}</td>
            <td>{safeDateTimeLabelJa(t.updatedAt, t.updatedAt || "-")}</td>
          </tr>
        ))}
      </List>
    </div>
  );
}