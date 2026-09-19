// frontend/console/shell/src/pages/inventoryManagement.tsx

import type { KeyboardEvent } from "react";

import { buildInventoryHeaders } from "../features/inventory/application/inventoryManagementService";
import { useInventoryManagement } from "../features/inventory/presentation/hook/useInventoryManagement";
import List from "../layout/List/List";
import { TableCell, TableRow } from "../shared/ui/table";

import "../styles/inventory.css";

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
    isResetting,
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
        isResetting={isResetting}
        onReset={handleReset}
      >
        {rows.map((row) => (
          <TableRow
            key={row.id}
            className="inv__clickable-row"
            role="button"
            tabIndex={0}
            onClick={() => handleRowClick(row)}
            onKeyDown={(e: KeyboardEvent<HTMLTableRowElement>) => {
              if (e.key === "Enter" || e.key === " ") {
                e.preventDefault();
                handleRowClick(row);
              }
            }}
          >
            <TableCell>{row.productName}</TableCell>
            <TableCell>{row.tokenName || "-"}</TableCell>
            <TableCell>{row.shippingAddressName || "-"}</TableCell>
            <TableCell>
              <span className="inv__total-pill">{row.availableStock}</span>
            </TableCell>
            <TableCell>
              <span className="inv__total-pill">{row.reservedCount}</span>
            </TableCell>
          </TableRow>
        ))}
      </List>
    </div>
  );
}