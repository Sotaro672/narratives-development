// frontend/console/permission/src/presentation/pages/permissionList.tsx

import List from "../../../../shell/src/layout/List/List";
import { usePermissionList } from "../hook/usePermissionList";

export default function PermissionList() {
  const { headers, filteredRows, goDetail, handleReset } = usePermissionList();

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
          <tr
            key={p.id}
            role="button"
            tabIndex={0}
            className="cursor-pointer hover:bg-slate-50 transition-colors"
            onClick={() => goDetail(p.id)}
            onKeyDown={(e) => {
              if (e.key === "Enter" || e.key === " ") {
                e.preventDefault();
                goDetail(p.id);
              }
            }}
          >
            <td>{p.name}</td>
            <td>
              <span className="lp-brand-pill">{p.category}</span>
            </td>
            <td>{p.description}</td>
          </tr>
        ))}
      </List>
    </div>
  );
}
