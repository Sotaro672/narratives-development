// frontend/console/inventory/src/presentation/pages/inventoryManagement.tsx

import React from "react";
import List from "../../../../shell/src/layout/List/List";
import "../styles/inventory.css";

import { useInventoryManagement } from "../hook/useInventoryManagement";
import { buildInventoryHeaders } from "../../application/inventoryManagementService";

/** 在庫管理ページ（スタイル＋レイアウト中心） */
export default function InventoryManagementPage() {
  const {
    rows,
    options: { productOptions, tokenOptions },
    state: { productFilter, tokenFilter, sortKey, sortDir },
    handlers: {
      setProductFilter,
      setTokenFilter,
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
        headerCells={buildInventoryHeaders(productOptions, tokenOptions, {
          productFilter,
          tokenFilter,
          setProductFilter,
          setTokenFilter,
          sortKey,
          sortDir,
          setSortKey,
          setSortDir,
        })}
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

            {/* トークン名 */}
            <td>{row.tokenName || "-"}</td>

            {/* 型番 */}
            <td>{row.modelNumber || "-"}</td>

            {/* 在庫数 */}
            <td>
              <span className="inv__total-pill">{row.stock}</span>
            </td>
          </tr>
        ))}
      </List>
    </div>
  );
}
