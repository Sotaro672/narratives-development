// frontend/list/src/presentation/pages/listManagement.tsx
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
            {/* ✅ 左から：タイトル、プロダクト名、トークン名、担当者、ステータス、作成日 */}
            <td>{l.title}</td>
            <td>{l.productName}</td>
            <td>{l.tokenName}</td>
            <td>{l.assigneeName}</td>
            <td>
              <span className={l.statusBadgeClass}>{l.statusBadgeText}</span>
            </td>
            <td>{(l as any).createdAt ?? ""}</td>
          </tr>
        ))}
      </List>
    </div>
  );
}
