// frontend/console/shell/src/pages/orderManagement.tsx

import type { KeyboardEvent } from "react";

import { getOrderManagementStatus } from "../features/order/application/orderManagementFilter";
import { useOrderManagement } from "../features/order/presentation/hooks/useOrderManagement";
import List from "../layout/List/List";
import { Badge, type BadgeVariant } from "../shared/ui/badge";
import { TableCell, TableRow } from "../shared/ui/table";
import { safeDateTimeLabelJa } from "../shared/util/dateJa";

function getOrderStatusBadgeVariant(status: string): BadgeVariant {
  switch (status) {
    case "発送済":
      return "success";
    case "移譲済":
      return "active";
    case "返品対応中":
      return "warning";
    case "返品済":
      return "secondary";
    case "キャンセル":
      return "danger";
    case "未発送":
    default:
      return "default";
  }
}

export default function OrderManagementPage() {
  const {
    rows,
    headers,
    errorMsg,
    isResetting,
    goDetail,
    reset,
  } = useOrderManagement();

  return (
    <div className="p-0">
      <List
        title="注文管理"
        headerCells={headers}
        showCreateButton={false}
        showResetButton
        isResetting={isResetting}
        onReset={reset}
      >
        {errorMsg ? (
          <TableRow>
            <TableCell colSpan={headers.length}>
              {errorMsg}
            </TableCell>
          </TableRow>
        ) : (
          rows.map((order) => {
            const status = getOrderManagementStatus(order);

            return (
              <TableRow
                key={`${order.orderId}__${order.inventoryId}__${order.listReadableId ?? ""}`}
                onClick={() => goDetail(order.orderId)}
                onKeyDown={(event: KeyboardEvent<HTMLTableRowElement>) => {
                  if (event.key !== "Enter" && event.key !== " ") {
                    return;
                  }

                  event.preventDefault();
                  goDetail(order.orderId);
                }}
                className="is-rowlink cursor-pointer"
                tabIndex={0}
                role="button"
              >
                <TableCell>
                  <span className="text-blue-600 hover:underline">
                    {order.orderId}
                  </span>
                </TableCell>
                <TableCell>{order.listReadableId || "-"}</TableCell>
                <TableCell>{order.productName || "-"}</TableCell>
                <TableCell>{order.tokenName || "-"}</TableCell>
                <TableCell>{order.userName || "-"}</TableCell>
                <TableCell>
                  {safeDateTimeLabelJa(order.createdAt, "-")}
                </TableCell>
                <TableCell>
                  <Badge variant={getOrderStatusBadgeVariant(status)}>
                    {status}
                  </Badge>
                </TableCell>
              </TableRow>
            );
          })
        )}
      </List>
    </div>
  );
}