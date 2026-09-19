// frontend/console/shell/src/pages/listManagement.tsx

import type { KeyboardEvent } from "react";

import { useListManagement } from "../features/list/presentation/hook/useListManagement";
import List from "../layout/List/List";
import { Badge } from "../shared/ui/badge";
import { TableCell, TableRow } from "../shared/ui/table";

export default function ListManagementPage() {
  const { vm, handlers, isResetting } = useListManagement();

  return (
    <div className="p-0">
      <List
        title={vm.title}
        headerCells={vm.headers}
        showResetButton
        isResetting={isResetting}
        onReset={handlers.onReset}
      >
        {vm.rows.map((l) => (
          <TableRow
            key={l.id}
            role="button"
            tabIndex={0}
            className="cursor-pointer"
            onClick={() => handlers.onRowClick(l.id)}
            onKeyDown={(e: KeyboardEvent<HTMLTableRowElement>) =>
              handlers.onRowKeyDown(e, l.id)
            }
          >
            <TableCell>{l.readableId || l.id}</TableCell>
            <TableCell>{l.productName}</TableCell>
            <TableCell>{l.tokenName}</TableCell>
            <TableCell>¥{l.totalSalesAmount.toLocaleString()}</TableCell>
            <TableCell>{l.totalOrderCount}</TableCell>
            <TableCell>{l.assigneeName}</TableCell>
            <TableCell>
              <Badge variant={l.status === "listing" ? "active" : "danger"}>
                {l.statusBadgeText}
              </Badge>
            </TableCell>
            <TableCell>{l.createdAt}</TableCell>
          </TableRow>
        ))}
      </List>
    </div>
  );
}