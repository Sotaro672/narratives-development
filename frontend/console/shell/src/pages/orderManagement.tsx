// frontend/console/shell/src/pages/orderManagement.tsx  
  
import List from "../layout/List/List";  
import "../styles/order.css";  
  
import { safeDateTimeLabelJa } from "../shared/util/dateJa";  
import { useOrderManagement } from "../features/order/presentation/hooks/useOrderManagement";  
import type { OrderItemInventoryRowDTO } from "../features/order/infrastructure/repository";  
  
function getOrderStatus(order: OrderItemInventoryRowDTO): string {  
  if (order.isCanceled) {  
    return "キャンセル";  
  }  
  
  if (order.transferred) {  
    return "移譲済";  
  }  
  
  if (order.paid) {  
    return "支払済";  
  }  
  
  if (order.isDispatched) {  
    return "発送済";  
  }  
  
  return "未発送";  
}  
  
export default function OrderManagementPage() {  
  const { rows, headers, errorMsg, isResetting, goDetail, reset } = useOrderManagement();  
  
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
          rows.map((order) => (  
            <tr  
              key={`${order.orderId}__${order.inventoryId}__${order.listReadableId ?? ""}`}  
              onClick={() => goDetail(order.orderId)}  
              onKeyDown={(event) => {  
                if (event.key !== "Enter" && event.key !== " ") return;  
  
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
              <td>{order.avatarName || "-"}</td>  
              <td>{safeDateTimeLabelJa(order.createdAt, "-")}</td>  
              <td>{getOrderStatus(order)}</td>  
            </tr>  
          ))  
        )}  
      </List>  
    </div>  
  );  
}