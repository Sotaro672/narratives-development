// frontend/list/src/presentation/pages/listManagement.tsx

import React from "react";
import List from "../../../../shell/src/layout/List/List";
import "../styles/list.css";

import { useListManagement } from "../hook/useListManagement";

export default function ListManagementPage() {
  const { vm, handlers } = useListManagement();

  return (
    <div className="p-0">
      <List
        title={vm.title}
        headerCells={vm.headers}
        showResetButton
        onReset={handlers.onReset}
      >
        {vm.rows.map((l) => (
          <tr
            key={l.id}
            role="button"
            tabIndex={0}
            className="cursor-pointer"
            onClick={() => handlers.onRowClick(l.id)}
            onKeyDown={(e) => handlers.onRowKeyDown(e, l.id)}
          >
            <td>{l.id}</td>
            <td>{l.productName}</td>
            <td>
              <span className="lp-brand-pill">{l.brandName}</span>
            </td>
            <td>
              <span className="lp-brand-pill">{l.tokenName}</span>
            </td>
            <td>
              <span className="list-stock-pill">{l.stock}</span>
            </td>
            <td>{l.assigneeName}</td>
            <td>
              <span className={l.statusBadgeClass}>{l.statusBadgeText}</span>
            </td>
          </tr>
        ))}
      </List>
    </div>
  );
}
