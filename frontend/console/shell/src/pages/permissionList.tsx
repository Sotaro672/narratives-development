// frontend/console/shell/src/pages/permissionList.tsx

import type { KeyboardEvent } from "react";

import { usePermissionList } from "../features/permission/presentation/hook/usePermissionList";
import List from "../layout/List/List";
import { TableCell, TableRow } from "../shared/ui/table";

export default function PermissionList() {
  const {
    headers,
    filteredRows,
    goDetail,
    handleReset,
  } = usePermissionList();

  return (
    <div className="p-0">
      <List
        title="権限管理"
        headerCells={headers}
        showCreateButton={false}
        showResetButton
        onReset={handleReset}
      >
        {filteredRows.map((p) => (
          <TableRow
            key={p.id}
            role="button"
            tabIndex={0}
            className="cursor-pointer"
            onClick={() => goDetail(p.id)}
            onKeyDown={(e: KeyboardEvent<HTMLTableRowElement>) => {
              if (e.key === "Enter" || e.key === " ") {
                e.preventDefault();
                goDetail(p.id);
              }
            }}
          >
            <TableCell>{p.name}</TableCell>
            <TableCell>
              <span className="lp-brand-pill">{p.category}</span>
            </TableCell>
            <TableCell>{p.description}</TableCell>
          </TableRow>
        ))}
      </List>
    </div>
  );
}