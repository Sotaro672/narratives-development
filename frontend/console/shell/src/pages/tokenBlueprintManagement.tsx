// frontend/console/shell/src/pages/tokenBlueprintManagement.tsx

import React, { type KeyboardEvent } from "react";

import { useTokenBlueprintManagement } from "../features/tokenBlueprint/presentation/hook/useTokenBlueprintManagement";
import List, {
  FilterableTableHeader,
  SortableTableHeader,
} from "../layout/List/List";
import type { TokenBlueprint } from "../shared/types/tokenBlueprint";
import { TableCell, TableRow } from "../shared/ui/table";

export default function TokenBlueprintManagementPage() {
  const {
    rows,
    brandOptions,
    assigneeOptions,
    mintedOptions,
    brandFilter,
    assigneeFilter,
    mintedFilter,
    sortKey,
    sortDir,
    handleChangeBrandFilter,
    handleChangeAssigneeFilter,
    handleChangeMintedFilter,
    handleChangeSort,
    handleReset,
    handleCreate,
    handleRowClick,
    isResetting,
  } = useTokenBlueprintManagement();

  const headers: React.ReactNode[] = [
    "トークン名",

    <FilterableTableHeader
      key="brand"
      label="ブランド"
      options={brandOptions}
      selected={brandFilter}
      onChange={handleChangeBrandFilter}
    />,

    <FilterableTableHeader
      key="assignee"
      label="担当者"
      options={assigneeOptions}
      selected={assigneeFilter}
      onChange={handleChangeAssigneeFilter}
    />,

    <FilterableTableHeader
      key="minted"
      label="ミント"
      options={mintedOptions}
      selected={mintedFilter}
      onChange={handleChangeMintedFilter}
    />,

    <SortableTableHeader
      key="createdAt"
      label="作成日"
      sortKey="createdAt"
      activeKey={sortKey}
      direction={sortDir}
      onChange={handleChangeSort}
    />,

    <SortableTableHeader
      key="updatedAt"
      label="更新日"
      sortKey="updatedAt"
      activeKey={sortKey}
      direction={sortDir}
      onChange={handleChangeSort}
    />,
  ];

  return (
    <div className="p-0">
      <List
        title="トークン設計"
        headerCells={headers}
        showCreateButton
        createLabel="トークン設計を作成"
        showResetButton
        isResetting={isResetting}
        onCreate={handleCreate}
        onReset={handleReset}
      >
        {rows.map((tokenBlueprint: TokenBlueprint) => (
          <TableRow
            key={tokenBlueprint.id}
            role="button"
            tabIndex={0}
            className="cursor-pointer"
            onClick={() => handleRowClick(tokenBlueprint.id)}
            onKeyDown={(event: KeyboardEvent<HTMLTableRowElement>) => {
              if (event.key === "Enter" || event.key === " ") {
                event.preventDefault();
                handleRowClick(tokenBlueprint.id);
              }
            }}
          >
            <TableCell>{tokenBlueprint.name}</TableCell>
            <TableCell>
              {tokenBlueprint.brandName || tokenBlueprint.brandId}
            </TableCell>
            <TableCell>{tokenBlueprint.assigneeName || ""}</TableCell>
            <TableCell>{String(tokenBlueprint.minted)}</TableCell>
            <TableCell>{tokenBlueprint.createdAt}</TableCell>
            <TableCell>{tokenBlueprint.updatedAt}</TableCell>
          </TableRow>
        ))}
      </List>
    </div>
  );
}