// frontend/console/order/src/presentation/pages/orderManagement.tsx
import List from "../../../../shell/src/layout/List/List";
import "../styles/order.css";

import { safeDateLabelJa } from "../../../../shell/src/shared/util/dateJa";
import { useOrderManagement } from "../hooks/useOrderManagement";

export default function OrderManagementPage() {
  const { rows, headers, errorMsg, isResetting, goDetail, reset } =
    useOrderManagement();

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
          rows.map((o) => (
            <tr
              key={`${o.orderId}__${o.inventoryId}__${o.listId}`}
              onClick={() => goDetail(o.orderId)}
              className="is-rowlink cursor-pointer hover:bg-slate-50 transition-colors"
              tabIndex={0}
              role="button"
            >
              <td>
                <a
                  href="#"
                  onClick={(e) => {
                    e.preventDefault();
                    goDetail(o.orderId);
                  }}
                  className="text-blue-600 hover:underline"
                >
                  {o.orderId}
                </a>
              </td>

              <td>{o.listId || "-"}</td>

              <td>{o.productName || "-"}</td>
              <td>{o.tokenName || "-"}</td>

              <td>{o.avatarName || "-"}</td>

              <td>{safeDateLabelJa(o.createdAt, "-")}</td>
              <td>{o.transferred ? "移譲済" : "未移譲"}</td>
            </tr>
          ))
        )}
      </List>
    </div>
  );
}