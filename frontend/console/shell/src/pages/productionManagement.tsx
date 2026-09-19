// frontend/console/shell/src/pages/productionManagement.tsx

import { useProductionManagement } from "../features/production/presentation/hook/useProductionManagement";
import List from "../layout/List/List";
import { TableCell, TableRow } from "../shared/ui/table";

export default function ProductionManagement() {
  const {
    headers,
    rows,
    handleCreate,
    handleReset,
    handleRowClick,
    isResetting,
  } = useProductionManagement();

  return (
    <div className="p-0">
      <List
        title="商品生産"
        headerCells={headers}
        showCreateButton
        createLabel="生産計画を作成"
        showResetButton
        isResetting={isResetting}
        onCreate={handleCreate}
        onReset={handleReset}
      >
        {rows.map((p) => (
          <TableRow
            key={p.id}
            className="cursor-pointer"
            onClick={() => handleRowClick(p.id)}
          >
            <TableCell>{p.productName || p.productBlueprintId}</TableCell>
            <TableCell>{p.brandName || ""}</TableCell>
            <TableCell>{p.assigneeName || p.assigneeId}</TableCell>
            <TableCell>{p.printed ? "印刷済" : "印刷前"}</TableCell>
            <TableCell>{p.totalQuantity}</TableCell>
            <TableCell>{p.printedAtLabel}</TableCell>
            <TableCell>{p.createdAtLabel}</TableCell>
          </TableRow>
        ))}
      </List>
    </div>
  );
}