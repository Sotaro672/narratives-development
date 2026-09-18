// frontend/console/shell/src/pages/listManagement.tsx

import List from "../layout/List/List";
import { Badge } from "../shared/ui/badge";

import { useListManagement } from "../features/list/presentation/hook/useListManagement";

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
          <tr
            key={l.id}
            role="button"
            tabIndex={0}
            className="cursor-pointer"
            onClick={() => handlers.onRowClick(l.id)}
            onKeyDown={(e) => handlers.onRowKeyDown(e, l.id)}
          >
            <td>{l.readableId || l.id}</td>
            <td>{l.productName}</td>
            <td>{l.tokenName}</td>
            <td>¥{l.totalSalesAmount.toLocaleString()}</td>
            <td>{l.totalOrderCount}</td>
            <td>{l.assigneeName}</td>
            <td>
              <Badge variant={l.status === "listing" ? "active" : "danger"}>
                {l.statusBadgeText}
              </Badge>
            </td>
            <td>{l.createdAt}</td>
          </tr>
        ))}
      </List>
    </div>
  );
}