// frontend/console/shell/src/pages/orderManagement.tsx

import List from "../layout/List/List";

import { getOrderManagementStatus } from "../features/order/application/orderManagementFilter";
import { useOrderManagement } from "../features/order/presentation/hooks/useOrderManagement";
import { Badge, type BadgeVariant } from "../shared/ui/badge";
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
          <tr>
            <td colSpan={headers.length} style={{ padding: 16 }}>
              {errorMsg}
            </td>
          </tr>
        ) : (
          rows.map((order) => {
            const status = getOrderManagementStatus(order);

            return (
              <tr
                key={`${order.orderId}__${order.inventoryId}__${order.listReadableId ?? ""}`}
                onClick={() => goDetail(order.orderId)}
                onKeyDown={(event) => {
                  if (event.key !== "Enter" && event.key !== " ") {
                    return;
                  }

                  event.preventDefault();
                  goDetail(order.orderId);
                }}
                className="is-rowlink cursor-pointer hover:bg-slate-50 transition-colors"
                tabIndex={0}
                role="button"
              >
                <td>
                  <span className="text-blue-600 hover:underline">
                    {order.orderId}
                  </span>
                </td>
                <td>{order.listReadableId || "-"}</td>
                <td>{order.productName || "-"}</td>
                <td>{order.tokenName || "-"}</td>
                <td>{order.userName || "-"}</td>
                <td>{safeDateTimeLabelJa(order.createdAt, "-")}</td>
                <td>
                  <Badge variant={getOrderStatusBadgeVariant(status)}>
                    {status}
                  </Badge>
                </td>
              </tr>
            );
          })
        )}
      </List>
    </div>
  );
}