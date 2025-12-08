// frontend/console/inventory/src/presentation/pages/inventoryManagement.tsx

import React from "react";
import List from "../../../../shell/src/layout/List/List";
import "../styles/inventory.css";

import {
  useInventoryManagement,
  type InventorySortKey as SortKey,
} from "../hook/useInventoryManagement";
import { buildInventoryHeaders } from "../../application/inventoryManagementService";

/** 在庫管理ページ（スタイル＋レイアウト中心） */
export default function InventoryManagementPage() {
  const {
    rows,
    options: { productOptions, brandOptions, assigneeOptions },
    state: {
      productFilter,
      brandFilter,
      assigneeFilter,
      sortKey,
      sortDir,
    },
    handlers: {
      setProductFilter,
      setBrandFilter,
      setAssigneeFilter,
      setSortKey,
      setSortDir,
      handleRowClick,
      handleReset,
    },
  } = useInventoryManagement();

  return (
    <div className="p-0 inv-page">
      <List
        title="在庫管理"
        headerCells={buildInventoryHeaders(
          productOptions,
          brandOptions,
          assigneeOptions,
          {
            productFilter,
            brandFilter,
            assigneeFilter,
            setProductFilter,
            setBrandFilter,
            setAssigneeFilter,
            sortKey,
            sortDir,
            setSortKey,
            setSortDir,
          },
        )}
        showCreateButton={false}
        showResetButton
        onReset={handleReset}
      >
        {rows.map((row) => (
          <tr
            key={row.id}
            className="inv__clickable-row"
            role="button"
            tabIndex={0}
            onClick={() => handleRowClick(row)}
            onKeyDown={(e) => {
              if (e.key === "Enter" || e.key === " ") {
                e.preventDefault();
                handleRowClick(row);
              }
            }}
          >
            {/* プロダクト名 */}
            <td>{row.productName}</td>

            {/* ★ ブランド列：brandName をラベル表示 */}
            <td>
              {row.brandName ? (
                <span className="lp-brand-pill">{row.brandName}</span>
              ) : (
                "-"
              )}
            </td>

            {/* ★ 担当者列：assigneeName をそのまま表示 */}
            <td>{row.assigneeName ?? "-"}</td>

            {/* 総在庫数 */}
            <td>
              <span className="inv__total-pill">{row.totalQuantity}</span>
            </td>
          </tr>
        ))}
      </List>
    </div>
  );
}
