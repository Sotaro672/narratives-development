// frontend/console/shell/src/pages/listManagement.tsx 
import List from "../layout/List/List"; 
import "../styles/list.css"; 
 
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
            {/* ✅ 左から：出品ID、プロダクト名、トークン名、累計売上、注文数、担当者、ステータス、作成日 */} 
            <td>{l.readableId || l.id}</td> 
            <td>{l.productName}</td> 
            <td>{l.tokenName}</td> 
            <td>¥{l.totalSalesAmount.toLocaleString()}</td> 
            <td>{l.totalOrderCount}</td> 
            <td>{l.assigneeName}</td> 
            <td> 
              <span className={l.statusBadgeClass}>{l.statusBadgeText}</span> 
            </td> 
            <td>{l.createdAt}</td> 
          </tr> 
        ))} 
      </List> 
    </div> 
  ); 
}